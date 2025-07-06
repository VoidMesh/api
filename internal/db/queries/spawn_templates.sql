-- name: GetSpawnTemplates :many
SELECT template_id, node_type, node_subtype, spawn_type, min_yield, max_yield, regeneration_rate, respawn_delay_hours, spawn_weight, biome_restriction
FROM node_spawn_templates
WHERE spawn_type = ?;

-- name: GetRespawnDelay :one
SELECT respawn_delay_hours FROM node_spawn_templates
WHERE node_type = ? AND node_subtype = ?;