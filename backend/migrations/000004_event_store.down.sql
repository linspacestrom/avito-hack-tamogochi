DROP FUNCTION IF EXISTS app_api.save_projection_checkpoint(TEXT, BIGINT);
DROP FUNCTION IF EXISTS app_api.list_aggregate_events(UUID, TEXT, UUID, BIGINT, INTEGER);
DROP FUNCTION IF EXISTS app_api.get_aggregate_version(UUID, TEXT, UUID);
DROP FUNCTION IF EXISTS app_api.list_events_by_command_id(UUID, UUID);
DROP FUNCTION IF EXISTS app_api.get_event_by_id(UUID, UUID);
DROP FUNCTION IF EXISTS app_api.append_event(
    UUID, TEXT, UUID, UUID, BIGINT, TEXT, INTEGER, JSONB, JSONB,
    UUID, UUID, SMALLINT, TIMESTAMPTZ
);
DROP TABLE IF EXISTS projection_checkpoints;
DROP TABLE IF EXISTS event_store;
DROP TABLE IF EXISTS aggregate_streams;
DROP TABLE IF EXISTS event_store_position;
DROP FUNCTION IF EXISTS reject_event_store_mutation();
DROP SCHEMA IF EXISTS app_api;
