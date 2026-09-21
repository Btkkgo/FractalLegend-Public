CREATE TABLE accounts (
    id text PRIMARY KEY CHECK (id <> '' AND char_length(id) <= 128),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE characters (
    id text PRIMARY KEY CHECK (id <> '' AND char_length(id) <= 128),
    account_id text NOT NULL REFERENCES accounts(id),
    name text NOT NULL CHECK (name <> '' AND char_length(name) <= 128),
    class_id text NOT NULL CHECK (class_id <> '' AND char_length(class_id) <= 256),
    class_name text NOT NULL CHECK (class_name <> '' AND char_length(class_name) <= 128),
    level integer NOT NULL CHECK (level >= 1),
    exp bigint NOT NULL CHECK (exp >= 0),
    current_hp double precision NOT NULL CHECK (current_hp >= 0),
    current_mp double precision NOT NULL CHECK (current_mp >= 0),
    revision bigint NOT NULL CHECK (revision >= 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE character_world_state (
    character_id text PRIMARY KEY REFERENCES characters(id) ON DELETE CASCADE,
    map_id text NOT NULL CHECK (map_id <> '' AND char_length(map_id) <= 512),
    x integer NOT NULL,
    y integer NOT NULL
);

CREATE TABLE character_inventory_items (
    instance_id text PRIMARY KEY CHECK (instance_id <> '' AND char_length(instance_id) <= 128),
    character_id text NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    definition_id text NOT NULL CHECK (definition_id <> '' AND char_length(definition_id) <= 512),
    legacy_id integer NOT NULL,
    name text NOT NULL CHECK (name <> '' AND char_length(name) <= 128),
    item_type text NOT NULL CHECK (item_type <> '' AND char_length(item_type) <= 64),
    quantity integer NOT NULL CHECK (quantity >= 1),
    slot_index integer NOT NULL CHECK (slot_index >= 0),
    location text NOT NULL CHECK (location IN ('INVENTORY', 'EQUIPMENT')),
    equipment_slot text,
    UNIQUE (character_id, instance_id),
    UNIQUE (character_id, slot_index),
    CHECK ((location = 'INVENTORY' AND equipment_slot IS NULL) OR (location = 'EQUIPMENT' AND equipment_slot IS NOT NULL))
);

CREATE TABLE character_equipment (
    character_id text NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    slot text NOT NULL CHECK (slot <> '' AND char_length(slot) <= 64),
    item_instance_id text NOT NULL UNIQUE,
    PRIMARY KEY (character_id, slot),
    FOREIGN KEY (character_id, item_instance_id)
        REFERENCES character_inventory_items(character_id, instance_id)
        ON DELETE CASCADE
);

CREATE TABLE character_skills (
    character_id text NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    skill_id text NOT NULL CHECK (skill_id <> '' AND char_length(skill_id) <= 512),
    learned boolean NOT NULL,
    skill_level integer NOT NULL CHECK (skill_level >= 1),
    PRIMARY KEY (character_id, skill_id)
);

CREATE INDEX character_inventory_owner_idx ON character_inventory_items(character_id);
CREATE INDEX characters_account_idx ON characters(account_id);
