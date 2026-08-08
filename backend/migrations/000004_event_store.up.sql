-- Event Store uses the users table and database roles created during initialization.
CREATE SCHEMA IF NOT EXISTS app_api AUTHORIZATION app_owner;

CREATE TABLE event_store_position (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    last_position BIGINT NOT NULL DEFAULT 0 CHECK (last_position >= 0)
);

INSERT INTO event_store_position (singleton, last_position) VALUES (TRUE, 0);

CREATE TABLE aggregate_streams (
    aggregate_type TEXT NOT NULL
        CHECK (aggregate_type <> '' AND octet_length(aggregate_type) <= 32),
    aggregate_id UUID NOT NULL,
    owner_user_id UUID NOT NULL,
    current_version BIGINT NOT NULL CHECK (current_version > 0),
    PRIMARY KEY (aggregate_type, aggregate_id),
    CONSTRAINT aggregate_streams_owner_unique
        UNIQUE (aggregate_type, aggregate_id, owner_user_id),
    CONSTRAINT aggregate_streams_owner_fk
        FOREIGN KEY (owner_user_id) REFERENCES users (id) ON DELETE RESTRICT
);

CREATE TABLE event_store (
    global_position BIGINT PRIMARY KEY CHECK (global_position > 0),
    event_id UUID NOT NULL UNIQUE,
    aggregate_type TEXT NOT NULL
        CHECK (aggregate_type <> '' AND octet_length(aggregate_type) <= 32),
    aggregate_id UUID NOT NULL,
    owner_user_id UUID NOT NULL,
    aggregate_version BIGINT NOT NULL CHECK (aggregate_version > 0),
    event_type TEXT NOT NULL
        CHECK (event_type <> '' AND octet_length(event_type) <= 64),
    schema_version INTEGER NOT NULL DEFAULT 1 CHECK (schema_version > 0),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (
            jsonb_typeof(payload) = 'object'
            AND octet_length(payload::text) <= 65536
        ),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (
            jsonb_typeof(metadata) = 'object'
            AND octet_length(metadata::text) <= 16384
        ),
    actor_user_id UUID,
    command_id UUID NOT NULL,
    command_event_index SMALLINT NOT NULL DEFAULT 0
        CHECK (command_event_index >= 0 AND command_event_index < 1000),
    occurred_at TIMESTAMPTZ NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT event_store_aggregate_version_unique
        UNIQUE (aggregate_type, aggregate_id, aggregate_version),
    CONSTRAINT event_store_owner_command_event_unique
        UNIQUE (owner_user_id, command_id, command_event_index),
    CONSTRAINT event_store_owner_fk
        FOREIGN KEY (owner_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT event_store_actor_fk
        FOREIGN KEY (actor_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT event_store_stream_fk
        FOREIGN KEY (aggregate_type, aggregate_id, owner_user_id)
        REFERENCES aggregate_streams (aggregate_type, aggregate_id, owner_user_id)
);

CREATE INDEX event_store_owner_recorded_idx
    ON event_store (owner_user_id, recorded_at DESC);

CREATE INDEX event_store_actor_recorded_idx
    ON event_store (actor_user_id, recorded_at DESC)
    WHERE actor_user_id IS NOT NULL;

CREATE TABLE projection_checkpoints (
    projection_name TEXT PRIMARY KEY
        CHECK (projection_name <> '' AND octet_length(projection_name) <= 64),
    last_position BIGINT NOT NULL DEFAULT 0 CHECK (last_position >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE FUNCTION reject_event_store_mutation()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
    RAISE EXCEPTION 'event_store is append-only';
END;
$$;

CREATE TRIGGER event_store_append_only
BEFORE UPDATE OR DELETE ON event_store
FOR EACH STATEMENT
EXECUTE FUNCTION reject_event_store_mutation();

CREATE TRIGGER event_store_no_truncate
BEFORE TRUNCATE ON event_store
FOR EACH STATEMENT
EXECUTE FUNCTION reject_event_store_mutation();

CREATE FUNCTION app_api.lock_event_command(
    p_owner_user_id UUID,
    p_command_id UUID
)
RETURNS VOID
LANGUAGE sql
STRICT
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            p_owner_user_id::TEXT || ':' || p_command_id::TEXT,
            0
        )
    );
$$;

CREATE FUNCTION app_api.append_event(
    p_event_id UUID,
    p_aggregate_type TEXT,
    p_aggregate_id UUID,
    p_owner_user_id UUID,
    p_expected_aggregate_version BIGINT,
    p_event_type TEXT,
    p_schema_version INTEGER,
    p_payload JSONB,
    p_metadata JSONB,
    p_actor_user_id UUID,
    p_command_id UUID,
    p_command_event_index SMALLINT,
    p_occurred_at TIMESTAMPTZ
)
RETURNS SETOF event_store
LANGUAGE sql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    WITH position_lock AS MATERIALIZED (
        SELECT last_position
        FROM public.event_store_position
        WHERE singleton = TRUE
        FOR UPDATE
    ),
    advanced_existing_stream AS (
        UPDATE public.aggregate_streams
        SET current_version = current_version + 1
        WHERE aggregate_type = p_aggregate_type
          AND aggregate_id = p_aggregate_id
          AND owner_user_id = p_owner_user_id
          AND current_version = p_expected_aggregate_version
          AND p_expected_aggregate_version > 0
          AND EXISTS (SELECT 1 FROM position_lock)
        RETURNING current_version
    ),
    created_stream AS (
        INSERT INTO public.aggregate_streams (
            aggregate_type,
            aggregate_id,
            owner_user_id,
            current_version
        )
        SELECT p_aggregate_type, p_aggregate_id, p_owner_user_id, 1
        FROM position_lock
        WHERE p_expected_aggregate_version = 0
        ON CONFLICT (aggregate_type, aggregate_id) DO NOTHING
        RETURNING current_version
    ),
    advanced_stream AS (
        SELECT current_version FROM advanced_existing_stream
        UNION ALL
        SELECT current_version FROM created_stream
    ),
    next_position AS (
        UPDATE public.event_store_position
        SET last_position = last_position + 1
        WHERE singleton = TRUE
          AND EXISTS (SELECT 1 FROM advanced_stream)
        RETURNING last_position
    )
    INSERT INTO public.event_store (
        global_position,
        event_id,
        aggregate_type,
        aggregate_id,
        owner_user_id,
        aggregate_version,
        event_type,
        schema_version,
        payload,
        metadata,
        actor_user_id,
        command_id,
        command_event_index,
        occurred_at
    )
    SELECT
        next_position.last_position,
        p_event_id,
        p_aggregate_type,
        p_aggregate_id,
        p_owner_user_id,
        advanced_stream.current_version,
        p_event_type,
        p_schema_version,
        p_payload,
        p_metadata,
        p_actor_user_id,
        p_command_id,
        p_command_event_index,
        p_occurred_at
    FROM next_position
    CROSS JOIN advanced_stream
    RETURNING *;
$$;

CREATE FUNCTION app_api.get_event_by_id(
    p_owner_user_id UUID,
    p_event_id UUID
)
RETURNS SETOF event_store
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT *
    FROM public.event_store
    WHERE owner_user_id = p_owner_user_id
      AND event_id = p_event_id;
$$;

CREATE FUNCTION app_api.list_events_by_command_id(
    p_owner_user_id UUID,
    p_command_id UUID
)
RETURNS SETOF event_store
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT *
    FROM public.event_store
    WHERE owner_user_id = p_owner_user_id
      AND command_id = p_command_id
    ORDER BY command_event_index ASC;
$$;

CREATE FUNCTION app_api.get_aggregate_version(
    p_owner_user_id UUID,
    p_aggregate_type TEXT,
    p_aggregate_id UUID
)
RETURNS BIGINT
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT COALESCE((
        SELECT current_version
        FROM public.aggregate_streams
        WHERE owner_user_id = p_owner_user_id
          AND aggregate_type = p_aggregate_type
          AND aggregate_id = p_aggregate_id
    ), 0)::BIGINT;
$$;

CREATE FUNCTION app_api.list_aggregate_events(
    p_owner_user_id UUID,
    p_aggregate_type TEXT,
    p_aggregate_id UUID,
    p_after_version BIGINT,
    p_page_size INTEGER
)
RETURNS SETOF event_store
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT *
    FROM public.event_store
    WHERE owner_user_id = p_owner_user_id
      AND aggregate_type = p_aggregate_type
      AND aggregate_id = p_aggregate_id
      AND aggregate_version > p_after_version
    ORDER BY aggregate_version ASC
    LIMIT LEAST(GREATEST(p_page_size, 1), 500);
$$;

CREATE FUNCTION app_api.save_projection_checkpoint(
    p_projection_name TEXT,
    p_last_position BIGINT
)
RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
    IF p_last_position < 0 OR octet_length(p_projection_name) NOT BETWEEN 1 AND 64 THEN
        RAISE EXCEPTION 'invalid projection checkpoint';
    END IF;

    IF p_last_position > 0 AND NOT EXISTS (
        SELECT 1 FROM public.event_store WHERE global_position = p_last_position
    ) THEN
        RAISE EXCEPTION 'checkpoint event does not exist';
    END IF;

    INSERT INTO public.projection_checkpoints (projection_name, last_position)
    VALUES (p_projection_name, p_last_position)
    ON CONFLICT (projection_name) DO UPDATE SET
        last_position = EXCLUDED.last_position,
        updated_at = now()
    WHERE projection_checkpoints.last_position < EXCLUDED.last_position;
END;
$$;

REVOKE ALL ON FUNCTION app_api.append_event(
    UUID, TEXT, UUID, UUID, BIGINT, TEXT, INTEGER, JSONB, JSONB,
    UUID, UUID, SMALLINT, TIMESTAMPTZ
) FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.lock_event_command(UUID, UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.get_event_by_id(UUID, UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.list_events_by_command_id(UUID, UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.get_aggregate_version(UUID, TEXT, UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.list_aggregate_events(
    UUID, TEXT, UUID, BIGINT, INTEGER
) FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.save_projection_checkpoint(TEXT, BIGINT) FROM PUBLIC;

REVOKE ALL ON TABLE event_store_position FROM PUBLIC;
REVOKE ALL ON TABLE aggregate_streams FROM PUBLIC;
REVOKE ALL ON TABLE event_store FROM PUBLIC;
REVOKE ALL ON TABLE projection_checkpoints FROM PUBLIC;

GRANT USAGE ON SCHEMA app_api TO app_runtime, app_projector;

ALTER TABLE event_store_position OWNER TO app_owner;
ALTER TABLE aggregate_streams OWNER TO app_owner;
ALTER TABLE event_store OWNER TO app_owner;
ALTER TABLE projection_checkpoints OWNER TO app_owner;
ALTER FUNCTION reject_event_store_mutation() OWNER TO app_owner;
ALTER FUNCTION app_api.lock_event_command(UUID, UUID) OWNER TO app_owner;
ALTER FUNCTION app_api.append_event(
    UUID, TEXT, UUID, UUID, BIGINT, TEXT, INTEGER, JSONB, JSONB,
    UUID, UUID, SMALLINT, TIMESTAMPTZ
) OWNER TO app_owner;
ALTER FUNCTION app_api.get_event_by_id(UUID, UUID) OWNER TO app_owner;
ALTER FUNCTION app_api.list_events_by_command_id(UUID, UUID) OWNER TO app_owner;
ALTER FUNCTION app_api.get_aggregate_version(UUID, TEXT, UUID) OWNER TO app_owner;
ALTER FUNCTION app_api.list_aggregate_events(
    UUID, TEXT, UUID, BIGINT, INTEGER
) OWNER TO app_owner;
ALTER FUNCTION app_api.save_projection_checkpoint(TEXT, BIGINT) OWNER TO app_owner;

GRANT EXECUTE ON FUNCTION app_api.append_event(
    UUID, TEXT, UUID, UUID, BIGINT, TEXT, INTEGER, JSONB, JSONB,
    UUID, UUID, SMALLINT, TIMESTAMPTZ
) TO app_runtime;
GRANT EXECUTE ON FUNCTION app_api.lock_event_command(UUID, UUID) TO app_runtime;
GRANT EXECUTE ON FUNCTION app_api.get_event_by_id(UUID, UUID) TO app_runtime;
GRANT EXECUTE ON FUNCTION app_api.list_events_by_command_id(UUID, UUID) TO app_runtime;
GRANT EXECUTE ON FUNCTION app_api.get_aggregate_version(UUID, TEXT, UUID) TO app_runtime;
GRANT EXECUTE ON FUNCTION app_api.list_aggregate_events(
    UUID, TEXT, UUID, BIGINT, INTEGER
) TO app_runtime;

GRANT SELECT ON event_store, projection_checkpoints TO app_projector;
GRANT EXECUTE ON FUNCTION app_api.save_projection_checkpoint(TEXT, BIGINT)
    TO app_projector;
