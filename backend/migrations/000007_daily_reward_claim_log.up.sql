-- Append-only history of daily reward claims. user_daily_reward_progress only tracks current
-- state (mutated in place on every claim); this table is what "what did I earn yesterday"
-- style features (e.g. the daily summary) will need to query instead of relying on progress.
--
-- reward_definition_id is captured at claim time rather than derived by joining through
-- daily_reward_cycle by day_number, so the log stays accurate even if that catalog's
-- day-to-reward mapping changes later.

CREATE TABLE daily_reward_claim_log (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL,
    day_number            INTEGER NOT NULL CHECK (day_number BETWEEN 1 AND 14),
    reward_definition_id  UUID NOT NULL REFERENCES reward_definitions (id),
    claimed_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX daily_reward_claim_log_user_id_claimed_at_idx
    ON daily_reward_claim_log (user_id, claimed_at DESC);

ALTER TABLE daily_reward_claim_log OWNER TO app_owner;

REVOKE ALL ON TABLE daily_reward_claim_log FROM PUBLIC;
GRANT SELECT, INSERT ON TABLE daily_reward_claim_log TO app_runtime;
