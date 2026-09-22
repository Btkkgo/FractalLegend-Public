-- G15: immutable server-side spend intent and links to the existing ledgers.
-- Refunded amount and status are derived from immutable G14 compensation rows.
CREATE TABLE system_spends (
    spend_id text PRIMARY KEY CHECK (spend_id <> '' AND char_length(spend_id) <= 128),
    operation_id text NOT NULL UNIQUE CHECK (operation_id <> '' AND char_length(operation_id) <= 120),
    player_id text NOT NULL CHECK (player_id <> '' AND char_length(player_id) <= 128),
    player_fb_account_id text NOT NULL REFERENCES fb_ledger_accounts(account_id),
    system_fb_account_id text NOT NULL REFERENCES fb_ledger_accounts(account_id),
    producer_type text NOT NULL CHECK (producer_type <> '' AND char_length(producer_type) <= 64),
    producer_reference text NOT NULL CHECK (producer_reference <> '' AND char_length(producer_reference) <= 128),
    fb_amount bigint NOT NULL CHECK (fb_amount > 0),
    eligible boolean NOT NULL,
    rule_version text NOT NULL CHECK (char_length(rule_version) <= 64),
    contribution_amount bigint NOT NULL CHECK (contribution_amount >= 0),
    fb_transaction_id text NOT NULL UNIQUE REFERENCES fb_ledger_transactions(transaction_id),
    contribution_entry_id text UNIQUE REFERENCES contribution_entries(entry_id),
    refundable boolean NOT NULL,
    partial_refund_allowed boolean NOT NULL,
    status text NOT NULL CHECK (status = 'COMPLETED'),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    created_at timestamptz NOT NULL,
    completed_at timestamptz NOT NULL,
    UNIQUE (producer_type, producer_reference),
    CHECK (player_fb_account_id <> system_fb_account_id),
    CHECK (NOT partial_refund_allowed OR refundable),
    CHECK ((eligible AND rule_version <> '' AND contribution_amount > 0 AND contribution_entry_id IS NOT NULL)
        OR (NOT eligible AND rule_version = '' AND contribution_amount = 0 AND contribution_entry_id IS NULL))
);
CREATE INDEX system_spends_player_idx ON system_spends(player_id, created_at, spend_id);
CREATE TRIGGER system_spends_immutable BEFORE UPDATE OR DELETE ON system_spends
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE FUNCTION validate_system_spend_links() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM fb_ledger_transactions t
        JOIN fb_ledger_entries debit ON debit.transaction_id=t.transaction_id AND debit.account_id=NEW.player_fb_account_id
        JOIN fb_ledger_entries credit ON credit.transaction_id=t.transaction_id AND credit.account_id=NEW.system_fb_account_id
        JOIN fb_ledger_accounts player_account ON player_account.account_id=debit.account_id
        JOIN fb_ledger_accounts system_account ON system_account.account_id=credit.account_id
        WHERE t.transaction_id=NEW.fb_transaction_id
          AND t.transaction_type='SYSTEM_SPEND' AND t.status='POSTED'
          AND t.spend_classification=CASE WHEN NEW.eligible THEN 'ELIGIBLE' ELSE 'NON_ELIGIBLE' END
          AND debit.amount=-NEW.fb_amount AND credit.amount=NEW.fb_amount
          AND player_account.owner_type='PLAYER' AND player_account.owner_id=NEW.player_id
          AND system_account.owner_type='SYSTEM'
          AND (SELECT count(*) FROM fb_ledger_entries WHERE transaction_id=t.transaction_id)=2
    ) THEN
        RAISE EXCEPTION 'system spend lacks matching FB debit' USING ERRCODE='23514';
    END IF;
    IF NEW.eligible AND NOT EXISTS (
        SELECT 1 FROM contribution_entries c
        WHERE c.entry_id=NEW.contribution_entry_id
          AND c.fb_transaction_id=NEW.fb_transaction_id
          AND c.player_id=NEW.player_id
          AND c.player_fb_account_id=NEW.player_fb_account_id
          AND c.system_fb_account_id=NEW.system_fb_account_id
          AND c.source='SYSTEM_SERVICE'
          AND c.source_id='g15:' || NEW.operation_id
          AND c.eligible_spend=NEW.fb_amount
          AND c.amount=NEW.contribution_amount
          AND c.rule_version=NEW.rule_version
    ) THEN
        RAISE EXCEPTION 'eligible system spend lacks matching Contribution credit' USING ERRCODE='23514';
    END IF;
    IF NOT NEW.eligible AND EXISTS (
        SELECT 1 FROM contribution_entries WHERE fb_transaction_id=NEW.fb_transaction_id
    ) THEN
        RAISE EXCEPTION 'non-eligible system spend minted Contribution' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER system_spend_links_valid BEFORE INSERT ON system_spends
    FOR EACH ROW EXECUTE FUNCTION validate_system_spend_links();
