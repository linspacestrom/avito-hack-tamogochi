-- name: InsertLeaderboardPet :execrows
INSERT INTO leaderboard_pet_levels (
    user_id,
    pet_id,
    pet_name,
    species,
    level,
    level_reached_position
)
VALUES ($1, $2, $3, $4, 1, $5)
ON CONFLICT (user_id) DO NOTHING;

-- name: AdvanceLeaderboardPetLevel :execrows
UPDATE leaderboard_pet_levels
SET level = $3,
    level_reached_position = $4,
    updated_at = now()
WHERE user_id = $1
  AND pet_id = $2
  AND level < $3
  AND level_reached_position < $4;

-- name: UpsertLeaderboardGameScore :execrows
INSERT INTO leaderboard_game_scores (
    user_id,
    session_id,
    best_score,
    achieved_position
)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET session_id = EXCLUDED.session_id,
    best_score = EXCLUDED.best_score,
    achieved_position = EXCLUDED.achieved_position,
    updated_at = now()
WHERE leaderboard_game_scores.best_score < EXCLUDED.best_score;

-- name: RecordLeaderboardEventFailure :execrows
INSERT INTO leaderboard_event_failures (
    event_id,
    user_id,
    global_position,
    event_type,
    schema_version,
    reason
)
SELECT
    stored.event_id,
    stored.owner_user_id,
    stored.global_position,
    stored.event_type,
    stored.schema_version,
    sqlc.arg(reason)::TEXT
FROM event_store AS stored
WHERE stored.event_id = sqlc.arg(event_id)::UUID
  AND stored.owner_user_id = sqlc.arg(user_id)::UUID
  AND stored.global_position = sqlc.arg(global_position)::BIGINT
  AND stored.event_type = sqlc.arg(event_type)::TEXT
  AND stored.schema_version = sqlc.arg(schema_version)::INTEGER
ON CONFLICT (event_id) DO UPDATE
SET reason = EXCLUDED.reason,
    detected_at = now()
WHERE leaderboard_event_failures.user_id = EXCLUDED.user_id
  AND leaderboard_event_failures.global_position = EXCLUDED.global_position
  AND leaderboard_event_failures.event_type = EXCLUDED.event_type
  AND leaderboard_event_failures.schema_version = EXCLUDED.schema_version;

-- name: ListPetLevelLeaderboard :many
WITH ranked AS (
    SELECT
        ROW_NUMBER() OVER (
            ORDER BY pet.level DESC, pet.level_reached_position ASC, pet.user_id ASC
        ) AS rank,
        pet.user_id,
        users.display_name,
        pet.pet_id,
        pet.pet_name,
        pet.species,
        pet.level
    FROM leaderboard_pet_levels AS pet
    JOIN users ON users.id = pet.user_id
    WHERE users.status = 'active'
)
SELECT rank, user_id, display_name, pet_id, pet_name, species, level
FROM ranked
ORDER BY rank ASC
LIMIT LEAST(GREATEST(sqlc.arg(page_size)::INTEGER, 1), 100)
OFFSET GREATEST(sqlc.arg(page_offset)::INTEGER, 0);

-- name: ListGameScoreLeaderboard :many
WITH ranked AS (
    SELECT
        ROW_NUMBER() OVER (
            ORDER BY game.best_score DESC, game.achieved_position ASC, game.user_id ASC
        ) AS rank,
        game.user_id,
        users.display_name,
        game.best_score
    FROM leaderboard_game_scores AS game
    JOIN users ON users.id = game.user_id
    WHERE users.status = 'active'
)
SELECT rank, user_id, display_name, best_score
FROM ranked
ORDER BY rank ASC
LIMIT LEAST(GREATEST(sqlc.arg(page_size)::INTEGER, 1), 100)
OFFSET GREATEST(sqlc.arg(page_offset)::INTEGER, 0);

-- name: GetUserLeaderboardPositions :one
WITH pet_ranked AS (
    SELECT
        ROW_NUMBER() OVER (
            ORDER BY pet.level DESC, pet.level_reached_position ASC, pet.user_id ASC
        ) AS rank,
        pet.user_id
    FROM leaderboard_pet_levels AS pet
    JOIN users ON users.id = pet.user_id
    WHERE users.status = 'active'
),
game_ranked AS (
    SELECT
        ROW_NUMBER() OVER (
            ORDER BY game.best_score DESC, game.achieved_position ASC, game.user_id ASC
        ) AS rank,
        game.user_id
    FROM leaderboard_game_scores AS game
    JOIN users ON users.id = game.user_id
    WHERE users.status = 'active'
)
SELECT
    COALESCE((
        SELECT rank FROM pet_ranked WHERE user_id = sqlc.arg(user_id)::UUID
    ), 0)::BIGINT AS pet_level_rank,
    COALESCE((
        SELECT rank FROM game_ranked WHERE user_id = sqlc.arg(user_id)::UUID
    ), 0)::BIGINT AS game_score_rank;
