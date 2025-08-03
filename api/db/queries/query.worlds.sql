-- name: GetWorldByID :one
SELECT * FROM worlds
WHERE id = $1
LIMIT 1;

-- name: GetDefaultWorld :one
SELECT * FROM worlds
ORDER BY created_at ASC
LIMIT 1;

-- name: ListWorlds :many
SELECT * FROM worlds
ORDER BY created_at;

-- name: CreateWorld :one
INSERT INTO worlds (
  name, seed
) VALUES (
  $1, $2
)
RETURNING *;

-- name: UpdateWorld :one
UPDATE worlds
SET name = $2
WHERE id = $1
RETURNING *;

-- name: DeleteWorld :exec
DELETE FROM worlds
WHERE id = $1;