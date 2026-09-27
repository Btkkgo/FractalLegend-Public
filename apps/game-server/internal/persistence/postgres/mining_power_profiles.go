package postgres

import (
	"context"
	"encoding/json"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
)

type miningPowerReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func miningPowerRead[T any](ctx context.Context, q miningPowerReader, sql string, args ...any) (T, error) {
	var value T
	var raw []byte
	if err := q.QueryRow(ctx, sql, args...).Scan(&raw); err != nil {
		return value, err
	}
	err := json.Unmarshal(raw, &value)
	if err == nil {
		if canonical, ok := any(value).(interface{ Canonical() T }); ok {
			value = canonical.Canonical()
		}
	}
	return value, err
}

// Resolvers stay bound to the acceptance transaction. No equipment/world path.
type miningPowerProfiles struct{ tx pgx.Tx }

func (p miningPowerProfiles) ResolveTool(ctx context.Context, ref, rule string) (miningpower.ToolProfile, error) {
	return miningPowerRead[miningpower.ToolProfile](ctx, p.tx, `SELECT data FROM mining_power_tool_profiles WHERE rule_version=$1 AND tool_reference=$2`, rule, ref)
}
func (p miningPowerProfiles) ResolveMap(ctx context.Context, ref, rule string) (miningpower.MapProfile, error) {
	return miningPowerRead[miningpower.MapProfile](ctx, p.tx, `SELECT data FROM mining_power_map_profiles WHERE rule_version=$1 AND map_reference=$2`, rule, ref)
}

var _ miningpower.MiningToolEligibility = miningPowerProfiles{}
var _ miningpower.MiningMapEligibility = miningPowerProfiles{}
