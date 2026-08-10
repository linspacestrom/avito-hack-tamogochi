-- name: ListTasksWithProgress :many
SELECT
    td.id, td.code, td.type, td.title, td.description, td.target_metric, td.target_value,
    td.reward_definition_id,
    rd.code AS reward_code, rd.title AS reward_title, rd.description AS reward_description,
    utp.current_value, utp.status, utp.completed_at, utp.claimed_at
FROM task_definitions td
JOIN reward_definitions rd ON rd.id = td.reward_definition_id
LEFT JOIN user_task_progress utp ON utp.task_definition_id = td.id AND utp.user_id = $1
WHERE td.is_active = true
ORDER BY td.type, td.code;

-- name: GetTaskWithProgressByCode :one
SELECT
    td.id, td.code, td.type, td.title, td.description, td.target_metric, td.target_value,
    td.reward_definition_id,
    rd.code AS reward_code, rd.title AS reward_title, rd.description AS reward_description,
    utp.current_value, utp.status, utp.completed_at, utp.claimed_at
FROM task_definitions td
JOIN reward_definitions rd ON rd.id = td.reward_definition_id
LEFT JOIN user_task_progress utp ON utp.task_definition_id = td.id AND utp.user_id = $1
WHERE td.code = $2 AND td.is_active = true;

-- name: ClaimTaskProgress :one
UPDATE user_task_progress
SET status = 'claimed', claimed_at = now()
WHERE user_id = $1 AND task_definition_id = $2 AND status = 'completed'
RETURNING id;
