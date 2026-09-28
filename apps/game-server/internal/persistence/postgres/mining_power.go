package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

const miningPowerMaxAttempts = 5
const miningPowerValidationTimeout = 30 * time.Second

func (s *Store) miningPowerFailure(stage string) error {
	if s.miningPowerFailureInjector != nil {
		return s.miningPowerFailureInjector(stage)
	}
	return nil
}
func miningPowerReject(status, reason string) miningpower.ValidationResult {
	return miningpower.ValidationResult{Status: status, ReasonCode: reason}
}
func miningPowerOwner(ctx context.Context, q miningPowerReader, p miningpower.Principal) (bool, error) {
	var owned bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM characters WHERE id=$1 AND account_id=$2)`, p.PlayerID, p.AccountID).Scan(&owned)
	return owned, err
}
func miningPowerPrior(ctx context.Context, q miningPowerReader, i miningpower.ActionIntent) (miningpower.ValidatedMiningActivity, error) {
	return miningPowerRead[miningpower.ValidatedMiningActivity](ctx, q, `SELECT data FROM mining_power_activities WHERE source_event_id=$1 OR activity_id=$2 ORDER BY activity_id LIMIT 1`, i.SourceEventID, i.ActivityID)
}
func miningPowerReplay(p miningpower.Principal, i miningpower.ActionIntent, rule string, a miningpower.ValidatedMiningActivity) miningpower.ValidationResult {
	if a.PlayerID != p.PlayerID {
		return miningPowerReject(miningpower.StatusNotEligible, "EVENT_OWNER_MISMATCH")
	}
	if a.ActivityID != i.ActivityID || a.SourceEventID != i.SourceEventID || a.ActivitySessionID != i.ActivitySessionID || a.BlockInstanceID != i.BlockInstanceID || a.BlockID != i.BlockID || a.RuleVersion != rule {
		return miningPowerReject(miningpower.StatusReplayed, "BINDING_CHANGED")
	}
	return miningpower.ValidationResult{Status: miningpower.StatusDuplicate, ReasonCode: "ALREADY_ACCEPTED", Original: &a}
}
func (s *Store) ValidateMiningActivity(ctx context.Context, p miningpower.Principal, i miningpower.ActionIntent, rule string) (miningpower.ValidationResult, error) {
	if s == nil || s.pool == nil {
		return miningpower.ValidationResult{}, miningpower.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, miningPowerValidationTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if miningpower.ValidateIntent(i) != nil {
		return miningPowerReject(miningpower.StatusInvalid, "INVALID_IDENTITY"), nil
	}
	if !miningpower.ValidID(p.PlayerID, 128) || !miningpower.ValidID(p.AccountID, 128) {
		return miningPowerReject(miningpower.StatusNotEligible, "INVALID_PRINCIPAL"), nil
	}
	owned, err := miningPowerOwner(ctx, s.pool, p)
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	if !owned {
		return miningPowerReject(miningpower.StatusNotEligible, "PLAYER_OWNER_MISMATCH"), nil
	}
	// Exact committed replay never resolves a reused display ID or a live G18 row.
	if prior, e := miningPowerPrior(ctx, s.pool, i); e == nil {
		replay := miningPowerReplay(p, i, rule, prior)
		if replay.Status != miningpower.StatusDuplicate {
			return replay, nil
		}
		snapshot, e := s.SnapshotMiningPower(ctx, prior.BlockInstanceID, prior.RuleVersion)
		if e != nil {
			return miningpower.ValidationResult{}, e
		}
		report, e := miningpower.ReconcileSnapshot(snapshot)
		if e != nil {
			return miningpower.ValidationResult{}, e
		}
		if report.Status != "PASS" {
			return miningpower.ValidationResult{}, miningpower.ErrInvariant
		}
		return replay, nil
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return miningpower.ValidationResult{}, e
	}
	if rule != miningpower.DevelopmentRuleVersion {
		return miningPowerReject(miningpower.StatusInvalid, "UNKNOWN_RULE"), nil
	}
	// Existing accepted facts were handled above. A committed seal takes
	// precedence over the current G18 FINALIZED state for a new source.
	var gateState string
	if e := s.pool.QueryRow(ctx, `SELECT state FROM mining_power_acceptance_states WHERE block_instance_id=$1`, i.BlockInstanceID).Scan(&gateState); e == nil {
		if gateState == "SEALED" {
			return miningPowerReject(miningpower.StatusNotEligible, "BLOCK_SEALED"), nil
		}
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return miningpower.ValidationResult{}, e
	}
	// This cross-domain read precedes every G20 transaction; no upstream locking.
	block, err := s.ReadBlock(ctx, i.BlockInstanceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningPowerReject(miningpower.StatusNotEligible, "BLOCK_INSTANCE_NOT_FOUND"), nil
	}
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	if block.BlockInstanceID != i.BlockInstanceID || block.ID != i.BlockID || block.Status != "OPEN" {
		// An acceptance that began while G18 was OPEN can still be committing.
		// Join its existing PostgreSQL admission lock and recheck exact replay
		// after it finishes before rejecting the now-finalized context.
		if block.BlockInstanceID == i.BlockInstanceID && block.ID == i.BlockID && block.Status == "FINALIZED" {
			prior, e := s.miningPowerPriorAfterAdmission(ctx, i)
			if e == nil {
				replay := miningPowerReplay(p, i, rule, prior)
				if replay.Status == miningpower.StatusDuplicate {
					snapshot, auditErr := s.SnapshotMiningPower(ctx, prior.BlockInstanceID, prior.RuleVersion)
					if auditErr != nil {
						return miningpower.ValidationResult{}, auditErr
					}
					report, auditErr := miningpower.ReconcileSnapshot(snapshot)
					if auditErr != nil || report.Status != "PASS" {
						return miningpower.ValidationResult{}, miningpower.ErrInvariant
					}
				}
				return replay, nil
			}
			if !errors.Is(e, pgx.ErrNoRows) {
				return miningpower.ValidationResult{}, e
			}
			var latestState string
			if gateErr := s.pool.QueryRow(ctx, `SELECT state FROM mining_power_acceptance_states WHERE block_instance_id=$1`, i.BlockInstanceID).Scan(&latestState); gateErr == nil && latestState == "SEALED" {
				return miningPowerReject(miningpower.StatusNotEligible, "BLOCK_SEALED"), nil
			} else if gateErr != nil && !errors.Is(gateErr, pgx.ErrNoRows) {
				return miningpower.ValidationResult{}, gateErr
			}
		}
		return miningPowerReject(miningpower.StatusNotEligible, "BLOCK_CONTEXT_MISMATCH"), nil
	}
	if err = s.miningPowerFailure("after_validation"); err != nil {
		return miningpower.ValidationResult{}, err
	}
	for attempt := 1; attempt <= miningPowerMaxAttempts; attempt++ {
		if err = ctx.Err(); err != nil {
			return miningpower.ValidationResult{}, err
		}
		result, e := s.acceptMiningPowerAttempt(ctx, p, i, rule, block)
		if e == nil {
			return result, nil
		}
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || (pg.Code != "40001" && pg.Code != "40P01") {
			return miningpower.ValidationResult{}, e
		}
		if attempt == miningPowerMaxAttempts {
			return miningpower.ValidationResult{}, fmt.Errorf("%w: %s after %d attempts", miningpower.ErrRetryExhausted, pg.Code, attempt)
		}
		timer := time.NewTimer(time.Duration(1<<min(attempt-1, 3)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return miningpower.ValidationResult{}, ctx.Err()
		case <-timer.C:
		}
	}
	return miningpower.ValidationResult{}, miningpower.ErrRetryExhausted
}

func (s *Store) miningPowerPriorAfterAdmission(ctx context.Context, i miningpower.ActionIntent) (miningpower.ValidatedMiningActivity, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return miningpower.ValidatedMiningActivity{}, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "G20_ACTIVITY_ACCEPTANCE_V1"); err != nil {
		return miningpower.ValidatedMiningActivity{}, err
	}
	return miningPowerPrior(ctx, tx, i)
}

func (s *Store) acceptMiningPowerAttempt(ctx context.Context, p miningpower.Principal, i miningpower.ActionIntent, rule string, b miningpower.BlockContext) (miningpower.ValidationResult, error) {
	// Preserve G20's existing global admission serialization and SERIALIZABLE
	// validation snapshot. The PostgreSQL state row below is the authoritative
	// Seal boundary, including other processes and direct Activity inserts.
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	defer conn.Release()
	key := "G20_ACTIVITY_ACCEPTANCE_V1"
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, key); err != nil {
		return miningpower.ValidationResult{}, err
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.Exec(unlockCtx, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, key); err != nil {
			_ = conn.Conn().Close(unlockCtx)
		}
	}()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	defer tx.Rollback(context.Background())
	if err = s.miningPowerFailure("after_begin"); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO mining_power_acceptance_states(block_instance_id,state,revision)
		VALUES($1,'OPEN',1) ON CONFLICT(block_instance_id) DO NOTHING`, i.BlockInstanceID); err != nil {
		return miningpower.ValidationResult{}, err
	}
	var acceptanceState string
	if err = tx.QueryRow(ctx, `SELECT state FROM mining_power_acceptance_states WHERE block_instance_id=$1 FOR SHARE`, i.BlockInstanceID).Scan(&acceptanceState); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if acceptanceState != "OPEN" {
		return miningPowerReject(miningpower.StatusNotEligible, "BLOCK_SEALED"), nil
	}
	owned, err := miningPowerOwner(ctx, tx, p)
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	if !owned {
		return miningPowerReject(miningpower.StatusNotEligible, "PLAYER_OWNER_MISMATCH"), nil
	}
	if prior, e := miningPowerPrior(ctx, tx, i); e == nil {
		replay := miningPowerReplay(p, i, rule, prior)
		if replay.Status == miningpower.StatusDuplicate {
			snapshot, e := snapshotMiningPowerTx(ctx, tx, prior.BlockInstanceID, prior.RuleVersion)
			if e != nil {
				return miningpower.ValidationResult{}, e
			}
			report, e := miningpower.ReconcileSnapshot(snapshot)
			if e != nil {
				return miningpower.ValidationResult{}, e
			}
			if report.Status != "PASS" {
				return miningpower.ValidationResult{}, miningpower.ErrInvariant
			}
		}
		return replay, nil
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return miningpower.ValidationResult{}, e
	}
	source, err := miningPowerRead[miningpower.SourceEvent](ctx, tx, `SELECT data FROM mining_power_source_events WHERE source_event_id=$1`, i.SourceEventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningPowerReject(miningpower.StatusNotEligible, "SOURCE_NOT_FOUND"), nil
	}
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	if source.PlayerID != p.PlayerID {
		return miningPowerReject(miningpower.StatusNotEligible, "EVENT_OWNER_MISMATCH"), nil
	}
	if source.ActivityID != i.ActivityID || source.ActivitySessionID != i.ActivitySessionID || source.BlockInstanceID != i.BlockInstanceID || source.BlockID != i.BlockID || source.RuleVersion != rule {
		return miningPowerReject(miningpower.StatusReplayed, "SOURCE_BINDING_CHANGED"), nil
	}
	session, err := miningPowerRead[miningpower.ActivitySession](ctx, tx, `SELECT data FROM mining_power_sessions WHERE session_id=$1 FOR SHARE`, i.ActivitySessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningPowerReject(miningpower.StatusNotEligible, "SESSION_NOT_FOUND"), nil
	}
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	if session.AccountID != p.AccountID || session.PlayerID != p.PlayerID || session.BlockInstanceID != i.BlockInstanceID || session.BlockID != i.BlockID || session.RuleVersion != rule || session.ToolReference != source.ToolReference || session.MapReference != source.MapReference || session.State != "ACTIVE" {
		return miningPowerReject(miningpower.StatusNotEligible, "SESSION_CONTEXT_MISMATCH"), nil
	}
	if err = s.miningPowerFailure("after_session"); err != nil {
		return miningpower.ValidationResult{}, err
	}
	var now time.Time
	if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return miningpower.ValidationResult{}, err
	}
	now = miningpower.CanonicalTime(now)
	if !now.Before(session.ExpiresAt) || !now.Before(source.ExpiresAt) || !now.Before(b.ScheduledEndAt) {
		return miningPowerReject(miningpower.StatusExpired, "WINDOW_EXPIRED"), nil
	}
	if source.ObservedAt.Before(session.OpenedAt) || source.ObservedAt.Before(b.StartedAt) || !source.ObservedAt.Before(b.ScheduledEndAt) || source.ObservedAt.After(now) || source.ObservedAt.After(b.ValidatedAt) || session.ExpiresAt.After(b.ScheduledEndAt) || source.ExpiresAt.Before(source.ObservedAt) || source.ServerEligibility != miningpower.StatusValid || source.EvidenceKind != miningpower.SyntheticEvidence || source.ActivityWeightScaled <= 0 {
		return miningPowerReject(miningpower.StatusInvalid, "INVALID_SERVER_EVIDENCE"), nil
	}
	manifest, err := miningPowerRead[miningpower.RuleManifest](ctx, tx, `SELECT data FROM mining_power_rules WHERE rule_version=$1`, rule)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningPowerReject(miningpower.StatusInvalid, "UNKNOWN_RULE"), nil
	}
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	profiles := miningPowerProfiles{tx: tx}
	tool, err := profiles.ResolveTool(ctx, session.ToolReference, rule)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningPowerReject(miningpower.StatusNotEligible, "TOOL_NOT_FOUND"), nil
	}
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	m, err := profiles.ResolveMap(ctx, session.MapReference, rule)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningPowerReject(miningpower.StatusNotEligible, "MAP_NOT_FOUND"), nil
	}
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	if tool.Kind != miningpower.SyntheticKind || m.Kind != miningpower.SyntheticKind || tool.RuleVersion != rule || m.RuleVersion != rule {
		return miningPowerReject(miningpower.StatusNotEligible, "NON_TEST_PROFILE"), nil
	}
	inputs := miningpower.ValidatedInputs{BasePowerUnits: tool.BasePowerUnits, EfficiencyScaled: tool.EfficiencyScaled, ActivityWeightScaled: source.ActivityWeightScaled, MapModifierScaled: m.ModifierScaled}
	power, err := miningpower.CalculatePower(manifest, inputs)
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	validation := miningpower.MiningBlockValidationSnapshot{BlockInstanceID: b.BlockInstanceID, BlockID: b.ID, BlockHeight: b.Height, CreateCommandID: b.CreateCommandID, Status: b.Status, G18RuleVersion: b.RuleVersion, G20RuleVersion: rule, SourceEvidenceVersion: b.SourceEvidenceVersion, StartedAt: b.StartedAt, ScheduledEndAt: b.ScheduledEndAt, ValidatedAt: b.ValidatedAt}
	a := miningpower.ValidatedMiningActivity{ActivityID: i.ActivityID, SourceEventID: i.SourceEventID, PlayerID: p.PlayerID, BlockID: i.BlockID, BlockInstanceID: i.BlockInstanceID, ActivitySessionID: i.ActivitySessionID, ToolReference: tool.Reference, MapReference: m.Reference, RuleVersion: rule, Inputs: inputs, ValidatedPower: power, AcceptedAt: now, ObservedAt: source.ObservedAt, ExpiresAt: source.ExpiresAt, Decision: miningpower.DecisionAccepted, Status: miningpower.StatusValid, ValidationSnapshot: validation}
	a = a.Canonical()
	raw, err := json.Marshal(a)
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	if err = s.miningPowerFailure("before_activity"); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO mining_power_activities(data) VALUES($1::jsonb)`, string(raw)); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if err = s.miningPowerFailure("after_activity"); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if err = captureMiningBeneficiaryTx(ctx, tx, p, a, now); err != nil {
		return miningpower.ValidationResult{}, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO mining_power_participants(player_id,block_instance_id,block_id,session_id,rule_version,tool_reference,map_reference,validated_power,activity_count,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,1,$9,$9)
 ON CONFLICT(player_id,block_instance_id,session_id,rule_version) DO UPDATE SET validated_power=mining_power_participants.validated_power+excluded.validated_power,activity_count=mining_power_participants.activity_count+1,created_at=least(mining_power_participants.created_at,excluded.created_at),updated_at=greatest(mining_power_participants.updated_at,excluded.updated_at)
 WHERE mining_power_participants.validated_power<=9223372036854775807-excluded.validated_power AND mining_power_participants.activity_count<9223372036854775807`, a.PlayerID, a.BlockInstanceID, a.BlockID, a.ActivitySessionID, a.RuleVersion, a.ToolReference, a.MapReference, power, now)
	if err != nil {
		return miningpower.ValidationResult{}, err
	}
	if tag.RowsAffected() != 1 {
		return miningpower.ValidationResult{}, miningpower.ErrOverflow
	}
	if err = s.miningPowerFailure("after_participant"); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if err = s.miningPowerFailure("before_commit"); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return miningpower.ValidationResult{}, err
	}
	if err = s.miningPowerFailure("after_commit_before_response"); err != nil {
		return miningpower.ValidationResult{}, err
	}
	return miningpower.ValidationResult{Status: miningpower.StatusValid, ReasonCode: miningpower.DecisionAccepted, AppliedPower: power, Original: &a}, nil
}
