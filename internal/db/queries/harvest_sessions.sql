-- name: CreateHarvestSession :one
INSERT INTO harvest_sessions (node_id, player_id)
VALUES (?, ?)
RETURNING session_id, node_id, player_id, started_at, last_activity, resources_gathered;

-- name: GetHarvestSession :one
SELECT session_id, node_id, player_id, started_at, last_activity, resources_gathered
FROM harvest_sessions
WHERE session_id = ?;

-- name: GetPlayerActiveSession :one
SELECT session_id, node_id, player_id, started_at, last_activity, resources_gathered
FROM harvest_sessions
WHERE player_id = ? AND last_activity > ?;

-- name: UpdateSessionActivity :exec
UPDATE harvest_sessions SET last_activity = CURRENT_TIMESTAMP, resources_gathered = resources_gathered + ?
WHERE session_id = ?;

-- name: CleanupExpiredSessions :exec
DELETE FROM harvest_sessions WHERE last_activity < ?;

-- name: GetPlayerSessions :many
SELECT session_id, node_id, player_id, started_at, last_activity, resources_gathered
FROM harvest_sessions
WHERE player_id = ?;