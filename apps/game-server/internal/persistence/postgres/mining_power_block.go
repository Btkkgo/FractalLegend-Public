package postgres

import (
	"context"
	"fractallegend/game-server/internal/miningpower"
)

// ReadBlock is deliberately an ordinary SELECT by concrete instance identity.
// It has no lifecycle, reservation, pool, debt or row-lock operation.
func (s *Store) ReadBlock(ctx context.Context, instanceID string) (miningpower.BlockContext, error) {
	var b miningpower.BlockContext
	if s == nil || s.pool == nil {
		return b, miningpower.ErrUnavailable
	}
	if !miningpower.ValidID(instanceID, 128) {
		return b, miningpower.ErrInvalidIntent
	}
	err := s.pool.QueryRow(ctx, `SELECT block_instance_id,block_id,block_height,create_command_id,status,rule_version,started_at,scheduled_end_at,clock_timestamp() FROM mining_blocks WHERE block_instance_id=$1`, instanceID).Scan(&b.BlockInstanceID, &b.ID, &b.Height, &b.CreateCommandID, &b.Status, &b.RuleVersion, &b.StartedAt, &b.ScheduledEndAt, &b.ValidatedAt)
	b.SourceEvidenceVersion = "G18_SCHEMA_0012"
	return b.Canonical(), err
}
