package postgres

import (
	"context"
	"testing"
)

func TestG21SettlementSchemaHasImmutableEconomicIdentities(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{
		"mining_reward_commands", "mining_reward_settlement_states",
		"mining_reward_settlement_receipts", "mining_reward_reservation_consumptions",
		"mining_reward_grants",
	} {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("missing G21 table %s", table)
		}
	}
	for _, table := range []string{
		"mining_reward_commands", "mining_reward_settlement_receipts",
		"mining_reward_reservation_consumptions", "mining_reward_grants",
	} {
		var rowGuard, truncateGuard int
		if err := s.pool.QueryRow(ctx, `SELECT
			count(*) FILTER (WHERE (tgtype & 1) = 1 AND (tgtype & 2) = 2 AND (tgtype & 8) = 8),
			count(*) FILTER (WHERE (tgtype & 1) = 0 AND (tgtype & 32) = 32)
			FROM pg_trigger WHERE tgrelid=to_regclass('public.' || $1) AND NOT tgisinternal`, table).
			Scan(&rowGuard, &truncateGuard); err != nil {
			t.Fatal(err)
		}
		if rowGuard == 0 || truncateGuard == 0 {
			t.Errorf("mutable G21 history %s: row=%d truncate=%d", table, rowGuard, truncateGuard)
		}
	}
}
