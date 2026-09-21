package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/ledger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateContributionAccount(ctx context.Context, playerID string) (contribution.Account, error) {
	if s == nil || s.pool == nil || playerID == "" || len(playerID) > 128 {
		return contribution.Account{}, contribution.ErrInvalidAccount
	}
	now := time.Now().UTC()
	value := contribution.Account{PlayerID: playerID, Revision: 1, CreatedAt: now, UpdatedAt: now}
	_, err := s.pool.Exec(ctx, `INSERT INTO contribution_accounts(player_id,balance,revision,created_at,updated_at) VALUES($1,0,1,$2,$2)`, playerID, now)
	return value, classifyContributionError(err)
}

func (s *Store) LoadContributionAccount(ctx context.Context, playerID string) (contribution.Account, error) {
	var value contribution.Account
	err := s.pool.QueryRow(ctx, `SELECT player_id,balance,revision,created_at,updated_at FROM contribution_accounts WHERE player_id=$1`, playerID).Scan(&value.PlayerID, &value.Balance, &value.Revision, &value.CreatedAt, &value.UpdatedAt)
	return value, classifyContributionError(err)
}

func (s *Store) PostContributionSystemSpend(ctx context.Context, request contribution.SpendRequest) (contribution.PostingResult, error) {
	if s == nil || s.pool == nil {
		return contribution.PostingResult{}, contribution.ErrUnavailable
	}
	amount, err := (contribution.EligibilityPolicy{}).Evaluate(request.Source, request.EligibleSpend, request.RuleVersion)
	if err != nil {
		return contribution.PostingResult{}, err
	}
	if amount == 0 {
		return contribution.PostingResult{}, contribution.ErrIneligibleSource
	}
	for attempt := 0; attempt < 24; attempt++ {
		tx, beginErr := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
		if beginErr != nil {
			return contribution.PostingResult{}, classifyContributionError(beginErr)
		}
		result, postErr := s.postContributionSystemSpendTx(ctx, tx, request)
		if postErr == nil {
			postErr = s.checkContributionFailure(contribution.FailureBeforeCommit)
		}
		if postErr == nil {
			postErr = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		if postErr == nil {
			return result, nil
		}
		if !retryContributionTransaction(postErr) {
			return contribution.PostingResult{}, classifyContributionError(postErr)
		}
		if err = ctx.Err(); err != nil {
			return contribution.PostingResult{}, err
		}
		// Bounded backoff reduces repeated serializable conflicts on a hot account.
		wait := time.Duration((attempt%8)+1) * time.Millisecond
		select {
		case <-ctx.Done():
			return contribution.PostingResult{}, ctx.Err()
		case <-time.After(wait):
		}
	}
	return contribution.PostingResult{}, contribution.ErrRetryExhausted
}

func (s *Store) postContributionSystemSpendTx(ctx context.Context, tx pgx.Tx, request contribution.SpendRequest) (contribution.PostingResult, error) {
	key := string(request.Source) + ":" + request.SourceID + ":" + request.RuleVersion
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "contribution:"+key); err != nil {
		return contribution.PostingResult{}, err
	}
	existing, err := loadContributionBySourceTx(ctx, tx, request)
	if err == nil {
		if existing.PlayerID != request.PlayerID || existing.PlayerFBAccountID != request.PlayerFBAccountID || existing.SystemFBAccountID != request.SystemFBAccountID || existing.EligibleSpend != request.EligibleSpend || existing.Amount != request.EligibleSpend {
			return contribution.PostingResult{}, contribution.ErrSourceConflict
		}
		posted, loadErr := loadLedgerTransactionTx(ctx, tx, "transaction_id=$1", existing.FBTransactionID)
		if loadErr != nil {
			return contribution.PostingResult{}, loadErr
		}
		return contribution.PostingResult{Entry: existing, FBTransaction: posted}, nil
	}
	if !errors.Is(err, contribution.ErrNotFound) {
		return contribution.PostingResult{}, err
	}
	playerFB, err := loadLedgerAccountRow(tx.QueryRow(ctx, `SELECT account_id,owner_id,owner_type,currency,balance,revision,created_at,updated_at FROM fb_ledger_accounts WHERE account_id=$1`, request.PlayerFBAccountID))
	if err != nil {
		return contribution.PostingResult{}, err
	}
	systemFB, err := loadLedgerAccountRow(tx.QueryRow(ctx, `SELECT account_id,owner_id,owner_type,currency,balance,revision,created_at,updated_at FROM fb_ledger_accounts WHERE account_id=$1`, request.SystemFBAccountID))
	if err != nil {
		return contribution.PostingResult{}, err
	}
	if playerFB.OwnerType != ledger.OwnerPlayer || playerFB.OwnerID != request.PlayerID || systemFB.OwnerType != ledger.OwnerSystem || playerFB.Currency != ledger.CurrencyFB || systemFB.Currency != ledger.CurrencyFB || playerFB.ID == systemFB.ID {
		return contribution.PostingResult{}, contribution.ErrInvalidSource
	}
	var accountID string
	if err = tx.QueryRow(ctx, `SELECT player_id FROM contribution_accounts WHERE player_id=$1`, request.PlayerID).Scan(&accountID); err != nil {
		return contribution.PostingResult{}, classifyContributionError(err)
	}
	if err = s.checkContributionFailure(contribution.FailureBeforeFBDebit); err != nil {
		return contribution.PostingResult{}, err
	}
	now := time.Now().UTC()
	draft := ledger.PostDraft{
		ID: newContributionID("fb-transaction"), Type: ledger.TransactionSystemSpend,
		Reference:           ledger.LedgerReference{Type: "CONTRIBUTION_SYSTEM_SPEND", ID: key},
		SpendClassification: ledger.SpendEligible, Reason: "G13 eligible internal system spend", CreatedAt: now,
		Entries: []ledger.EntryDraft{
			{ID: newContributionID("fb-entry"), AccountID: request.PlayerFBAccountID, Amount: -request.EligibleSpend},
			{ID: newContributionID("fb-entry"), AccountID: request.SystemFBAccountID, Amount: request.EligibleSpend},
		},
	}
	posted, err := s.postLedgerTransactionTx(ctx, tx, draft)
	if err != nil {
		return contribution.PostingResult{}, err
	}
	if posted.ID != draft.ID {
		return contribution.PostingResult{}, contribution.ErrSourceConflict
	}
	if err = s.checkContributionFailure(contribution.FailureAfterFBDebit); err != nil {
		return contribution.PostingResult{}, err
	}
	var account contribution.Account
	err = tx.QueryRow(ctx, `SELECT player_id,balance,revision,created_at,updated_at FROM contribution_accounts WHERE player_id=$1 FOR UPDATE`, request.PlayerID).Scan(&account.PlayerID, &account.Balance, &account.Revision, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return contribution.PostingResult{}, classifyContributionError(err)
	}
	after, err := ledger.AddAmount(account.Balance, request.EligibleSpend)
	if err != nil {
		return contribution.PostingResult{}, contribution.ErrOverflow
	}
	if err = s.checkContributionFailure(contribution.FailureBeforeEntry); err != nil {
		return contribution.PostingResult{}, err
	}
	entry := contribution.Entry{
		ID: newContributionID("contribution-entry"), PlayerID: request.PlayerID,
		PlayerFBAccountID: request.PlayerFBAccountID, SystemFBAccountID: request.SystemFBAccountID,
		Source: request.Source, SourceID: request.SourceID, EligibleSpend: request.EligibleSpend,
		Amount: request.EligibleSpend, BalanceBefore: account.Balance, BalanceAfter: after,
		RuleVersion: request.RuleVersion, FBTransactionID: posted.ID, CreatedAt: now,
	}
	_, err = tx.Exec(ctx, `INSERT INTO contribution_entries(entry_id,player_id,player_fb_account_id,system_fb_account_id,source,source_id,eligible_spend,amount,balance_before,balance_after,rule_version,fb_transaction_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, entry.ID, entry.PlayerID, entry.PlayerFBAccountID, entry.SystemFBAccountID, entry.Source, entry.SourceID, entry.EligibleSpend, entry.Amount, entry.BalanceBefore, entry.BalanceAfter, entry.RuleVersion, entry.FBTransactionID, entry.CreatedAt)
	if err != nil {
		return contribution.PostingResult{}, classifyContributionError(err)
	}
	if err = s.checkContributionFailure(contribution.FailureAfterEntry); err != nil {
		return contribution.PostingResult{}, err
	}
	var revision int64
	err = tx.QueryRow(ctx, `UPDATE contribution_accounts SET balance=$1,revision=revision+1,updated_at=$2 WHERE player_id=$3 AND revision=$4 RETURNING revision`, after, now, account.PlayerID, account.Revision).Scan(&revision)
	if err != nil {
		return contribution.PostingResult{}, classifyContributionError(err)
	}
	if err = s.checkContributionFailure(contribution.FailureAfterAccountUpdate); err != nil {
		return contribution.PostingResult{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO contribution_audit_events(entry_id,player_id,source,source_id,rule_version,amount,fb_transaction_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, entry.ID, entry.PlayerID, entry.Source, entry.SourceID, entry.RuleVersion, entry.Amount, entry.FBTransactionID, entry.CreatedAt)
	if err != nil {
		return contribution.PostingResult{}, classifyContributionError(err)
	}
	if err = s.checkContributionFailure(contribution.FailureBeforeFinalState); err != nil {
		return contribution.PostingResult{}, err
	}
	return contribution.PostingResult{Entry: entry, FBTransaction: posted}, nil
}

func loadContributionBySourceTx(ctx context.Context, tx pgx.Tx, request contribution.SpendRequest) (contribution.Entry, error) {
	var value contribution.Entry
	err := tx.QueryRow(ctx, `SELECT entry_id,player_id,player_fb_account_id,system_fb_account_id,source,source_id,eligible_spend,amount,balance_before,balance_after,rule_version,fb_transaction_id,created_at FROM contribution_entries WHERE source=$1 AND source_id=$2 AND rule_version=$3`, request.Source, request.SourceID, request.RuleVersion).Scan(&value.ID, &value.PlayerID, &value.PlayerFBAccountID, &value.SystemFBAccountID, &value.Source, &value.SourceID, &value.EligibleSpend, &value.Amount, &value.BalanceBefore, &value.BalanceAfter, &value.RuleVersion, &value.FBTransactionID, &value.CreatedAt)
	return value, classifyContributionError(err)
}

func (s *Store) ContributionEntries(ctx context.Context, playerID string) ([]contribution.Entry, error) {
	rows, err := s.pool.Query(ctx, `SELECT entry_id,player_id,player_fb_account_id,system_fb_account_id,source,source_id,eligible_spend,amount,balance_before,balance_after,rule_version,fb_transaction_id,created_at FROM contribution_entries WHERE player_id=$1 ORDER BY balance_before,entry_id`, playerID)
	if err != nil {
		return nil, classifyContributionError(err)
	}
	defer rows.Close()
	var result []contribution.Entry
	for rows.Next() {
		var value contribution.Entry
		if err = rows.Scan(&value.ID, &value.PlayerID, &value.PlayerFBAccountID, &value.SystemFBAccountID, &value.Source, &value.SourceID, &value.EligibleSpend, &value.Amount, &value.BalanceBefore, &value.BalanceAfter, &value.RuleVersion, &value.FBTransactionID, &value.CreatedAt); err != nil {
			return nil, classifyContributionError(err)
		}
		result = append(result, value)
	}
	return result, classifyContributionError(rows.Err())
}

func (s *Store) ContributionAuditEvents(ctx context.Context, playerID string) ([]contribution.AuditEvent, error) {
	rows, err := s.pool.Query(ctx, `SELECT sequence,entry_id,player_id,source,source_id,rule_version,amount,fb_transaction_id,created_at FROM contribution_audit_events WHERE player_id=$1 ORDER BY sequence`, playerID)
	if err != nil {
		return nil, classifyContributionError(err)
	}
	defer rows.Close()
	var result []contribution.AuditEvent
	for rows.Next() {
		var value contribution.AuditEvent
		if err = rows.Scan(&value.Sequence, &value.EntryID, &value.PlayerID, &value.Source, &value.SourceID, &value.RuleVersion, &value.Amount, &value.FBTransactionID, &value.CreatedAt); err != nil {
			return nil, classifyContributionError(err)
		}
		result = append(result, value)
	}
	return result, classifyContributionError(rows.Err())
}

func (s *Store) ReconcileContribution(ctx context.Context) (contribution.ReconciliationReport, error) {
	rows, err := s.pool.Query(ctx, `SELECT a.player_id,a.balance,COALESCE(sum(e.amount),0)::bigint FROM contribution_accounts a LEFT JOIN contribution_entries e ON e.player_id=a.player_id GROUP BY a.player_id,a.balance ORDER BY a.player_id`)
	if err != nil {
		return contribution.ReconciliationReport{}, classifyContributionError(err)
	}
	defer rows.Close()
	report := contribution.ReconciliationReport{Balanced: true}
	for rows.Next() {
		var value contribution.ReconciliationMismatch
		if err = rows.Scan(&value.PlayerID, &value.Balance, &value.EntriesTotal); err != nil {
			return contribution.ReconciliationReport{}, classifyContributionError(err)
		}
		if value.Balance != value.EntriesTotal {
			report.Balanced = false
			report.Mismatches = append(report.Mismatches, value)
		}
	}
	return report, classifyContributionError(rows.Err())
}

func (s *Store) checkContributionFailure(point string) error {
	if s.contributionFailureInjector != nil {
		return s.contributionFailureInjector(point)
	}
	return nil
}

func retryContributionTransaction(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "40P01")
}

func classifyContributionError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return contribution.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return contribution.ErrSourceConflict
	}
	return err
}

func newContributionID(prefix string) string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		panic(fmt.Sprintf("secure random source unavailable: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(random[:])
}

var _ contribution.Repository = (*Store)(nil)
