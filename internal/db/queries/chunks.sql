-- name: GetChunk :one
SELECT chunk_x, chunk_z, created_at, last_modified FROM chunks
WHERE chunk_x = ? AND chunk_z = ?;

-- name: CreateChunk :exec
INSERT OR IGNORE INTO chunks (chunk_x, chunk_z) VALUES (?, ?);

-- name: UpdateChunkModified :exec
UPDATE chunks SET last_modified = CURRENT_TIMESTAMP WHERE chunk_x = ? AND chunk_z = ?;