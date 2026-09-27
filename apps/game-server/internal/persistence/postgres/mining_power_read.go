package postgres

import (
	"context"
	"encoding/json"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
)

func miningPowerReadRows[T any](ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]T, error) {
	result := []T{}
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var v T
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		if canonical, ok := any(v).(interface{ Canonical() T }); ok {
			v = canonical.Canonical()
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func snapshotMiningPowerTx(ctx context.Context, tx pgx.Tx, instance, rule string) (miningpower.Snapshot, error) {
	var snap miningpower.Snapshot
	var err error
	if snap.Rules, err = miningPowerReadRows[miningpower.RuleManifest](ctx, tx, `SELECT data FROM mining_power_rules ORDER BY rule_version`); err != nil {
		return snap, err
	}
	if snap.Tools, err = miningPowerReadRows[miningpower.ToolProfile](ctx, tx, `SELECT data FROM mining_power_tool_profiles ORDER BY rule_version,tool_reference`); err != nil {
		return snap, err
	}
	if snap.Maps, err = miningPowerReadRows[miningpower.MapProfile](ctx, tx, `SELECT data FROM mining_power_map_profiles ORDER BY rule_version,map_reference`); err != nil {
		return snap, err
	}
	filter := ` WHERE ($1='' OR block_instance_id=$1) AND ($2='' OR rule_version=$2)`
	if snap.Sessions, err = miningPowerReadRows[miningpower.ActivitySession](ctx, tx, `SELECT data FROM mining_power_sessions`+filter+` ORDER BY session_id`, instance, rule); err != nil {
		return snap, err
	}
	if snap.Sources, err = miningPowerReadRows[miningpower.SourceEvent](ctx, tx, `SELECT data FROM mining_power_source_events`+filter+` ORDER BY source_event_id`, instance, rule); err != nil {
		return snap, err
	}
	if snap.Activities, err = miningPowerReadRows[miningpower.ValidatedMiningActivity](ctx, tx, `SELECT data FROM mining_power_activities`+filter+` ORDER BY activity_id`, instance, rule); err != nil {
		return snap, err
	}
	rows, err := tx.Query(ctx, `SELECT player_id,block_instance_id,block_id,session_id,rule_version,tool_reference,map_reference,validated_power,activity_count,created_at,updated_at FROM mining_power_participants`+filter+` ORDER BY player_id,block_instance_id,session_id,rule_version`, instance, rule)
	if err != nil {
		return snap, err
	}
	snap.Participants = []miningpower.MiningParticipant{}
	for rows.Next() {
		var p miningpower.MiningParticipant
		if err = rows.Scan(&p.PlayerID, &p.BlockInstanceID, &p.BlockID, &p.ActivitySessionID, &p.RuleVersion, &p.ToolReference, &p.MapReference, &p.ValidatedPower, &p.ActivityCount, &p.CreatedAt, &p.UpdatedAt); err != nil {
			rows.Close()
			return snap, err
		}
		snap.Participants = append(snap.Participants, p.Canonical())
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return snap, err
	}
	// Historical G18 evidence comes only from immutable accepted facts. No live
	// upstream lookup can substitute a newly recreated business ID for this one.
	snap.Blocks = []miningpower.BlockContext{}
	seen := map[string]bool{}
	for _, a := range snap.Activities {
		v := a.ValidationSnapshot
		if seen[v.BlockInstanceID] {
			continue
		}
		seen[v.BlockInstanceID] = true
		snap.Blocks = append(snap.Blocks, miningpower.BlockContext{ID: v.BlockID, BlockInstanceID: v.BlockInstanceID, Height: v.BlockHeight, CreateCommandID: v.CreateCommandID, Status: v.Status, RuleVersion: v.G18RuleVersion, StartedAt: v.StartedAt, ScheduledEndAt: v.ScheduledEndAt, ValidatedAt: v.ValidatedAt, SourceEvidenceVersion: v.SourceEvidenceVersion})
	}
	return snap, nil
}
func (s *Store) SnapshotMiningPower(ctx context.Context, instance, rule string) (miningpower.Snapshot, error) {
	if s == nil || s.pool == nil {
		return miningpower.Snapshot{}, miningpower.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return miningpower.Snapshot{}, err
	}
	defer tx.Rollback(context.Background())
	if err = s.miningPowerFailure("read_snapshot"); err != nil {
		return miningpower.Snapshot{}, err
	}
	snap, err := snapshotMiningPowerTx(ctx, tx, instance, rule)
	if err != nil {
		return snap, err
	}
	return snap, tx.Commit(ctx)
}
func (s *Store) ReconcileMiningPower(ctx context.Context, instance, rule string) (miningpower.ReconciliationReport, error) {
	snap, err := s.SnapshotMiningPower(ctx, instance, rule)
	if err != nil {
		return miningpower.ReconciliationReport{}, err
	}
	return miningpower.ReconcileSnapshot(snap)
}
func (s *Store) TotalValidatedPower(ctx context.Context, instance, rule string) (int64, error) {
	snap, err := s.SnapshotMiningPower(ctx, instance, rule)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, p := range snap.Participants {
		if rule == "" && p.RuleVersion != miningpower.DevelopmentRuleVersion {
			return 0, miningpower.ErrUnknownRule
		}
		if p.ValidatedPower < 0 || total > miningpower.MaxPower-p.ValidatedPower {
			return 0, miningpower.ErrOverflow
		}
		total += p.ValidatedPower
	}
	return total, nil
}

var _ miningpower.Repository = (*Store)(nil)
