-- name: CreateChunk :one
INSERT INTO chunks (world_id, chunk_x, chunk_y, chunk_data)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetChunk :one
SELECT * FROM chunks
WHERE world_id = $1 AND chunk_x = $2 AND chunk_y = $3;

-- name: GetChunks :many
SELECT * FROM chunks
WHERE world_id = $1
AND chunk_x >= $2 AND chunk_x <= $3 
AND chunk_y >= $4 AND chunk_y <= $5
ORDER BY chunk_x, chunk_y;

-- name: ChunkExists :one
SELECT EXISTS(
    SELECT 1 FROM chunks
    WHERE world_id = $1 AND chunk_x = $2 AND chunk_y = $3
);

-- name: DeleteChunk :exec
DELETE FROM chunks
WHERE world_id = $1 AND chunk_x = $2 AND chunk_y = $3;