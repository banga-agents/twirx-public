-- TWIRX E3.3 Genesis relational schema design.
-- NON-NORMATIVE AND NOT AN EXECUTABLE MIGRATION.
-- No deployment process consumes this file. S2 must produce reviewed,
-- versioned migrations after the host, recovery, S1, and founder gates pass.

CREATE SCHEMA atlas;
CREATE SCHEMA semantic;
CREATE SCHEMA query;
CREATE SCHEMA economics;
CREATE SCHEMA learning;

CREATE EXTENSION pg_trgm;
CREATE EXTENSION unaccent;
-- pgvector is deliberately absent and disabled.

CREATE TABLE atlas.origin_bindings (
    origin_id text PRIMARY KEY,
    canonical_origin text NOT NULL UNIQUE,
    registry_artifact_digest bytea NOT NULL CHECK (octet_length(registry_artifact_digest) = 32),
    admission_decision_digest bytea NOT NULL CHECK (octet_length(admission_decision_digest) = 32),
    policy_review_state text NOT NULL CHECK (policy_review_state IN ('pending', 'completed')),
    policy_decision text NOT NULL CHECK (policy_decision IN (
        'permit_live', 'permit_with_constraints', 'profile_only',
        'catalog_only', 'deny', 'uncertain'
    )),
    technical_stage text NOT NULL CHECK (technical_stage IN (
        'unprofiled', 'profiled', 'observed', 'native_schema',
        'compiled', 'semantically_linked', 'live'
    )),
    scope text NOT NULL CHECK (scope IN ('public_origin', 'test_fixture')),
    imported_at timestamptz NOT NULL
);

CREATE TABLE semantic.packet_batches (
    batch_id bytea PRIMARY KEY CHECK (octet_length(batch_id) = 32),
    origin_id text NOT NULL REFERENCES atlas.origin_bindings(origin_id),
    manifest_size bigint NOT NULL CHECK (manifest_size BETWEEN 1 AND 4194304),
    compiler_contract_digest bytea NOT NULL CHECK (octet_length(compiler_contract_digest) = 32),
    policy_decision_digest bytea NOT NULL CHECK (octet_length(policy_decision_digest) = 32),
    previous_batch_id bytea REFERENCES semantic.packet_batches(batch_id),
    started_at timestamptz NOT NULL,
    completed_at timestamptz NOT NULL,
    admitted_at timestamptz NOT NULL,
    CHECK (completed_at >= started_at),
    CHECK (admitted_at >= completed_at)
);

-- The non-partitioned identity table makes packet_digest globally unique.
CREATE TABLE semantic.packet_identities (
    packet_digest bytea PRIMARY KEY CHECK (octet_length(packet_digest) = 32),
    observed_at timestamptz NOT NULL,
    canonical_size bigint NOT NULL CHECK (canonical_size BETWEEN 1 AND 4194304),
    UNIQUE (packet_digest, observed_at)
);

CREATE TABLE semantic.packet_log (
    packet_digest bytea NOT NULL,
    observed_at timestamptz NOT NULL,
    batch_id bytea NOT NULL REFERENCES semantic.packet_batches(batch_id),
    origin_id text NOT NULL REFERENCES atlas.origin_bindings(origin_id),
    semantic_key_digest bytea NOT NULL CHECK (octet_length(semantic_key_digest) = 32),
    packet_kind text NOT NULL CHECK (packet_kind IN (
        'claim', 'state', 'capability', 'offer', 'relationship',
        'event', 'measurement', 'document'
    )),
    semantic_lane text NOT NULL CHECK (semantic_lane IN (
        'observed_native', 'provisional_semantic', 'attested_semantic'
    )),
    lifecycle_state text NOT NULL CHECK (lifecycle_state IN (
        'current', 'superseded', 'withdrawn', 'stale', 'retracted', 'invalid'
    )),
    subject_native text NOT NULL,
    subject_canonical text,
    predicate_native text NOT NULL,
    predicate_semantic text,
    native_locator text NOT NULL,
    native_lexical text NOT NULL,
    typed_type text,
    typed_lexical text,
    language text,
    jurisdiction text,
    mapping_status text NOT NULL,
    authority_class text NOT NULL,
    representation_digest bytea NOT NULL CHECK (octet_length(representation_digest) = 32),
    observation_digest bytea NOT NULL CHECK (octet_length(observation_digest) = 32),
    adapter_digest bytea NOT NULL CHECK (octet_length(adapter_digest) = 32),
    semantic_closure_digest bytea CHECK (
        semantic_closure_digest IS NULL OR octet_length(semantic_closure_digest) = 32
    ),
    valid_from timestamptz,
    valid_until timestamptz,
    PRIMARY KEY (packet_digest, observed_at),
    FOREIGN KEY (packet_digest, observed_at)
      REFERENCES semantic.packet_identities(packet_digest, observed_at),
    CHECK (valid_until IS NULL OR valid_from IS NULL OR valid_until >= valid_from),
    CHECK ((typed_type IS NULL) = (typed_lexical IS NULL)),
    CHECK (semantic_lane = 'observed_native' OR predicate_semantic IS NOT NULL),
    CHECK (semantic_lane <> 'attested_semantic' OR semantic_closure_digest IS NOT NULL)
) PARTITION BY RANGE (observed_at);

-- S2 creates monthly partitions explicitly. There is no DEFAULT partition.

CREATE TABLE semantic.delta_identities (
    sequence bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    delta_digest bytea NOT NULL UNIQUE CHECK (octet_length(delta_digest) = 32),
    occurred_at timestamptz NOT NULL,
    canonical_size bigint NOT NULL CHECK (canonical_size BETWEEN 1 AND 4194304),
    UNIQUE (sequence, occurred_at),
    UNIQUE (sequence, occurred_at, delta_digest)
);

CREATE TABLE semantic.delta_log (
    sequence bigint NOT NULL,
    occurred_at timestamptz NOT NULL,
    delta_digest bytea NOT NULL CHECK (octet_length(delta_digest) = 32),
    delta_class text NOT NULL CHECK (delta_class IN ('origin', 'semantic', 'canon')),
    delta_kind text NOT NULL,
    semantic_key_digest bytea NOT NULL CHECK (octet_length(semantic_key_digest) = 32),
    before_packet_digest bytea CHECK (
        before_packet_digest IS NULL OR octet_length(before_packet_digest) = 32
    ),
    after_packet_digest bytea CHECK (
        after_packet_digest IS NULL OR octet_length(after_packet_digest) = 32
    ),
    origin_id text NOT NULL REFERENCES atlas.origin_bindings(origin_id),
    batch_id bytea NOT NULL REFERENCES semantic.packet_batches(batch_id),
    PRIMARY KEY (sequence, occurred_at),
    FOREIGN KEY (sequence, occurred_at, delta_digest)
      REFERENCES semantic.delta_identities(sequence, occurred_at, delta_digest),
    FOREIGN KEY (before_packet_digest) REFERENCES semantic.packet_identities(packet_digest),
    FOREIGN KEY (after_packet_digest) REFERENCES semantic.packet_identities(packet_digest)
) PARTITION BY RANGE (occurred_at);

CREATE TABLE semantic.packet_heads (
    semantic_key_digest bytea PRIMARY KEY CHECK (octet_length(semantic_key_digest) = 32),
    current_packet_digest bytea NOT NULL,
    current_observed_at timestamptz NOT NULL,
    previous_packet_digest bytea CHECK (
        previous_packet_digest IS NULL OR octet_length(previous_packet_digest) = 32
    ),
    update_delta_digest bytea NOT NULL REFERENCES semantic.delta_identities(delta_digest),
    current_sequence bigint NOT NULL REFERENCES semantic.delta_identities(sequence),
    FOREIGN KEY (current_packet_digest, current_observed_at)
      REFERENCES semantic.packet_log(packet_digest, observed_at),
    FOREIGN KEY (previous_packet_digest) REFERENCES semantic.packet_identities(packet_digest)
);

CREATE TABLE semantic.canon_modules (
    module_id text NOT NULL,
    module_version text NOT NULL,
    module_digest bytea NOT NULL UNIQUE CHECK (octet_length(module_digest) = 32),
    status text NOT NULL CHECK (status IN ('candidate', 'reviewed', 'superseded', 'revoked')),
    admitted_at timestamptz,
    PRIMARY KEY (module_id, module_version)
);

CREATE TABLE semantic.concepts (
    concept_id text NOT NULL,
    module_id text NOT NULL,
    module_version text NOT NULL,
    canonical_label text NOT NULL,
    labels_text text NOT NULL,
    search_document tsvector NOT NULL,
    description text,
    PRIMARY KEY (concept_id, module_id, module_version),
    FOREIGN KEY (module_id, module_version)
      REFERENCES semantic.canon_modules(module_id, module_version)
);

CREATE TABLE semantic.concept_edges (
    edge_digest bytea PRIMARY KEY CHECK (octet_length(edge_digest) = 32),
    subject_concept_id text NOT NULL,
    predicate text NOT NULL,
    object_concept_id text NOT NULL,
    module_id text NOT NULL,
    module_version text NOT NULL,
    path_cost_millionths integer NOT NULL CHECK (path_cost_millionths BETWEEN 0 AND 1000000),
    mapping_status text NOT NULL CHECK (mapping_status IN (
        'candidate', 'reviewed', 'disputed', 'revoked'
    )),
    FOREIGN KEY (module_id, module_version)
      REFERENCES semantic.canon_modules(module_id, module_version)
);

CREATE TABLE semantic.concept_closures (
    module_set_digest bytea NOT NULL CHECK (octet_length(module_set_digest) = 32),
    from_concept_id text NOT NULL,
    to_concept_id text NOT NULL,
    depth smallint NOT NULL CHECK (depth BETWEEN 0 AND 16),
    total_cost_millionths bigint NOT NULL CHECK (total_cost_millionths BETWEEN 0 AND 16000000),
    path_digest bytea NOT NULL CHECK (octet_length(path_digest) = 32),
    PRIMARY KEY (module_set_digest, from_concept_id, to_concept_id, path_digest)
);

CREATE TABLE semantic.materialization_definitions (
    materialization_id text NOT NULL,
    version text NOT NULL,
    definition_digest bytea NOT NULL UNIQUE CHECK (octet_length(definition_digest) = 32),
    canon_module_set_digest bytea NOT NULL CHECK (octet_length(canon_module_set_digest) = 32),
    status text NOT NULL CHECK (status IN ('candidate', 'active', 'suspended', 'superseded')),
    PRIMARY KEY (materialization_id, version)
);

CREATE TABLE semantic.materialization_runs (
    manifest_digest bytea PRIMARY KEY CHECK (octet_length(manifest_digest) = 32),
    materialization_id text NOT NULL,
    version text NOT NULL,
    through_sequence bigint NOT NULL REFERENCES semantic.delta_identities(sequence),
    result_artifact_digest bytea NOT NULL CHECK (octet_length(result_artifact_digest) = 32),
    row_count bigint NOT NULL CHECK (row_count >= 0),
    built_at timestamptz NOT NULL,
    FOREIGN KEY (materialization_id, version)
      REFERENCES semantic.materialization_definitions(materialization_id, version)
);

CREATE TABLE semantic.materialized_rows (
    materialization_id text NOT NULL,
    version text NOT NULL,
    row_key_digest bytea NOT NULL CHECK (octet_length(row_key_digest) = 32),
    current_manifest_digest bytea NOT NULL REFERENCES semantic.materialization_runs(manifest_digest),
    row_artifact_digest bytea NOT NULL CHECK (octet_length(row_artifact_digest) = 32),
    conflict_state text NOT NULL CHECK (conflict_state IN ('none', 'preserved', 'unresolved')),
    through_sequence bigint NOT NULL REFERENCES semantic.delta_identities(sequence),
    PRIMARY KEY (materialization_id, version, row_key_digest),
    FOREIGN KEY (materialization_id, version)
      REFERENCES semantic.materialization_definitions(materialization_id, version)
);

CREATE TABLE semantic.materialized_row_packets (
    materialization_id text NOT NULL,
    version text NOT NULL,
    row_key_digest bytea NOT NULL,
    packet_digest bytea NOT NULL REFERENCES semantic.packet_identities(packet_digest),
    PRIMARY KEY (materialization_id, version, row_key_digest, packet_digest),
    FOREIGN KEY (materialization_id, version, row_key_digest)
      REFERENCES semantic.materialized_rows(materialization_id, version, row_key_digest)
);

CREATE TABLE query.runs (
    run_id text PRIMARY KEY,
    query_digest bytea NOT NULL CHECK (octet_length(query_digest) = 32),
    plan_digest bytea CHECK (plan_digest IS NULL OR octet_length(plan_digest) = 32),
    snapshot_sequence bigint REFERENCES semantic.delta_identities(sequence),
    preference_policy text NOT NULL,
    status text NOT NULL CHECK (status IN ('planned', 'running', 'resolved', 'partial', 'unresolved', 'failed')),
    started_at timestamptz NOT NULL,
    completed_at timestamptz,
    result_digest bytea CHECK (result_digest IS NULL OR octet_length(result_digest) = 32),
    CHECK (completed_at IS NULL OR completed_at >= started_at)
);

CREATE TABLE query.subscriptions (
    subscription_id text PRIMARY KEY,
    subscription_digest bytea NOT NULL UNIQUE CHECK (octet_length(subscription_digest) = 32),
    query_digest bytea NOT NULL CHECK (octet_length(query_digest) = 32),
    current_sequence bigint NOT NULL DEFAULT 0 CHECK (current_sequence >= 0),
    status text NOT NULL CHECK (status IN ('active', 'paused', 'expired', 'revoked')),
    created_at timestamptz NOT NULL,
    expires_at timestamptz,
    CHECK (expires_at IS NULL OR expires_at >= created_at)
);

CREATE TABLE economics.events (
    event_digest bytea PRIMARY KEY CHECK (octet_length(event_digest) = 32),
    event_id text NOT NULL UNIQUE,
    occurred_at timestamptz NOT NULL,
    origin_id text REFERENCES atlas.origin_bindings(origin_id),
    work_type text NOT NULL,
    query_digest bytea CHECK (query_digest IS NULL OR octet_length(query_digest) = 32),
    batch_id bytea REFERENCES semantic.packet_batches(batch_id),
    requests bigint NOT NULL CHECK (requests >= 0),
    transferred_bytes bigint NOT NULL CHECK (transferred_bytes >= 0),
    cpu_milliseconds bigint NOT NULL CHECK (cpu_milliseconds >= 0),
    peak_memory_bytes bigint NOT NULL CHECK (peak_memory_bytes >= 0),
    evidence_bytes_written bigint NOT NULL CHECK (evidence_bytes_written >= 0),
    proof_bytes_returned bigint NOT NULL CHECK (proof_bytes_returned >= 0),
    human_review_seconds bigint NOT NULL CHECK (human_review_seconds >= 0),
    cost_currency char(3) NOT NULL,
    cost_amount_decimal text NOT NULL,
    revenue_currency char(3) NOT NULL,
    revenue_amount_decimal text NOT NULL,
    funding_class text NOT NULL,
    measurement_method text NOT NULL
);

CREATE TABLE semantic.outbox (
    outbox_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    delta_sequence bigint NOT NULL REFERENCES semantic.delta_identities(sequence),
    topic text NOT NULL,
    stable_key text NOT NULL,
    payload_digest bytea NOT NULL CHECK (octet_length(payload_digest) = 32),
    created_at timestamptz NOT NULL,
    claimed_by text,
    claimed_until timestamptz,
    delivered_at timestamptz,
    UNIQUE (topic, stable_key, delta_sequence)
);

CREATE TABLE learning.export_manifests (
    export_digest bytea PRIMARY KEY CHECK (octet_length(export_digest) = 32),
    through_sequence bigint NOT NULL REFERENCES semantic.delta_identities(sequence),
    schema_version text NOT NULL,
    row_count bigint NOT NULL CHECK (row_count >= 0),
    origin_count integer NOT NULL CHECK (origin_count >= 0),
    created_at timestamptz NOT NULL,
    split_policy text NOT NULL
);

CREATE INDEX packet_log_origin_time_idx
    ON semantic.packet_log (origin_id, observed_at DESC);
CREATE INDEX packet_log_semantic_lookup_idx
    ON semantic.packet_log (subject_canonical, predicate_semantic, observed_at DESC);
CREATE INDEX packet_log_native_predicate_trgm_idx
    ON semantic.packet_log USING gin (predicate_native gin_trgm_ops);
CREATE INDEX packet_log_native_lexical_search_idx
    ON semantic.packet_log USING gin (to_tsvector('simple', native_lexical));
CREATE INDEX packet_log_lane_lifecycle_idx
    ON semantic.packet_log (semantic_lane, lifecycle_state, observed_at DESC);
CREATE INDEX packet_log_proof_idx
    ON semantic.packet_log (observation_digest, representation_digest);
CREATE INDEX delta_log_cursor_idx
    ON semantic.delta_log (sequence);
CREATE INDEX delta_log_origin_class_idx
    ON semantic.delta_log (origin_id, delta_class, occurred_at DESC);
CREATE INDEX concepts_search_idx
    ON semantic.concepts USING gin (search_document);
CREATE INDEX concepts_label_trgm_idx
    ON semantic.concepts USING gin (canonical_label gin_trgm_ops);
CREATE INDEX concept_edges_lookup_idx
    ON semantic.concept_edges (subject_concept_id, predicate, mapping_status);
CREATE INDEX concept_closure_lookup_idx
    ON semantic.concept_closures (from_concept_id, depth, total_cost_millionths);
CREATE INDEX outbox_pending_idx
    ON semantic.outbox (outbox_id)
    WHERE delivered_at IS NULL;
CREATE INDEX economics_origin_time_idx
    ON economics.events (origin_id, occurred_at DESC);

-- Runtime privilege design (roles are created by an operator-only migration):
-- * migration owner: schema/DDL only; no application login
-- * compiler: INSERT/SELECT and bounded SECURITY DEFINER admission procedures
-- * query: SELECT views plus bounded query procedures
-- * subscriber: SELECT admitted outbox/delta views
-- * analytics exporter: bounded read-only export views
-- No runtime role receives UPDATE/DELETE on packet_identities, packet_log,
-- delta_identities, delta_log, or canon module history.
