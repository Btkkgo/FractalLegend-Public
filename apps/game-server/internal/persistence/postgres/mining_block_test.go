package postgres

import (
	"context"
	"errors"
	"io/fs"
	"math"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/ledger"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/systemspend"
)

func g18Fund(t *testing.T, amount int64) (*Store, *miningblock.Service, *contribution.Service, systemspend.SystemSpend) {
	t.Helper()
	store, contributions, _ := g13Fixture(t)
	ctx := context.Background()
	spend, err := systemspend.NewService(store, g15Registry(t)).Post(ctx, g15Intent("g18-fund", amount))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = emission.NewService(store, emission.DevelopmentRuleVersion).Apply(ctx, spend.ID); err != nil {
		t.Fatal(err)
	}
	return store, miningblock.NewService(store, miningblock.DevelopmentRuleVersion, false), contributions, spend
}

func TestG18CreateCancelReplayRestartSnapshotAndConservation(t *testing.T) {
	store, service, _, _ := g18Fund(t, 20)
	ctx := context.Background()
	first, err := service.Create(ctx, "g18-create-1")
	if err != nil || first.BlockHeight != 1 || first.RewardReserved != 10 || first.PoolBefore != 20 || first.PoolAfter != 10 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	replay, err := service.Create(ctx, "g18-create-1")
	if err != nil || !reflect.DeepEqual(first, replay) {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	block, err := service.Recover(ctx, first.BlockID)
	if err != nil || block.Status != miningblock.StatusOpen || block.Height != 1 || !block.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("block=%+v err=%v", block, err)
	}
	opened, err := store.SnapshotMiningBlocks(ctx)
	if err != nil || len(opened.Blocks) != 1 || len(opened.Reservations) != 1 || len(opened.Entries) != 1 || len(opened.Receipts) != 1 || !reflect.DeepEqual(opened.Receipts[0], first) {
		t.Fatalf("snapshot=%+v err=%v", opened, err)
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	restarted := miningblock.NewService(reopened, miningblock.DevelopmentRuleVersion, false)
	replayed, err := restarted.Create(ctx, "g18-create-1")
	if err != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatalf("restart replay=%+v err=%v", replayed, err)
	}
	reload, err := reopened.SnapshotMiningBlocks(ctx)
	if err != nil || !reflect.DeepEqual(opened, reload) {
		t.Fatalf("reload mismatch=%+v err=%v", reload, err)
	}
	cancelled, err := restarted.Cancel(ctx, first.BlockID)
	if err != nil || cancelled.BlockStatus != miningblock.StatusCancelled || cancelled.PoolBefore != 10 || cancelled.PoolAfter != 20 {
		t.Fatalf("cancel=%+v err=%v", cancelled, err)
	}
	duplicate, err := service.Cancel(ctx, first.BlockID)
	if err != nil || !reflect.DeepEqual(cancelled, duplicate) {
		t.Fatalf("duplicate cancel=%+v err=%v", duplicate, err)
	}
	if _, err = service.Finalize(ctx, first.BlockID); !errors.Is(err, miningblock.ErrConflict) {
		t.Fatalf("finalize after cancel=%v", err)
	}
	snapshot, err := service.Recover(ctx, first.BlockID)
	if err != nil || snapshot.Status != miningblock.StatusCancelled || snapshot.RewardReturned != 10 {
		t.Fatalf("recovered=%+v err=%v", snapshot, err)
	}
	pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
	if err != nil || pool.Pool.TotalReserved != 0 || pool.Pool.RemainingCapacity != 20 || pool.Pool.TotalDistributed != 0 {
		t.Fatalf("pool=%+v err=%v", pool.Pool, err)
	}
	report, err := service.Recover(ctx, first.BlockID)
	if err != nil || report.Status != miningblock.StatusCancelled {
		t.Fatalf("recovery=%+v err=%v", report, err)
	}
	if _, err = store.pool.Exec(ctx, `DELETE FROM mining_block_entries WHERE block_id=$1 AND action='CANCEL'`, first.BlockID); err == nil {
		t.Fatal("immutable release entry was deletable")
	}
	second, err := service.Create(ctx, "g18-create-2")
	if err != nil || second.BlockHeight != 2 {
		t.Fatalf("next height=%+v err=%v", second, err)
	}
}

func TestG18CapacityConcurrentHeightAndFinalize(t *testing.T) {
	store, service, _, _ := g18Fund(t, 100)
	ctx := context.Background()
	const requests = 14
	results := make(chan miningblock.Receipt, requests)
	errs := make(chan error, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, e := service.Create(ctx, "g18-concurrent-"+string(rune('a'+i)))
			if e != nil {
				errs <- e
			} else {
				results <- r
			}
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	heights := map[int64]bool{}
	for r := range results {
		if heights[r.BlockHeight] {
			t.Fatalf("duplicate height %d", r.BlockHeight)
		}
		heights[r.BlockHeight] = true
	}
	if len(heights) != 10 {
		t.Fatalf("successful reservations=%d", len(heights))
	}
	for e := range errs {
		if !errors.Is(e, miningblock.ErrInsufficientCapacity) {
			t.Fatalf("unexpected concurrency error=%v", e)
		}
	}
	for i := int64(1); i <= 10; i++ {
		if !heights[i] {
			t.Fatalf("missing height %d", i)
		}
	}
	if _, err := service.Create(ctx, "g18-eleventh"); !errors.Is(err, miningblock.ErrInsufficientCapacity) {
		t.Fatalf("over-reserve=%v", err)
	}
	pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
	if err != nil || pool.Pool.TotalReserved != 100 || pool.Pool.RemainingCapacity != 0 || pool.Pool.TotalDistributed != 0 {
		t.Fatalf("pool=%+v err=%v", pool.Pool, err)
	}
	report, err := service.Recover(ctx, "nonexistent")
	if !errors.Is(err, miningblock.ErrNotFound) || report.ID != "" {
		t.Fatalf("recover missing=%+v err=%v", report, err)
	}
	miningReport, err := store.ReconcileMiningBlocks(ctx)
	if err != nil || !miningReport.Balanced || miningReport.Checked != 10 {
		t.Fatalf("reconcile=%+v err=%v", miningReport, err)
	}
	emissionReport, err := store.ReconcileBlackIronEmission(ctx)
	if err != nil || !emissionReport.Balanced {
		t.Fatalf("G17 reconcile=%+v err=%v", emissionReport, err)
	}
}

func TestG18FinalizeCancelRaceAndExactReplay(t *testing.T) {
	store, service, _, _ := g18Fund(t, 10)
	ctx := context.Background()
	open, err := service.Create(ctx, "g18-race")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan miningblock.Receipt, 2)
	errs := make(chan error, 2)
	for _, action := range []string{miningblock.ActionFinalize, miningblock.ActionCancel} {
		wg.Add(1)
		go func(action string) {
			defer wg.Done()
			var r miningblock.Receipt
			var e error
			if action == miningblock.ActionFinalize {
				r, e = service.Finalize(ctx, open.BlockID)
			} else {
				r, e = service.Cancel(ctx, open.BlockID)
			}
			results <- r
			errs <- e
		}(action)
	}
	wg.Wait()
	close(results)
	close(errs)
	success, conflict := 0, 0
	for e := range errs {
		if e == nil {
			success++
		} else if errors.Is(e, miningblock.ErrConflict) {
			conflict++
		} else {
			t.Fatalf("race error=%v", e)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
	block, err := store.LoadMiningBlock(ctx, open.BlockID)
	if err != nil {
		t.Fatal(err)
	}
	if block.Status == miningblock.StatusFinalized {
		first, err := service.Finalize(ctx, open.BlockID)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := service.Finalize(ctx, open.BlockID)
		if err != nil || !reflect.DeepEqual(first, replay) {
			t.Fatalf("finalize replay=%+v err=%v", replay, err)
		}
		if _, err = service.Cancel(ctx, open.BlockID); !errors.Is(err, miningblock.ErrConflict) {
			t.Fatalf("cancel finalized=%v", err)
		}
	} else if block.Status == miningblock.StatusCancelled {
		first, err := service.Cancel(ctx, open.BlockID)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := service.Cancel(ctx, open.BlockID)
		if err != nil || !reflect.DeepEqual(first, replay) {
			t.Fatalf("cancel replay=%+v err=%v", replay, err)
		}
		if _, err = service.Finalize(ctx, open.BlockID); !errors.Is(err, miningblock.ErrConflict) {
			t.Fatalf("finalize cancelled=%v", err)
		}
	} else {
		t.Fatalf("illegal terminal state %s", block.Status)
	}
	report, err := store.ReconcileMiningBlocks(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG18InjectedFailuresRollbackAndRestart(t *testing.T) {
	for _, point := range []string{"after_block", "after_reserve", "after_entry", "after_receipt", "before_commit"} {
		t.Run("create/"+point, func(t *testing.T) {
			store, service, _, _ := g18Fund(t, 10)
			ctx := context.Background()
			store.miningBlockFailureInjector = func(at string) error {
				if at == point {
					return errors.New("injected")
				}
				return nil
			}
			if _, err := service.Create(ctx, "g18-failure"); err == nil {
				t.Fatal("failure committed")
			}
			store.miningBlockFailureInjector = nil
			snapshot, err := store.SnapshotMiningBlocks(ctx)
			if err != nil || len(snapshot.Blocks) != 0 || len(snapshot.Entries) != 0 {
				t.Fatalf("ghost block=%+v err=%v", snapshot, err)
			}
			pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
			if err != nil || pool.Pool.RemainingCapacity != 10 || pool.Pool.TotalReserved != 0 {
				t.Fatalf("leaked capacity=%+v err=%v", pool.Pool, err)
			}
			if _, err = service.Create(ctx, "g18-failure"); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, action := range []string{miningblock.ActionFinalize, miningblock.ActionCancel} {
		t.Run(action, func(t *testing.T) {
			store, service, _, _ := g18Fund(t, 10)
			ctx := context.Background()
			open, err := service.Create(ctx, "g18-transition-failure")
			if err != nil {
				t.Fatal(err)
			}
			store.miningBlockFailureInjector = func(at string) error {
				if at == "after_state_change" {
					return errors.New("injected")
				}
				return nil
			}
			if action == miningblock.ActionFinalize {
				_, err = service.Finalize(ctx, open.BlockID)
			} else {
				_, err = service.Cancel(ctx, open.BlockID)
			}
			if err == nil {
				t.Fatal("transition committed")
			}
			store.miningBlockFailureInjector = nil
			block, err := store.LoadMiningBlock(ctx, open.BlockID)
			if err != nil || block.Status != miningblock.StatusOpen {
				t.Fatalf("state=%+v err=%v", block, err)
			}
			pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
			if err != nil || pool.Pool.TotalReserved != 10 || pool.Pool.RemainingCapacity != 0 {
				t.Fatalf("pool=%+v err=%v", pool.Pool, err)
			}
			if action == miningblock.ActionFinalize {
				_, err = service.Finalize(ctx, open.BlockID)
			} else {
				_, err = service.Cancel(ctx, open.BlockID)
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestG18ReservedRefundCreatesDebtAndCancelRepaysIt(t *testing.T) {
	store, service, contributions, spend := g18Fund(t, 10)
	ctx := context.Background()
	open, err := service.Create(ctx, "g18-refund")
	if err != nil {
		t.Fatal(err)
	}
	first, err := contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-refund-10", 10))
	if err != nil {
		t.Fatal(err)
	}
	pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
	if err != nil || pool.Pool.TotalRefundedSpend != 10 || pool.Pool.TotalReserved != 10 || pool.Pool.RemainingCapacity != 0 || pool.Pool.RecoveryDebt != 10 || pool.Pool.TotalEmissionCapacity != 0 {
		t.Fatalf("debt pool=%+v err=%v", pool.Pool, err)
	}
	replayed, err := contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-refund-10", 10))
	if err != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatalf("refund replay=%+v err=%v", replayed, err)
	}
	if _, err = service.Cancel(ctx, open.BlockID); err != nil {
		t.Fatal(err)
	}
	pool, err = emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
	if err != nil || pool.Pool.TotalRefundedSpend != 10 || pool.Pool.TotalEmissionCapacity != 0 || pool.Pool.RemainingCapacity != 0 || pool.Pool.RecoveryDebt != 0 || pool.Pool.TotalReserved != 0 {
		t.Fatalf("refunded pool=%+v err=%v", pool.Pool, err)
	}
}

func TestG18RuleFailClosedAndExistingNineToTenMigration(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	old := fstest.MapFS{}
	for version := 1; version <= 9; version++ {
		entries, err := fs.ReadDir(migrations, "migrations")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "000"+string(rune('0'+version))+"_") {
				data, err := fs.ReadFile(migrations, "migrations/"+entry.Name())
				if err != nil {
					t.Fatal(err)
				}
				old["migrations/"+entry.Name()] = &fstest.MapFile{Data: data}
			}
		}
	}
	if len(old) != 9 {
		t.Fatalf("old migration count=%d", len(old))
	}
	if err := store.MigrateFS(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var versionCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&versionCount); err != nil || versionCount != 10 {
		t.Fatalf("versions=%d err=%v", versionCount, err)
	}
	for _, rule := range []string{"", "PROD_UNKNOWN", miningblock.DevelopmentRuleVersion} {
		production := rule == miningblock.DevelopmentRuleVersion
		service := miningblock.NewService(store, rule, production)
		if _, err := service.Create(ctx, "g18-closed"); !errors.Is(err, miningblock.ErrUnknownRule) {
			t.Fatalf("rule=%q production=%v err=%v", rule, production, err)
		}
	}
	if _, err := store.CreateMiningBlock(ctx, "g18-invalid", "UNKNOWN"); !errors.Is(err, miningblock.ErrUnknownRule) {
		t.Fatalf("direct unknown rule=%v", err)
	}
}

func TestG18MigrationBackfillsExistingG17HistoryWithoutRewritingIt(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	old := fstest.MapFS{}
	files, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "0010_") {
			continue
		}
		data, err := fs.ReadFile(migrations, "migrations/"+file.Name())
		if err != nil {
			t.Fatal(err)
		}
		old["migrations/"+file.Name()] = &fstest.MapFile{Data: data}
	}
	if len(old) != 9 {
		t.Fatalf("old migration count=%d", len(old))
	}
	if err := store.MigrateFS(ctx, old); err != nil {
		t.Fatal(err)
	}
	ledgerService := ledger.NewService(store, ledger.Options{})
	for _, account := range []struct{ id, owner string }{
		{"pg-fb-a", "pg-player-a"},
		{"pg-fb-system", "pg-system-spend"},
		{postgresFixtureLiabilityAccountID, "pg-test-fixture-liability"},
	} {
		if account.id == "pg-fb-a" {
			_, err = ledgerService.CreatePlayerAccount(ctx, account.id, account.owner)
		} else {
			_, err = ledgerService.CreateSystemAccount(ctx, account.id, account.owner)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	postgresCredit(t, postgresLedgerFixture{store: store, service: ledgerService}, "pg-fb-a", 1000, "g18-upgrade")
	contributions := contribution.NewService(store)
	if _, err = contributions.CreateAccount(ctx, "pg-player-a"); err != nil {
		t.Fatal(err)
	}
	spend, err := systemspend.NewService(store, g15Registry(t)).Post(ctx, g15Intent("g18-existing-nine-spend", 30))
	if err != nil {
		t.Fatal(err)
	}
	// Persist a valid G17 emission using the historical 0009 schema. The
	// current G17 writer requires the new 0010 debt column, so it cannot be
	// used before the migration under test.
	const historicalEntryID = "g18-existing-nine-emission-entry"
	const historicalReceiptID = "g18-existing-nine-emission-receipt"
	if _, err = store.pool.Exec(ctx, `INSERT INTO black_iron_emission_entries(entry_id,source_type,source_id,spend_id,eligible_spend_amount,emission_amount,rule_version,created_at,pool_revision)
		VALUES($1,'G15_ELIGIBLE_SYSTEM_SPEND',$2,$2,30,30,$3,now(),1)`, historicalEntryID, spend.ID, emission.DevelopmentRuleVersion); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO black_iron_emission_receipts(receipt_id,entry_id,source_id,eligible_spend,emission_added,pool_before,pool_after,remaining_capacity,rule_version,created_at)
		SELECT $1,entry_id,source_id,30,30,0,30,30,rule_version,created_at FROM black_iron_emission_entries WHERE entry_id=$2`, historicalReceiptID, historicalEntryID); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE black_iron_emission_pools SET total_eligible_spend_observed=30,total_emission_capacity=30,remaining_capacity=30,rule_version=$1,revision=1,updated_at=now() WHERE pool_id='GLOBAL'`, emission.DevelopmentRuleVersion); err != nil {
		t.Fatal(err)
	}
	var beforeEntries, beforeReceipts string
	if err = store.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(e) ORDER BY e.pool_revision)::text,'[]') FROM black_iron_emission_entries e`).Scan(&beforeEntries); err != nil {
		t.Fatal(err)
	}
	if err = store.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(r) ORDER BY r.receipt_id)::text,'[]') FROM black_iron_emission_receipts r`).Scan(&beforeReceipts); err != nil {
		t.Fatal(err)
	}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var afterEntries, afterReceipts string
	if err = store.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(e) ORDER BY e.pool_revision)::text,'[]') FROM black_iron_emission_entries e`).Scan(&afterEntries); err != nil {
		t.Fatal(err)
	}
	if err = store.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(r) ORDER BY r.receipt_id)::text,'[]') FROM black_iron_emission_receipts r`).Scan(&afterReceipts); err != nil {
		t.Fatal(err)
	}
	if beforeEntries != afterEntries || beforeReceipts != afterReceipts {
		t.Fatal("migration rewrote G17 history")
	}
	snapshot, err := store.SnapshotBlackIronEmission(ctx)
	if err != nil || len(snapshot.Entries) != 1 || len(snapshot.RecoveryEntries) != 1 || snapshot.Pool.RecoveryDebt != 0 {
		t.Fatalf("upgraded snapshot=%+v err=%v", snapshot, err)
	}
	for i, entry := range snapshot.Entries {
		recovery := snapshot.RecoveryEntries[i]
		if recovery.EmissionEntryID == nil || *recovery.EmissionEntryID != entry.ID || recovery.PoolRevision != entry.PoolRevision || !recovery.CreatedAt.Equal(entry.CreatedAt) {
			t.Fatalf("historical backfill[%d]=%+v entry=%+v", i, recovery, entry)
		}
	}
	report, err := store.ReconcileBlackIronEmission(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("upgraded reconciliation=%+v err=%v", report, err)
	}
}

func TestG18ConcurrentDuplicateCommandsAndCorruptionDetection(t *testing.T) {
	store, service, _, _ := g18Fund(t, 10)
	ctx := context.Background()
	const count = 8
	var wg sync.WaitGroup
	created := make(chan miningblock.Receipt, count)
	failures := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := service.Create(ctx, "g18-same-command")
			created <- r
			failures <- err
		}()
	}
	wg.Wait()
	close(created)
	close(failures)
	var first miningblock.Receipt
	for r := range created {
		if first.ID == "" {
			first = r
		} else if !reflect.DeepEqual(first, r) {
			t.Fatalf("concurrent create replay differs: %+v / %+v", first, r)
		}
	}
	for err := range failures {
		if err != nil {
			t.Fatalf("concurrent create=%v", err)
		}
	}
	if first.BlockHeight != 1 {
		t.Fatalf("height=%d", first.BlockHeight)
	}
	finalized := make(chan miningblock.Receipt, count)
	finalizeErrors := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := service.Finalize(ctx, first.BlockID)
			finalized <- r
			finalizeErrors <- err
		}()
	}
	wg.Wait()
	close(finalized)
	close(finalizeErrors)
	var terminal miningblock.Receipt
	for r := range finalized {
		if terminal.ID == "" {
			terminal = r
		} else if !reflect.DeepEqual(terminal, r) {
			t.Fatalf("concurrent finalize replay differs")
		}
	}
	for err := range finalizeErrors {
		if err != nil {
			t.Fatalf("concurrent finalize=%v", err)
		}
	}
	if terminal.BlockStatus != miningblock.StatusFinalized {
		t.Fatalf("terminal=%+v", terminal)
	}
	pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
	if err != nil || pool.Pool.TotalReserved != 10 || pool.Pool.TotalDistributed != 0 {
		t.Fatalf("pool=%+v err=%v", pool.Pool, err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE mining_block_reservations SET amount=9 WHERE block_id=$1`, first.BlockID); err != nil {
		t.Fatal(err)
	}
	report, err := store.ReconcileMiningBlocks(ctx)
	if err != nil || report.Balanced {
		t.Fatalf("reservation corruption undetected: %+v err=%v", report, err)
	}
}

func TestG18ConcurrentDuplicateCancelAndInputGuards(t *testing.T) {
	store, service, _, _ := g18Fund(t, 10)
	ctx := context.Background()
	open, err := service.Create(ctx, "g18-cancel-duplicates")
	if err != nil {
		t.Fatal(err)
	}
	const count = 8
	var wg sync.WaitGroup
	receipts := make(chan miningblock.Receipt, count)
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); r, e := service.Cancel(ctx, open.BlockID); receipts <- r; errs <- e }()
	}
	wg.Wait()
	close(receipts)
	close(errs)
	var first miningblock.Receipt
	for r := range receipts {
		if first.ID == "" {
			first = r
		} else if !reflect.DeepEqual(first, r) {
			t.Fatal("cancel replay differs")
		}
	}
	for e := range errs {
		if e != nil {
			t.Fatalf("cancel error=%v", e)
		}
	}
	pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
	if err != nil || pool.Pool.TotalReserved != 0 || pool.Pool.RemainingCapacity != 10 {
		t.Fatalf("pool=%+v err=%v", pool.Pool, err)
	}
	for _, reward := range []int64{0, -1} {
		_, err = store.pool.Exec(ctx, `INSERT INTO mining_blocks(block_id,block_height,create_command_id,status,rule_version,started_at,scheduled_end_at,reward_reserved,pool_revision_at_reservation,created_at,updated_at) VALUES($1,99,$2,'OPEN',$3,now(),now()+interval '1 minute',$4,1,now(),now())`, "fake-block-"+string(rune('a'-reward)), "fake-command-"+string(rune('a'-reward)), miningblock.DevelopmentRuleVersion, reward)
		if err == nil {
			t.Fatalf("accepted nonpositive reward %d", reward)
		}
	}
	if _, err = service.Finalize(ctx, "fake-block-id"); !errors.Is(err, miningblock.ErrNotFound) {
		t.Fatalf("fake block=%v", err)
	}
	if _, err = service.Create(ctx, ""); !errors.Is(err, miningblock.ErrInvalidCommand) {
		t.Fatalf("empty command=%v", err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE black_iron_emission_pools SET revision=$1 WHERE pool_id='GLOBAL'`, int64(math.MaxInt64)); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Create(ctx, "g18-overflow"); !errors.Is(err, miningblock.ErrOverflow) {
		t.Fatalf("revision overflow=%v", err)
	}
}

func TestG18FinalizedReservationRetainsDebtOnUpstreamRefund(t *testing.T) {
	store, service, contributions, spend := g18Fund(t, 10)
	ctx := context.Background()
	opened, err := service.Create(ctx, "g18-finalized-refund")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Finalize(ctx, opened.BlockID); err != nil {
		t.Fatal(err)
	}
	if _, err = contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-finalized-refund-request", 10)); err != nil {
		t.Fatalf("finalized refund=%v", err)
	}
	if _, err = service.Cancel(ctx, opened.BlockID); !errors.Is(err, miningblock.ErrConflict) {
		t.Fatalf("cancel finalized=%v", err)
	}
	pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
	if err != nil || pool.Pool.TotalReserved != 10 || pool.Pool.TotalRefundedSpend != 10 || pool.Pool.TotalDistributed != 0 || pool.Pool.RecoveryDebt != 10 || pool.Pool.TotalEmissionCapacity != 0 {
		t.Fatalf("pool=%+v err=%v", pool.Pool, err)
	}
}
