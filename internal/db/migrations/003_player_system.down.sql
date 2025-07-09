-- Drop indexes
DROP INDEX IF EXISTS idx_player_sessions_activity;
DROP INDEX IF EXISTS idx_player_sessions_expires;
DROP INDEX IF EXISTS idx_player_sessions_token;
DROP INDEX IF EXISTS idx_player_sessions_player;

DROP INDEX IF EXISTS idx_player_stats_updated;
DROP INDEX IF EXISTS idx_player_stats_player;

DROP INDEX IF EXISTS idx_player_inventories_updated;
DROP INDEX IF EXISTS idx_player_inventories_resource;
DROP INDEX IF EXISTS idx_player_inventories_player;

DROP INDEX IF EXISTS idx_players_position;
DROP INDEX IF EXISTS idx_players_online;
DROP INDEX IF EXISTS idx_players_email;
DROP INDEX IF EXISTS idx_players_username;

-- Drop view
DROP VIEW IF EXISTS player_validation;

-- Drop tables
DROP TABLE IF EXISTS player_sessions;
DROP TABLE IF EXISTS player_stats;
DROP TABLE IF EXISTS player_inventories;
DROP TABLE IF EXISTS players;