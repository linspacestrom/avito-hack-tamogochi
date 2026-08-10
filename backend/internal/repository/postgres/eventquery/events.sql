-- name: LockEventCommand :exec
SELECT app_api.lock_event_command(
    sqlc.arg(owner_user_id)::UUID,
    sqlc.arg(command_id)::UUID
);

-- name: AppendEvent :one
-- owner_user_id identifies the user from the authenticated request context.
SELECT *
FROM app_api.append_event(
    sqlc.arg(event_id)::UUID,
    sqlc.arg(aggregate_type)::TEXT,
    sqlc.arg(aggregate_id)::UUID,
    sqlc.arg(owner_user_id)::UUID,
    sqlc.arg(expected_aggregate_version)::BIGINT,
    sqlc.arg(event_type)::TEXT,
    sqlc.arg(schema_version)::INTEGER,
    sqlc.arg(payload)::JSONB,
    sqlc.arg(metadata)::JSONB,
    sqlc.narg(actor_user_id)::UUID,
    sqlc.arg(command_id)::UUID,
    sqlc.arg(command_event_index)::SMALLINT,
    sqlc.arg(occurred_at)::TIMESTAMPTZ
);

-- name: GetEventByID :one
-- owner_user_id limits the lookup to the authenticated user.
SELECT *
FROM app_api.get_event_by_id(
    sqlc.arg(owner_user_id)::UUID,
    sqlc.arg(event_id)::UUID
);

-- name: ListEventsByCommandID :many
SELECT *
FROM app_api.list_events_by_command_id(
    sqlc.arg(owner_user_id)::UUID,
    sqlc.arg(command_id)::UUID
);

-- name: GetAggregateVersion :one
SELECT app_api.get_aggregate_version(
    sqlc.arg(owner_user_id)::UUID,
    sqlc.arg(aggregate_type)::TEXT,
    sqlc.arg(aggregate_id)::UUID
) AS version;

-- name: ListAggregateEvents :many
SELECT *
FROM app_api.list_aggregate_events(
    sqlc.arg(owner_user_id)::UUID,
    sqlc.arg(aggregate_type)::TEXT,
    sqlc.arg(aggregate_id)::UUID,
    sqlc.arg(after_version)::BIGINT,
    sqlc.arg(page_size)::INTEGER
);

-- Projection workers consume events in global order when updating read models.
-- name: ListEventsAfterPosition :many
SELECT *
FROM event_store
WHERE global_position > sqlc.arg(after_position)
ORDER BY global_position ASC
LIMIT LEAST(GREATEST(sqlc.arg(page_size)::INTEGER, 1), 500);
