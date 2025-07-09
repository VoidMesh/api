-- name: CreatePlayer :one
INSERT INTO players (username, password_hash, salt, email)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetPlayerByUsername :one
SELECT * FROM players
WHERE username = ? LIMIT 1;

-- name: GetPlayerByID :one
SELECT * FROM players
WHERE player_id = ? LIMIT 1;

-- name: GetPlayerByEmail :one
SELECT * FROM players
WHERE email = ? LIMIT 1;

-- name: UpdatePlayerPosition :exec
UPDATE players
SET world_x = ?, world_y = ?, world_z = ?,
    current_chunk_x = ?, current_chunk_z = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE player_id = ?;

-- name: SetPlayerOnline :exec
UPDATE players
SET is_online = 1, last_login = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE player_id = ?;

-- name: SetPlayerOffline :exec
UPDATE players
SET is_online = 0, last_logout = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE player_id = ?;

-- name: GetOnlinePlayers :many
SELECT * FROM players
WHERE is_online = 1;

-- name: GetPlayersInChunk :many
SELECT * FROM players
WHERE current_chunk_x = ? AND current_chunk_z = ? AND is_online = 1;

-- name: UpdatePlayerEmail :exec
UPDATE players
SET email = ?, updated_at = CURRENT_TIMESTAMP
WHERE player_id = ?;

-- name: UpdatePlayerPassword :exec
UPDATE players
SET password_hash = ?, salt = ?, updated_at = CURRENT_TIMESTAMP
WHERE player_id = ?;

-- name: DeletePlayer :exec
DELETE FROM players
WHERE player_id = ?;

-- name: GetPlayerStats :one
SELECT * FROM player_stats
WHERE player_id = ? LIMIT 1;

-- name: CreatePlayerStats :one
INSERT INTO player_stats (player_id)
VALUES (?)
RETURNING *;

-- name: UpdatePlayerStats :exec
UPDATE player_stats
SET total_resources_harvested = total_resources_harvested + ?,
    total_harvest_sessions = total_harvest_sessions + ?,
    iron_ore_harvested = iron_ore_harvested + ?,
    gold_ore_harvested = gold_ore_harvested + ?,
    wood_harvested = wood_harvested + ?,
    stone_harvested = stone_harvested + ?,
    unique_nodes_discovered = ?,
    total_nodes_harvested = total_nodes_harvested + ?,
    last_harvest = CURRENT_TIMESTAMP,
    stats_updated = CURRENT_TIMESTAMP
WHERE player_id = ?;

-- name: IncrementPlayerSessions :exec
UPDATE player_stats
SET sessions_count = sessions_count + 1,
    stats_updated = CURRENT_TIMESTAMP
WHERE player_id = ?;

-- name: UpdatePlayerPlaytime :exec
UPDATE player_stats
SET total_playtime_minutes = total_playtime_minutes + ?,
    stats_updated = CURRENT_TIMESTAMP
WHERE player_id = ?;

-- name: GetPlayerInventory :many
SELECT * FROM player_inventories
WHERE player_id = ?
ORDER BY resource_type, resource_subtype;

-- name: GetPlayerInventoryResource :one
SELECT * FROM player_inventories
WHERE player_id = ? AND resource_type = ? AND resource_subtype = ?
LIMIT 1;

-- name: AddToPlayerInventory :exec
INSERT INTO player_inventories (player_id, resource_type, resource_subtype, quantity)
VALUES (?, ?, ?, ?)
ON CONFLICT(player_id, resource_type, resource_subtype) DO UPDATE SET
    quantity = quantity + excluded.quantity,
    last_updated = CURRENT_TIMESTAMP;

-- name: RemoveFromPlayerInventory :exec
UPDATE player_inventories
SET quantity = quantity - ?,
    last_updated = CURRENT_TIMESTAMP
WHERE player_id = ? AND resource_type = ? AND resource_subtype = ? AND quantity >= ?;

-- name: SetPlayerInventoryQuantity :exec
INSERT INTO player_inventories (player_id, resource_type, resource_subtype, quantity)
VALUES (?, ?, ?, ?)
ON CONFLICT(player_id, resource_type, resource_subtype) DO UPDATE SET
    quantity = excluded.quantity,
    last_updated = CURRENT_TIMESTAMP;

-- name: ClearPlayerInventory :exec
DELETE FROM player_inventories
WHERE player_id = ?;

-- name: GetTotalResourcesInInventory :one
SELECT COALESCE(SUM(quantity), 0) as total_resources
FROM player_inventories
WHERE player_id = ?;

-- name: CreatePlayerSession :one
INSERT INTO player_sessions (player_id, session_token, ip_address, user_agent, expires_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPlayerSession :one
SELECT * FROM player_sessions
WHERE session_token = ? AND expires_at > CURRENT_TIMESTAMP
LIMIT 1;

-- name: UpdatePlayerSessionActivity :exec
UPDATE player_sessions
SET last_activity = CURRENT_TIMESTAMP
WHERE session_token = ?;

-- name: DeletePlayerSession :exec
DELETE FROM player_sessions
WHERE session_token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM player_sessions
WHERE expires_at <= CURRENT_TIMESTAMP;

-- name: GetPlayerActiveSessions :many
SELECT * FROM player_sessions
WHERE player_id = ? AND expires_at > CURRENT_TIMESTAMP
ORDER BY last_activity DESC;

-- name: DeleteAllPlayerSessions :exec
DELETE FROM player_sessions
WHERE player_id = ?;

-- name: GetPlayersWithStats :many
SELECT p.*, 
       COALESCE(ps.total_resources_harvested, 0) as total_resources_harvested,
       COALESCE(ps.total_harvest_sessions, 0) as total_harvest_sessions,
       COALESCE(ps.sessions_count, 0) as sessions_count,
       COALESCE(ps.total_playtime_minutes, 0) as total_playtime_minutes
FROM players p
LEFT JOIN player_stats ps ON p.player_id = ps.player_id
ORDER BY p.created_at DESC;

-- name: GetTopPlayersByResources :many
SELECT p.username, ps.total_resources_harvested
FROM players p
JOIN player_stats ps ON p.player_id = ps.player_id
ORDER BY ps.total_resources_harvested DESC
LIMIT ?;

-- name: GetTopPlayersByPlaytime :many
SELECT p.username, ps.total_playtime_minutes
FROM players p
JOIN player_stats ps ON p.player_id = ps.player_id
ORDER BY ps.total_playtime_minutes DESC
LIMIT ?;