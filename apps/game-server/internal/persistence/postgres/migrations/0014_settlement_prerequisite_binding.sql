-- G21-P0: only new reservations obtain an authoritative instance binding.
-- Existing reservations have no provable birth relationship and remain unbound.
CREATE TABLE mining_reservation_instance_bindings (
    binding_id text PRIMARY KEY CHECK (binding_id <> '' AND char_length(binding_id) <= 128),
    reservation_source_id text NOT NULL UNIQUE CHECK (reservation_source_id <> '' AND char_length(reservation_source_id) <= 128),
    reservation_receipt_id text NOT NULL UNIQUE CHECK (reservation_receipt_id <> '' AND char_length(reservation_receipt_id) <= 128),
    block_instance_id text NOT NULL UNIQUE CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    display_block_id text NOT NULL CHECK (display_block_id <> '' AND char_length(display_block_id) <= 128),
    g18_rule_version text NOT NULL CHECK (g18_rule_version <> '' AND char_length(g18_rule_version) <= 64),
    reservation_amount bigint NOT NULL CHECK (reservation_amount > 0),
    binding_version text NOT NULL CHECK (binding_version = 'G21_P0_RESERVATION_V1'),
    evidence_digest text NOT NULL CHECK (evidence_digest ~ '^[0-9a-f]{64}$'),
    created_at timestamptz NOT NULL
);
CREATE TRIGGER mining_reservation_instance_bindings_immutable BEFORE UPDATE OR DELETE ON mining_reservation_instance_bindings
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reservation_instance_bindings_truncate BEFORE TRUNCATE ON mining_reservation_instance_bindings
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- This is G20-owned evidence of the actual character row checked at accept time.
-- Historical G20 facts without this evidence remain settlement-ineligible.
CREATE TABLE mining_power_beneficiary_bindings (
    binding_id text PRIMARY KEY CHECK (binding_id <> '' AND char_length(binding_id) <= 128),
    activity_id text NOT NULL UNIQUE REFERENCES mining_power_activities(activity_id),
    source_event_id text NOT NULL UNIQUE REFERENCES mining_power_source_events(source_event_id),
    block_instance_id text NOT NULL CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    account_id text NOT NULL CHECK (mining_power_valid_id(account_id,128)),
    player_id text NOT NULL CHECK (mining_power_valid_id(player_id,128)),
    character_id text NOT NULL CHECK (mining_power_valid_id(character_id,128)),
    authority_source text NOT NULL CHECK (authority_source = 'CHARACTERS_OWNER_ROW_V1'),
    binding_version text NOT NULL CHECK (binding_version = 'G21_P0_BENEFICIARY_V1'),
    bound_at timestamptz NOT NULL,
    digest text NOT NULL CHECK (digest ~ '^[0-9a-f]{64}$'),
    FOREIGN KEY(character_id,account_id) REFERENCES characters(id,account_id)
);
CREATE INDEX mining_power_beneficiary_by_instance ON mining_power_beneficiary_bindings(block_instance_id,character_id,player_id);
CREATE TRIGGER mining_power_beneficiary_bindings_immutable BEFORE UPDATE OR DELETE ON mining_power_beneficiary_bindings
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_power_beneficiary_bindings_truncate BEFORE TRUNCATE ON mining_power_beneficiary_bindings
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TABLE mining_power_acceptance_states (
    block_instance_id text PRIMARY KEY CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    state text NOT NULL CHECK (state IN ('OPEN','SEALED')),
    seal_id text UNIQUE,
    seal_rule_version text,
    sealed_at timestamptz,
    revision bigint NOT NULL CHECK (revision BETWEEN 1 AND 2),
    CHECK ((state='OPEN' AND seal_id IS NULL AND seal_rule_version IS NULL AND sealed_at IS NULL AND revision=1)
        OR (state='SEALED' AND seal_id IS NOT NULL AND seal_rule_version='G21_P0_SEAL_V1' AND sealed_at IS NOT NULL AND revision=2))
);
CREATE FUNCTION mining_power_guard_acceptance_state() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.block_instance_id <> NEW.block_instance_id OR OLD.state <> 'OPEN' OR NEW.state <> 'SEALED'
       OR OLD.revision <> 1 OR NEW.revision <> 2 THEN
        RAISE EXCEPTION 'mining acceptance state is monotonic' USING ERRCODE='55000';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER mining_power_acceptance_state_update BEFORE UPDATE ON mining_power_acceptance_states
    FOR EACH ROW EXECUTE FUNCTION mining_power_guard_acceptance_state();
CREATE TRIGGER mining_power_acceptance_state_delete BEFORE DELETE ON mining_power_acceptance_states
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_power_acceptance_state_truncate BEFORE TRUNCATE ON mining_power_acceptance_states
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- Every accepted fact, including a direct application-role insert, must be
-- admitted under the same persistent row lock. Existing facts are untouched.
CREATE FUNCTION mining_power_guard_new_activity() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE current_state text;
BEGIN
    -- Generated stored columns are not populated in a BEFORE INSERT trigger.
    SELECT state INTO current_state FROM mining_power_acceptance_states
        WHERE block_instance_id=NEW.data->>'BlockInstanceID' FOR SHARE;
    IF current_state IS NULL OR current_state <> 'OPEN' THEN
        RAISE EXCEPTION 'BLOCK_SEALED or acceptance gate missing' USING ERRCODE='55000';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER mining_power_activity_acceptance_gate BEFORE INSERT ON mining_power_activities
    FOR EACH ROW EXECUTE FUNCTION mining_power_guard_new_activity();

CREATE TABLE mining_power_input_seals (
    seal_id text PRIMARY KEY CHECK (seal_id <> '' AND char_length(seal_id) <= 128),
    block_instance_id text NOT NULL UNIQUE CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    seal_rule_version text NOT NULL CHECK (seal_rule_version='G21_P0_SEAL_V1'),
    schema_version text NOT NULL CHECK (schema_version='G21_P0_SCHEMA_V1'),
    activity_count integer NOT NULL CHECK (activity_count >= 0),
    participant_count integer NOT NULL CHECK (participant_count BETWEEN 0 AND 500),
    total_valid_power bigint NOT NULL CHECK (total_valid_power >= 0),
    eligibility text NOT NULL CHECK (eligibility IN ('SETTLEMENT_INPUT','NOT_SETTLEMENT_ELIGIBLE')),
    canonical_bytes bytea NOT NULL,
    canonical_digest text NOT NULL CHECK (canonical_digest ~ '^[0-9a-f]{64}$'),
    sealed_at timestamptz NOT NULL
);
CREATE INDEX mining_power_seal_by_instance ON mining_power_input_seals(block_instance_id,seal_id);
CREATE TRIGGER mining_power_input_seals_immutable BEFORE UPDATE OR DELETE ON mining_power_input_seals
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_power_input_seals_truncate BEFORE TRUNCATE ON mining_power_input_seals
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- P4 allows a TEST-only accounting transition S -> D. M stays fixed because
-- G18 OPEN already deducted the reservation. The source is not an ore issue.
ALTER TABLE black_iron_emission_pools DROP CONSTRAINT black_iron_emission_pools_total_distributed_check;
ALTER TABLE black_iron_emission_pools ADD CONSTRAINT black_iron_emission_pools_distributed_nonnegative CHECK (total_distributed >= 0);
CREATE TABLE mining_prerequisite_distributions (
    distribution_id text PRIMARY KEY CHECK (distribution_id <> '' AND char_length(distribution_id) <= 128),
    reservation_source_id text NOT NULL UNIQUE REFERENCES mining_reservation_instance_bindings(reservation_source_id),
    amount bigint NOT NULL CHECK (amount > 0),
    pool_revision bigint NOT NULL UNIQUE CHECK (pool_revision > 0),
    created_at timestamptz NOT NULL
);
CREATE TRIGGER mining_prerequisite_distributions_immutable BEFORE UPDATE OR DELETE ON mining_prerequisite_distributions
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_prerequisite_distributions_truncate BEFORE TRUNCATE ON mining_prerequisite_distributions
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();
ALTER TABLE black_iron_emission_recovery_entries
    ADD COLUMN test_distribution_id text UNIQUE REFERENCES mining_prerequisite_distributions(distribution_id),
    ADD COLUMN distributed_delta bigint NOT NULL DEFAULT 0,
    ADD COLUMN distributed_before bigint NOT NULL DEFAULT 0 CHECK (distributed_before >= 0),
    ADD COLUMN distributed_after bigint NOT NULL DEFAULT 0 CHECK (distributed_after >= 0),
    ADD CONSTRAINT black_iron_recovery_distributed_transition CHECK (distributed_after = distributed_before + distributed_delta);
ALTER TABLE black_iron_emission_recovery_entries DROP CONSTRAINT black_iron_emission_recovery_entries_source_type_check;
ALTER TABLE black_iron_emission_recovery_entries ADD CONSTRAINT black_iron_emission_recovery_source_type
    CHECK (source_type IN ('G15_ELIGIBLE_SYSTEM_SPEND','G14_REFUND_COMPENSATION','G18_BLOCK_OPEN','G18_BLOCK_CANCEL','G21_P0_TEST_DISTRIBUTION'));
ALTER TABLE black_iron_emission_recovery_entries DROP CONSTRAINT black_iron_emission_recovery_entries_check;
ALTER TABLE black_iron_emission_recovery_entries ADD CONSTRAINT black_iron_emission_recovery_source_shape CHECK (
    (source_type IN ('G15_ELIGIBLE_SYSTEM_SPEND','G14_REFUND_COMPENSATION')
        AND emission_entry_id IS NOT NULL AND block_entry_id IS NULL AND test_distribution_id IS NULL
        AND reserved_delta=0 AND distributed_delta=0
        AND ((source_type='G15_ELIGIBLE_SYSTEM_SPEND' AND net_emission_delta>0)
          OR (source_type='G14_REFUND_COMPENSATION' AND net_emission_delta<0)))
    OR (source_type='G18_BLOCK_OPEN' AND emission_entry_id IS NULL AND block_entry_id IS NOT NULL
        AND test_distribution_id IS NULL AND net_emission_delta=0 AND reserved_delta>0
        AND distributed_delta=0 AND debt_before=debt_after)
    OR (source_type='G18_BLOCK_CANCEL' AND emission_entry_id IS NULL AND block_entry_id IS NOT NULL
        AND test_distribution_id IS NULL AND net_emission_delta=0 AND reserved_delta<0 AND distributed_delta=0)
    OR (source_type='G21_P0_TEST_DISTRIBUTION' AND emission_entry_id IS NULL AND block_entry_id IS NULL
        AND test_distribution_id IS NOT NULL AND net_emission_delta=0 AND reserved_delta<0
        AND distributed_delta=-reserved_delta AND remaining_before=remaining_after AND debt_before=debt_after)
);

-- P5: a separate MINING_REWARD origin. The G19 identity catalog remains the
-- only canonical material definition; neither its migration assets nor its
-- receipts are modified. These tables are inert without the TEST-only adapter.
CREATE TABLE mining_reward_issuance_lots (
    issuance_id text PRIMARY KEY CHECK (issuance_id <> '' AND char_length(issuance_id) <= 128),
    source_type text NOT NULL CHECK (source_type='MINING_REWARD'),
    source_key text NOT NULL UNIQUE CHECK (source_key <> '' AND char_length(source_key) <= 128),
    settlement_id text NOT NULL CHECK (settlement_id='G21_P0_TEST_PLACEHOLDER'),
    block_instance_id text NOT NULL CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    character_id text NOT NULL CHECK (character_id <> '' AND char_length(character_id) <= 128),
    quantity integer NOT NULL CHECK (quantity > 0),
    material_definition_id text NOT NULL CHECK (material_definition_id <> '' AND char_length(material_definition_id) <= 512),
    rule_version text NOT NULL CHECK (rule_version='G21_P0_ISSUANCE_V1'),
    created_at timestamptz NOT NULL,
    source_digest text NOT NULL CHECK (source_digest ~ '^[0-9a-f]{64}$'),
    status text NOT NULL CHECK (status='P0_SYNTHETIC_ONLY'),
    UNIQUE(block_instance_id,character_id,source_key)
);
CREATE TRIGGER mining_reward_issuance_lots_immutable BEFORE UPDATE OR DELETE ON mining_reward_issuance_lots
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reward_issuance_lots_truncate BEFORE TRUNCATE ON mining_reward_issuance_lots
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TABLE mining_reward_inventory_projections (
    issuance_id text PRIMARY KEY REFERENCES mining_reward_issuance_lots(issuance_id),
    instance_id text NOT NULL UNIQUE CHECK (instance_id <> '' AND char_length(instance_id) <= 128),
    character_id text NOT NULL CHECK (character_id <> '' AND char_length(character_id) <= 128),
    block_instance_id text NOT NULL CHECK (block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'),
    quantity integer NOT NULL CHECK (quantity > 0),
    definition_id text NOT NULL CHECK (definition_id <> '' AND char_length(definition_id) <= 512),
    revision_before bigint NOT NULL CHECK (revision_before >= 1),
    revision_after bigint NOT NULL CHECK (revision_after = revision_before+1),
    created_at timestamptz NOT NULL
);
CREATE TRIGGER mining_reward_inventory_projections_immutable BEFORE UPDATE OR DELETE ON mining_reward_inventory_projections
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_reward_inventory_projections_truncate BEFORE TRUNCATE ON mining_reward_inventory_projections
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- Preserve G19 migrated-asset protection while recognizing the new, separate
-- source. Canonical inventory without either source still fails at commit.
CREATE OR REPLACE FUNCTION guard_migrated_black_iron_asset() RETURNS trigger AS $$
DECLARE asset_id text; migrated black_iron_migration_assets%ROWTYPE; reward mining_reward_inventory_projections%ROWTYPE;
BEGIN
    IF TG_OP = 'DELETE' THEN asset_id := OLD.instance_id; ELSE asset_id := NEW.instance_id; END IF;
    SELECT * INTO migrated FROM black_iron_migration_assets WHERE instance_id=asset_id;
    SELECT * INTO reward FROM mining_reward_inventory_projections WHERE instance_id=asset_id;
    IF migrated.instance_id IS NOT NULL AND reward.instance_id IS NOT NULL THEN
        RAISE EXCEPTION 'black iron asset has two origins' USING ERRCODE='23514';
    END IF;
    IF migrated.instance_id IS NOT NULL AND NOT EXISTS(
      SELECT 1 FROM character_inventory_items i WHERE i.instance_id=asset_id
        AND i.character_id=migrated.character_id AND i.definition_id=migrated.definition_id
        AND i.legacy_id=migrated.legacy_id AND i.quantity=migrated.quantity
        AND i.name='黑铁矿石' AND i.item_type='MATERIAL'
        AND i.location='INVENTORY' AND i.equipment_slot IS NULL
    ) THEN
        RAISE EXCEPTION 'black iron migration movement is not implemented' USING ERRCODE='23514';
    END IF;
    IF reward.instance_id IS NOT NULL AND NOT EXISTS(
      SELECT 1 FROM character_inventory_items i JOIN mining_reward_issuance_lots l ON l.issuance_id=reward.issuance_id
      JOIN black_iron_identity_aliases a ON a.definition_id=l.material_definition_id
      WHERE i.instance_id=asset_id AND i.character_id=reward.character_id
        AND i.definition_id=reward.definition_id AND i.definition_id=l.material_definition_id
        AND i.legacy_id=a.legacy_id AND i.quantity=reward.quantity AND i.quantity=l.quantity
        AND i.name='黑铁矿石' AND i.item_type='MATERIAL'
        AND i.location='INVENTORY' AND i.equipment_slot IS NULL
        AND reward.block_instance_id=l.block_instance_id AND reward.character_id=l.character_id
    ) THEN
        RAISE EXCEPTION 'mining reward projection differs from immutable lot' USING ERRCODE='23514';
    END IF;
    IF migrated.instance_id IS NULL AND reward.instance_id IS NULL AND EXISTS(
      SELECT 1 FROM character_inventory_items i JOIN black_iron_identity_aliases a ON a.definition_id=i.definition_id
      WHERE i.instance_id=asset_id AND i.name='黑铁矿石'
    ) THEN
        RAISE EXCEPTION 'black iron asset lacks provenance' USING ERRCODE='23514';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
