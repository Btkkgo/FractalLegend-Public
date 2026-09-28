package postgres

import (
	"context"
	"testing"

	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/systemspend"
)

// The G18 OPEN already removed R from Remaining. A TEST accounting transfer
// must only move it from Reserved to Distributed, with no player inventory.
func TestG21P0SyntheticDistributionDoesNotDeductRemainingTwice(t *testing.T) {
	s, service := g20Fund(t, 20)
	ctx := context.Background()
	receipt, err := service.Create(ctx, "g21-p0-accounting")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Finalize(ctx, receipt.BlockID); err != nil {
		t.Fatal(err)
	}
	var source string
	if err = s.pool.QueryRow(ctx, `SELECT reservation_source_id FROM mining_reservation_instance_bindings WHERE display_block_id=$1`, receipt.BlockID).Scan(&source); err != nil {
		t.Fatal(err)
	}
	if err = s.transferReservedToDistributedPrerequisiteTEST(ctx, source); err != nil {
		t.Fatal(err)
	}
	var capacity, reserved, distributed, remaining, debt int64
	if err = s.pool.QueryRow(ctx, `SELECT total_emission_capacity,total_reserved,total_distributed,remaining_capacity,recovery_debt
		FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).
		Scan(&capacity, &reserved, &distributed, &remaining, &debt); err != nil {
		t.Fatal(err)
	}
	if capacity != 20 || reserved != 0 || distributed != 10 || remaining != 10 || debt != 0 {
		t.Fatalf("double deduction or bad conservation: C/S/D/M/H=%d/%d/%d/%d/%d", capacity, reserved, distributed, remaining, debt)
	}
	report, err := s.ReconcileBlackIronEmission(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("recovery rebuild=%+v %v", report, err)
	}
}

func TestG21P0DistributedRefundDebtAndRecoveryRebuild(t *testing.T) {
	s, service, contributions, spend := g18Fund(t, 20)
	ctx := context.Background()
	opened, err := service.Create(ctx, "g21-p0-refund-distributed")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Finalize(ctx, opened.BlockID); err != nil {
		t.Fatal(err)
	}
	var source string
	if err = s.pool.QueryRow(ctx, `SELECT reservation_source_id FROM mining_reservation_instance_bindings WHERE display_block_id=$1`, opened.BlockID).Scan(&source); err != nil {
		t.Fatal(err)
	}
	if err = s.transferReservedToDistributedPrerequisiteTEST(ctx, source); err != nil {
		t.Fatal(err)
	}
	if _, err = contributions.RefundSystemSpend(ctx, g15Refund(spend, "g21-p0-distributed-refund-15", 15)); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.SnapshotBlackIronEmission(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p := snapshot.Pool
	if p.TotalEmissionCapacity != 5 || p.TotalReserved != 0 || p.TotalDistributed != 10 || p.RemainingCapacity != 0 || p.RecoveryDebt != 5 {
		t.Fatalf("post-distribution refund C/S/D/M/H=%d/%d/%d/%d/%d", p.TotalEmissionCapacity, p.TotalReserved, p.TotalDistributed, p.RemainingCapacity, p.RecoveryDebt)
	}
	ready, err := s.MiningSettlementPrerequisiteReady(ctx)
	if err != nil || ready {
		t.Fatalf("debt gate=%v %v", ready, err)
	}
	report, err := s.ReconcileBlackIronEmission(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("recovery rebuild=%+v %v", report, err)
	}
	newSpend, err := systemspend.NewService(s, g15Registry(t)).Post(ctx, g15Intent("g21-p0-repay-5", 5))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, newSpend.ID); err != nil {
		t.Fatal(err)
	}
	p, err = func() (emission.Pool, error) { v, e := s.SnapshotBlackIronEmission(ctx); return v.Pool, e }()
	if err != nil || p.RecoveryDebt != 0 || p.RemainingCapacity != 0 || p.TotalDistributed != 10 {
		t.Fatalf("debt repayment=%+v %v", p, err)
	}
	ready, err = s.MiningSettlementPrerequisiteReady(ctx)
	if err != nil || !ready {
		t.Fatalf("repaid gate=%v %v", ready, err)
	}
	report, err = s.ReconcileBlackIronEmission(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("repaid rebuild=%+v %v", report, err)
	}
}
