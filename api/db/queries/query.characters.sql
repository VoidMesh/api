-- name: CreateCharacter :one
INSERT INTO characters (user_id, name, x, y, chunk_x, chunk_y)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCharacterById :one
SELECT * FROM characters
WHERE id = $1;

-- name: GetCharacterByUserAndName :one
SELECT * FROM characters
WHERE user_id = $1 AND name = $2;

-- name: GetCharactersByUser :many
SELECT * FROM characters
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateCharacterPosition :one
UPDATE characters
SET x = $2, y = $3, chunk_x = $4, chunk_y = $5
WHERE id = $1
RETURNING *;

-- name: DeleteCharacter :exec
DELETE FROM characters
WHERE id = $1;

-- name: GetCharactersInChunk :many
SELECT * FROM characters
WHERE chunk_x = $1 AND chunk_y = $2;