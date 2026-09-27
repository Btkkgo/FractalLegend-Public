-- G18.1: identity of the concrete current row, independent of business fields.
-- The volatile default assigns a fresh value to every existing row atomically.
-- No historical block/economic field or journal is rewritten.
ALTER TABLE mining_blocks ADD COLUMN block_instance_id text NOT NULL
    DEFAULT ('mining-block-instance-' || replace(gen_random_uuid()::text, '-', ''));
ALTER TABLE mining_blocks ADD CONSTRAINT mining_blocks_instance_id_unique UNIQUE (block_instance_id);
ALTER TABLE mining_blocks ADD CONSTRAINT mining_blocks_instance_id_valid CHECK (
    block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$'
    AND block_instance_id <> 'mining-block-instance-00000000000000000000000000000000'
);

CREATE FUNCTION reject_mining_block_instance_identity_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.block_instance_id IS DISTINCT FROM OLD.block_instance_id THEN
        RAISE EXCEPTION 'mining block instance identity is immutable' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER mining_blocks_instance_identity_immutable BEFORE UPDATE ON mining_blocks
    FOR EACH ROW EXECUTE FUNCTION reject_mining_block_instance_identity_mutation();
