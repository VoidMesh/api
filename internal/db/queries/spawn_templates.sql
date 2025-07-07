-- name: GetSpawnTemplates :many
SELECT template_id, node_type, node_subtype, spawn_type, min_yield, max_yield, regeneration_rate, respawn_delay_hours, spawn_weight, biome_restriction, cluster_size_min, cluster_size_max, cluster_spread_min, cluster_spread_max, clusters_per_chunk, noise_scale, noise_threshold, noise_octaves, noise_persistence
FROM node_spawn_templates;

-- name: GetRespawnDelay :one
SELECT respawn_delay_hours FROM node_spawn_templates
WHERE node_type = ? AND node_subtype = ?;

-- name: GetWorldConfig :one
SELECT config_value FROM world_config WHERE config_key = ?;

-- name: SetWorldConfig :exec
INSERT OR REPLACE INTO world_config (config_key, config_value) VALUES (?, ?);