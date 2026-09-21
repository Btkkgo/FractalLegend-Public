-- G14: additive recovery state; historical G13 entries remain unchanged.
ALTER TABLE contribution_accounts
    ADD COLUMN recovery_debt bigint NOT NULL DEFAULT 0 CHECK (recovery_debt >= 0),
    ADD COLUMN review_required boolean NOT NULL DEFAULT false;
ALTER TABLE contribution_accounts ADD CONSTRAINT contribution_recovery_hold_check
    CHECK (recovery_debt = 0 OR balance = 0);

ALTER TABLE contribution_entries
    ADD COLUMN debt_settled bigint NOT NULL DEFAULT 0 CHECK (debt_settled >= 0 AND debt_settled <= amount);
ALTER TABLE contribution_entries DROP CONSTRAINT contribution_entries_check;
ALTER TABLE contribution_entries ADD CONSTRAINT contribution_entries_balance_after_check
    CHECK (balance_after >= 0 AND balance_after = balance_before + amount - debt_settled);

-- No production writer is exposed in G14. This immutable relation accounts for
-- historical/synthetic consumption when testing insufficient-balance recovery.
CREATE TABLE contribution_consumptions (
    consumption_id text PRIMARY KEY CHECK (consumption_id <> '' AND char_length(consumption_id) <= 128),
    player_id text NOT NULL REFERENCES contribution_accounts(player_id),
    amount bigint NOT NULL CHECK (amount > 0),
    created_at timestamptz NOT NULL
);
CREATE TRIGGER contribution_consumptions_immutable BEFORE UPDATE OR DELETE ON contribution_consumptions
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TABLE contribution_compensations (
    compensation_id text PRIMARY KEY CHECK (compensation_id <> '' AND char_length(compensation_id) <= 128),
    original_entry_id text NOT NULL REFERENCES contribution_entries(entry_id),
    original_fb_transaction_id text NOT NULL REFERENCES fb_ledger_transactions(transaction_id),
    fb_transaction_id text NOT NULL UNIQUE REFERENCES fb_ledger_transactions(transaction_id),
    player_id text NOT NULL REFERENCES contribution_accounts(player_id),
    reference_id text NOT NULL UNIQUE CHECK (reference_id <> '' AND char_length(reference_id) <= 128),
    amount bigint NOT NULL CHECK (amount > 0),
    available_reversed bigint NOT NULL CHECK (available_reversed >= 0),
    debt_created bigint NOT NULL CHECK (debt_created >= 0),
    balance_before bigint NOT NULL CHECK (balance_before >= 0),
    balance_after bigint NOT NULL CHECK (balance_after >= 0),
    debt_before bigint NOT NULL CHECK (debt_before >= 0),
    debt_after bigint NOT NULL CHECK (debt_after >= 0),
    rule_version text NOT NULL CHECK (rule_version <> ''),
    created_at timestamptz NOT NULL,
    CHECK (amount = available_reversed + debt_created),
    CHECK (balance_after = balance_before - available_reversed),
    CHECK (debt_after = debt_before + debt_created)
);
CREATE INDEX contribution_compensations_original_idx ON contribution_compensations(original_entry_id);
CREATE INDEX contribution_compensations_player_idx ON contribution_compensations(player_id, created_at);
CREATE TRIGGER contribution_compensations_immutable BEFORE UPDATE OR DELETE ON contribution_compensations
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- Keep G11's single-compensation rule for every existing use. Only the
-- orchestrated G14 reference may have multiple partial refunds.
DROP INDEX fb_ledger_single_compensation_idx;
CREATE UNIQUE INDEX fb_ledger_single_compensation_idx
    ON fb_ledger_transactions(original_transaction_id)
    WHERE transaction_type IN ('REFUND', 'REVERSAL') AND reference_type <> 'CONTRIBUTION_REFUND';

CREATE FUNCTION validate_contribution_compensation() RETURNS trigger AS $$
DECLARE
    original_amount bigint;
    compensated bigint;
BEGIN
    SELECT amount INTO original_amount FROM contribution_entries WHERE entry_id=NEW.original_entry_id FOR UPDATE;
    SELECT COALESCE(sum(amount),0)::bigint INTO compensated FROM contribution_compensations WHERE original_entry_id=NEW.original_entry_id;
    IF original_amount IS NULL OR compensated+NEW.amount>original_amount THEN
        RAISE EXCEPTION 'contribution refund exceeds original entitlement' USING ERRCODE='23514';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM contribution_entries e
        JOIN fb_ledger_transactions original ON original.transaction_id=e.fb_transaction_id
        JOIN fb_ledger_transactions refund ON refund.transaction_id=NEW.fb_transaction_id
        JOIN fb_ledger_entries player_entry ON player_entry.transaction_id=refund.transaction_id AND player_entry.account_id=e.player_fb_account_id
        JOIN fb_ledger_entries system_entry ON system_entry.transaction_id=refund.transaction_id AND system_entry.account_id=e.system_fb_account_id
        WHERE e.entry_id=NEW.original_entry_id AND e.player_id=NEW.player_id
          AND e.fb_transaction_id=NEW.original_fb_transaction_id
          AND e.rule_version=NEW.rule_version AND e.amount>=NEW.amount
          AND original.transaction_type='SYSTEM_SPEND' AND original.spend_classification='ELIGIBLE'
          AND refund.original_transaction_id=original.transaction_id
          AND refund.transaction_type IN ('REFUND','REVERSAL')
          AND refund.reference_type='CONTRIBUTION_REFUND' AND refund.reference_id=NEW.reference_id
          AND player_entry.amount=NEW.amount AND system_entry.amount=-NEW.amount
          AND (SELECT count(*) FROM fb_ledger_entries WHERE transaction_id=refund.transaction_id)=2
    ) THEN
        RAISE EXCEPTION 'invalid contribution refund linkage' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER contribution_compensation_valid BEFORE INSERT ON contribution_compensations
    FOR EACH ROW EXECUTE FUNCTION validate_contribution_compensation();

CREATE FUNCTION require_contribution_compensation() RETURNS trigger AS $$
DECLARE
    original_reference text;
BEGIN
    IF NEW.transaction_type IN ('REFUND','REVERSAL') AND NEW.original_transaction_id IS NOT NULL THEN
        SELECT reference_type INTO original_reference FROM fb_ledger_transactions WHERE transaction_id=NEW.original_transaction_id;
    END IF;
    IF NEW.reference_type='CONTRIBUTION_REFUND' OR original_reference='CONTRIBUTION_SYSTEM_SPEND' THEN
        IF NEW.reference_type<>'CONTRIBUTION_REFUND' OR original_reference<>'CONTRIBUTION_SYSTEM_SPEND' OR NOT EXISTS (
            SELECT 1 FROM contribution_compensations c WHERE c.fb_transaction_id=NEW.transaction_id
        ) THEN
            RAISE EXCEPTION 'FB contribution refund lacks matching compensation' USING ERRCODE='23514';
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
CREATE CONSTRAINT TRIGGER fb_contribution_refund_atomic
    AFTER INSERT ON fb_ledger_transactions DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION require_contribution_compensation();
