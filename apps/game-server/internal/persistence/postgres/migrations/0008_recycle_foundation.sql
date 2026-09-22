-- G16: per-instance revision and permanent consumption guard on the existing inventory.
CREATE TABLE item_instance_lifecycle (
    instance_id text PRIMARY KEY CHECK (instance_id <> '' AND char_length(instance_id) <= 128),
    revision bigint NOT NULL CHECK (revision >= 1),
    consumed boolean NOT NULL DEFAULT false
);
INSERT INTO item_instance_lifecycle(instance_id,revision)
    SELECT instance_id,1 FROM character_inventory_items;

CREATE FUNCTION track_item_instance_revision() RETURNS trigger AS $$
DECLARE next_revision bigint;
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.instance_id <> OLD.instance_id THEN
        RAISE EXCEPTION 'item instance ID cannot change' USING ERRCODE='23514';
    END IF;
    INSERT INTO item_instance_lifecycle(instance_id,revision,consumed)
        VALUES(NEW.instance_id,1,false)
        ON CONFLICT(instance_id) DO UPDATE
            SET revision=item_instance_lifecycle.revision+1
            WHERE NOT item_instance_lifecycle.consumed
        RETURNING revision INTO next_revision;
    IF next_revision IS NULL THEN
        RAISE EXCEPTION 'consumed item instance cannot reenter inventory' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER inventory_item_lifecycle_insert BEFORE INSERT ON character_inventory_items
    FOR EACH ROW EXECUTE FUNCTION track_item_instance_revision();
CREATE TRIGGER inventory_item_lifecycle_update BEFORE UPDATE ON character_inventory_items
    FOR EACH ROW EXECUTE FUNCTION track_item_instance_revision();

CREATE TABLE reputation_accounts (
    player_id text PRIMARY KEY REFERENCES characters(id),
    balance bigint NOT NULL DEFAULT 0 CHECK (balance >= 0),
    revision bigint NOT NULL DEFAULT 1 CHECK (revision >= 1)
);
CREATE TABLE recycle_receipts (
    operation_id text PRIMARY KEY CHECK (operation_id <> '' AND char_length(operation_id) <= 120),
    player_id text NOT NULL REFERENCES characters(id),
    item_instance_id text NOT NULL UNIQUE REFERENCES item_instance_lifecycle(instance_id),
    item_template_id text NOT NULL,
    item_revision bigint NOT NULL CHECK (item_revision >= 1),
    rule_id text NOT NULL,
    rule_version text NOT NULL,
    rule_snapshot jsonb NOT NULL CHECK (jsonb_typeof(rule_snapshot) = 'object'),
    requested_at timestamptz NOT NULL,
    materials_awarded jsonb NOT NULL CHECK (jsonb_typeof(materials_awarded) = 'array'),
    reputation_awarded bigint NOT NULL CHECK (reputation_awarded >= 0),
    fb_awarded bigint NOT NULL DEFAULT 0 CHECK (fb_awarded = 0),
    contribution_awarded bigint NOT NULL DEFAULT 0 CHECK (contribution_awarded = 0),
    black_iron_awarded bigint NOT NULL DEFAULT 0 CHECK (black_iron_awarded = 0),
    created_at timestamptz NOT NULL
);
CREATE TABLE reputation_entries (
    operation_id text PRIMARY KEY CHECK (operation_id <> '' AND char_length(operation_id) <= 120),
    player_id text NOT NULL REFERENCES characters(id),
    amount bigint NOT NULL CHECK (amount >= 0),
    created_at timestamptz NOT NULL,
    FOREIGN KEY (operation_id) REFERENCES recycle_receipts(operation_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE recycle_material_credits (
    operation_id text NOT NULL REFERENCES recycle_receipts(operation_id) DEFERRABLE INITIALLY DEFERRED,
    material_id text NOT NULL CHECK (material_id <> 'BLACK_IRON_ORE'),
    quantity bigint NOT NULL CHECK (quantity >= 0),
    inventory_instance_id text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (operation_id, material_id)
);
CREATE TABLE recycle_audit_events (
    operation_id text PRIMARY KEY REFERENCES recycle_receipts(operation_id) DEFERRABLE INITIALLY DEFERRED,
    kind text NOT NULL CHECK (kind = 'RECYCLE_COMMITTED'),
    occurred_at timestamptz NOT NULL
);
CREATE INDEX recycle_receipts_player_idx ON recycle_receipts(player_id,created_at);
CREATE TRIGGER reputation_entries_immutable BEFORE UPDATE OR DELETE ON reputation_entries
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER recycle_receipts_immutable BEFORE UPDATE OR DELETE ON recycle_receipts
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER recycle_material_credits_immutable BEFORE UPDATE OR DELETE ON recycle_material_credits
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER recycle_audit_events_immutable BEFORE UPDATE OR DELETE ON recycle_audit_events
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
