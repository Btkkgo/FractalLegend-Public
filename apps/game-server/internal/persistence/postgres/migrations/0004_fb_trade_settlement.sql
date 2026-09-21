ALTER TABLE trade_sessions
    ADD COLUMN player_a_fb_offer bigint NOT NULL DEFAULT 0 CHECK (player_a_fb_offer >= 0),
    ADD COLUMN player_b_fb_offer bigint NOT NULL DEFAULT 0 CHECK (player_b_fb_offer >= 0);

ALTER TABLE trade_settlements
    ADD COLUMN ledger_transaction_ids text[] NOT NULL DEFAULT '{}';

ALTER TABLE trade_audit_events
    ADD COLUMN ledger_transaction_ids text[] NOT NULL DEFAULT '{}';

CREATE INDEX trade_settlements_ledger_transactions_idx
    ON trade_settlements USING gin (ledger_transaction_ids);
