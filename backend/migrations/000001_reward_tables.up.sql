-- reward_definitions and user_rewards, per ARCHITECTURE.md schema (RewardDefinition / UserReward).
--
-- NOTE: no FK to users(id) / events(id) yet — those tables aren't migrated anywhere in the repo
-- yet (pet/auth module owns them). user_id and source_event_id are left as plain UUID columns;
-- add the FK constraints in a follow-up migration once those tables exist.
--
-- NOTE: reward_type and value are proposed additions, not yet confirmed with the team (see
-- ARCHITECTURE.md schema, which does not list them) — included now per explicit request to lay
-- the groundwork, to be run past @sonofche separately.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE reward_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    required_level  INTEGER NOT NULL DEFAULT 0,
    validity_days   INTEGER,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,

    -- Proposed, not yet confirmed with the team.
    reward_type     TEXT NOT NULL
        CHECK (reward_type IN ('game_currency', 'game_energy', 'avito_promo', 'avito_discount')),
    value           JSONB NOT NULL DEFAULT '{}'::jsonb
);

-- status values are an assumption (not specified anywhere yet) — confirm with the team before
-- relying on them elsewhere.
CREATE TABLE user_rewards (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID NOT NULL,
    reward_definition_id   UUID NOT NULL REFERENCES reward_definitions (id),
    source_event_id        UUID,
    status                 TEXT NOT NULL DEFAULT 'issued'
        CHECK (status IN ('issued', 'redeemed', 'expired', 'revoked')),
    issued_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at             TIMESTAMPTZ,
    redeemed_at            TIMESTAMPTZ,

    -- Protection against issuing the same reward to the same user twice. Also serves as the
    -- lookup index for "list this user's rewards" (user_id is the leading column), so no
    -- separate index on user_id alone is needed.
    UNIQUE (user_id, reward_definition_id)
);
