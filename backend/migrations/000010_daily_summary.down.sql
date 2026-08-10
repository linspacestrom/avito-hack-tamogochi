DROP FUNCTION IF EXISTS app_api.record_daily_summary_event_failure(UUID, UUID, BIGINT, TEXT, INTEGER, TEXT);
DROP FUNCTION IF EXISTS app_api.list_daily_summary_events_by_position(UUID, BIGINT, BIGINT, INTEGER);
REVOKE SELECT ON TABLE event_store_position FROM app_runtime;
DROP VIEW IF EXISTS daily_summary_event_view;
DROP INDEX IF EXISTS event_store_owner_position_idx;
DROP TABLE IF EXISTS daily_summary_event_failures;
DROP TABLE IF EXISTS daily_summary_checkpoints;
