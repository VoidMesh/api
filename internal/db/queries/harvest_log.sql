-- name: CreateHarvestLog :exec
INSERT INTO harvest_log (node_id, player_id, amount_harvested, node_yield_before, node_yield_after)
VALUES (?, ?, ?, ?, ?);

-- name: GetPlayerDailyHarvest :one
SELECT COUNT(*) FROM harvest_log
WHERE player_id = ? AND node_id = ? AND DATE(harvested_at) = DATE('now');