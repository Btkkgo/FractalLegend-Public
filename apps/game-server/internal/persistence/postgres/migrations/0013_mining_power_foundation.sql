-- G20: TEST / NON-PRODUCTION facts only. No seeds, reward or upstream writes.
-- JSON retains the complete immutable domain record; generated columns make
-- its identities/bounds the very same values enforced by relational constraints.
CREATE FUNCTION mining_power_valid_id(value text, maximum integer) RETURNS boolean
LANGUAGE sql IMMUTABLE AS $$ SELECT value IS NOT NULL AND length(value) BETWEEN 1 AND maximum AND value ~ '^[A-Za-z0-9_.:-]+$' $$;
CREATE FUNCTION mining_power_has_nonnull(value jsonb, keys text[]) RETURNS boolean
LANGUAGE sql IMMUTABLE AS $$ SELECT coalesce(bool_and(value->>key IS NOT NULL),true) FROM unnest(keys) AS key $$;
CREATE UNIQUE INDEX mining_power_character_owner ON characters(id,account_id);

CREATE TABLE mining_power_rules (
 data jsonb NOT NULL,
 rule_version text GENERATED ALWAYS AS (data->>'Version') STORED PRIMARY KEY,
 CHECK (data = '{"Version":"DEV_G20_MINING_POWER_V1","Kind":"TEST_NON_PRODUCTION","AlgorithmID":"INTEGER_PRODUCT_FLOOR_ONCE_V1","Scale":1000000,"MaxBasePower":9223372036854775807,"MaxMultiplier":1000000000,"MaxPower":9223372036854775807}'::jsonb)
);
CREATE TABLE mining_power_tool_profiles (
 data jsonb NOT NULL,
 rule_version text GENERATED ALWAYS AS (data->>'RuleVersion') STORED NOT NULL REFERENCES mining_power_rules(rule_version),
 tool_reference text GENERATED ALWAYS AS (data->>'Reference') STORED NOT NULL,
 base_power bigint GENERATED ALWAYS AS ((data->>'BasePowerUnits')::bigint) STORED NOT NULL CHECK(base_power>=0),
 efficiency bigint GENERATED ALWAYS AS ((data->>'EfficiencyScaled')::bigint) STORED NOT NULL CHECK(efficiency BETWEEN 1 AND 1000000000),
 PRIMARY KEY(rule_version,tool_reference),
 CHECK(mining_power_valid_id(tool_reference,128)), CHECK(data->>'Kind' IS NOT NULL AND data->>'Kind'='TEST_NON_PRODUCTION')
);
CREATE TABLE mining_power_map_profiles (
 data jsonb NOT NULL,
 rule_version text GENERATED ALWAYS AS (data->>'RuleVersion') STORED NOT NULL REFERENCES mining_power_rules(rule_version),
 map_reference text GENERATED ALWAYS AS (data->>'Reference') STORED NOT NULL,
 modifier bigint GENERATED ALWAYS AS ((data->>'ModifierScaled')::bigint) STORED NOT NULL CHECK(modifier BETWEEN 1 AND 1000000000),
 PRIMARY KEY(rule_version,map_reference),
 CHECK(mining_power_valid_id(map_reference,128)), CHECK(data->>'Kind' IS NOT NULL AND data->>'Kind'='TEST_NON_PRODUCTION')
);
CREATE TABLE mining_power_sessions (
 data jsonb NOT NULL,
 session_id text GENERATED ALWAYS AS (data->>'ID') STORED PRIMARY KEY,
 player_id text GENERATED ALWAYS AS (data->>'PlayerID') STORED NOT NULL,
 account_id text GENERATED ALWAYS AS (data->>'AccountID') STORED NOT NULL,
 block_instance_id text GENERATED ALWAYS AS (data->>'BlockInstanceID') STORED NOT NULL,
 block_id text GENERATED ALWAYS AS (data->>'BlockID') STORED NOT NULL,
 rule_version text GENERATED ALWAYS AS (data->>'RuleVersion') STORED NOT NULL,
 tool_reference text GENERATED ALWAYS AS (data->>'ToolReference') STORED NOT NULL,
 map_reference text GENERATED ALWAYS AS (data->>'MapReference') STORED NOT NULL,
 state text GENERATED ALWAYS AS (data->>'State') STORED NOT NULL CHECK(state IN ('ACTIVE','CLOSED')),
 FOREIGN KEY(player_id,account_id) REFERENCES characters(id,account_id),
 FOREIGN KEY(rule_version,tool_reference) REFERENCES mining_power_tool_profiles(rule_version,tool_reference),
 FOREIGN KEY(rule_version,map_reference) REFERENCES mining_power_map_profiles(rule_version,map_reference),
 UNIQUE(session_id,player_id,block_instance_id,block_id,rule_version,tool_reference,map_reference),
 CHECK(mining_power_valid_id(session_id,128) AND mining_power_valid_id(player_id,128) AND mining_power_valid_id(account_id,128) AND mining_power_valid_id(block_id,128)),
 CHECK(block_instance_id ~ '^mining-block-instance-[0-9a-f]{32}$' AND block_instance_id<>'mining-block-instance-00000000000000000000000000000000'),
 CHECK(data ?& ARRAY['OpenedAt','ExpiresAt','CreatedAt','UpdatedAt']),
 CHECK((data->>'ExpiresAt')::timestamptz>(data->>'OpenedAt')::timestamptz AND (data->>'UpdatedAt')::timestamptz>=(data->>'CreatedAt')::timestamptz)
);
CREATE TABLE mining_power_source_events (
 data jsonb NOT NULL,
 source_event_id text GENERATED ALWAYS AS (data->>'ID') STORED PRIMARY KEY,
 activity_id text GENERATED ALWAYS AS (data->>'ActivityID') STORED NOT NULL UNIQUE,
 player_id text GENERATED ALWAYS AS (data->>'PlayerID') STORED NOT NULL,
 block_instance_id text GENERATED ALWAYS AS (data->>'BlockInstanceID') STORED NOT NULL,
 block_id text GENERATED ALWAYS AS (data->>'BlockID') STORED NOT NULL,
 session_id text GENERATED ALWAYS AS (data->>'ActivitySessionID') STORED NOT NULL,
 rule_version text GENERATED ALWAYS AS (data->>'RuleVersion') STORED NOT NULL,
 tool_reference text GENERATED ALWAYS AS (data->>'ToolReference') STORED NOT NULL,
 map_reference text GENERATED ALWAYS AS (data->>'MapReference') STORED NOT NULL,
 activity_weight bigint GENERATED ALWAYS AS ((data->>'ActivityWeightScaled')::bigint) STORED NOT NULL CHECK(activity_weight BETWEEN 0 AND 1000000000),
 FOREIGN KEY(session_id,player_id,block_instance_id,block_id,rule_version,tool_reference,map_reference) REFERENCES mining_power_sessions(session_id,player_id,block_instance_id,block_id,rule_version,tool_reference,map_reference),
 UNIQUE(source_event_id,activity_id,player_id,block_instance_id,block_id,session_id,rule_version,tool_reference,map_reference),
 CHECK(mining_power_valid_id(source_event_id,96) AND activity_id='mpa:'||source_event_id),
 CHECK(data->>'EvidenceKind' IS NOT NULL AND data->>'EvidenceKind'='SYNTHETIC_TEST'),
 CHECK(data->>'ServerEligibility' IS NOT NULL AND data->>'ServerEligibility' IN ('VALID','INVALID')),
 CHECK(data ?& ARRAY['ObservedAt','ExpiresAt'] AND (data->>'ExpiresAt')::timestamptz>(data->>'ObservedAt')::timestamptz)
);
CREATE TABLE mining_power_activities (
 data jsonb NOT NULL,
 activity_id text GENERATED ALWAYS AS (data->>'ActivityID') STORED PRIMARY KEY,
 source_event_id text GENERATED ALWAYS AS (data->>'SourceEventID') STORED NOT NULL UNIQUE,
 player_id text GENERATED ALWAYS AS (data->>'PlayerID') STORED NOT NULL,
 block_instance_id text GENERATED ALWAYS AS (data->>'BlockInstanceID') STORED NOT NULL,
 block_id text GENERATED ALWAYS AS (data->>'BlockID') STORED NOT NULL,
 session_id text GENERATED ALWAYS AS (data->>'ActivitySessionID') STORED NOT NULL,
 rule_version text GENERATED ALWAYS AS (data->>'RuleVersion') STORED NOT NULL,
 tool_reference text GENERATED ALWAYS AS (data->>'ToolReference') STORED NOT NULL,
 map_reference text GENERATED ALWAYS AS (data->>'MapReference') STORED NOT NULL,
 validated_power bigint GENERATED ALWAYS AS ((data->>'ValidatedPower')::bigint) STORED NOT NULL CHECK(validated_power>=0),
 base_power bigint GENERATED ALWAYS AS ((data->'Inputs'->>'BasePowerUnits')::bigint) STORED NOT NULL CHECK(base_power>=0),
 efficiency bigint GENERATED ALWAYS AS ((data->'Inputs'->>'EfficiencyScaled')::bigint) STORED NOT NULL CHECK(efficiency BETWEEN 1 AND 1000000000),
 activity_weight bigint GENERATED ALWAYS AS ((data->'Inputs'->>'ActivityWeightScaled')::bigint) STORED NOT NULL CHECK(activity_weight BETWEEN 1 AND 1000000000),
 map_modifier bigint GENERATED ALWAYS AS ((data->'Inputs'->>'MapModifierScaled')::bigint) STORED NOT NULL CHECK(map_modifier BETWEEN 1 AND 1000000000),
 FOREIGN KEY(source_event_id,activity_id,player_id,block_instance_id,block_id,session_id,rule_version,tool_reference,map_reference) REFERENCES mining_power_source_events(source_event_id,activity_id,player_id,block_instance_id,block_id,session_id,rule_version,tool_reference,map_reference),
 CHECK(activity_id='mpa:'||source_event_id),
 CHECK(data->>'Status' IS NOT NULL AND data->>'Status'='VALID' AND data->>'Decision' IS NOT NULL AND data->>'Decision'='ACCEPTED'),
 CHECK(validated_power::numeric = floor(base_power::numeric*efficiency::numeric*activity_weight::numeric*map_modifier::numeric/1000000000000000000::numeric)),
 CHECK(data ?& ARRAY['AcceptedAt','ObservedAt','ExpiresAt','ValidationSnapshot']),
 CHECK((data->>'AcceptedAt')::timestamptz>=(data->>'ObservedAt')::timestamptz AND (data->>'AcceptedAt')::timestamptz<(data->>'ExpiresAt')::timestamptz),
 CHECK(jsonb_typeof(data->'ValidationSnapshot')='object' AND data->'ValidationSnapshot' ?& ARRAY['BlockInstanceID','BlockID','BlockHeight','CreateCommandID','Status','G18RuleVersion','G20RuleVersion','SourceEvidenceVersion','StartedAt','ScheduledEndAt','ValidatedAt']),
 CHECK(data->'ValidationSnapshot'->>'BlockInstanceID'=block_instance_id AND data->'ValidationSnapshot'->>'BlockID'=block_id AND data->'ValidationSnapshot'->>'G20RuleVersion'=rule_version AND data->'ValidationSnapshot'->>'Status'='OPEN' AND data->'ValidationSnapshot'->>'SourceEvidenceVersion'='G18_SCHEMA_0012'),
 CHECK((data->'ValidationSnapshot'->>'BlockHeight')::bigint>0 AND mining_power_valid_id(data->'ValidationSnapshot'->>'CreateCommandID',128) AND mining_power_valid_id(data->'ValidationSnapshot'->>'G18RuleVersion',128)),
 CHECK((data->>'ObservedAt')::timestamptz>=(data->'ValidationSnapshot'->>'StartedAt')::timestamptz AND (data->>'AcceptedAt')::timestamptz<(data->'ValidationSnapshot'->>'ScheduledEndAt')::timestamptz AND (data->'ValidationSnapshot'->>'ValidatedAt')::timestamptz<=(data->>'AcceptedAt')::timestamptz)
);
CREATE INDEX mining_power_activity_context ON mining_power_activities(block_instance_id,rule_version,player_id,activity_id);
CREATE TABLE mining_power_participants (
 player_id text NOT NULL,
 block_instance_id text NOT NULL,
 block_id text NOT NULL,
 session_id text NOT NULL,
 rule_version text NOT NULL,
 tool_reference text NOT NULL,
 map_reference text NOT NULL,
 validated_power bigint NOT NULL CHECK(validated_power>=0),
 activity_count bigint NOT NULL CHECK(activity_count>0),
 created_at timestamptz NOT NULL,
 updated_at timestamptz NOT NULL CHECK(updated_at>=created_at),
 PRIMARY KEY(player_id,block_instance_id,session_id,rule_version),
 FOREIGN KEY(session_id,player_id,block_instance_id,block_id,rule_version,tool_reference,map_reference) REFERENCES mining_power_sessions(session_id,player_id,block_instance_id,block_id,rule_version,tool_reference,map_reference)
);
CREATE INDEX mining_power_participant_context ON mining_power_participants(block_instance_id,rule_version,player_id);

-- Reuse the existing immutable ledger guard without touching its definition.
DO $$ DECLARE relation text; BEGIN
 FOREACH relation IN ARRAY ARRAY['mining_power_rules','mining_power_tool_profiles','mining_power_map_profiles','mining_power_source_events','mining_power_activities'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation()',relation||'_immutable',relation);
  EXECUTE format('CREATE TRIGGER %I BEFORE TRUNCATE ON %I FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation()',relation||'_truncate',relation);
 END LOOP;
END $$;
CREATE FUNCTION mining_power_guard_session() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF (NEW.data - ARRAY['State','UpdatedAt']) IS DISTINCT FROM (OLD.data - ARRAY['State','UpdatedAt']) OR (OLD.data->>'State'='CLOSED' AND NEW.data->>'State'<>'CLOSED') THEN
  RAISE EXCEPTION 'mining session binding is immutable' USING ERRCODE='55000';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER mining_power_session_binding BEFORE UPDATE ON mining_power_sessions FOR EACH ROW EXECUTE FUNCTION mining_power_guard_session();
CREATE TRIGGER mining_power_session_delete BEFORE DELETE ON mining_power_sessions FOR EACH ROW EXECUTE FUNCTION reject_fb_ledger_history_mutation();
CREATE TRIGGER mining_power_session_truncate BEFORE TRUNCATE ON mining_power_sessions FOR EACH STATEMENT EXECUTE FUNCTION reject_fb_ledger_history_mutation();

-- CHECK permits SQL NULL; explicitly reject missing/JSON-null historical evidence.
ALTER TABLE mining_power_sessions ADD CHECK(mining_power_has_nonnull(data,ARRAY['OpenedAt','ExpiresAt','CreatedAt','UpdatedAt']));
ALTER TABLE mining_power_source_events ADD CHECK(mining_power_has_nonnull(data,ARRAY['ObservedAt','ExpiresAt']));
ALTER TABLE mining_power_activities ADD CHECK(mining_power_has_nonnull(data,ARRAY['AcceptedAt','ObservedAt','ExpiresAt','ValidationSnapshot']));
ALTER TABLE mining_power_activities ADD CHECK(mining_power_has_nonnull(data->'ValidationSnapshot',ARRAY['BlockInstanceID','BlockID','BlockHeight','CreateCommandID','Status','G18RuleVersion','G20RuleVersion','SourceEvidenceVersion','StartedAt','ScheduledEndAt','ValidatedAt']));
ALTER TABLE mining_power_activities ADD CHECK((data->'ValidationSnapshot'->>'ValidatedAt')::timestamptz>=(data->>'ObservedAt')::timestamptz);

-- Closed, case-sensitive JSON keys prevent Go's case-insensitive field matching
-- from interpreting an alias differently from the exact SQL-generated column.
CREATE FUNCTION mining_power_only_keys(value jsonb, allowed text[]) RETURNS boolean
LANGUAGE sql IMMUTABLE AS $$ SELECT jsonb_typeof(value)='object' AND value-allowed='{}'::jsonb $$;
ALTER TABLE mining_power_tool_profiles ADD CHECK(mining_power_only_keys(data,ARRAY['Reference','RuleVersion','Kind','BasePowerUnits','EfficiencyScaled']));
ALTER TABLE mining_power_map_profiles ADD CHECK(mining_power_only_keys(data,ARRAY['Reference','RuleVersion','Kind','ModifierScaled']));
ALTER TABLE mining_power_sessions ADD CHECK(mining_power_only_keys(data,ARRAY['ID','PlayerID','AccountID','BlockID','BlockInstanceID','ToolReference','MapReference','RuleVersion','State','OpenedAt','ExpiresAt','CreatedAt','UpdatedAt']));
ALTER TABLE mining_power_source_events ADD CHECK(mining_power_only_keys(data,ARRAY['ID','ActivityID','PlayerID','BlockID','BlockInstanceID','ActivitySessionID','ToolReference','MapReference','RuleVersion','ObservedAt','ExpiresAt','ActivityWeightScaled','ServerEligibility','EvidenceKind']));
ALTER TABLE mining_power_activities ADD CHECK(mining_power_only_keys(data,ARRAY['ActivityID','SourceEventID','PlayerID','BlockID','BlockInstanceID','ActivitySessionID','ToolReference','MapReference','RuleVersion','ValidationSnapshot','Inputs','ValidatedPower','AcceptedAt','ObservedAt','ExpiresAt','Decision','Status']));
ALTER TABLE mining_power_activities ADD CHECK(mining_power_only_keys(data->'Inputs',ARRAY['BasePowerUnits','EfficiencyScaled','ActivityWeightScaled','MapModifierScaled']));
ALTER TABLE mining_power_activities ADD CHECK(mining_power_only_keys(data->'ValidationSnapshot',ARRAY['BlockInstanceID','BlockID','BlockHeight','CreateCommandID','Status','G18RuleVersion','G20RuleVersion','SourceEvidenceVersion','StartedAt','ScheduledEndAt','ValidatedAt']));
