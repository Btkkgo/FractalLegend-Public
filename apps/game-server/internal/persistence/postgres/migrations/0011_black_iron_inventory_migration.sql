-- G19: compatibility only. No production aliases, balances or ore are seeded.
CREATE TABLE black_iron_identity_aliases (
    definition_id text PRIMARY KEY CHECK (definition_id <> '' AND char_length(definition_id) <= 512),
    legacy_id integer NOT NULL CHECK (legacy_id >= 0),
    legacy_name text NOT NULL CHECK (legacy_name <> '' AND legacy_name <> '黑铁矿石' AND char_length(legacy_name) <= 128),
    evidence text NOT NULL CHECK (btrim(evidence) <> '' AND char_length(evidence) <= 512)
);
CREATE TABLE black_iron_migration_receipts (
    version text NOT NULL CHECK (version = 'G19_BLACK_IRON_V1'),
    character_id text NOT NULL REFERENCES characters(id),
    status text NOT NULL CHECK (status = 'COMPLETE'),
    target_kind text NOT NULL CHECK (target_kind = 'BLACK_IRON_ORE'),
    pre_total bigint NOT NULL CHECK (pre_total >= 0),
    post_total bigint NOT NULL CHECK (post_total = pre_total),
    receipt jsonb NOT NULL CHECK (jsonb_typeof(receipt) = 'object'),
    created_at timestamptz NOT NULL,
    completed_at timestamptz NOT NULL CHECK (completed_at >= created_at),
    PRIMARY KEY (version,character_id)
);
CREATE TRIGGER black_iron_migration_receipts_immutable BEFORE UPDATE OR DELETE ON black_iron_migration_receipts
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER black_iron_identity_aliases_immutable BEFORE UPDATE OR DELETE ON black_iron_identity_aliases
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- Persistent freeze row: a stale Repeatable Read writer fails serialization.
CREATE TABLE black_iron_migration_catalog (
    singleton boolean PRIMARY KEY CHECK (singleton),
    frozen boolean NOT NULL DEFAULT false
);
INSERT INTO black_iron_migration_catalog(singleton) VALUES(true);
CREATE FUNCTION guard_black_iron_identity_catalog() RETURNS trigger AS $$
DECLARE is_frozen boolean;
BEGIN
    SELECT frozen INTO STRICT is_frozen FROM black_iron_migration_catalog WHERE singleton FOR UPDATE;
    IF is_frozen THEN
        RAISE EXCEPTION 'black iron identity catalog is frozen' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER black_iron_identity_catalog_insert BEFORE INSERT ON black_iron_identity_aliases
    FOR EACH ROW EXECUTE FUNCTION guard_black_iron_identity_catalog();
CREATE FUNCTION guard_black_iron_catalog_freeze() RETURNS trigger AS $$
BEGIN
    IF OLD.frozen AND NOT NEW.frozen THEN
        RAISE EXCEPTION 'black iron catalog cannot be unfrozen' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER black_iron_catalog_freeze_update BEFORE UPDATE ON black_iron_migration_catalog
    FOR EACH ROW EXECUTE FUNCTION guard_black_iron_catalog_freeze();
CREATE TRIGGER black_iron_catalog_delete BEFORE DELETE ON black_iron_migration_catalog
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER black_iron_catalog_truncate BEFORE TRUNCATE ON black_iron_migration_catalog
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER black_iron_aliases_truncate BEFORE TRUNCATE ON black_iron_identity_aliases
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER black_iron_receipts_truncate BEFORE TRUNCATE ON black_iron_migration_receipts
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- Legacy-compatible aggregate writers keep the one canonical representation.
-- Quantity/identity/ownership are not created or changed by this trigger.
CREATE FUNCTION normalize_migrated_black_iron_item() RETURNS trigger AS $$
DECLARE alias black_iron_identity_aliases%ROWTYPE;
BEGIN
    IF EXISTS(SELECT 1 FROM black_iron_migration_receipts WHERE character_id=NEW.character_id) THEN
        SELECT * INTO alias FROM black_iron_identity_aliases WHERE definition_id=NEW.definition_id;
        IF FOUND THEN
            IF NEW.legacy_id <> alias.legacy_id OR NEW.item_type <> 'MATERIAL'
              OR NEW.name NOT IN (alias.legacy_name,'黑铁矿石') OR NEW.location <> 'INVENTORY'
              OR NEW.equipment_slot IS NOT NULL THEN
                RAISE EXCEPTION 'invalid black iron compatibility identity' USING ERRCODE='23514';
            END IF;
            NEW.name := '黑铁矿石';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER inventory_black_iron_compatibility BEFORE INSERT OR UPDATE ON character_inventory_items
    FOR EACH ROW EXECUTE FUNCTION normalize_migrated_black_iron_item();

-- Commitments are migration provenance, not a second inventory or balance.
CREATE TABLE black_iron_migration_assets (
    instance_id text PRIMARY KEY REFERENCES item_instance_lifecycle(instance_id),
    version text NOT NULL,
    character_id text NOT NULL,
    definition_id text NOT NULL REFERENCES black_iron_identity_aliases(definition_id),
    legacy_id integer NOT NULL,
    quantity integer NOT NULL CHECK (quantity > 0),
    FOREIGN KEY (version,character_id) REFERENCES black_iron_migration_receipts(version,character_id)
);
CREATE TRIGGER black_iron_assets_immutable BEFORE UPDATE OR DELETE ON black_iron_migration_assets
    FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER black_iron_assets_truncate BEFORE TRUNCATE ON black_iron_migration_assets
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- Ore movement/consumption is deliberately deferred. Aggregate replacement and
-- ordinary inventory slot reordering are allowed if identity/quantity/owner
-- survive at commit; whole/partial trade and destruction fail atomically.
CREATE FUNCTION guard_migrated_black_iron_asset() RETURNS trigger AS $$
DECLARE asset_id text; expected black_iron_migration_assets%ROWTYPE;
BEGIN
    IF TG_OP = 'DELETE' THEN asset_id := OLD.instance_id; ELSE asset_id := NEW.instance_id; END IF;
    SELECT * INTO expected FROM black_iron_migration_assets WHERE instance_id=asset_id;
    IF FOUND AND NOT EXISTS(
      SELECT 1 FROM character_inventory_items i WHERE i.instance_id=asset_id
        AND i.character_id=expected.character_id AND i.definition_id=expected.definition_id
        AND i.legacy_id=expected.legacy_id AND i.quantity=expected.quantity
        AND i.name='黑铁矿石' AND i.item_type='MATERIAL'
        AND i.location='INVENTORY' AND i.equipment_slot IS NULL
    ) THEN
        RAISE EXCEPTION 'black iron movement is not implemented' USING ERRCODE='23514';
    END IF;
    IF NOT FOUND AND EXISTS(
      SELECT 1 FROM character_inventory_items i JOIN black_iron_identity_aliases a ON a.definition_id=i.definition_id
      WHERE i.instance_id=asset_id AND i.name='黑铁矿石'
    ) THEN
        RAISE EXCEPTION 'black iron asset lacks migration provenance' USING ERRCODE='23514';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
CREATE CONSTRAINT TRIGGER inventory_black_iron_integrity AFTER INSERT OR UPDATE OR DELETE ON character_inventory_items
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION guard_migrated_black_iron_asset();
CREATE TRIGGER inventory_black_iron_truncate BEFORE TRUNCATE ON character_inventory_items
    FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();
