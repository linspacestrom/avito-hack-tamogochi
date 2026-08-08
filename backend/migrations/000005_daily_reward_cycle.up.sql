-- daily_reward_cycle and user_daily_reward_progress, per ARCHITECTURE.md ("Ежедневный цикл
-- наград") — a repeating 14-day login calendar, separate from the achievement-style rewards
-- in reward_definitions/user_rewards.
--
-- daily_reward_cycle is the shared catalog (exactly 14 rows, one per day of the cycle).
-- user_daily_reward_progress is where each user currently stands in their own cycle.

CREATE TABLE daily_reward_cycle (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    day_number            INTEGER NOT NULL UNIQUE CHECK (day_number BETWEEN 1 AND 14),
    reward_definition_id  UUID NOT NULL REFERENCES reward_definitions (id)
);

-- One row per user — current_day is where they are in the 14-day loop, last_claimed_at is
-- what the lazy day-advance/reset math in the service layer is computed from.
CREATE TABLE user_daily_reward_progress (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL UNIQUE,
    current_day       INTEGER NOT NULL DEFAULT 1 CHECK (current_day BETWEEN 1 AND 14),
    cycle_started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_claimed_at   TIMESTAMPTZ
);

ALTER TABLE daily_reward_cycle OWNER TO app_owner;
ALTER TABLE user_daily_reward_progress OWNER TO app_owner;

REVOKE ALL ON TABLE daily_reward_cycle, user_daily_reward_progress FROM PUBLIC;
GRANT SELECT ON TABLE daily_reward_cycle TO app_runtime;
GRANT SELECT, INSERT, UPDATE ON TABLE user_daily_reward_progress TO app_runtime;
