-- name: CreateChunk :one
INSERT INTO chunks (chunk_x, chunk_y, seed, chunk_data)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetChunk :one
SELECT * FROM chunks
WHERE chunk_x = $1 AND chunk_y = $2;

-- name: GetChunks :many
SELECT * FROM chunks
WHERE chunk_x >= $1 AND chunk_x <= $2 
AND chunk_y >= $3 AND chunk_y <= $4
ORDER BY chunk_x, chunk_y;

-- name: ChunkExists :one
SELECT EXISTS(
    SELECT 1 FROM chunks
    WHERE chunk_x = $1 AND chunk_y = $2
);

-- name: DeleteChunk :exec
DELETE FROM chunks
WHERE chunk_x = $1 AND chunk_y = $2;