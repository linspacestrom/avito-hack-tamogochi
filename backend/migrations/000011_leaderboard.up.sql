CREATE TABLE leaderboard_pet_levels (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    pet_id UUID NOT NULL UNIQUE,
    pet_name TEXT NOT NULL
        CHECK (btrim(pet_name) <> '' AND octet_length(pet_name) <= 128),
    species TEXT NOT NULL
        CHECK (btrim(species) <> '' AND octet_length(species) <= 64),
    level INTEGER NOT NULL CHECK (level > 0),
    level_reached_position BIGINT NOT NULL
        REFERENCES event_store (global_position) ON DELETE RESTRICT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX leaderboard_pet_levels_rank_idx
    ON leaderboard_pet_levels (
        level DESC,
        level_reached_position ASC,
        user_id ASC
    );

CREATE TABLE leaderboard_game_scores (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    session_id UUID NOT NULL,
    best_score BIGINT NOT NULL CHECK (best_score >= 0),
    achieved_position BIGINT NOT NULL
        REFERENCES event_store (global_position) ON DELETE RESTRICT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX leaderboard_game_scores_rank_idx
    ON leaderboard_game_scores (
        best_score DESC,
        achieved_position ASC,
        user_id ASC
    );

CREATE TABLE leaderboard_event_failures (
    event_id UUID PRIMARY KEY REFERENCES event_store (event_id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    global_position BIGINT NOT NULL
        REFERENCES event_store (global_position) ON DELETE RESTRICT,
    event_type TEXT NOT NULL,
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    reason TEXT NOT NULL CHECK (
        reason IN (
            'INVALID_PAYLOAD',
            'INVALID_AGGREGATE',
            'INVALID_SEQUENCE',
            'UNSUPPORTED_SCHEMA'
        )
    ),
    detected_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE FUNCTION app_api.reset_leaderboard_projection()
RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('leaderboard_v1', 0));
    TRUNCATE TABLE
        public.leaderboard_event_failures,
        public.leaderboard_game_scores,
        public.leaderboard_pet_levels;
    DELETE FROM public.projection_checkpoints
    WHERE projection_name = 'leaderboard_v1';
END;
$$;

ALTER TABLE leaderboard_pet_levels OWNER TO app_owner;
ALTER TABLE leaderboard_game_scores OWNER TO app_owner;
ALTER TABLE leaderboard_event_failures OWNER TO app_owner;
ALTER FUNCTION app_api.reset_leaderboard_projection() OWNER TO app_owner;

REVOKE ALL ON TABLE leaderboard_pet_levels FROM PUBLIC;
REVOKE ALL ON TABLE leaderboard_game_scores FROM PUBLIC;
REVOKE ALL ON TABLE leaderboard_event_failures FROM PUBLIC;
REVOKE ALL ON FUNCTION app_api.reset_leaderboard_projection() FROM PUBLIC;

GRANT SELECT ON TABLE leaderboard_pet_levels, leaderboard_game_scores TO app_runtime;
GRANT SELECT, INSERT, UPDATE ON TABLE
    leaderboard_pet_levels,
    leaderboard_game_scores,
    leaderboard_event_failures
TO app_projector;
