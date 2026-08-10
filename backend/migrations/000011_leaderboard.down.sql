SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('leaderboard_v1', 0)
);
DROP FUNCTION IF EXISTS app_api.reset_leaderboard_projection();
DELETE FROM public.projection_checkpoints WHERE projection_name = 'leaderboard_v1';
DROP TABLE IF EXISTS leaderboard_event_failures;
DROP TABLE IF EXISTS leaderboard_game_scores;
DROP TABLE IF EXISTS leaderboard_pet_levels;
