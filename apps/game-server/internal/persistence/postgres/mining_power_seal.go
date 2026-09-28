package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
)

// SealMiningPowerTEST is an internal-only prerequisite operation. No runtime
// route calls it; the database name check also prevents use on a production
// database. It does not allocate reward or touch the economic pool.
func (s *Store) SealMiningPowerTEST(ctx context.Context, instanceID string) (miningpower.SettlementInputSeal, error) {
	var empty miningpower.SettlementInputSeal
	if s == nil || s.pool == nil || !miningpower.ValidID(instanceID, 128) {
		return empty, miningpower.ErrUnavailable
	}
	var dbName string
	if err := s.pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName); err != nil {
		return empty, err
	}
	if !strings.HasPrefix(dbName, "fractal_g9_test_") {
		return empty, miningpower.ErrUnavailable
	}
	block, err := s.ReadBlock(ctx, instanceID)
	if err != nil {
		return empty, err
	}
	if block.BlockInstanceID != instanceID || block.Status != "FINALIZED" {
		return empty, miningpower.ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return empty, err
	}
	defer tx.Rollback(context.Background())
	// Existing immutable facts require their original admission state. Do not
	// silently recreate a missing gate behind accepted history.
	var orphanHistory bool
	if err = tx.QueryRow(ctx, `SELECT NOT EXISTS(SELECT 1 FROM mining_power_acceptance_states WHERE block_instance_id=$1)
		AND EXISTS(SELECT 1 FROM mining_power_activities WHERE block_instance_id=$1)`, instanceID).Scan(&orphanHistory); err != nil {
		return empty, err
	}
	if orphanHistory {
		return empty, miningpower.ErrInvariant
	}
	// Provisioning and locking use different commands. Concurrent first
	// acceptance/Seal insertions contend on this unique instance identity.
	if _, err = tx.Exec(ctx, `INSERT INTO mining_power_acceptance_states(block_instance_id,state,revision)
		VALUES($1,'OPEN',1) ON CONFLICT(block_instance_id) DO NOTHING`, instanceID); err != nil {
		return empty, err
	}
	var state string
	if err = tx.QueryRow(ctx, `SELECT state FROM mining_power_acceptance_states WHERE block_instance_id=$1 FOR UPDATE`, instanceID).Scan(&state); err != nil {
		return empty, err
	}
	if state == "SEALED" {
		prior, e := loadMiningPowerSealTx(ctx, tx, instanceID)
		if e != nil {
			return empty, e
		}
		return prior, tx.Commit(ctx)
	}
	if state != "OPEN" {
		return empty, miningpower.ErrInvariant
	}
	var sourceID, receiptID, bindingRule, bindingDigest, displayBlockID string
	var amount int64
	err = tx.QueryRow(ctx, `SELECT reservation_source_id,reservation_receipt_id,g18_rule_version,evidence_digest,display_block_id,reservation_amount
		FROM mining_reservation_instance_bindings WHERE block_instance_id=$1`, instanceID).
		Scan(&sourceID, &receiptID, &bindingRule, &bindingDigest, &displayBlockID, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return empty, miningpower.ErrInvalidInput // Historical unbound reservation.
	}
	if err != nil {
		return empty, err
	}
	if displayBlockID != block.ID || bindingRule != block.RuleVersion || amount <= 0 {
		return empty, miningpower.ErrInvariant
	}
	var actualSource, actualReceipt, status string
	var actualR int64
	err = tx.QueryRow(ctx, `SELECT e.entry_id,r.receipt_id,b.status,q.amount
		FROM mining_block_entries e JOIN mining_block_receipts r ON r.entry_id=e.entry_id
		JOIN mining_blocks b ON b.block_id=e.block_id JOIN mining_block_reservations q ON q.block_id=b.block_id
		WHERE e.entry_id=$1 AND e.action='OPEN' AND b.block_instance_id=$2`, sourceID, instanceID).
		Scan(&actualSource, &actualReceipt, &status, &actualR)
	if err != nil || actualSource != sourceID || actualReceipt != receiptID || status != "FINALIZED" || actualR != amount {
		return empty, miningpower.ErrInvariant
	}
	sealID := newContributionID("mining-power-seal")
	sealedAt := miningpower.CanonicalTime(time.Now().UTC().Truncate(time.Microsecond))
	if _, err = tx.Exec(ctx, `UPDATE mining_power_acceptance_states SET state='SEALED',seal_id=$2,
		seal_rule_version=$3,sealed_at=$4,revision=2 WHERE block_instance_id=$1 AND state='OPEN'`,
		instanceID, sealID, miningpower.SealRuleVersion, sealedAt); err != nil {
		return empty, err
	}
	// A new READ COMMITTED statement after the exclusive lock sees every
	// admitted acceptance that committed while Seal was waiting.
	if err := validateMiningPowerSealRulesTx(ctx, tx, instanceID, miningpower.DevelopmentRuleVersion); err != nil {
		return empty, err
	}
	snapshot, err := snapshotMiningPowerTx(ctx, tx, instanceID, miningpower.DevelopmentRuleVersion)
	if err != nil {
		return empty, err
	}
	report, err := miningpower.ReconcileSnapshot(snapshot)
	if err != nil || report.Status != "PASS" {
		return empty, miningpower.ErrInvariant
	}
	seal, err := buildMiningPowerSealTx(ctx, tx, sealID, sealedAt, block, sourceID, bindingDigest, snapshot)
	if err != nil {
		return empty, err
	}
	raw, err := miningpower.CanonicalSealBytes(seal)
	if err != nil {
		return empty, err
	}
	if len(raw) > settlementMaxSealBytes || seal.ActivityCount > settlementMaxActivities {
		return empty, miningpower.ErrInvalidInput
	}
	seal.CanonicalDigest = miningpower.DigestSealBytes(raw)
	_, err = tx.Exec(ctx, `INSERT INTO mining_power_input_seals(seal_id,block_instance_id,seal_rule_version,schema_version,
		activity_count,participant_count,total_valid_power,eligibility,canonical_bytes,canonical_digest,sealed_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, seal.SealID, seal.BlockInstanceID,
		seal.SealRuleVersion, seal.SchemaVersion, seal.ActivityCount, seal.ParticipantCount,
		seal.TotalValidMiningPower, seal.Eligibility, raw, seal.CanonicalDigest, seal.SealedAt)
	if err != nil {
		return empty, err
	}
	if s.prerequisiteFailureInjector != nil {
		if err = s.prerequisiteFailureInjector("seal_before_commit"); err != nil {
			return empty, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return empty, err
	}
	if s.prerequisiteFailureInjector != nil {
		if err = s.prerequisiteFailureInjector("seal_after_commit_before_response"); err != nil {
			return empty, err
		}
	}
	return seal, nil
}

func buildMiningPowerSealTx(ctx context.Context, tx pgx.Tx, sealID string, at time.Time, block miningpower.BlockContext,
	sourceID, bindingDigest string, snapshot miningpower.Snapshot) (miningpower.SettlementInputSeal, error) {
	s := miningpower.SettlementInputSeal{
		SchemaVersion: miningpower.SealSchemaVersion, SealRuleVersion: miningpower.SealRuleVersion,
		SealID: sealID, BlockInstanceID: block.BlockInstanceID, DisplayBlockID: block.ID,
		G18ReservationSourceID: sourceID, G18ReservationEvidenceDigest: bindingDigest,
		G18RuleVersion: block.RuleVersion, G20RuleVersion: miningpower.DevelopmentRuleVersion,
		WindowIdentity: block.BlockInstanceID, WindowStartedAt: block.StartedAt, WindowEndedAt: block.ScheduledEndAt,
		SealedAt: at, ParticipantWeights: []miningpower.SealedWeight{},
		AcceptedActivityIdentities: []miningpower.SealedActivityIdentity{},
	}
	byCharacter := map[string]*miningpower.SealedWeight{}
	bindingDigests := make([]string, 0, len(snapshot.Activities))
	for _, a := range snapshot.Activities {
		if a.BlockInstanceID != block.BlockInstanceID || a.RuleVersion != miningpower.DevelopmentRuleVersion ||
			a.Status != miningpower.StatusValid || a.Decision != miningpower.DecisionAccepted || a.ValidatedPower < 0 {
			return s, miningpower.ErrInvariant
		}
		s.AcceptedActivityIdentities = append(s.AcceptedActivityIdentities,
			miningpower.SealedActivityIdentity{ActivityID: a.ActivityID, SourceEventID: a.SourceEventID})
		var account, player, character, digest, sourceEvent, authority, version sql.NullString
		var boundAt sql.NullTime
		err := tx.QueryRow(ctx, `SELECT account_id,player_id,character_id,digest,source_event_id,authority_source,binding_version,bound_at FROM mining_power_beneficiary_bindings
			WHERE activity_id=$1 AND block_instance_id=$2`, a.ActivityID, block.BlockInstanceID).
			Scan(&account, &player, &character, &digest, &sourceEvent, &authority, &version, &boundAt)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return s, err
		}
		if errors.Is(err, pgx.ErrNoRows) {
			s.Eligibility = "NOT_SETTLEMENT_ELIGIBLE"
			bindingDigests = append(bindingDigests, "")
			character.String = "UNBOUND:" + a.PlayerID
			player.String = a.PlayerID
		} else {
			if !account.Valid || !player.Valid || !character.Valid || !digest.Valid || !sourceEvent.Valid ||
				!authority.Valid || !version.Valid || !boundAt.Valid || player.String != a.PlayerID ||
				sourceEvent.String != a.SourceEventID || authority.String != "CHARACTERS_OWNER_ROW_V1" ||
				version.String != miningBeneficiaryVersion {
				return s, miningpower.ErrInvariant
			}
			actualDigest, e := miningpower.BeneficiaryEvidenceDigest(miningpower.BeneficiaryEvidence{
				Version: version.String, ActivityID: a.ActivityID, SourceEventID: sourceEvent.String,
				AccountID: account.String, PlayerID: player.String, CharacterID: character.String,
				BlockInstanceID: block.BlockInstanceID, AuthoritySource: authority.String, BoundAt: boundAt.Time,
			})
			if e != nil || actualDigest != digest.String {
				return s, miningpower.ErrInvariant
			}
			bindingDigests = append(bindingDigests, digest.String)
		}
		key := character.String
		weight := byCharacter[key]
		if weight == nil {
			weight = &miningpower.SealedWeight{PlayerID: player.String, AccountID: account.String, CharacterID: character.String}
			byCharacter[key] = weight
		} else if weight.PlayerID != player.String || weight.AccountID != account.String {
			return s, miningpower.ErrInvariant
		}
		if weight.Power > math.MaxInt64-a.ValidatedPower || weight.ActivityCount == math.MaxInt64 ||
			s.TotalValidMiningPower > math.MaxInt64-a.ValidatedPower {
			return s, miningpower.ErrOverflow
		}
		weight.Power += a.ValidatedPower
		weight.ActivityCount++
		s.TotalValidMiningPower += a.ValidatedPower
		s.ActivityCount++
	}
	if len(byCharacter) > miningpower.TestParticipantCap {
		return s, fmt.Errorf("%w: 501 participants exceed TEST cap 500", miningpower.ErrInvalidInput)
	}
	for _, weight := range byCharacter {
		s.ParticipantWeights = append(s.ParticipantWeights, *weight)
	}
	miningpower.SortSealedWeights(s.ParticipantWeights)
	s.ParticipantCount = len(s.ParticipantWeights)
	if s.Eligibility == "" {
		s.Eligibility = "SETTLEMENT_INPUT"
	}
	if s.TotalValidMiningPower == 0 {
		s.Eligibility = "NOT_SETTLEMENT_ELIGIBLE"
	}
	activityDigest, err := miningpower.AcceptedActivityIdentityDigest(s.AcceptedActivityIdentities)
	if err != nil {
		return s, err
	}
	s.AcceptedActivityIdentityDigest = activityDigest
	beneficiaryDigest, err := miningpower.BeneficiaryBindingOrderDigest(bindingDigests)
	if err != nil {
		return s, err
	}
	s.BeneficiaryBindingDigest = beneficiaryDigest
	return s, nil
}

func loadMiningPowerSealTx(ctx context.Context, tx pgx.Tx, instanceID string) (miningpower.SettlementInputSeal, error) {
	var raw []byte
	var digest, sealID, rule, schema, eligibility string
	var activities, participants int
	var total int64
	var at time.Time
	err := tx.QueryRow(ctx, `SELECT canonical_bytes,canonical_digest,seal_id,seal_rule_version,schema_version,
		activity_count,participant_count,total_valid_power,eligibility,sealed_at
		FROM mining_power_input_seals WHERE block_instance_id=$1`, instanceID).
		Scan(&raw, &digest, &sealID, &rule, &schema, &activities, &participants, &total, &eligibility, &at)
	if err != nil {
		return miningpower.SettlementInputSeal{}, err
	}
	if len(raw) > settlementMaxSealBytes {
		return miningpower.SettlementInputSeal{}, miningpower.ErrInvariant
	}
	var s miningpower.SettlementInputSeal
	if err = json.Unmarshal(raw, &s); err != nil {
		return s, miningpower.ErrInvariant
	}
	encoded, err := miningpower.CanonicalSealBytes(s)
	if err != nil || !bytes.Equal(raw, encoded) || miningpower.DigestSealBytes(raw) != digest ||
		s.BlockInstanceID != instanceID || s.SealID != sealID || s.SealRuleVersion != rule ||
		s.SchemaVersion != schema || s.ActivityCount != activities || s.ParticipantCount != participants ||
		s.TotalValidMiningPower != total || s.Eligibility != eligibility || !s.SealedAt.Equal(at) {
		return miningpower.SettlementInputSeal{}, miningpower.ErrInvariant
	}
	s.CanonicalDigest = digest
	return s, nil
}

func (s *Store) LoadMiningPowerSeal(ctx context.Context, instanceID string) (miningpower.SettlementInputSeal, error) {
	if s == nil || s.pool == nil {
		return miningpower.SettlementInputSeal{}, miningpower.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return miningpower.SettlementInputSeal{}, err
	}
	defer tx.Rollback(context.Background())
	result, err := loadMiningPowerSealTx(ctx, tx, instanceID)
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}

// RebuildMiningPowerSeal compares the immutable seal with a fresh read-only
// reconstruction of accepted activities and beneficiary bindings. It never
// uses the mutable participant aggregate as a source of reward weights.
func (s *Store) RebuildMiningPowerSeal(ctx context.Context, instanceID string) (miningpower.SettlementInputSeal, error) {
	var empty miningpower.SettlementInputSeal
	if s == nil || s.pool == nil {
		return empty, miningpower.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return empty, err
	}
	defer tx.Rollback(ctx)
	stored, err := loadMiningPowerSealTx(ctx, tx, instanceID)
	if err != nil {
		return empty, err
	}
	rebuilt, err := validateMiningPowerSealSnapshotTx(ctx, tx, stored)
	if err != nil {
		return empty, err
	}
	return rebuilt, tx.Commit(ctx)
}

// validateMiningPowerSealSnapshotTx compares the stored Seal with its source
// identities inside the caller's transaction, before any economic mutation.
func validateMiningPowerSealSnapshotTx(ctx context.Context, tx pgx.Tx, stored miningpower.SettlementInputSeal) (miningpower.SettlementInputSeal, error) {
	var empty miningpower.SettlementInputSeal
	instanceID := stored.BlockInstanceID
	var sourceID, evidenceDigest string
	err := tx.QueryRow(ctx, `SELECT reservation_source_id,evidence_digest FROM mining_reservation_instance_bindings WHERE block_instance_id=$1`, instanceID).Scan(&sourceID, &evidenceDigest)
	if err != nil || sourceID != stored.G18ReservationSourceID || evidenceDigest != stored.G18ReservationEvidenceDigest {
		return empty, miningpower.ErrInvariant
	}
	if err := validateMiningPowerSealRulesTx(ctx, tx, instanceID, stored.G20RuleVersion); err != nil {
		return empty, err
	}
	snapshot, err := snapshotMiningPowerTx(ctx, tx, instanceID, stored.G20RuleVersion)
	if err != nil {
		return empty, err
	}
	report, err := miningpower.ReconcileSnapshot(snapshot)
	if err != nil || report.Status != "PASS" {
		return empty, miningpower.ErrInvariant
	}
	block := miningpower.BlockContext{ID: stored.DisplayBlockID, BlockInstanceID: stored.BlockInstanceID,
		RuleVersion: stored.G18RuleVersion, StartedAt: stored.WindowStartedAt, ScheduledEndAt: stored.WindowEndedAt}
	rebuilt, err := buildMiningPowerSealTx(ctx, tx, stored.SealID, stored.SealedAt, block, sourceID, evidenceDigest, snapshot)
	if err != nil {
		return empty, err
	}
	raw, err := miningpower.CanonicalSealBytes(rebuilt)
	if err != nil || miningpower.DigestSealBytes(raw) != stored.CanonicalDigest {
		return empty, miningpower.ErrInvariant
	}
	rebuilt.CanonicalDigest = stored.CanonicalDigest
	return rebuilt, nil
}

// The filtered G20 snapshot is not authority to omit accepted facts from another
// rule. Every immutable accepted fact in this instance must use the sealed rule.
func validateMiningPowerSealRulesTx(ctx context.Context, tx pgx.Tx, instanceID, rule string) error {
	var mixed bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mining_power_activities
		WHERE block_instance_id=$1 AND rule_version<>$2)`, instanceID, rule).Scan(&mixed); err != nil {
		return err
	}
	if mixed {
		return miningpower.ErrInvariant
	}
	return nil
}
