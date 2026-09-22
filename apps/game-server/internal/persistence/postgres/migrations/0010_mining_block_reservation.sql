-- G18 relaxes G17's zero-reservation guard while retaining one authoritative pool.
ALTER TABLE black_iron_emission_pools DROP CONSTRAINT black_iron_emission_pools_total_reserved_check;
ALTER TABLE black_iron_emission_pools DROP CONSTRAINT black_iron_emission_pools_check1;
ALTER TABLE black_iron_emission_pools ADD COLUMN recovery_debt bigint NOT NULL DEFAULT 0 CHECK (recovery_debt >= 0);
ALTER TABLE black_iron_emission_pools ADD CONSTRAINT black_iron_emission_pools_reserved_nonnegative CHECK (total_reserved >= 0);
ALTER TABLE black_iron_emission_pools ADD CONSTRAINT black_iron_emission_pools_conservation CHECK (total_emission_capacity = total_reserved + total_distributed + remaining_capacity - recovery_debt);
ALTER TABLE black_iron_emission_pools ADD CONSTRAINT black_iron_emission_pools_debt_bound CHECK (recovery_debt <= total_reserved + total_distributed);
ALTER TABLE black_iron_emission_receipts DROP CONSTRAINT black_iron_emission_receipts_check;
ALTER TABLE black_iron_emission_receipts ADD CONSTRAINT black_iron_emission_receipts_remaining_bound CHECK (remaining_capacity <= pool_after);

CREATE TABLE mining_blocks (
    block_id text PRIMARY KEY CHECK (block_id <> '' AND char_length(block_id) <= 128),
    block_height bigint NOT NULL UNIQUE CHECK (block_height > 0),
    create_command_id text NOT NULL UNIQUE CHECK (create_command_id <> '' AND char_length(create_command_id) <= 128),
    status text NOT NULL CHECK (status IN ('OPEN','FINALIZED','CANCELLED')),
    rule_version text NOT NULL CHECK (rule_version <> '' AND char_length(rule_version) <= 64),
    started_at timestamptz NOT NULL,
    scheduled_end_at timestamptz NOT NULL,
    finalized_at timestamptz,
    cancelled_at timestamptz,
    reward_reserved bigint NOT NULL CHECK (reward_reserved > 0),
    reward_released bigint NOT NULL DEFAULT 0 CHECK (reward_released = 0),
    reward_returned bigint NOT NULL DEFAULT 0 CHECK (reward_returned >= 0 AND reward_returned <= reward_reserved),
    pool_revision_at_reservation bigint NOT NULL CHECK (pool_revision_at_reservation >= 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK (scheduled_end_at > started_at),
    CHECK ((status='OPEN' AND finalized_at IS NULL AND cancelled_at IS NULL AND reward_returned=0)
        OR (status='FINALIZED' AND finalized_at IS NOT NULL AND cancelled_at IS NULL AND reward_returned=0)
        OR (status='CANCELLED' AND finalized_at IS NULL AND cancelled_at IS NOT NULL AND reward_returned=reward_reserved))
);

CREATE TABLE mining_block_reservations (
    block_id text PRIMARY KEY REFERENCES mining_blocks(block_id),
    amount bigint NOT NULL CHECK (amount > 0),
    released bigint NOT NULL DEFAULT 0 CHECK (released >= 0 AND released <= amount),
    status text NOT NULL CHECK (status IN ('ACTIVE','RELEASED')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK ((status='ACTIVE' AND released=0) OR (status='RELEASED' AND released=amount))
);

CREATE TABLE mining_block_entries (
    entry_id text PRIMARY KEY CHECK (entry_id <> '' AND char_length(entry_id) <= 128),
    block_id text NOT NULL REFERENCES mining_blocks(block_id),
    block_height bigint NOT NULL CHECK (block_height > 0),
    action text NOT NULL CHECK (action IN ('OPEN','FINALIZE','CANCEL')),
    capacity_delta bigint NOT NULL,
    pool_before bigint NOT NULL CHECK (pool_before >= 0),
    pool_after bigint NOT NULL CHECK (pool_after >= 0),
    block_status_before text NOT NULL,
    block_status_after text NOT NULL,
    rule_version text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (block_id,action),
    CHECK (pool_after = pool_before - capacity_delta),
    CHECK ((action='OPEN' AND capacity_delta > 0 AND block_status_before='' AND block_status_after='OPEN')
        OR (action='FINALIZE' AND capacity_delta=0 AND block_status_before='OPEN' AND block_status_after='FINALIZED')
        OR (action='CANCEL' AND capacity_delta <= 0 AND block_status_before='OPEN' AND block_status_after='CANCELLED'))
);
CREATE TRIGGER mining_block_entries_immutable BEFORE UPDATE OR DELETE ON mining_block_entries
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TABLE mining_block_receipts (
    receipt_id text PRIMARY KEY CHECK (receipt_id <> '' AND char_length(receipt_id) <= 128),
    entry_id text NOT NULL UNIQUE REFERENCES mining_block_entries(entry_id),
    block_id text NOT NULL REFERENCES mining_blocks(block_id),
    block_height bigint NOT NULL CHECK (block_height > 0),
    action text NOT NULL CHECK (action IN ('OPEN','FINALIZE','CANCEL')),
    reward_reserved bigint NOT NULL CHECK (reward_reserved > 0),
    pool_before bigint NOT NULL CHECK (pool_before >= 0),
    pool_after bigint NOT NULL CHECK (pool_after >= 0),
    block_status text NOT NULL CHECK (block_status IN ('OPEN','FINALIZED','CANCELLED')),
    rule_version text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (block_id,action)
);
CREATE TRIGGER mining_block_receipts_immutable BEFORE UPDATE OR DELETE ON mining_block_receipts
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- One immutable recovery audit for every pool revision, including a zero-debt
-- operation. Its source FK and unique revision bind it to the authoritative
-- emission/block journal. Existing G17 history had no reservations or debt.
CREATE TABLE black_iron_emission_recovery_entries (
    recovery_entry_id text PRIMARY KEY CHECK (recovery_entry_id <> '' AND char_length(recovery_entry_id) <= 128),
    source_type text NOT NULL CHECK (source_type IN ('G15_ELIGIBLE_SYSTEM_SPEND','G14_REFUND_COMPENSATION','G18_BLOCK_OPEN','G18_BLOCK_CANCEL')),
    source_id text NOT NULL CHECK (source_id <> '' AND char_length(source_id) <= 128),
    emission_entry_id text UNIQUE REFERENCES black_iron_emission_entries(entry_id),
    block_entry_id text UNIQUE REFERENCES mining_block_entries(entry_id),
    net_emission_delta bigint NOT NULL,
    reserved_delta bigint NOT NULL,
    remaining_before bigint NOT NULL CHECK (remaining_before >= 0),
    remaining_after bigint NOT NULL CHECK (remaining_after >= 0),
    debt_before bigint NOT NULL CHECK (debt_before >= 0),
    debt_after bigint NOT NULL CHECK (debt_after >= 0),
    pool_revision bigint NOT NULL UNIQUE CHECK (pool_revision > 0),
    created_at timestamptz NOT NULL,
    UNIQUE (source_type,source_id),
    CHECK ((source_type IN ('G15_ELIGIBLE_SYSTEM_SPEND','G14_REFUND_COMPENSATION')
            AND emission_entry_id IS NOT NULL AND block_entry_id IS NULL AND reserved_delta=0
            AND ((source_type='G15_ELIGIBLE_SYSTEM_SPEND' AND net_emission_delta>0)
              OR (source_type='G14_REFUND_COMPENSATION' AND net_emission_delta<0)))
        OR (source_type='G18_BLOCK_OPEN' AND emission_entry_id IS NULL AND block_entry_id IS NOT NULL
            AND net_emission_delta=0 AND reserved_delta>0 AND debt_before=debt_after)
        OR (source_type='G18_BLOCK_CANCEL' AND emission_entry_id IS NULL AND block_entry_id IS NOT NULL
            AND net_emission_delta=0 AND reserved_delta<0))
);
CREATE TRIGGER black_iron_emission_recovery_entries_immutable BEFORE UPDATE OR DELETE ON black_iron_emission_recovery_entries
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

INSERT INTO black_iron_emission_recovery_entries
    (recovery_entry_id,source_type,source_id,emission_entry_id,net_emission_delta,reserved_delta,
     remaining_before,remaining_after,debt_before,debt_after,pool_revision,created_at)
SELECT e.entry_id,e.source_type,e.source_id,e.entry_id,e.emission_amount,0,
       r.pool_before,r.remaining_capacity,0,0,e.pool_revision,e.created_at
FROM black_iron_emission_entries e
JOIN black_iron_emission_receipts r ON r.entry_id=e.entry_id;
