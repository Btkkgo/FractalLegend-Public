package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/ledger"
	"fractallegend/game-server/internal/systemspend"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const g15SourcePrefix = "g15:"

// PostSystemSpend is the G15 internal coordinator. It reuses G13's FB +
// Contribution writer inside the same transaction that creates the immutable
// SystemSpend row. No gameplay producer or client route calls this in G15.
func (s *Store) PostSystemSpend(ctx context.Context, intent systemspend.ResolvedIntent) (systemspend.SystemSpend, error) {
	if s == nil || s.pool == nil {
		return systemspend.SystemSpend{}, systemspend.ErrUnavailable
	}
	if intent.ProducerType != systemspend.ProducerInternalTestEligible && intent.ProducerType != systemspend.ProducerInternalTestNonEligible {
		return systemspend.SystemSpend{}, systemspend.ErrDisabledProducer
	}
	if intent.FBAmount <= 0 || intent.OperationID == "" || intent.ProducerReference == "" {
		return systemspend.SystemSpend{}, systemspend.ErrInvalidIntent
	}
	if intent.Eligible {
		amount, err := (contribution.EligibilityPolicy{}).Evaluate(contribution.SourceSystemService, intent.FBAmount, intent.RuleVersion)
		if err != nil || amount != intent.ContributionAmount || intent.ProducerType != systemspend.ProducerInternalTestEligible {
			return systemspend.SystemSpend{}, systemspend.ErrUnknownRule
		}
	} else if intent.RuleVersion != "" || intent.ContributionAmount != 0 || intent.Refundable || intent.PartialRefundAllowed || intent.ProducerType != systemspend.ProducerInternalTestNonEligible {
		return systemspend.SystemSpend{}, systemspend.ErrInvalidIntent
	}
	for attempt := 0; attempt < 32; attempt++ {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
		if err != nil {
			return systemspend.SystemSpend{}, err
		}
		value, err := s.postSystemSpendTx(ctx, tx, intent)
		if err == nil && s.systemSpendFailureInjector != nil {
			err = s.systemSpendFailureInjector("before_commit")
		}
		if err == nil {
			err = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		if err == nil {
			return value, nil
		}
		if !retryContributionTransaction(err) {
			return systemspend.SystemSpend{}, classifySystemSpendError(err)
		}
		select {
		case <-ctx.Done():
			return systemspend.SystemSpend{}, ctx.Err()
		case <-time.After(time.Duration((attempt%8)+1) * time.Millisecond):
		}
	}
	return systemspend.SystemSpend{}, contribution.ErrRetryExhausted
}

func (s *Store) postSystemSpendTx(ctx context.Context, tx pgx.Tx, intent systemspend.ResolvedIntent) (systemspend.SystemSpend, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "g15-operation:"+intent.OperationID); err != nil {
		return systemspend.SystemSpend{}, err
	}
	existing, err := loadSystemSpendTx(ctx, tx, intent.OperationID)
	if err == nil {
		if !sameSystemSpendIntent(existing, intent) {
			return systemspend.SystemSpend{}, systemspend.ErrConflict
		}
		if err = validateSystemSpendTx(ctx, tx, existing); err != nil {
			return systemspend.SystemSpend{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, systemspend.ErrNotFound) {
		return systemspend.SystemSpend{}, err
	}
	var conflicting bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM system_spends WHERE producer_type=$1 AND producer_reference=$2)`, intent.ProducerType, intent.ProducerReference).Scan(&conflicting); err != nil {
		return systemspend.SystemSpend{}, err
	}
	if conflicting {
		return systemspend.SystemSpend{}, systemspend.ErrReferenceConflict
	}
	value := systemspend.SystemSpend{
		ID: newContributionID("system-spend"), OperationID: intent.OperationID,
		PlayerID: intent.PlayerID, PlayerFBAccountID: intent.PlayerFBAccountID,
		SystemFBAccountID: intent.SystemFBAccountID, ProducerType: intent.ProducerType,
		ProducerReference: intent.ProducerReference, FBAmount: intent.FBAmount,
		Eligible: intent.Eligible, RuleVersion: intent.RuleVersion,
		ContributionAmount: intent.ContributionAmount, RefundStatus: systemspend.RefundNone,
		Refundable: intent.Refundable, PartialRefundAllowed: intent.PartialRefundAllowed,
		Status: "COMPLETED", Metadata: normalizeMetadata(intent.Metadata),
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond), CompletedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if intent.Eligible {
		var preexisting bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contribution_entries WHERE source='SYSTEM_SERVICE' AND source_id=$1)`, g15SourcePrefix+intent.OperationID).Scan(&preexisting); err != nil {
			return systemspend.SystemSpend{}, err
		}
		if preexisting {
			return systemspend.SystemSpend{}, systemspend.ErrReconciliation
		}
		posted, postErr := s.postContributionSystemSpendTx(ctx, tx, contribution.SpendRequest{
			PlayerID: intent.PlayerID, PlayerFBAccountID: intent.PlayerFBAccountID,
			SystemFBAccountID: intent.SystemFBAccountID, Source: contribution.SourceSystemService,
			SourceID: g15SourcePrefix + intent.OperationID, EligibleSpend: intent.FBAmount,
			RuleVersion: intent.RuleVersion,
		})
		if postErr != nil {
			return systemspend.SystemSpend{}, postErr
		}
		value.FBTransactionID = posted.FBTransaction.ID
		value.ContributionEntryID = posted.Entry.ID
	} else {
		var playerOwner string
		var playerType ledger.OwnerType
		var systemType ledger.OwnerType
		if err = tx.QueryRow(ctx, `SELECT owner_id,owner_type FROM fb_ledger_accounts WHERE account_id=$1`, intent.PlayerFBAccountID).Scan(&playerOwner, &playerType); err != nil {
			return systemspend.SystemSpend{}, classifyContributionError(err)
		}
		if err = tx.QueryRow(ctx, `SELECT owner_type FROM fb_ledger_accounts WHERE account_id=$1`, intent.SystemFBAccountID).Scan(&systemType); err != nil {
			return systemspend.SystemSpend{}, classifyContributionError(err)
		}
		if playerOwner != intent.PlayerID || playerType != ledger.OwnerPlayer || systemType != ledger.OwnerSystem || intent.PlayerFBAccountID == intent.SystemFBAccountID {
			return systemspend.SystemSpend{}, systemspend.ErrInvalidIntent
		}
		draft := ledger.PostDraft{ID: newContributionID("fb-transaction"), Type: ledger.TransactionSystemSpend,
			Reference:           ledger.LedgerReference{Type: "SYSTEM_SPEND_NON_ELIGIBLE", ID: g15SourcePrefix + intent.OperationID},
			SpendClassification: ledger.SpendNonEligible, Reason: "G15 internal non-eligible system spend",
			CreatedAt: value.CreatedAt,
			Entries: []ledger.EntryDraft{
				{ID: newContributionID("fb-entry"), AccountID: intent.PlayerFBAccountID, Amount: -intent.FBAmount},
				{ID: newContributionID("fb-entry"), AccountID: intent.SystemFBAccountID, Amount: intent.FBAmount},
			},
		}
		posted, postErr := s.postLedgerTransactionTx(ctx, tx, draft)
		if postErr != nil {
			return systemspend.SystemSpend{}, postErr
		}
		if posted.ID != draft.ID {
			return systemspend.SystemSpend{}, systemspend.ErrReconciliation
		}
		value.FBTransactionID = posted.ID
	}
	if s.systemSpendFailureInjector != nil {
		if err = s.systemSpendFailureInjector("after_economic_effect"); err != nil {
			return systemspend.SystemSpend{}, err
		}
	}
	metadata, err := json.Marshal(value.Metadata)
	if err != nil || len(metadata) > 2048 {
		return systemspend.SystemSpend{}, systemspend.ErrInvalidIntent
	}
	var contributionID any
	if value.Eligible {
		contributionID = value.ContributionEntryID
	}
	_, err = tx.Exec(ctx, `INSERT INTO system_spends(spend_id,operation_id,player_id,player_fb_account_id,system_fb_account_id,producer_type,producer_reference,fb_amount,eligible,rule_version,contribution_amount,fb_transaction_id,contribution_entry_id,refundable,partial_refund_allowed,status,metadata,created_at,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`, value.ID, value.OperationID, value.PlayerID, value.PlayerFBAccountID, value.SystemFBAccountID, value.ProducerType, value.ProducerReference, value.FBAmount, value.Eligible, value.RuleVersion, value.ContributionAmount, value.FBTransactionID, contributionID, value.Refundable, value.PartialRefundAllowed, value.Status, metadata, value.CreatedAt, value.CompletedAt)
	if err != nil {
		return systemspend.SystemSpend{}, err
	}
	if s.systemSpendFailureInjector != nil {
		if err = s.systemSpendFailureInjector("after_spend_record"); err != nil {
			return systemspend.SystemSpend{}, err
		}
	}
	// Return PostgreSQL's stored value, as the replay and snapshot paths do.
	return loadSystemSpendTx(ctx, tx, intent.OperationID)
}

func sameSystemSpendIntent(value systemspend.SystemSpend, intent systemspend.ResolvedIntent) bool {
	return value.PlayerID == intent.PlayerID && value.PlayerFBAccountID == intent.PlayerFBAccountID &&
		value.SystemFBAccountID == intent.SystemFBAccountID && value.ProducerType == intent.ProducerType &&
		value.ProducerReference == intent.ProducerReference && value.FBAmount == intent.FBAmount &&
		value.Eligible == intent.Eligible && value.RuleVersion == intent.RuleVersion &&
		value.ContributionAmount == intent.ContributionAmount && value.Refundable == intent.Refundable &&
		value.PartialRefundAllowed == intent.PartialRefundAllowed && reflect.DeepEqual(normalizeMetadata(value.Metadata), normalizeMetadata(intent.Metadata))
}

func normalizeMetadata(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func loadSystemSpendTx(ctx context.Context, tx pgx.Tx, operationID string) (systemspend.SystemSpend, error) {
	var value systemspend.SystemSpend
	var metadata []byte
	var contributionID *string
	err := tx.QueryRow(ctx, `SELECT s.spend_id,s.operation_id,s.player_id,s.player_fb_account_id,s.system_fb_account_id,s.producer_type,s.producer_reference,s.fb_amount,s.eligible,s.rule_version,s.contribution_amount,s.fb_transaction_id,s.contribution_entry_id,s.refundable,s.partial_refund_allowed,s.status,s.metadata,s.created_at,s.completed_at,
        COALESCE((SELECT sum(amount) FROM contribution_compensations WHERE original_fb_transaction_id=s.fb_transaction_id),0)::bigint
        FROM system_spends s WHERE s.operation_id=$1`, operationID).Scan(
		&value.ID, &value.OperationID, &value.PlayerID, &value.PlayerFBAccountID,
		&value.SystemFBAccountID, &value.ProducerType, &value.ProducerReference,
		&value.FBAmount, &value.Eligible, &value.RuleVersion, &value.ContributionAmount,
		&value.FBTransactionID, &contributionID, &value.Refundable,
		&value.PartialRefundAllowed, &value.Status, &metadata, &value.CreatedAt,
		&value.CompletedAt, &value.RefundedAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		return systemspend.SystemSpend{}, systemspend.ErrNotFound
	}
	if err != nil {
		return systemspend.SystemSpend{}, err
	}
	value.CreatedAt = value.CreatedAt.UTC()
	value.CompletedAt = value.CompletedAt.UTC()
	if contributionID != nil {
		value.ContributionEntryID = *contributionID
	}
	if err = json.Unmarshal(metadata, &value.Metadata); err != nil {
		return systemspend.SystemSpend{}, systemspend.ErrReconciliation
	}
	value.RefundStatus = systemspend.RefundNone
	if value.RefundedAmount > 0 {
		value.RefundStatus = systemspend.RefundPartial
		if value.RefundedAmount == value.FBAmount {
			value.RefundStatus = systemspend.RefundFull
		}
	}
	return value, nil
}

func (s *Store) LoadSystemSpend(ctx context.Context, operationID string) (systemspend.SystemSpend, error) {
	if s == nil || s.pool == nil {
		return systemspend.SystemSpend{}, systemspend.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return systemspend.SystemSpend{}, err
	}
	defer tx.Rollback(ctx)
	value, err := loadSystemSpendTx(ctx, tx, operationID)
	if err != nil {
		return systemspend.SystemSpend{}, err
	}
	if err = validateSystemSpendTx(ctx, tx, value); err != nil {
		return systemspend.SystemSpend{}, err
	}
	return value, tx.Commit(ctx)
}

func classifySystemSpendError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return systemspend.ErrReferenceConflict
	}
	return err
}

var _ systemspend.Repository = (*Store)(nil)
