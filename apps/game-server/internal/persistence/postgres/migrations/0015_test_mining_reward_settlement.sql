-- G21 TEST-only atomic settlement. Existing 0001-0014 rows remain untouched.
-- Count admitted Activities at the database boundary. The row-level upsert
-- serializes direct SQL writers as well as the G20 service, so a concurrent
-- 100001st insert cannot slip through between admission and Seal.
CREATE TABLE mining_power_activity_admission_counts (
    block_instance_id text PRIMARY KEY REFERENCES mining_power_acceptance_states(block_instance_id),
    activity_count integer NOT NULL CHECK (activity_count BETWEEN 1 AND 100000)
);
INSERT INTO mining_power_activity_admission_counts(block_instance_id,activity_count)
SELECT block_instance_id,count(*)::integer FROM mining_power_activities
GROUP BY block_instance_id;
CREATE FUNCTION mining_power_guard_activity_limit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO mining_power_activity_admission_counts(block_instance_id,activity_count)
    VALUES(NEW.block_instance_id,1)
    ON CONFLICT(block_instance_id) DO UPDATE
        SET activity_count=mining_power_activity_admission_counts.activity_count+1
        WHERE mining_power_activity_admission_counts.activity_count<100000;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'mining activity input limit exceeded' USING ERRCODE='54000';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER mining_power_activity_limit AFTER INSERT ON mining_power_activities
    FOR EACH ROW EXECUTE FUNCTION mining_power_guard_activity_limit();

-- G21 treats the reviewed G19 alias as issuance authority. Preserve it
-- against table-wide erasure as well as the existing row UPDATE/DELETE guard.
CREATE TRIGGER black_iron_identity_aliases_truncate BEFORE TRUNCATE ON black_iron_identity_aliases
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- An origin inserted after an otherwise untouched stack must receive the same
-- deferred exact-provenance check as inventory insertion/update. This closes
-- both orders of G19/G21 double-origin insertion without blocking atomic
-- creation of a valid lot, projection and stack in one transaction.
CREATE CONSTRAINT TRIGGER black_iron_migration_origin_exclusive
    AFTER INSERT ON black_iron_migration_assets DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION guard_migrated_black_iron_asset();
CREATE CONSTRAINT TRIGGER mining_reward_projection_origin_exclusive
    AFTER INSERT ON mining_reward_inventory_projections DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION guard_migrated_black_iron_asset();

-- A new reward stack starts at item revision 1. Persist the revision so the
-- receipt's 0 -> 1 claim has an independent inventory oracle on replay.
ALTER TABLE character_inventory_items ADD COLUMN item_revision bigint NOT NULL DEFAULT 1
    CHECK (item_revision >= 1);

CREATE TABLE mining_reward_commands (
    command_id text PRIMARY KEY CHECK (command_id <> '' AND char_length(command_id) <= 128),
    block_instance_id text NOT NULL UNIQUE CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    fingerprint text NOT NULL CHECK (fingerprint ~ '^[0-9a-f]{64}$'),
    seal_digest text NOT NULL CHECK (seal_digest ~ '^[0-9a-f]{64}$'),
    status text NOT NULL CHECK (status IN ('COMPLETED','SEALED_NO_ELIGIBLE_POWER')),
    settlement_id text UNIQUE,
    created_at timestamptz NOT NULL,
    CHECK ((status='COMPLETED' AND settlement_id IS NOT NULL)
       OR (status='SEALED_NO_ELIGIBLE_POWER' AND settlement_id IS NULL))
);
CREATE TRIGGER mining_reward_commands_immutable BEFORE UPDATE OR DELETE ON mining_reward_commands
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reward_commands_truncate BEFORE TRUNCATE ON mining_reward_commands
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TABLE mining_reward_settlement_states (
    block_instance_id text PRIMARY KEY CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    seal_id text NOT NULL CHECK (seal_id <> '' AND char_length(seal_id) <= 128),
    state text NOT NULL CHECK (state IN ('SEALED','COMPLETED','SEALED_NO_ELIGIBLE_POWER')),
    command_id text UNIQUE,
    settlement_id text UNIQUE,
    revision bigint NOT NULL CHECK (revision IN (1,2)),
    completed_at timestamptz,
    CHECK ((state='SEALED' AND revision=1 AND command_id IS NULL AND settlement_id IS NULL AND completed_at IS NULL)
        OR (state='COMPLETED' AND revision=2 AND command_id IS NOT NULL AND settlement_id IS NOT NULL AND completed_at IS NOT NULL)
        OR (state='SEALED_NO_ELIGIBLE_POWER' AND revision=2 AND command_id IS NOT NULL AND settlement_id IS NULL AND completed_at IS NOT NULL))
);
CREATE FUNCTION mining_reward_guard_state() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.block_instance_id <> NEW.block_instance_id OR OLD.seal_id <> NEW.seal_id
       OR OLD.state <> 'SEALED' OR OLD.revision <> 1 OR NEW.revision <> 2
       OR NEW.state NOT IN ('COMPLETED','SEALED_NO_ELIGIBLE_POWER') THEN
        RAISE EXCEPTION 'mining reward state is monotonic' USING ERRCODE='55000';
    END IF;
    IF NEW.state='COMPLETED' AND NOT EXISTS (
        SELECT 1 FROM mining_reward_commands c
        JOIN mining_reward_settlement_receipts r ON r.command_id=c.command_id
        JOIN mining_reward_reservation_consumptions x ON x.settlement_id=r.settlement_id
        WHERE c.command_id=NEW.command_id AND c.block_instance_id=NEW.block_instance_id
          AND c.status='COMPLETED' AND c.settlement_id=NEW.settlement_id
          AND r.settlement_id=NEW.settlement_id AND r.block_instance_id=NEW.block_instance_id
          AND r.seal_id=NEW.seal_id AND x.block_instance_id=NEW.block_instance_id
    ) THEN
        RAISE EXCEPTION 'mining reward completion lacks economic history' USING ERRCODE='55000';
    END IF;
    IF NEW.state='SEALED_NO_ELIGIBLE_POWER' AND NOT EXISTS (
        SELECT 1 FROM mining_reward_commands c
        JOIN mining_power_input_seals s ON s.block_instance_id=c.block_instance_id
        WHERE c.command_id=NEW.command_id AND c.block_instance_id=NEW.block_instance_id
          AND c.status='SEALED_NO_ELIGIBLE_POWER' AND s.seal_id=NEW.seal_id
          AND s.total_valid_power=0 AND s.eligibility='NOT_SETTLEMENT_ELIGIBLE'
    ) THEN
        RAISE EXCEPTION 'mining reward no-power state lacks authority' USING ERRCODE='55000';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER mining_reward_state_update BEFORE UPDATE ON mining_reward_settlement_states
    FOR EACH ROW EXECUTE FUNCTION mining_reward_guard_state();
CREATE TRIGGER mining_reward_state_delete BEFORE DELETE ON mining_reward_settlement_states
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reward_state_truncate BEFORE TRUNCATE ON mining_reward_settlement_states
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TABLE mining_reward_settlement_receipts (
    settlement_id text PRIMARY KEY CHECK (settlement_id <> '' AND char_length(settlement_id) <= 128),
    command_id text NOT NULL UNIQUE,
    block_instance_id text NOT NULL UNIQUE CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    reservation_source_id text NOT NULL UNIQUE CHECK (reservation_source_id <> '' AND char_length(reservation_source_id) <= 128),
    seal_id text NOT NULL,
    seal_digest text NOT NULL CHECK (seal_digest ~ '^[0-9a-f]{64}$'),
    total_ore bigint NOT NULL CHECK (total_ore > 0),
    canonical_bytes bytea NOT NULL,
    canonical_digest text NOT NULL CHECK (canonical_digest ~ '^[0-9a-f]{64}$'),
    created_at timestamptz NOT NULL
);
CREATE TRIGGER mining_reward_receipts_immutable BEFORE UPDATE OR DELETE ON mining_reward_settlement_receipts
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reward_receipts_truncate BEFORE TRUNCATE ON mining_reward_settlement_receipts
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TABLE mining_reward_reservation_consumptions (
    consumption_id text PRIMARY KEY CHECK (consumption_id <> '' AND char_length(consumption_id) <= 128),
    settlement_id text NOT NULL UNIQUE,
    block_instance_id text NOT NULL UNIQUE CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    reservation_source_id text NOT NULL UNIQUE CHECK (reservation_source_id <> '' AND char_length(reservation_source_id) <= 128),
    amount bigint NOT NULL CHECK (amount > 0),
    pool_revision bigint NOT NULL UNIQUE CHECK (pool_revision > 0),
    created_at timestamptz NOT NULL
);
CREATE TRIGGER mining_reward_consumptions_immutable BEFORE UPDATE OR DELETE ON mining_reward_reservation_consumptions
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reward_consumptions_truncate BEFORE TRUNCATE ON mining_reward_reservation_consumptions
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TABLE mining_reward_grants (
    grant_id text PRIMARY KEY CHECK (grant_id <> '' AND char_length(grant_id) <= 128),
    settlement_id text NOT NULL,
    block_instance_id text NOT NULL CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    player_id text NOT NULL,
    account_id text NOT NULL,
    character_id text NOT NULL,
    power bigint NOT NULL CHECK (power >= 0),
    quotient bigint NOT NULL CHECK (quotient >= 0),
    remainder bigint NOT NULL CHECK (remainder >= 0),
    bonus bigint NOT NULL CHECK (bonus IN (0,1)),
    quantity bigint NOT NULL CHECK (quantity = quotient + bonus AND quantity >= 0),
    issuance_id text UNIQUE,
    inventory_instance_id text UNIQUE,
    created_at timestamptz NOT NULL,
    UNIQUE (settlement_id,character_id),
    CHECK ((quantity=0 AND issuance_id IS NULL AND inventory_instance_id IS NULL)
        OR (quantity>0 AND issuance_id IS NOT NULL AND inventory_instance_id IS NOT NULL))
);
CREATE TRIGGER mining_reward_grants_immutable BEFORE UPDATE OR DELETE ON mining_reward_grants
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reward_grants_truncate BEFORE TRUNCATE ON mining_reward_grants
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- Keep the P0 synthetic lot contract intact while admitting real settlement IDs.
ALTER TABLE mining_reward_issuance_lots DROP CONSTRAINT mining_reward_issuance_lots_settlement_id_check;
ALTER TABLE mining_reward_issuance_lots ADD CONSTRAINT mining_reward_lot_settlement_shape CHECK (
    (status='P0_SYNTHETIC_ONLY' AND settlement_id='G21_P0_TEST_PLACEHOLDER' AND rule_version='G21_P0_ISSUANCE_V1')
    OR (status='G21_SETTLED' AND settlement_id<>'G21_P0_TEST_PLACEHOLDER'
        AND settlement_id<>'' AND char_length(settlement_id)<=128 AND rule_version='G21_TEST_ISSUANCE_V1')
);
ALTER TABLE mining_reward_issuance_lots DROP CONSTRAINT mining_reward_issuance_lots_status_check;
ALTER TABLE mining_reward_issuance_lots DROP CONSTRAINT mining_reward_issuance_lots_rule_version_check;
CREATE UNIQUE INDEX mining_reward_lot_settlement_character ON mining_reward_issuance_lots(settlement_id,character_id)
    WHERE status='G21_SETTLED';

-- P0 synthetic evidence and real G21 settlement can never share one instance.
-- The unique row participates in MVCC conflict detection, including stale
-- Repeatable Read writers; a SELECT-only cross-table check is insufficient.
CREATE TABLE mining_reward_origin_modes (
    block_instance_id text PRIMARY KEY REFERENCES mining_reservation_instance_bindings(block_instance_id),
    origin_mode text NOT NULL CHECK(origin_mode IN ('P0_SYNTHETIC_ONLY','G21_SETTLED'))
);
-- Existing 0014 lots are exclusively P0. This records their origin metadata,
-- creating no lot, stack, economic receipt, reservation or pool mutation.
INSERT INTO mining_reward_origin_modes(block_instance_id,origin_mode)
SELECT DISTINCT block_instance_id,status FROM mining_reward_issuance_lots;
CREATE FUNCTION mining_reward_guard_origin_mode() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD IS DISTINCT FROM NEW THEN
        RAISE EXCEPTION 'mining reward origin mode is immutable' USING ERRCODE='55000';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER mining_reward_origin_mode_update BEFORE UPDATE ON mining_reward_origin_modes
    FOR EACH ROW EXECUTE FUNCTION mining_reward_guard_origin_mode();
CREATE TRIGGER mining_reward_origin_mode_delete BEFORE DELETE ON mining_reward_origin_modes
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reward_origin_mode_truncate BEFORE TRUNCATE ON mining_reward_origin_modes
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE FUNCTION mining_reward_guard_lot_origin() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO mining_reward_origin_modes(block_instance_id,origin_mode)
    VALUES(NEW.block_instance_id,NEW.status)
    ON CONFLICT(block_instance_id) DO UPDATE
        SET origin_mode=mining_reward_origin_modes.origin_mode
        WHERE mining_reward_origin_modes.origin_mode=EXCLUDED.origin_mode;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'mining instance already has a different reward origin' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER mining_reward_lot_origin BEFORE INSERT ON mining_reward_issuance_lots
    FOR EACH ROW EXECUTE FUNCTION mining_reward_guard_lot_origin();

-- Both TEST accounting operations serialize against one binding identity.
-- This prevents a P0 synthetic transfer and a real settlement consuming R twice.
CREATE FUNCTION mining_reward_guard_exclusive_reservation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM mining_reservation_instance_bindings
        WHERE reservation_source_id=NEW.reservation_source_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'unbound mining reservation' USING ERRCODE='23514';
    END IF;
    IF TG_TABLE_NAME='mining_reward_reservation_consumptions' THEN
        IF EXISTS(SELECT 1 FROM mining_prerequisite_distributions WHERE reservation_source_id=NEW.reservation_source_id) THEN
            RAISE EXCEPTION 'reservation already transferred by P0' USING ERRCODE='23514';
        END IF;
    ELSIF EXISTS(SELECT 1 FROM mining_reward_reservation_consumptions WHERE reservation_source_id=NEW.reservation_source_id) THEN
        RAISE EXCEPTION 'reservation already settled' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER mining_reward_consumption_exclusive BEFORE INSERT ON mining_reward_reservation_consumptions
    FOR EACH ROW EXECUTE FUNCTION mining_reward_guard_exclusive_reservation();
CREATE TRIGGER mining_prerequisite_distribution_exclusive BEFORE INSERT ON mining_prerequisite_distributions
    FOR EACH ROW EXECUTE FUNCTION mining_reward_guard_exclusive_reservation();

-- A distinct immutable recovery source preserves old P0 and G17-G18 replay.
ALTER TABLE black_iron_emission_recovery_entries
    ADD COLUMN settlement_consumption_id text UNIQUE;
ALTER TABLE black_iron_emission_recovery_entries DROP CONSTRAINT black_iron_emission_recovery_source_type;
ALTER TABLE black_iron_emission_recovery_entries ADD CONSTRAINT black_iron_emission_recovery_source_type
    CHECK (source_type IN ('G15_ELIGIBLE_SYSTEM_SPEND','G14_REFUND_COMPENSATION',
        'G18_BLOCK_OPEN','G18_BLOCK_CANCEL','G21_P0_TEST_DISTRIBUTION','G21_TEST_MINING_SETTLEMENT'));
ALTER TABLE black_iron_emission_recovery_entries DROP CONSTRAINT black_iron_emission_recovery_source_shape;
ALTER TABLE black_iron_emission_recovery_entries ADD CONSTRAINT black_iron_emission_recovery_source_shape CHECK (
    (source_type IN ('G15_ELIGIBLE_SYSTEM_SPEND','G14_REFUND_COMPENSATION')
        AND emission_entry_id IS NOT NULL AND block_entry_id IS NULL
        AND test_distribution_id IS NULL AND settlement_consumption_id IS NULL
        AND reserved_delta=0 AND distributed_delta=0
        AND ((source_type='G15_ELIGIBLE_SYSTEM_SPEND' AND net_emission_delta>0)
          OR (source_type='G14_REFUND_COMPENSATION' AND net_emission_delta<0)))
    OR (source_type='G18_BLOCK_OPEN' AND emission_entry_id IS NULL AND block_entry_id IS NOT NULL
        AND test_distribution_id IS NULL AND settlement_consumption_id IS NULL
        AND net_emission_delta=0 AND reserved_delta>0 AND distributed_delta=0 AND debt_before=debt_after)
    OR (source_type='G18_BLOCK_CANCEL' AND emission_entry_id IS NULL AND block_entry_id IS NOT NULL
        AND test_distribution_id IS NULL AND settlement_consumption_id IS NULL
        AND net_emission_delta=0 AND reserved_delta<0 AND distributed_delta=0)
    OR (source_type='G21_P0_TEST_DISTRIBUTION' AND emission_entry_id IS NULL AND block_entry_id IS NULL
        AND test_distribution_id IS NOT NULL AND settlement_consumption_id IS NULL
        AND net_emission_delta=0 AND reserved_delta<0 AND distributed_delta=-reserved_delta
        AND remaining_before=remaining_after AND debt_before=debt_after)
    OR (source_type='G21_TEST_MINING_SETTLEMENT' AND emission_entry_id IS NULL AND block_entry_id IS NULL
        AND test_distribution_id IS NULL AND settlement_consumption_id IS NOT NULL
        AND net_emission_delta=0 AND reserved_delta<0 AND distributed_delta=-reserved_delta
        AND remaining_before=remaining_after AND debt_before=debt_after)
);
CREATE TRIGGER black_iron_emission_recovery_truncate BEFORE TRUNCATE ON black_iron_emission_recovery_entries
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();
