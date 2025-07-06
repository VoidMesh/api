-- Drop indexes
DROP INDEX IF EXISTS idx_harvest_log_player;
DROP INDEX IF EXISTS idx_harvest_log_node;
DROP INDEX IF EXISTS idx_harvest_sessions_player;
DROP INDEX IF EXISTS idx_harvest_sessions_node;
DROP INDEX IF EXISTS idx_resource_nodes_respawn;
DROP INDEX IF EXISTS idx_resource_nodes_active;
DROP INDEX IF EXISTS idx_resource_nodes_chunk;

-- Drop tables
DROP TABLE IF EXISTS node_spawn_templates;
DROP TABLE IF EXISTS harvest_log;
DROP TABLE IF EXISTS harvest_sessions;
DROP TABLE IF EXISTS resource_nodes;
DROP TABLE IF EXISTS chunks;