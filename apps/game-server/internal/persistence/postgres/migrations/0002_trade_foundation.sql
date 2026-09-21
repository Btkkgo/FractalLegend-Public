CREATE TABLE trade_sessions (
    trade_id text PRIMARY KEY CHECK (trade_id <> '' AND char_length(trade_id) <= 128),
    player_a_id text NOT NULL REFERENCES characters(id),
    player_b_id text NOT NULL REFERENCES characters(id),
    state text NOT NULL CHECK (state IN ('NEGOTIATING', 'READY_TO_SETTLE', 'COMPLETED', 'CANCELLED', 'EXPIRED')),
    revision bigint NOT NULL CHECK (revision >= 1),
    player_a_confirmed_revision bigint NOT NULL DEFAULT 0 CHECK (player_a_confirmed_revision >= 0),
    player_b_confirmed_revision bigint NOT NULL DEFAULT 0 CHECK (player_b_confirmed_revision >= 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    completed_at timestamptz,
    cancelled_at timestamptz,
    CHECK (player_a_id <> player_b_id)
);

CREATE TABLE trade_offers (
    trade_id text NOT NULL REFERENCES trade_sessions(trade_id) ON DELETE CASCADE,
    player_id text NOT NULL REFERENCES characters(id),
    item_instance_id text NOT NULL,
    quantity integer NOT NULL CHECK (quantity >= 1),
    PRIMARY KEY (trade_id, player_id, item_instance_id)
);

CREATE TABLE trade_item_locks (
    item_instance_id text PRIMARY KEY,
    trade_id text NOT NULL REFERENCES trade_sessions(trade_id) ON DELETE CASCADE,
    owner_character_id text NOT NULL REFERENCES characters(id),
    quantity integer NOT NULL CHECK (quantity >= 1),
    created_at timestamptz NOT NULL
);

CREATE TABLE trade_settlements (
    settlement_id text PRIMARY KEY CHECK (settlement_id <> '' AND char_length(settlement_id) <= 128),
    trade_id text NOT NULL UNIQUE REFERENCES trade_sessions(trade_id),
    completed_at timestamptz NOT NULL
);

CREATE TABLE trade_audit_events (
    sequence bigserial PRIMARY KEY,
    trade_id text NOT NULL REFERENCES trade_sessions(trade_id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind <> '' AND char_length(kind) <= 64),
    revision bigint NOT NULL CHECK (revision >= 1),
    previous_state text NOT NULL,
    new_state text NOT NULL,
    player_a_id text NOT NULL,
    player_b_id text NOT NULL,
    occurred_at timestamptz NOT NULL,
    outcome text NOT NULL CHECK (outcome <> '' AND char_length(outcome) <= 64)
);

CREATE INDEX trade_sessions_participants_idx ON trade_sessions(player_a_id, player_b_id);
CREATE INDEX trade_item_locks_trade_idx ON trade_item_locks(trade_id);
CREATE INDEX trade_audit_events_trade_idx ON trade_audit_events(trade_id, sequence);
