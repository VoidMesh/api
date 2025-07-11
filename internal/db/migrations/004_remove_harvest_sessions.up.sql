-- Drop harvest_sessions table and related indexes
DROP INDEX IF EXISTS idx_harvest_sessions_node;
DROP INDEX IF EXISTS idx_harvest_sessions_player;
DROP TABLE IF EXISTS harvest_sessions;