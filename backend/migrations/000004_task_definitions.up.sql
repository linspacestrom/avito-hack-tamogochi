-- task_definitions and user_task_progress, per ARCHITECTURE.md schema (TaskDefinition /
-- UserTaskProgress) — backbone of the "Задания" feature.
--
-- Same catalog-vs-per-user-progress split as reward_definitions/user_rewards and
-- daily_reward_cycle/user_daily_reward_progress: task_definitions is the shared list of
-- quests, user_task_progress is where each user stands on each one.
--
-- NOTE: no FK to users(id) yet, same reason as in 000001 — that table isn't migrated on
-- main yet (pet/auth module owns it).

CREATE TABLE task_definitions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                  TEXT NOT NULL UNIQUE,
    type                  TEXT NOT NULL
        CHECK (type IN ('daily', 'general', 'avito_external')),
    title                 TEXT NOT NULL,
    description           TEXT NOT NULL DEFAULT '',
    target_metric         TEXT NOT NULL,
    target_value          INTEGER NOT NULL,
    reward_definition_id  UUID NOT NULL REFERENCES reward_definitions (id),
    reset_period          TEXT,
    is_active             BOOLEAN NOT NULL DEFAULT TRUE
);

-- One progress row per (user, task) — for repeating tasks (daily/periodic) the same row is
-- reset in place (current_value back to 0, period_started_at bumped), not re-inserted.
CREATE TABLE user_task_progress (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL,
    task_definition_id   UUID NOT NULL REFERENCES task_definitions (id),
    current_value        INTEGER NOT NULL DEFAULT 0,
    status               TEXT NOT NULL DEFAULT 'in_progress'
        CHECK (status IN ('in_progress', 'completed', 'claimed')),
    completed_at         TIMESTAMPTZ,
    claimed_at           TIMESTAMPTZ,
    period_started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, task_definition_id)
);

ALTER TABLE task_definitions OWNER TO app_owner;
ALTER TABLE user_task_progress OWNER TO app_owner;

REVOKE ALL ON TABLE task_definitions, user_task_progress FROM PUBLIC;
GRANT SELECT ON TABLE task_definitions TO app_runtime;
GRANT SELECT, INSERT, UPDATE ON TABLE user_task_progress TO app_runtime;
