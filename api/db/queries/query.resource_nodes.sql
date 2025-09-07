-- Resource Node Operations

-- name: CreateResourceNode :one
INSERT INTO resource_nodes (
  resource_node_type_id,
  world_id,
  chunk_x,
  chunk_y,
  cluster_id,
  x,
  y,
  size
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetResourceNode :one
SELECT
  rn.*
FROM resource_nodes rn
WHERE rn.id = $1;

-- name: GetResourceNodesInChunk :many
SELECT
  rn.*
FROM resource_nodes rn
WHERE rn.world_id = $1 AND rn.chunk_x = $2 AND rn.chunk_y = $3;

-- name: GetResourceNodesInChunks :many
SELECT
  rn.*
FROM resource_nodes rn
WHERE rn.world_id = $1 AND (
      (rn.chunk_x = $2 AND rn.chunk_y = $3) OR
      (rn.chunk_x = $4 AND rn.chunk_y = $5) OR
      (rn.chunk_x = $6 AND rn.chunk_y = $7) OR
      (rn.chunk_x = $8 AND rn.chunk_y = $9) OR
      (rn.chunk_x = $10 AND rn.chunk_y = $11)
);

-- name: GetResourceNodesInChunkRange :many
SELECT
  rn.*
FROM resource_nodes rn
WHERE rn.world_id = $1 AND
      rn.chunk_x >= $2 AND rn.chunk_x <= $3 AND
      rn.chunk_y >= $4 AND rn.chunk_y <= $5;

-- name: GetResourceNodesInCluster :many
SELECT
  rn.*
FROM resource_nodes rn
WHERE rn.cluster_id = $1;

-- name: DeleteResourceNodesInChunk :exec
DELETE FROM resource_nodes
WHERE world_id = $1 AND chunk_x = $2 AND chunk_y = $3;

-- name: ResourceNodeExistsAtPosition :one
SELECT EXISTS(
  SELECT 1 FROM resource_nodes
  WHERE world_id = $1 AND x = $2 AND y = $3
);

-- name: CountResourceNodesInChunk :one
SELECT COUNT(*) FROM resource_nodes
WHERE world_id = $1 AND chunk_x = $2 AND chunk_y = $3;

-- name: CountResourceNodesByType :one
SELECT COUNT(*) FROM resource_nodes
WHERE world_id = $1 AND chunk_x = $2 AND chunk_y = $3 AND resource_node_type_id = $4;
