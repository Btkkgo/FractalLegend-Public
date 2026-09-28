package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"fractallegend/game-server/internal/miningpower"
	"fractallegend/game-server/internal/miningreward"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const settlementAllocationVersion = "TEST_G21_LARGEST_REMAINDER_CHARACTER_BYTES_V1"
const settlementConversionVersion = "TEST_G21_ORE_1_TO_1_V1"
const settlementIssuanceVersion = "G21_TEST_ISSUANCE_V1"
const settlementMaxActivities = 100000
const settlementMaxSealBytes = 64 << 20
const settlementMaxReceiptBytes = 4 << 20
const settlementDefaultStatementTimeout = 60 * time.Second

var ErrMiningRewardConflict = errors.New("mining reward settlement conflict")
var ErrMiningRewardDebt = errors.New("mining reward settlement paused by recovery debt")
var ErrMiningRewardInvariant = errors.New("mining reward settlement invariant failed")

// Preserve the invariant error class for existing first-execution callers.
var ErrMiningRewardBindingChanged = fmt.Errorf("BINDING_CHANGED: %w", ErrMiningRewardInvariant)

func (s *Store) settlementBarrier(point string) error {
	if s.settlementFailureInjector != nil {
		return s.settlementFailureInjector(point)
	}
	return nil
}

type MiningRewardSettlementCommand struct {
	CommandID, BlockInstanceID, ExpectedSealDigest string
}

type MiningRewardSettlementReceipt struct {
	SchemaVersion, Status, SettlementID, CommandID, CommandFingerprint, BlockInstanceID string
	DisplayBlockID, ReservationSourceID, ReservationBindingID, ReservationBindingDigest string
	SealID, SealDigest, WindowID, G18RuleVersion, G20RuleVersion                        string
	AllocationVersion, ConversionVersion                                                string
	ActivityCount, ParticipantCount, PositiveParticipantCount                           int64
	TotalPower, ReservedAmount                                                          int64
	PoolNetCapacityBefore, PoolNetCapacityAfter                                         int64
	TotalOre, PoolReservedBefore, PoolReservedAfter                                     int64
	PoolDistributedBefore, PoolDistributedAfter                                         int64
	PoolRemainingBefore, PoolRemainingAfter                                             int64
	PoolDebtBefore, PoolDebtAfter, PoolRevisionBefore, PoolRevisionAfter                int64
	Grants                                                                              []MiningRewardGrant
	CreatedAt                                                                           time.Time
	CanonicalDigest                                                                     string `json:"-"`
}

type MiningRewardGrant struct {
	PlayerID, AccountID, CharacterID, BeneficiaryBindingID string
	Power, Quotient, Remainder, Bonus, Quantity            int64
	IssuanceID, InventoryInstanceID                        string
	InventoryRevisionBefore, InventoryRevisionAfter        int64
	ItemRevisionBefore, ItemRevisionAfter                  int64
}

// checkedMiningRewardQuantity checks the inventory storage bound without
// constructing or changing economic authority. Zero remains a grant only.
func checkedMiningRewardQuantity(quantity int64) (int32, error) {
	if quantity < 0 || quantity > math.MaxInt32 {
		return 0, ErrMiningRewardInvariant
	}
	return int32(quantity), nil
}

func rewardHash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func miningRewardFingerprint(cmd MiningRewardSettlementCommand, seal miningpower.SettlementInputSeal, bindingDigest string) (string, error) {
	type parameters struct {
		TieBreak              string      `json:"tieBreak"`
		ConversionNumerator   quotedInt64 `json:"conversionNumerator"`
		ConversionDenominator quotedInt64 `json:"conversionDenominator"`
		MaxParticipants       quotedInt64 `json:"maxParticipants"`
		MaxActivities         quotedInt64 `json:"maxActivities"`
		MaxSealBytes          quotedInt64 `json:"maxSealBytes"`
		MaxReceiptBytes       quotedInt64 `json:"maxReceiptBytes"`
	}
	plan := struct {
		SchemaVersion            string     `json:"schemaVersion"`
		Operation                string     `json:"operation"`
		BlockInstanceID          string     `json:"blockInstanceID"`
		ExpectedSealDigest       string     `json:"expectedSealDigest"`
		ReservationBindingDigest string     `json:"reservationBindingDigest"`
		BeneficiaryBindingDigest string     `json:"beneficiaryBindingDigest"`
		G20RuleVersion           string     `json:"g20RuleVersion"`
		AllocationVersion        string     `json:"allocationRuleVersion"`
		ConversionVersion        string     `json:"oreConversionRuleVersion"`
		AllocationParameters     parameters `json:"allocationParameters"`
	}{"G21_V1", "TEST_SETTLE", cmd.BlockInstanceID, cmd.ExpectedSealDigest,
		bindingDigest, seal.BeneficiaryBindingDigest, seal.G20RuleVersion,
		settlementAllocationVersion, settlementConversionVersion,
		parameters{"CHARACTER_ID_BYTES_ASC", 1, 1, miningpower.TestParticipantCap,
			settlementMaxActivities, settlementMaxSealBytes, settlementMaxReceiptBytes}}
	raw, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	return rewardHash(raw), nil
}

func loadMiningRewardReceiptTx(ctx context.Context, tx pgx.Tx, commandID string) (MiningRewardSettlementReceipt, error) {
	return loadMiningRewardReceiptValidatedTx(ctx, tx, commandID, true)
}

// Audits include current projections; completed replay validates immutable history only.
func loadMiningRewardReceiptValidatedTx(ctx context.Context, tx pgx.Tx, commandID string, auditCurrent bool) (MiningRewardSettlementReceipt, error) {
	var result MiningRewardSettlementReceipt
	var raw []byte
	var id, instance, source, sealID, sealDigest, digest string
	var total int64
	var createdAt time.Time
	err := tx.QueryRow(ctx, `SELECT settlement_id,block_instance_id,reservation_source_id,seal_id,seal_digest,total_ore,canonical_bytes,canonical_digest,created_at
		FROM mining_reward_settlement_receipts WHERE command_id=$1`, commandID).
		Scan(&id, &instance, &source, &sealID, &sealDigest, &total, &raw, &digest, &createdAt)
	if err != nil {
		return result, err
	}
	result, err = decodeCanonicalMiningRewardReceipt(raw)
	if err != nil || len(raw) > settlementMaxReceiptBytes || rewardHash(raw) != digest ||
		result.CanonicalDigest != "" || result.Status != "COMPLETED" ||
		result.SettlementID != id || result.CommandID != commandID ||
		result.BlockInstanceID != instance || result.ReservationSourceID != source || result.TotalOre != total ||
		result.SealID != sealID || result.SealDigest != sealDigest || !result.CreatedAt.Equal(createdAt) {
		return MiningRewardSettlementReceipt{}, ErrMiningRewardInvariant
	}
	result.CanonicalDigest = digest
	if err = validateMiningRewardReceiptTx(ctx, tx, result, auditCurrent); err != nil {
		return MiningRewardSettlementReceipt{}, err
	}
	return result, nil
}

func validateMiningRewardReceiptTx(ctx context.Context, tx pgx.Tx, receipt MiningRewardSettlementReceipt, auditCurrent bool) error {
	if receipt.SchemaVersion != "G21_V1" || receipt.AllocationVersion != settlementAllocationVersion ||
		receipt.ConversionVersion != settlementConversionVersion || receipt.TotalOre <= 0 ||
		receipt.TotalOre != receipt.ReservedAmount || receipt.ParticipantCount != int64(len(receipt.Grants)) ||
		receipt.TotalPower <= 0 || receipt.PoolReservedBefore-receipt.TotalOre != receipt.PoolReservedAfter ||
		receipt.PoolDistributedBefore+receipt.TotalOre != receipt.PoolDistributedAfter ||
		receipt.PoolNetCapacityBefore != receipt.PoolNetCapacityAfter ||
		receipt.PoolRemainingBefore != receipt.PoolRemainingAfter || receipt.PoolDebtBefore != receipt.PoolDebtAfter ||
		receipt.PoolRevisionBefore+1 != receipt.PoolRevisionAfter {
		return ErrMiningRewardInvariant
	}
	var commandFingerprint, commandSealDigest, commandInstance, commandSettlement, commandStatus string
	if err := tx.QueryRow(ctx, `SELECT fingerprint,seal_digest,block_instance_id,COALESCE(settlement_id,''),status FROM mining_reward_commands WHERE command_id=$1`,
		receipt.CommandID).Scan(&commandFingerprint, &commandSealDigest, &commandInstance, &commandSettlement, &commandStatus); err != nil ||
		commandFingerprint != receipt.CommandFingerprint || commandSealDigest != receipt.SealDigest ||
		commandInstance != receipt.BlockInstanceID || commandSettlement != receipt.SettlementID || commandStatus != "COMPLETED" {
		return ErrMiningRewardInvariant
	}
	seal, err := loadMiningPowerSealTx(ctx, tx, receipt.BlockInstanceID)
	if err != nil || seal.CanonicalDigest != receipt.SealDigest || seal.SealID != receipt.SealID ||
		seal.DisplayBlockID != receipt.DisplayBlockID || seal.WindowIdentity != receipt.WindowID ||
		seal.G18RuleVersion != receipt.G18RuleVersion || seal.G20RuleVersion != receipt.G20RuleVersion ||
		int64(seal.ActivityCount) != receipt.ActivityCount || int64(seal.ParticipantCount) != receipt.ParticipantCount ||
		seal.TotalValidMiningPower != receipt.TotalPower ||
		seal.G18ReservationSourceID != receipt.ReservationSourceID ||
		seal.G18ReservationEvidenceDigest != receipt.ReservationBindingDigest {
		return ErrMiningRewardInvariant
	}
	if auditCurrent {
		var bindingID, bindingDigest, bindingSource string
		var bindingAmount int64
		if err := tx.QueryRow(ctx, `SELECT binding_id,evidence_digest,reservation_source_id,reservation_amount
		FROM mining_reservation_instance_bindings WHERE block_instance_id=$1`, receipt.BlockInstanceID).
			Scan(&bindingID, &bindingDigest, &bindingSource, &bindingAmount); err != nil ||
			bindingID != receipt.ReservationBindingID || bindingDigest != receipt.ReservationBindingDigest ||
			bindingSource != receipt.ReservationSourceID || bindingAmount != receipt.ReservedAmount {
			return ErrMiningRewardInvariant
		}
	}
	var count int
	var total int64
	err = tx.QueryRow(ctx, `SELECT count(*),COALESCE(sum(quantity),0) FROM mining_reward_grants
		WHERE settlement_id=$1`, receipt.SettlementID).Scan(&count, &total)
	if err != nil || count != len(receipt.Grants) || total != receipt.TotalOre {
		return ErrMiningRewardInvariant
	}
	var consumptionID string
	var consumptionAmount, consumptionRevision int64
	err = tx.QueryRow(ctx, `SELECT consumption_id,amount,pool_revision FROM mining_reward_reservation_consumptions
		WHERE settlement_id=$1 AND block_instance_id=$2 AND reservation_source_id=$3`,
		receipt.SettlementID, receipt.BlockInstanceID, receipt.ReservationSourceID).
		Scan(&consumptionID, &consumptionAmount, &consumptionRevision)
	if err != nil || consumptionAmount != receipt.TotalOre || consumptionRevision != receipt.PoolRevisionAfter {
		return ErrMiningRewardInvariant
	}
	var recoveryCount, stateCount, commandCount int
	err = tx.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM black_iron_emission_recovery_entries WHERE
		 settlement_consumption_id=$1 AND source_type='G21_TEST_MINING_SETTLEMENT' AND
		 reserved_delta=$4 AND distributed_delta=$5 AND pool_revision=$6),
		(SELECT count(*) FROM mining_reward_settlement_states WHERE block_instance_id=$2 AND
		 state='COMPLETED' AND settlement_id=$3 AND command_id=$7),
		(SELECT count(*) FROM mining_reward_commands WHERE command_id=$7 AND settlement_id=$3 AND status='COMPLETED')`,
		consumptionID, receipt.BlockInstanceID, receipt.SettlementID, -receipt.TotalOre,
		receipt.TotalOre, receipt.PoolRevisionAfter, receipt.CommandID).
		Scan(&recoveryCount, &stateCount, &commandCount)
	if err != nil || recoveryCount != 1 || stateCount != 1 || commandCount != 1 {
		return ErrMiningRewardInvariant
	}
	positive := 0
	for i, expected := range receipt.Grants {
		if i > 0 && receipt.Grants[i-1].CharacterID >= expected.CharacterID {
			return ErrMiningRewardInvariant
		}
		var player, account, character, lotID, itemID string
		var power, quotient, remainder, bonus, quantity int64
		err = tx.QueryRow(ctx, `SELECT player_id,account_id,character_id,power,quotient,remainder,bonus,quantity,
			COALESCE(issuance_id,''),COALESCE(inventory_instance_id,'') FROM mining_reward_grants
			WHERE settlement_id=$1 AND character_id=$2`, receipt.SettlementID, expected.CharacterID).
			Scan(&player, &account, &character, &power, &quotient, &remainder, &bonus, &quantity, &lotID, &itemID)
		if err != nil || player != expected.PlayerID || account != expected.AccountID ||
			character != expected.CharacterID || power != expected.Power || quotient != expected.Quotient ||
			remainder != expected.Remainder || bonus != expected.Bonus || quantity != expected.Quantity ||
			lotID != expected.IssuanceID || itemID != expected.InventoryInstanceID {
			return ErrMiningRewardInvariant
		}
		if auditCurrent {
			var expectedBindingID string
			if err := tx.QueryRow(ctx, `SELECT min(binding_id) FROM mining_power_beneficiary_bindings
			WHERE block_instance_id=$1 AND character_id=$2 AND account_id=$3 AND player_id=$4`,
				receipt.BlockInstanceID, character, account, player).Scan(&expectedBindingID); err != nil ||
				expectedBindingID != expected.BeneficiaryBindingID {
				return ErrMiningRewardInvariant
			}
		}
		if quantity == 0 {
			if lotID != "" || itemID != "" || expected.InventoryRevisionBefore != expected.InventoryRevisionAfter ||
				expected.ItemRevisionBefore != 0 || expected.ItemRevisionAfter != 0 {
				return ErrMiningRewardInvariant
			}
			continue
		}
		positive++
		if expected.ItemRevisionBefore != 0 || expected.ItemRevisionAfter != 1 ||
			expected.InventoryRevisionAfter != expected.InventoryRevisionBefore+1 {
			return ErrMiningRewardInvariant
		}
		var sourceKey string
		err = tx.QueryRow(ctx, `SELECT source_key FROM mining_reward_issuance_lots WHERE issuance_id=$1`, lotID).Scan(&sourceKey)
		if err != nil {
			return ErrMiningRewardInvariant
		}
		lot, e := loadMiningRewardIssuanceTx(ctx, tx, sourceKey)
		if e != nil || lot.ID != lotID || lot.SettlementID != receipt.SettlementID ||
			lot.BlockInstanceID != receipt.BlockInstanceID || lot.CharacterID != character ||
			int64(lot.Quantity) != quantity || lot.Status != "G21_SETTLED" || lot.RuleVersion != settlementIssuanceVersion {
			return ErrMiningRewardInvariant
		}
		lotDigest, e := miningRewardSourceDigest(lot)
		if e != nil || lotDigest != lot.SourceDigest {
			return ErrMiningRewardInvariant
		}
		var projectionCount int
		// Historical projection is immutable issuance evidence, independent of the
		// current physical stack, owner revision and catalog.
		if auditCurrent {
			err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_reward_inventory_projections p
			JOIN character_inventory_items i ON i.instance_id=p.instance_id
			JOIN characters c ON c.id=p.character_id
            JOIN black_iron_identity_aliases a ON a.definition_id=i.definition_id
			WHERE p.issuance_id=$1 AND p.instance_id=$2 AND p.character_id=$3 AND
			p.block_instance_id=$4 AND p.quantity=$5 AND p.definition_id=$6 AND
			p.revision_before=$7 AND p.revision_after=$8 AND c.revision>=p.revision_after AND
			i.character_id=p.character_id AND i.definition_id=p.definition_id AND i.quantity=p.quantity AND
            i.item_revision=$9 AND c.account_id=$10 AND i.legacy_id=a.legacy_id AND
            i.name='黑铁矿石' AND i.item_type='MATERIAL' AND i.location='INVENTORY' AND i.equipment_slot IS NULL AND
            NOT EXISTS(SELECT 1 FROM black_iron_migration_assets m WHERE m.instance_id=i.instance_id)`,
				lotID, itemID, character, receipt.BlockInstanceID, quantity, lot.MaterialDefinitionID,
				expected.InventoryRevisionBefore, expected.InventoryRevisionAfter, expected.ItemRevisionAfter, account).Scan(&projectionCount)
		} else {
			err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_reward_inventory_projections
                WHERE issuance_id=$1 AND instance_id=$2 AND character_id=$3 AND
                block_instance_id=$4 AND quantity=$5 AND definition_id=$6 AND
                revision_before=$7 AND revision_after=$8`, lotID, itemID, character,
				receipt.BlockInstanceID, quantity, lot.MaterialDefinitionID,
				expected.InventoryRevisionBefore, expected.InventoryRevisionAfter).Scan(&projectionCount)
		}
		if err != nil || projectionCount != 1 {
			return ErrMiningRewardInvariant
		}
	}
	var lotCount, projectionCount int
	err = tx.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM mining_reward_issuance_lots WHERE settlement_id=$1 AND status='G21_SETTLED'),
		(SELECT count(*) FROM mining_reward_inventory_projections p
		 JOIN mining_reward_issuance_lots l ON l.issuance_id=p.issuance_id
		 WHERE l.settlement_id=$1 AND l.status='G21_SETTLED')`, receipt.SettlementID).
		Scan(&lotCount, &projectionCount)
	if err != nil || lotCount != positive || projectionCount != positive || receipt.PositiveParticipantCount != int64(positive) {
		return ErrMiningRewardInvariant
	}
	return nil
}

// SettleMiningRewardTEST has no route and requires an isolated TEST database.
// All payout effects are committed by one transaction; never call the separate
// P0 synthetic distribution/issuance helpers from this operation.
func (s *Store) SettleMiningRewardTEST(ctx context.Context, cmd MiningRewardSettlementCommand) (MiningRewardSettlementReceipt, error) {
	var empty MiningRewardSettlementReceipt
	if s == nil || s.pool == nil || !validMiningBlockID(cmd.CommandID) ||
		!miningpower.ValidID(cmd.BlockInstanceID, 128) || len(cmd.ExpectedSealDigest) != 64 {
		return empty, miningpower.ErrInvalidInput
	}
	var dbName string
	if err := s.pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName); err != nil {
		return empty, err
	}
	if !strings.HasPrefix(dbName, "fractal_g9_test_") {
		return empty, miningpower.ErrUnavailable
	}
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		result, err := s.settleMiningRewardAttempt(ctx, cmd)
		if err == nil {
			return result, nil
		}
		last = err
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || (pgErr.Code != "40001" && pgErr.Code != "40P01") {
			return empty, err
		}
		if ctx.Err() != nil {
			return empty, ctx.Err()
		}
	}
	return empty, fmt.Errorf("mining reward retry exhausted: %w", last)
}

func (s *Store) settleMiningRewardAttempt(ctx context.Context, cmd MiningRewardSettlementCommand) (MiningRewardSettlementReceipt, error) {
	var empty MiningRewardSettlementReceipt
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return empty, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL lock_timeout='5s'`); err != nil {
		return empty, err
	}
	statementTimeout := settlementDefaultStatementTimeout
	if s.settlementStatementTimeout >= time.Millisecond && s.settlementStatementTimeout < statementTimeout {
		statementTimeout = s.settlementStatementTimeout
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('statement_timeout',$1,true)`, fmt.Sprintf("%dms", statementTimeout.Milliseconds())); err != nil {
		return empty, err
	}
	// Transaction-scoped command identity lock is first. Hash collisions only
	// serialize unrelated commands; persisted CommandID uniqueness is authority.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, cmd.CommandID); err != nil {
		return empty, err
	}
	var oldFingerprint, oldStatus, oldInstance, oldSeal, oldSettlement string
	err = tx.QueryRow(ctx, `SELECT fingerprint,status,block_instance_id,seal_digest,COALESCE(settlement_id,'')
        FROM mining_reward_commands WHERE command_id=$1`, cmd.CommandID).
		Scan(&oldFingerprint, &oldStatus, &oldInstance, &oldSeal, &oldSettlement)
	knownCommand := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return empty, err
	}
	if knownCommand && (oldInstance != cmd.BlockInstanceID || oldSeal != cmd.ExpectedSealDigest) {
		return empty, ErrMiningRewardConflict
	}
	if knownCommand && oldStatus == "COMPLETED" {
		// Fingerprint inputs come from the immutable historical seal, never
		// reconstructed current bindings, rule catalog or inventory.
		historicalSeal, e := loadMiningPowerSealTx(ctx, tx, oldInstance)
		if e != nil {
			return empty, e
		}
		if historicalSeal.CanonicalDigest != oldSeal {
			return empty, ErrMiningRewardInvariant
		}
		fingerprint, e := miningRewardFingerprint(cmd, historicalSeal, historicalSeal.G18ReservationEvidenceDigest)
		if e != nil {
			return empty, e
		}
		if fingerprint != oldFingerprint {
			return empty, ErrMiningRewardConflict
		}
		prior, e := loadMiningRewardReceiptValidatedTx(ctx, tx, cmd.CommandID, false)
		if e != nil {
			return empty, e
		}
		if prior.CommandFingerprint != oldFingerprint || prior.BlockInstanceID != oldInstance ||
			prior.SettlementID != oldSettlement || prior.SealDigest != oldSeal {
			return empty, ErrMiningRewardInvariant
		}
		return prior, tx.Commit(ctx)
	}
	seal, err := loadMiningPowerSealTx(ctx, tx, cmd.BlockInstanceID)
	if err != nil {
		return empty, err
	}
	if seal.CanonicalDigest != cmd.ExpectedSealDigest || seal.BlockInstanceID != cmd.BlockInstanceID ||
		seal.ParticipantCount > miningpower.TestParticipantCap || seal.ActivityCount > settlementMaxActivities {
		return empty, ErrMiningRewardConflict
	}
	var bindingID, bindingSource, bindingReceipt, bindingDigest, displayID, bindingRule, bindingVersion string
	var reward int64
	err = tx.QueryRow(ctx, `SELECT binding_id,reservation_source_id,reservation_receipt_id,evidence_digest,display_block_id,g18_rule_version,reservation_amount,binding_version
		FROM mining_reservation_instance_bindings WHERE block_instance_id=$1`, cmd.BlockInstanceID).
		Scan(&bindingID, &bindingSource, &bindingReceipt, &bindingDigest, &displayID, &bindingRule, &reward, &bindingVersion)
	if err != nil {
		return empty, err
	}
	if reward <= 0 || bindingSource != seal.G18ReservationSourceID ||
		bindingDigest != seal.G18ReservationEvidenceDigest || displayID != seal.DisplayBlockID ||
		bindingRule != seal.G18RuleVersion || bindingID == "" || bindingReceipt == "" ||
		bindingVersion != miningpower.ReservationBindingVersion {
		return empty, ErrMiningRewardInvariant
	}
	actualBindingDigest, err := miningpower.ReservationBindingEvidenceDigest(miningpower.ReservationBindingEvidence{
		Version: bindingVersion, InstanceID: cmd.BlockInstanceID, SourceID: bindingSource,
		ReceiptID: bindingReceipt, BlockID: displayID, RuleVersion: bindingRule, Amount: reward,
	})
	if err != nil || actualBindingDigest != bindingDigest {
		return empty, ErrMiningRewardInvariant
	}
	if _, err = validateMiningPowerSealSnapshotTx(ctx, tx, seal); err != nil {
		if errors.Is(err, miningpower.ErrInvariant) {
			return empty, ErrMiningRewardBindingChanged
		}
		return empty, ErrMiningRewardInvariant
	}
	fingerprint, err := miningRewardFingerprint(cmd, seal, bindingDigest)
	if err != nil {
		return empty, err
	}
	if knownCommand {
		if oldFingerprint != fingerprint {
			return empty, ErrMiningRewardConflict
		}
		if oldStatus == "SEALED_NO_ELIGIBLE_POWER" {
			return MiningRewardSettlementReceipt{SchemaVersion: "G21_V1", Status: oldStatus, CommandID: cmd.CommandID,
				BlockInstanceID: cmd.BlockInstanceID, SealID: seal.SealID, SealDigest: seal.CanonicalDigest}, tx.Commit(ctx)
		}
		return empty, ErrMiningRewardInvariant
	}

	// P0 Seal is the durable SEALED authority. The G21 sidecar row is only
	// committed with a terminal command, so an aborted attempt leaves no state.
	_, err = tx.Exec(ctx, `INSERT INTO mining_reward_settlement_states(block_instance_id,seal_id,state,revision)
		VALUES($1,$2,'SEALED',1) ON CONFLICT(block_instance_id) DO NOTHING`, cmd.BlockInstanceID, seal.SealID)
	if err != nil {
		return empty, err
	}
	var state, stateSeal string
	err = tx.QueryRow(ctx, `SELECT state,seal_id FROM mining_reward_settlement_states WHERE block_instance_id=$1 FOR UPDATE`, cmd.BlockInstanceID).
		Scan(&state, &stateSeal)
	if err != nil {
		return empty, err
	}
	if stateSeal != seal.SealID || state != "SEALED" {
		return empty, ErrMiningRewardConflict
	}
	if seal.TotalValidMiningPower == 0 {
		if seal.Eligibility != "NOT_SETTLEMENT_ELIGIBLE" {
			return empty, ErrMiningRewardInvariant
		}
		now := miningreward.CanonicalTime(time.Now())
		if _, err = tx.Exec(ctx, `INSERT INTO mining_reward_commands
			(command_id,block_instance_id,fingerprint,seal_digest,status,created_at)
			VALUES($1,$2,$3,$4,'SEALED_NO_ELIGIBLE_POWER',$5)`,
			cmd.CommandID, cmd.BlockInstanceID, fingerprint, seal.CanonicalDigest, now); err != nil {
			return empty, err
		}
		if _, err = tx.Exec(ctx, `UPDATE mining_reward_settlement_states SET state='SEALED_NO_ELIGIBLE_POWER',
			command_id=$2,revision=2,completed_at=$3 WHERE block_instance_id=$1`,
			cmd.BlockInstanceID, cmd.CommandID, now); err != nil {
			return empty, err
		}
		if err = tx.Commit(ctx); err != nil {
			return empty, err
		}
		return MiningRewardSettlementReceipt{SchemaVersion: "G21_V1", Status: "SEALED_NO_ELIGIBLE_POWER",
			CommandID: cmd.CommandID, BlockInstanceID: cmd.BlockInstanceID, SealID: seal.SealID,
			SealDigest: seal.CanonicalDigest}, nil
	}
	if seal.Eligibility != "SETTLEMENT_INPUT" {
		return empty, ErrMiningRewardInvariant
	}
	pool, err := loadEmissionPoolTx(ctx, tx, true)
	if err != nil {
		return empty, err
	}
	if !emissionRecoveryConserved(pool) {
		return empty, ErrMiningRewardInvariant
	}
	if pool.RecoveryDebt > 0 {
		return empty, ErrMiningRewardDebt
	}
	if pool.TotalReserved < reward || pool.TotalDistributed > math.MaxInt64-reward || pool.Revision == math.MaxInt64 {
		return empty, ErrMiningRewardInvariant
	}
	var blockStatus, reservationStatus, sourceID, receiptID, blockInstance string
	var blockReward, reservationAmount int64
	err = tx.QueryRow(ctx, `SELECT m.status,r.status,e.entry_id,x.receipt_id,m.block_instance_id,m.reward_reserved,r.amount
		FROM mining_blocks m JOIN mining_block_reservations r ON r.block_id=m.block_id
		JOIN mining_block_entries e ON e.block_id=m.block_id AND e.action='OPEN'
		JOIN mining_block_receipts x ON x.entry_id=e.entry_id
		WHERE m.block_id=$1 FOR UPDATE OF m,r`, displayID).
		Scan(&blockStatus, &reservationStatus, &sourceID, &receiptID, &blockInstance, &blockReward, &reservationAmount)
	if err != nil {
		return empty, err
	}
	if blockStatus != "FINALIZED" || reservationStatus != "ACTIVE" ||
		blockInstance != cmd.BlockInstanceID || sourceID != bindingSource || receiptID != bindingReceipt ||
		blockReward != reward || reservationAmount != reward {
		return empty, ErrMiningRewardInvariant
	}
	var used int
	if err = tx.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM mining_prerequisite_distributions WHERE reservation_source_id=$1)+
		(SELECT count(*) FROM mining_reward_reservation_consumptions WHERE reservation_source_id=$1)`, bindingSource).
		Scan(&used); err != nil {
		return empty, err
	}
	if used != 0 {
		return empty, ErrMiningRewardConflict
	}
	var frozen bool
	// The row lock coordinates with G19 migration. The frozen flag controls
	// legacy migration, not whether the reviewed alias can back a new reward.
	if err = tx.QueryRow(ctx, `SELECT frozen FROM black_iron_migration_catalog WHERE singleton FOR SHARE`).Scan(&frozen); err != nil {
		return empty, err
	}
	var definition string
	var legacyID, aliasCount int32
	if err = tx.QueryRow(ctx, `SELECT min(definition_id),min(legacy_id),count(*) FROM black_iron_identity_aliases`).
		Scan(&definition, &legacyID, &aliasCount); err != nil {
		return empty, err
	}
	if aliasCount != 1 || definition == "" {
		return empty, ErrMiningRewardInvariant
	}
	allocation, err := miningreward.AllocateTEST(reward, seal.ParticipantWeights)
	if err != nil || allocation.Power != seal.TotalValidMiningPower || allocation.Total != reward {
		return empty, ErrMiningRewardInvariant
	}
	for _, g := range allocation.Grants {
		if _, err := checkedMiningRewardQuantity(g.Quantity); err != nil {
			return empty, ErrMiningRewardInvariant
		}
	}
	var syntheticCount int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_reward_issuance_lots
		WHERE block_instance_id=$1 AND status='P0_SYNTHETIC_ONLY'`, cmd.BlockInstanceID).Scan(&syntheticCount); err != nil {
		return empty, err
	}
	if syntheticCount != 0 {
		return empty, ErrMiningRewardConflict
	}
	now := miningreward.CanonicalTime(time.Now())
	result := MiningRewardSettlementReceipt{
		SchemaVersion: "G21_V1", Status: "COMPLETED", SettlementID: newContributionID("g21-settlement"),
		CommandID: cmd.CommandID, CommandFingerprint: fingerprint, BlockInstanceID: cmd.BlockInstanceID,
		DisplayBlockID: displayID, ReservationSourceID: bindingSource,
		ReservationBindingID: bindingID, ReservationBindingDigest: bindingDigest,
		SealID: seal.SealID, SealDigest: seal.CanonicalDigest, WindowID: seal.WindowIdentity,
		G18RuleVersion: seal.G18RuleVersion, G20RuleVersion: seal.G20RuleVersion,
		AllocationVersion: settlementAllocationVersion, ConversionVersion: settlementConversionVersion,
		ActivityCount: int64(seal.ActivityCount), ParticipantCount: int64(seal.ParticipantCount),
		TotalPower: seal.TotalValidMiningPower, ReservedAmount: reward,
		PoolNetCapacityBefore: pool.TotalEmissionCapacity, PoolNetCapacityAfter: pool.TotalEmissionCapacity,
		TotalOre: reward, PoolReservedBefore: pool.TotalReserved, PoolReservedAfter: pool.TotalReserved - reward,
		PoolDistributedBefore: pool.TotalDistributed, PoolDistributedAfter: pool.TotalDistributed + reward,
		PoolRemainingBefore: pool.RemainingCapacity, PoolRemainingAfter: pool.RemainingCapacity,
		PoolDebtBefore: pool.RecoveryDebt, PoolDebtAfter: pool.RecoveryDebt,
		PoolRevisionBefore: pool.Revision, PoolRevisionAfter: pool.Revision + 1,
		Grants: make([]MiningRewardGrant, 0, len(allocation.Grants)), CreatedAt: now,
	}
	if err = s.settlementBarrier("before_issuance"); err != nil {
		return empty, err
	}
	// Acquire every owner lock in canonical CharacterID order before writing
	// inventory or history for any owner.
	ownerRevisions := make(map[string]int64, len(allocation.Grants))
	beneficiaryBindings := make(map[string]string, len(allocation.Grants))
	for _, g := range allocation.Grants {
		var ownerAccount string
		var revision int64
		err = tx.QueryRow(ctx, `SELECT account_id,revision FROM characters WHERE id=$1 FOR UPDATE`, g.CharacterID).
			Scan(&ownerAccount, &revision)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return empty, ErrMiningRewardInvariant
			}
			return empty, err
		}
		if ownerAccount != g.AccountID || revision < 1 {
			return empty, ErrMiningRewardInvariant
		}
		var bindingCount int
		var beneficiaryBindingID string
		err = tx.QueryRow(ctx, `SELECT count(*),min(binding_id) FROM mining_power_beneficiary_bindings
			WHERE block_instance_id=$1 AND character_id=$2 AND account_id=$3 AND player_id=$4`,
			cmd.BlockInstanceID, g.CharacterID, g.AccountID, g.PlayerID).Scan(&bindingCount, &beneficiaryBindingID)
		if err != nil {
			return empty, err
		}
		if bindingCount == 0 {
			return empty, ErrMiningRewardInvariant
		}
		ownerRevisions[g.CharacterID] = revision
		beneficiaryBindings[g.CharacterID] = beneficiaryBindingID
	}
	for _, g := range allocation.Grants {
		revision := ownerRevisions[g.CharacterID]
		grant := MiningRewardGrant{PlayerID: g.PlayerID, AccountID: g.AccountID, CharacterID: g.CharacterID,
			BeneficiaryBindingID: beneficiaryBindings[g.CharacterID],
			Power:                g.Power, Quotient: g.Quotient, Remainder: g.Remainder, Bonus: g.Bonus, Quantity: g.Quantity,
			InventoryRevisionBefore: revision, InventoryRevisionAfter: revision}
		if g.Quantity > 0 {
			if revision == math.MaxInt64 {
				return empty, ErrMiningRewardInvariant
			}
			var slot int32
			if err = tx.QueryRow(ctx, `SELECT COALESCE(max(slot_index)+1,0) FROM character_inventory_items
				WHERE character_id=$1`, g.CharacterID).Scan(&slot); err != nil {
				return empty, err
			}
			lot := MiningRewardIssuance{ID: newContributionID("g21-issuance"),
				SourceType: "MINING_REWARD", SourceKey: newContributionID("g21-source"),
				SettlementID: result.SettlementID, BlockInstanceID: cmd.BlockInstanceID,
				CharacterID: g.CharacterID, Quantity: int32(g.Quantity), MaterialDefinitionID: definition,
				RuleVersion: settlementIssuanceVersion, CreatedAt: now, Status: "G21_SETTLED"}
			lot.SourceDigest, err = miningRewardSourceDigest(lot)
			if err != nil {
				return empty, err
			}
			grant.IssuanceID = lot.ID
			grant.InventoryInstanceID = newContributionID("g21-item")
			if _, err = tx.Exec(ctx, `INSERT INTO mining_reward_issuance_lots
				(issuance_id,source_type,source_key,settlement_id,block_instance_id,character_id,quantity,
				 material_definition_id,rule_version,created_at,source_digest,status)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
				lot.ID, lot.SourceType, lot.SourceKey, lot.SettlementID, lot.BlockInstanceID, lot.CharacterID,
				lot.Quantity, lot.MaterialDefinitionID, lot.RuleVersion, lot.CreatedAt, lot.SourceDigest, lot.Status); err != nil {
				return empty, err
			}
			if err = s.settlementBarrier("after_issuance"); err != nil {
				return empty, err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO mining_reward_inventory_projections
				(issuance_id,instance_id,character_id,block_instance_id,quantity,definition_id,
				 revision_before,revision_after,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				lot.ID, grant.InventoryInstanceID, g.CharacterID, cmd.BlockInstanceID, lot.Quantity, definition,
				revision, revision+1, now); err != nil {
				return empty, err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO character_inventory_items
				(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location,item_revision)
				VALUES($1,$2,$3,$4,'黑铁矿石','MATERIAL',$5,$6,'INVENTORY',1)`,
				grant.InventoryInstanceID, g.CharacterID, definition, legacyID, lot.Quantity, slot); err != nil {
				return empty, err
			}
			tag, e := tx.Exec(ctx, `UPDATE characters SET revision=revision+1,updated_at=$2
				WHERE id=$1 AND revision=$3`, g.CharacterID, now, revision)
			if e != nil {
				return empty, e
			}
			if tag.RowsAffected() != 1 {
				return empty, ErrMiningRewardInvariant
			}
			grant.InventoryRevisionAfter = revision + 1
			grant.ItemRevisionAfter = 1
		}
		result.Grants = append(result.Grants, grant)
	}
	for _, grant := range result.Grants {
		if grant.Quantity > 0 {
			result.PositiveParticipantCount++
		}
	}
	if err = s.settlementBarrier("after_inventory_projection"); err != nil {
		return empty, err
	}
	// Pool-first locking was done above. M is unchanged: G18 OPEN already
	// deducted R from Remaining.
	if _, err = tx.Exec(ctx, `UPDATE black_iron_emission_pools SET total_reserved=$1,
		total_distributed=$2,revision=$3,updated_at=$4 WHERE pool_id='GLOBAL'`,
		result.PoolReservedAfter, result.PoolDistributedAfter, result.PoolRevisionAfter, now); err != nil {
		return empty, err
	}
	if err = s.settlementBarrier("after_reserved_to_distributed"); err != nil {
		return empty, err
	}
	consumptionID := newContributionID("g21-consumption")
	if _, err = tx.Exec(ctx, `INSERT INTO mining_reward_reservation_consumptions
		(consumption_id,settlement_id,block_instance_id,reservation_source_id,amount,pool_revision,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		consumptionID, result.SettlementID, cmd.BlockInstanceID, bindingSource, reward, result.PoolRevisionAfter, now); err != nil {
		return empty, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO black_iron_emission_recovery_entries
		(recovery_entry_id,source_type,source_id,settlement_consumption_id,net_emission_delta,
		 reserved_delta,distributed_delta,distributed_before,distributed_after,remaining_before,
		 remaining_after,debt_before,debt_after,pool_revision,created_at)
		VALUES($1,'G21_TEST_MINING_SETTLEMENT',$2,$3,0,$4,$5,$6,$7,$8,$8,0,0,$9,$10)`,
		newContributionID("g21-recovery"), bindingSource, consumptionID, -reward, reward,
		pool.TotalDistributed, result.PoolDistributedAfter, pool.RemainingCapacity, result.PoolRevisionAfter, now); err != nil {
		return empty, err
	}
	if err = s.settlementBarrier("after_recovery_history"); err != nil {
		return empty, err
	}
	for _, g := range result.Grants {
		var lotID, itemID any
		if g.IssuanceID != "" {
			lotID, itemID = g.IssuanceID, g.InventoryInstanceID
		}
		if _, err = tx.Exec(ctx, `INSERT INTO mining_reward_grants
			(grant_id,settlement_id,block_instance_id,player_id,account_id,character_id,
			 power,quotient,remainder,bonus,quantity,issuance_id,inventory_instance_id,created_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			newContributionID("g21-grant"), result.SettlementID, cmd.BlockInstanceID,
			g.PlayerID, g.AccountID, g.CharacterID, g.Power, g.Quotient, g.Remainder, g.Bonus, g.Quantity,
			lotID, itemID, now); err != nil {
			return empty, err
		}
	}
	if err = s.settlementBarrier("after_grant"); err != nil {
		return empty, err
	}
	raw, err := canonicalMiningRewardReceipt(result)
	if err != nil {
		return empty, err
	}
	if len(raw) > settlementMaxReceiptBytes {
		return empty, ErrMiningRewardInvariant
	}
	result.CanonicalDigest = rewardHash(raw)
	if _, err = tx.Exec(ctx, `INSERT INTO mining_reward_settlement_receipts
		(settlement_id,command_id,block_instance_id,reservation_source_id,seal_id,seal_digest,
		 total_ore,canonical_bytes,canonical_digest,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, result.SettlementID, cmd.CommandID,
		cmd.BlockInstanceID, bindingSource, seal.SealID, seal.CanonicalDigest, reward, raw, result.CanonicalDigest, now); err != nil {
		return empty, err
	}
	if err = s.settlementBarrier("after_receipt"); err != nil {
		return empty, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO mining_reward_commands
		(command_id,block_instance_id,fingerprint,seal_digest,status,settlement_id,created_at)
		VALUES($1,$2,$3,$4,'COMPLETED',$5,$6)`, cmd.CommandID, cmd.BlockInstanceID,
		fingerprint, seal.CanonicalDigest, result.SettlementID, now); err != nil {
		return empty, err
	}
	if _, err = tx.Exec(ctx, `UPDATE mining_reward_settlement_states SET state='COMPLETED',
		command_id=$2,settlement_id=$3,revision=2,completed_at=$4 WHERE block_instance_id=$1`,
		cmd.BlockInstanceID, cmd.CommandID, result.SettlementID, now); err != nil {
		return empty, err
	}
	if err = s.settlementBarrier("before_commit"); err != nil {
		return empty, err
	}
	if err = tx.Commit(ctx); err != nil {
		return empty, err
	}
	if err = s.settlementBarrier("after_commit_ack_lost"); err != nil {
		return empty, err
	}
	return result, nil
}
