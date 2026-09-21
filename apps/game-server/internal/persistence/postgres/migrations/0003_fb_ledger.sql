CREATE TABLE fb_ledger_accounts (
    account_id text PRIMARY KEY CHECK (account_id <> '' AND char_length(account_id) <= 128),
    owner_id text NOT NULL CHECK (owner_id <> '' AND char_length(owner_id) <= 128),
    owner_type text NOT NULL CHECK (owner_type IN ('PLAYER', 'SYSTEM')),
    currency text NOT NULL DEFAULT 'FB' CHECK (currency = 'FB'),
    balance bigint NOT NULL DEFAULT 0 CHECK (owner_type = 'SYSTEM' OR balance >= 0),
    revision bigint NOT NULL DEFAULT 1 CHECK (revision >= 1),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (owner_type, owner_id)
);

CREATE TABLE fb_ledger_transactions (
    transaction_id text PRIMARY KEY CHECK (transaction_id <> '' AND char_length(transaction_id) <= 128),
    transaction_type text NOT NULL CHECK (transaction_type IN ('PLAYER_TRANSFER', 'SYSTEM_SPEND', 'REFUND', 'REVERSAL')),
    status text NOT NULL CHECK (status = 'POSTED'),
    reference_type text NOT NULL CHECK (reference_type <> '' AND char_length(reference_type) <= 64),
    reference_id text NOT NULL CHECK (reference_id <> '' AND char_length(reference_id) <= 256),
    original_transaction_id text REFERENCES fb_ledger_transactions(transaction_id),
    spend_classification text NOT NULL DEFAULT '' CHECK (spend_classification IN ('', 'ELIGIBLE', 'NON_ELIGIBLE')),
    reason text NOT NULL DEFAULT '' CHECK (char_length(reason) <= 512),
    created_at timestamptz NOT NULL,
    UNIQUE (reference_type, reference_id)
);

CREATE UNIQUE INDEX fb_ledger_single_compensation_idx
    ON fb_ledger_transactions(original_transaction_id)
    WHERE transaction_type IN ('REFUND', 'REVERSAL');

CREATE TABLE fb_ledger_entries (
    entry_id text PRIMARY KEY CHECK (entry_id <> '' AND char_length(entry_id) <= 128),
    transaction_id text NOT NULL REFERENCES fb_ledger_transactions(transaction_id),
    entry_order smallint NOT NULL CHECK (entry_order >= 0),
    account_id text NOT NULL REFERENCES fb_ledger_accounts(account_id),
    entry_type text NOT NULL CHECK (entry_type IN ('PLAYER_TRANSFER', 'SYSTEM_SPEND', 'REFUND', 'REVERSAL')),
    amount bigint NOT NULL CHECK (amount <> 0),
    direction text NOT NULL CHECK ((direction = 'DEBIT' AND amount < 0) OR (direction = 'CREDIT' AND amount > 0)),
    balance_before bigint NOT NULL,
    balance_after bigint NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK (balance_after = balance_before + amount),
    UNIQUE (transaction_id, account_id),
    UNIQUE (transaction_id, entry_order)
);

CREATE TABLE fb_ledger_audit_events (
    sequence bigserial PRIMARY KEY,
    transaction_id text NOT NULL REFERENCES fb_ledger_transactions(transaction_id),
    reference_type text NOT NULL,
    reference_id text NOT NULL,
    account_ids text[] NOT NULL CHECK (cardinality(account_ids) >= 1),
    amount bigint NOT NULL CHECK (amount > 0),
    event_type text NOT NULL CHECK (event_type <> '' AND char_length(event_type) <= 64),
    result text NOT NULL CHECK (result <> '' AND char_length(result) <= 64),
    occurred_at timestamptz NOT NULL
);

CREATE INDEX fb_ledger_entries_account_idx ON fb_ledger_entries(account_id, created_at, entry_id);
CREATE INDEX fb_ledger_audit_transaction_idx ON fb_ledger_audit_events(transaction_id, sequence);

CREATE FUNCTION validate_fb_ledger_transaction_balance() RETURNS trigger AS $$
DECLARE
    posting_total bigint;
BEGIN
    SELECT COALESCE(SUM(amount), 0)::bigint
      INTO posting_total
      FROM fb_ledger_entries
     WHERE transaction_id = NEW.transaction_id;
    IF posting_total <> 0 THEN
        RAISE EXCEPTION 'FB ledger transaction postings must sum to zero' USING ERRCODE = '23514';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER fb_ledger_transaction_balanced
    AFTER INSERT ON fb_ledger_entries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION validate_fb_ledger_transaction_balance();

CREATE FUNCTION reject_fb_ledger_history_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'posted FB ledger history is immutable' USING ERRCODE = '55000';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fb_ledger_transactions_immutable
    BEFORE UPDATE OR DELETE ON fb_ledger_transactions
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

CREATE TRIGGER fb_ledger_entries_immutable
    BEFORE UPDATE OR DELETE ON fb_ledger_entries
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
