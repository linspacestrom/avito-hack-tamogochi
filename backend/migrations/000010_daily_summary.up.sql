CREATE TABLE daily_summary_checkpoints (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    last_check_in_at TIMESTAMPTZ NOT NULL,
    last_event_position BIGINT NOT NULL CHECK (last_event_position >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE daily_summary_event_failures (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES event_store (event_id) ON DELETE RESTRICT,
    global_position BIGINT NOT NULL CHECK (global_position > 0),
    event_type TEXT NOT NULL,
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    reason TEXT NOT NULL CHECK (reason IN ('INVALID_PAYLOAD', 'UNSUPPORTED_SCHEMA')),
    detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, event_id)
);

CREATE INDEX event_store_owner_position_idx
    ON event_store (owner_user_id, global_position);

CREATE VIEW daily_summary_event_view AS
SELECT
    event.global_position,
    event.event_id,
    event.owner_user_id,
    event.event_type,
    event.schema_version,
    event.payload
FROM event_store AS event
WHERE event.event_type IN (
    'EXPERIENCE_GRANTED',
    'COINS_GRANTED',
    'GAME_SESSION_COMPLETED',
    'REWARD_GRANTED'
);

CREATE FUNCTION app_api.list_daily_summary_events_by_position(
    p_owner_user_id UUID,
    p_after_position BIGINT,
    p_to_position BIGINT,
    p_page_size INTEGER
)
RETURNS SETOF daily_summary_event_view
LANGUAGE sql
STABLE
STRICT
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT event.*
    FROM public.daily_summary_event_view AS event
    WHERE event.owner_user_id = p_owner_user_id
      AND event.global_position > GREATEST(p_after_position, 0)
      AND event.global_position <= GREATEST(p_to_position, 0)
    ORDER BY event.global_position ASC
    LIMIT LEAST(GREATEST(p_page_size, 1), 500);
$$;

CREATE FUNCTION app_api.record_daily_summary_event_failure(
    p_user_id UUID,
    p_event_id UUID,
    p_global_position BIGINT,
    p_event_type TEXT,
    p_schema_version INTEGER,
    p_reason TEXT
)
RETURNS BOOLEAN
LANGUAGE sql
VOLATILE
STRICT
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    WITH matching_event AS (
        SELECT
            event.owner_user_id AS user_id,
            event.event_id,
            event.global_position,
            event.event_type,
            event.schema_version
        FROM public.event_store AS event
        WHERE event.owner_user_id = p_user_id
          AND event.event_id = p_event_id
          AND event.global_position = p_global_position
          AND event.event_type = p_event_type
          AND event.schema_version = p_schema_version
    ),
    inserted AS (
        INSERT INTO public.daily_summary_event_failures (
            user_id,
            event_id,
            global_position,
            event_type,
            schema_version,
            reason
        )
        SELECT
            matching_event.user_id,
            matching_event.event_id,
            matching_event.global_position,
            matching_event.event_type,
            matching_event.schema_version,
            p_reason
        FROM matching_event
        ON CONFLICT (user_id, event_id) DO UPDATE
        SET reason = EXCLUDED.reason,
            detected_at = now()
        WHERE daily_summary_event_failures.global_position = EXCLUDED.global_position
          AND daily_summary_event_failures.event_type = EXCLUDED.event_type
          AND daily_summary_event_failures.schema_version = EXCLUDED.schema_version
        RETURNING TRUE
    )
    SELECT EXISTS (SELECT 1 FROM inserted);
$$;

ALTER TABLE daily_summary_checkpoints OWNER TO app_owner;
ALTER TABLE daily_summary_event_failures OWNER TO app_owner;
ALTER VIEW daily_summary_event_view OWNER TO app_owner;
ALTER FUNCTION app_api.list_daily_summary_events_by_position(UUID, BIGINT, BIGINT, INTEGER)
    OWNER TO app_owner;
ALTER FUNCTION app_api.record_daily_summary_event_failure(UUID, UUID, BIGINT, TEXT, INTEGER, TEXT)
    OWNER TO app_owner;

REVOKE ALL ON TABLE daily_summary_checkpoints FROM PUBLIC;
REVOKE ALL ON TABLE daily_summary_event_failures FROM PUBLIC;
REVOKE ALL ON TABLE daily_summary_event_view FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.list_daily_summary_events_by_position(UUID, BIGINT, BIGINT, INTEGER)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.record_daily_summary_event_failure(UUID, UUID, BIGINT, TEXT, INTEGER, TEXT)
    FROM PUBLIC;

GRANT SELECT, INSERT, UPDATE ON TABLE daily_summary_checkpoints TO app_runtime;
GRANT SELECT ON TABLE event_store_position TO app_runtime;
GRANT EXECUTE ON FUNCTION app_api.list_daily_summary_events_by_position(UUID, BIGINT, BIGINT, INTEGER)
    TO app_runtime;
GRANT EXECUTE ON FUNCTION app_api.record_daily_summary_event_failure(UUID, UUID, BIGINT, TEXT, INTEGER, TEXT)
    TO app_runtime;
