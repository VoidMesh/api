-- name: GetWorldSetting :one
SELECT * FROM world_settings
WHERE key = $1;

-- name: SetWorldSetting :one
INSERT INTO world_settings (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = $2
RETURNING *;

-- name: GetAllWorldSettings :many
SELECT * FROM world_settings
ORDER BY key;