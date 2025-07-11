// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: chunks.sql

package db

import (
	"context"
)

const createChunk = `-- name: CreateChunk :exec
INSERT OR IGNORE INTO chunks (chunk_x, chunk_z) VALUES (?, ?)
`

type CreateChunkParams struct {
	ChunkX int64 `json:"chunk_x"`
	ChunkZ int64 `json:"chunk_z"`
}

func (q *Queries) CreateChunk(ctx context.Context, arg CreateChunkParams) error {
	_, err := q.exec(ctx, q.createChunkStmt, createChunk, arg.ChunkX, arg.ChunkZ)
	return err
}

const getChunk = `-- name: GetChunk :one
SELECT chunk_x, chunk_z, created_at, last_modified FROM chunks
WHERE chunk_x = ? AND chunk_z = ?
`

type GetChunkParams struct {
	ChunkX int64 `json:"chunk_x"`
	ChunkZ int64 `json:"chunk_z"`
}

func (q *Queries) GetChunk(ctx context.Context, arg GetChunkParams) (Chunk, error) {
	row := q.queryRow(ctx, q.getChunkStmt, getChunk, arg.ChunkX, arg.ChunkZ)
	var i Chunk
	err := row.Scan(
		&i.ChunkX,
		&i.ChunkZ,
		&i.CreatedAt,
		&i.LastModified,
	)
	return i, err
}

const updateChunkModified = `-- name: UpdateChunkModified :exec
UPDATE chunks SET last_modified = CURRENT_TIMESTAMP WHERE chunk_x = ? AND chunk_z = ?
`

type UpdateChunkModifiedParams struct {
	ChunkX int64 `json:"chunk_x"`
	ChunkZ int64 `json:"chunk_z"`
}

func (q *Queries) UpdateChunkModified(ctx context.Context, arg UpdateChunkModifiedParams) error {
	_, err := q.exec(ctx, q.updateChunkModifiedStmt, updateChunkModified, arg.ChunkX, arg.ChunkZ)
	return err
}
