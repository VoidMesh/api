-- name: GetChunkNodes :many
SELECT node_id, chunk_x, chunk_z, local_x, local_z, node_type, node_subtype,
       max_yield, current_yield, regeneration_rate, spawned_at, last_harvest,
       respawn_timer, spawn_type, is_active
FROM resource_nodes
WHERE chunk_x = ? AND chunk_z = ? AND is_active = 1;

-- name: GetNode :one
SELECT node_id, chunk_x, chunk_z, local_x, local_z, node_type, node_subtype,
       max_yield, current_yield, regeneration_rate, spawned_at, last_harvest,
       respawn_timer, spawn_type, is_active
FROM resource_nodes
WHERE node_id = ?;

-- name: CreateNode :one
INSERT INTO resource_nodes (chunk_x, chunk_z, local_x, local_z, node_type, node_subtype, max_yield, current_yield, regeneration_rate, spawn_type)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING node_id;

-- name: UpdateNodeYield :exec
UPDATE resource_nodes SET current_yield = ?, last_harvest = CURRENT_TIMESTAMP WHERE node_id = ?;

-- name: DeactivateNode :exec
UPDATE resource_nodes SET is_active = 0, respawn_timer = ? WHERE node_id = ?;

-- name: ReactivateNode :exec
UPDATE resource_nodes SET current_yield = ?, is_active = 1, respawn_timer = NULL, spawned_at = CURRENT_TIMESTAMP WHERE node_id = ?;

-- name: CheckNodePosition :one
SELECT COUNT(*) FROM resource_nodes
WHERE chunk_x = ? AND chunk_z = ? AND local_x = ? AND local_z = ? AND is_active = 1;

-- name: GetNodesToRespawn :many
SELECT node_id, max_yield FROM resource_nodes
WHERE chunk_x = ? AND chunk_z = ? AND is_active = 0 AND respawn_timer IS NOT NULL AND respawn_timer <= ?;

-- name: GetDailyNodeCount :one
SELECT COUNT(*) FROM resource_nodes
WHERE chunk_x = ? AND chunk_z = ? AND spawn_type = 1 AND DATE(spawned_at) = DATE(?);

-- name: GetRandomNodeCount :one
SELECT COUNT(*) FROM resource_nodes
WHERE chunk_x = ? AND chunk_z = ? AND spawn_type = 0 AND is_active = 1;

-- name: GetChunkNodeCount :one
SELECT COUNT(*) FROM resource_nodes
WHERE chunk_x = ? AND chunk_z = ? AND node_type = ? AND node_subtype = ? AND is_active = 1;

-- name: RegenerateNodeYield :exec
UPDATE resource_nodes
SET current_yield = MIN(current_yield + regeneration_rate, max_yield)
WHERE regeneration_rate > 0 AND is_active = 1 AND current_yield < max_yield;