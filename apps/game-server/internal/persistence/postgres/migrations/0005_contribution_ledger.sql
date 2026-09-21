CREATE TABLE contribution_accounts (
    player_id text PRIMARY KEY CHECK (player_id <> '' AND char_length(player_id) <= 128),
    balance bigint NOT NULL DEFAULT 0 CHECK (balance >= 0),
    revision bigint NOT NULL DEFAULT 1 CHECK (revision >= 1),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE contribution_entries (
    entry_id text PRIMARY KEY CHECK (entry_id <> '' AND char_length(entry_id) <= 128),
    player_id text NOT NULL REFERENCES contribution_accounts(player_id),
    player_fb_account_id text NOT NULL REFERENCES fb_ledger_accounts(account_id),
    system_fb_account_id text NOT NULL REFERENCES fb_ledger_accounts(account_id),
    source text NOT NULL CHECK (source <> '' AND char_length(source) <= 64),
    source_id text NOT NULL CHECK (source_id <> '' AND char_length(source_id) <= 128),
    eligible_spend bigint NOT NULL CHECK (eligible_spend > 0),
    amount bigint NOT NULL CHECK (amount > 0),
    balance_before bigint NOT NULL CHECK (balance_before >= 0),
    balance_after bigint NOT NULL CHECK (balance_after >= 0 AND balance_after = balance_before + amount),
    rule_version text NOT NULL CHECK (rule_version <> '' AND char_length(rule_version) <= 64),
    fb_transaction_id text NOT NULL UNIQUE REFERENCES fb_ledger_transactions(transaction_id),
    created_at timestamptz NOT NULL,
    UNIQUE (source, source_id, rule_version),
    CHECK (rule_version = 'CONTRIBUTION_RULE_V1' AND source = 'SYSTEM_SERVICE' AND amount = eligible_spend)
);

CREATE TABLE contribution_audit_events (
    sequence bigserial PRIMARY KEY,
    entry_id text NOT NULL REFERENCES contribution_entries(entry_id),
    player_id text NOT NULL REFERENCES contribution_accounts(player_id),
    source text NOT NULL,
    source_id text NOT NULL,
    rule_version text NOT NULL,
    amount bigint NOT NULL CHECK (amount > 0),
    fb_transaction_id text NOT NULL REFERENCES fb_ledger_transactions(transaction_id),
    created_at timestamptz NOT NULL
);

CREATE INDEX contribution_entries_player_idx ON contribution_entries(player_id, created_at, entry_id);
CREATE INDEX contribution_audit_player_idx ON contribution_audit_events(player_id, sequence);

CREATE FUNCTION validate_contribution_spend_reference() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM fb_ledger_transactions t
          JOIN fb_ledger_entries debit ON debit.transaction_id = t.transaction_id
          JOIN fb_ledger_entries credit ON credit.transaction_id = t.transaction_id
          JOIN fb_ledger_accounts player_account ON player_account.account_id = debit.account_id
          JOIN fb_ledger_accounts system_account ON system_account.account_id = credit.account_id
         WHERE t.transaction_id = NEW.fb_transaction_id
           AND t.transaction_type = 'SYSTEM_SPEND'
           AND t.status = 'POSTED'
           AND t.spend_classification = 'ELIGIBLE'
           AND t.reference_type = 'CONTRIBUTION_SYSTEM_SPEND'
           AND t.reference_id = NEW.source || ':' || NEW.source_id || ':' || NEW.rule_version
           AND debit.account_id = NEW.player_fb_account_id
           AND debit.amount = -NEW.eligible_spend
           AND credit.account_id = NEW.system_fb_account_id
           AND credit.amount = NEW.eligible_spend
           AND player_account.owner_type = 'PLAYER'
           AND player_account.owner_id = NEW.player_id
           AND system_account.owner_type = 'SYSTEM'
           AND (SELECT count(*) FROM fb_ledger_entries WHERE transaction_id = t.transaction_id) = 2
    ) THEN
        RAISE EXCEPTION 'contribution requires the matching eligible FB system spend' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER contribution_requires_eligible_fb_spend
    BEFORE INSERT ON contribution_entries
    FOR EACH ROW EXECUTE FUNCTION validate_contribution_spend_reference();

CREATE TRIGGER contribution_entries_immutable
    BEFORE UPDATE OR DELETE ON contribution_entries
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TRIGGER contribution_audit_immutable
    BEFORE UPDATE OR DELETE ON contribution_audit_events
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
