-- Resource Type Operations

-- name: CreateResourceType :one
INSERT INTO resource_types (name, description, terrain_type, rarity, visual_data, properties)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetResourceType :one
SELECT * FROM resource_types
WHERE id = $1;

-- name: ListResourceTypes :many
SELECT * FROM resource_types
ORDER BY name;

-- name: ListResourceTypesByTerrain :many
SELECT * FROM resource_types
WHERE terrain_type = $1
ORDER BY name;

-- name: UpdateResourceType :one
UPDATE resource_types
SET 
  name = $2,
  description = $3,
  terrain_type = $4,
  rarity = $5,
  visual_data = $6,
  properties = $7
WHERE id = $1
RETURNING *;

-- name: DeleteResourceType :exec
DELETE FROM resource_types
WHERE id = $1;

-- Resource Node Operations

-- name: CreateResourceNode :one
INSERT INTO resource_nodes (
  resource_type_id, 
  chunk_x, 
  chunk_y, 
  cluster_id, 
  pos_x, 
  pos_y,
  size
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetResourceNode :one
SELECT
  rn.*,
  rt.name as resource_name,
  rt.terrain_type,
  rt.rarity
FROM resource_nodes rn
JOIN resource_types rt ON rn.resource_type_id = rt.id
WHERE rn.id = $1;

-- name: GetResourceNodesInChunk :many
SELECT
  rn.*,
  rt.name as resource_name,
  rt.terrain_type,
  rt.rarity
FROM resource_nodes rn
JOIN resource_types rt ON rn.resource_type_id = rt.id
WHERE rn.chunk_x = $1 AND rn.chunk_y = $2;

-- name: GetResourceNodesInChunks :many
SELECT
  rn.*,
  rt.name as resource_name,
  rt.terrain_type,
  rt.rarity
FROM resource_nodes rn
JOIN resource_types rt ON rn.resource_type_id = rt.id
WHERE (rn.chunk_x = $1 AND rn.chunk_y = $2) OR
      (rn.chunk_x = $3 AND rn.chunk_y = $4) OR
      (rn.chunk_x = $5 AND rn.chunk_y = $6) OR
      (rn.chunk_x = $7 AND rn.chunk_y = $8) OR
      (rn.chunk_x = $9 AND rn.chunk_y = $10);

-- name: GetResourceNodesInChunkRange :many
SELECT
  rn.*,
  rt.name as resource_name,
  rt.terrain_type,
  rt.rarity
FROM resource_nodes rn
JOIN resource_types rt ON rn.resource_type_id = rt.id
WHERE rn.chunk_x >= $1 AND rn.chunk_x <= $2 AND
      rn.chunk_y >= $3 AND rn.chunk_y <= $4;

-- name: GetResourceNodesInCluster :many
SELECT
  rn.*,
  rt.name as resource_name,
  rt.terrain_type,
  rt.rarity
FROM resource_nodes rn
JOIN resource_types rt ON rn.resource_type_id = rt.id
WHERE rn.cluster_id = $1;

-- name: DeleteResourceNodesInChunk :exec
DELETE FROM resource_nodes
WHERE chunk_x = $1 AND chunk_y = $2;

-- name: ResourceExistsAtPosition :one
SELECT EXISTS(
  SELECT 1 FROM resource_nodes
  WHERE chunk_x = $1 AND chunk_y = $2 AND pos_x = $3 AND pos_y = $4
);

-- name: CountResourceNodesInChunk :one
SELECT COUNT(*) FROM resource_nodes
WHERE chunk_x = $1 AND chunk_y = $2;

-- name: CountResourceNodesByType :one
SELECT COUNT(*) FROM resource_nodes
WHERE chunk_x = $1 AND chunk_y = $2 AND resource_type_id = $3;