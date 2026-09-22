-- G17 accounts for global future Black Iron capacity only. No player ore,
-- mining reward, or production emission ratio is created by this migration.
CREATE TABLE black_iron_emission_pools (
    pool_id text PRIMARY KEY CHECK (pool_id = 'GLOBAL'),
    total_eligible_spend_observed bigint NOT NULL DEFAULT 0 CHECK (total_eligible_spend_observed >= 0),
    total_refunded_spend bigint NOT NULL DEFAULT 0 CHECK (total_refunded_spend >= 0 AND total_refunded_spend <= total_eligible_spend_observed),
    total_emission_capacity bigint NOT NULL DEFAULT 0 CHECK (total_emission_capacity >= 0),
    total_reserved bigint NOT NULL DEFAULT 0 CHECK (total_reserved = 0),
    total_distributed bigint NOT NULL DEFAULT 0 CHECK (total_distributed = 0),
    remaining_capacity bigint NOT NULL DEFAULT 0 CHECK (remaining_capacity >= 0),
    rule_version text NOT NULL DEFAULT '' CHECK (char_length(rule_version) <= 64),
    revision bigint NOT NULL DEFAULT 0 CHECK (revision >= 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK (total_emission_capacity = remaining_capacity)
);
INSERT INTO black_iron_emission_pools(pool_id,created_at,updated_at) VALUES('GLOBAL',now(),now());

CREATE TABLE black_iron_emission_entries (
    entry_id text PRIMARY KEY CHECK (entry_id <> '' AND char_length(entry_id) <= 128),
    source_type text NOT NULL CHECK (source_type IN ('G15_ELIGIBLE_SYSTEM_SPEND','G14_REFUND_COMPENSATION')),
    source_id text NOT NULL CHECK (source_id <> '' AND char_length(source_id) <= 128),
    spend_id text NOT NULL REFERENCES system_spends(spend_id),
    compensation_id text UNIQUE REFERENCES contribution_compensations(compensation_id),
    eligible_spend_amount bigint NOT NULL CHECK (eligible_spend_amount <> 0),
    emission_amount bigint NOT NULL CHECK (emission_amount <> 0),
    rule_version text NOT NULL CHECK (rule_version <> '' AND char_length(rule_version) <= 64),
    created_at timestamptz NOT NULL,
    pool_revision bigint NOT NULL UNIQUE CHECK (pool_revision > 0),
    UNIQUE (source_type,source_id),
    CHECK ((source_type='G15_ELIGIBLE_SYSTEM_SPEND' AND compensation_id IS NULL AND eligible_spend_amount > 0 AND emission_amount > 0)
        OR (source_type='G14_REFUND_COMPENSATION' AND compensation_id IS NOT NULL AND eligible_spend_amount < 0 AND emission_amount < 0))
);
CREATE INDEX black_iron_emission_entries_spend_idx ON black_iron_emission_entries(spend_id,pool_revision);
CREATE TRIGGER black_iron_emission_entries_immutable BEFORE UPDATE OR DELETE ON black_iron_emission_entries
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TABLE black_iron_emission_receipts (
    receipt_id text PRIMARY KEY CHECK (receipt_id <> '' AND char_length(receipt_id) <= 128),
    entry_id text NOT NULL UNIQUE REFERENCES black_iron_emission_entries(entry_id),
    source_id text NOT NULL CHECK (source_id <> '' AND char_length(source_id) <= 128),
    eligible_spend bigint NOT NULL CHECK (eligible_spend <> 0),
    emission_added bigint NOT NULL CHECK (emission_added <> 0),
    pool_before bigint NOT NULL CHECK (pool_before >= 0),
    pool_after bigint NOT NULL CHECK (pool_after >= 0),
    remaining_capacity bigint NOT NULL CHECK (remaining_capacity >= 0),
    rule_version text NOT NULL CHECK (rule_version <> '' AND char_length(rule_version) <= 64),
    created_at timestamptz NOT NULL,
    CHECK (pool_after = remaining_capacity),
    CHECK ((emission_added > 0 AND pool_after > pool_before) OR (emission_added < 0 AND pool_after < pool_before))
);
CREATE TRIGGER black_iron_emission_receipts_immutable BEFORE UPDATE OR DELETE ON black_iron_emission_receipts
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
