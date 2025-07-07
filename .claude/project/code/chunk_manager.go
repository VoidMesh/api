package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	CHUNK_SIZE = 16

	// Node types
	IRON_ORE = 1
	GOLD_ORE = 2
	WOOD     = 3
	STONE    = 4

	// Node subtypes
	POOR_QUALITY   = 0
	NORMAL_QUALITY = 1
	RICH_QUALITY   = 2

	// Spawn types
	RANDOM_SPAWN     = 0
	STATIC_DAILY     = 1
	STATIC_PERMANENT = 2

	// Harvest session timeout (minutes)
	SESSION_TIMEOUT = 5
)

type ChunkManager struct {
	db *sql.DB
}

type ResourceNode struct {
	NodeID           int
	ChunkX, ChunkZ   int
	LocalX, LocalZ   int
	NodeType         int
	NodeSubtype      int
	MaxYield         int
	CurrentYield     int
	RegenerationRate int
	SpawnedAt        time.Time
	LastHarvest      *time.Time
	RespawnTimer     *time.Time
	SpawnType        int
	IsActive         bool
}

type HarvestSession struct {
	SessionID         int
	NodeID            int
	PlayerID          int
	StartedAt         time.Time
	LastActivity      time.Time
	ResourcesGathered int
}

type SpawnTemplate struct {
	TemplateID        int
	NodeType          int
	NodeSubtype       int
	SpawnType         int
	MinYield          int
	MaxYield          int
	RegenerationRate  int
	RespawnDelayHours int
	SpawnWeight       int
}

func NewChunkManager(dbPath string) (*ChunkManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	cm := &ChunkManager{db: db}

	// Initialize spawn templates
	cm.initializeSpawnTemplates()

	return cm, nil
}

func (cm *ChunkManager) initializeSpawnTemplates() {
	templates := []SpawnTemplate{
		// Static daily iron ore nodes
		{NodeType: IRON_ORE, NodeSubtype: NORMAL_QUALITY, SpawnType: STATIC_DAILY, MinYield: 100, MaxYield: 200, RegenerationRate: 5, RespawnDelayHours: 24, SpawnWeight: 3},
		{NodeType: IRON_ORE, NodeSubtype: RICH_QUALITY, SpawnType: STATIC_DAILY, MinYield: 300, MaxYield: 500, RegenerationRate: 10, RespawnDelayHours: 24, SpawnWeight: 1},

		// Random gold ore nodes
		{NodeType: GOLD_ORE, NodeSubtype: NORMAL_QUALITY, SpawnType: RANDOM_SPAWN, MinYield: 50, MaxYield: 100, RegenerationRate: 2, RespawnDelayHours: 12, SpawnWeight: 2},
		{NodeType: GOLD_ORE, NodeSubtype: RICH_QUALITY, SpawnType: RANDOM_SPAWN, MinYield: 150, MaxYield: 300, RegenerationRate: 5, RespawnDelayHours: 12, SpawnWeight: 1},

		// Permanent wood nodes (like trees)
		{NodeType: WOOD, NodeSubtype: NORMAL_QUALITY, SpawnType: STATIC_PERMANENT, MinYield: 50, MaxYield: 100, RegenerationRate: 1, RespawnDelayHours: 6, SpawnWeight: 4},
	}

	for _, template := range templates {
		cm.db.Exec(`INSERT OR IGNORE INTO node_spawn_templates
                    (node_type, node_subtype, spawn_type, min_yield, max_yield, regeneration_rate, respawn_delay_hours, spawn_weight)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			template.NodeType, template.NodeSubtype, template.SpawnType,
			template.MinYield, template.MaxYield, template.RegenerationRate,
			template.RespawnDelayHours, template.SpawnWeight)
	}
}

func (cm *ChunkManager) LoadChunk(chunkX, chunkZ int) ([]ResourceNode, error) {
	// Ensure chunk exists
	cm.db.Exec(`INSERT OR IGNORE INTO chunks (chunk_x, chunk_z) VALUES (?, ?)`, chunkX, chunkZ)

	// Generate nodes if this chunk is new or needs respawning
	err := cm.generateNodes(chunkX, chunkZ)
	if err != nil {
		return nil, err
	}

	// Load active nodes
	query := `SELECT node_id, chunk_x, chunk_z, local_x, local_z, node_type, node_subtype,
                     max_yield, current_yield, regeneration_rate, spawned_at, last_harvest,
                     respawn_timer, spawn_type, is_active
              FROM resource_nodes
              WHERE chunk_x = ? AND chunk_z = ? AND is_active = 1`

	rows, err := cm.db.Query(query, chunkX, chunkZ)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []ResourceNode
	for rows.Next() {
		var node ResourceNode
		var lastHarvest, respawnTimer sql.NullTime

		err := rows.Scan(&node.NodeID, &node.ChunkX, &node.ChunkZ, &node.LocalX, &node.LocalZ,
			&node.NodeType, &node.NodeSubtype, &node.MaxYield, &node.CurrentYield,
			&node.RegenerationRate, &node.SpawnedAt, &lastHarvest, &respawnTimer,
			&node.SpawnType, &node.IsActive)
		if err != nil {
			return nil, err
		}

		if lastHarvest.Valid {
			node.LastHarvest = &lastHarvest.Time
		}
		if respawnTimer.Valid {
			node.RespawnTimer = &respawnTimer.Time
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (cm *ChunkManager) generateNodes(chunkX, chunkZ int) error {
	// Check if we need to spawn daily nodes
	cm.spawnDailyNodes(chunkX, chunkZ)

	// Check if we need to spawn random nodes
	cm.spawnRandomNodes(chunkX, chunkZ)

	// Respawn depleted nodes whose timer has expired
	cm.respawnNodes(chunkX, chunkZ)

	return nil
}

func (cm *ChunkManager) spawnDailyNodes(chunkX, chunkZ int) {
	// Check if we have spawned daily nodes today
	today := time.Now().Format("2006-01-02")

	var count int
	cm.db.QueryRow(`SELECT COUNT(*) FROM resource_nodes
                    WHERE chunk_x = ? AND chunk_z = ? AND spawn_type = ? AND DATE(spawned_at) = ?`,
		chunkX, chunkZ, STATIC_DAILY, today).Scan(&count)

	if count > 0 {
		return // Already spawned today
	}

	// Spawn daily nodes
	templates := cm.getSpawnTemplates(STATIC_DAILY)
	for _, template := range templates {
		for i := 0; i < template.SpawnWeight; i++ {
			cm.spawnNode(chunkX, chunkZ, template)
		}
	}
}

func (cm *ChunkManager) spawnRandomNodes(chunkX, chunkZ int) {
	// Spawn random nodes based on current node density
	var activeCount int
	cm.db.QueryRow(`SELECT COUNT(*) FROM resource_nodes
                    WHERE chunk_x = ? AND chunk_z = ? AND spawn_type = ? AND is_active = 1`,
		chunkX, chunkZ, RANDOM_SPAWN).Scan(&activeCount)

	// Maintain 2-5 random nodes per chunk
	maxRandomNodes := 5
	if activeCount < maxRandomNodes {
		templates := cm.getSpawnTemplates(RANDOM_SPAWN)
		if len(templates) > 0 {
			template := templates[rand.Intn(len(templates))]
			cm.spawnNode(chunkX, chunkZ, template)
		}
	}
}

func (cm *ChunkManager) spawnNode(chunkX, chunkZ int, template SpawnTemplate) {
	// Find random position
	localX := rand.Intn(CHUNK_SIZE)
	localZ := rand.Intn(CHUNK_SIZE)

	// Check if position is occupied
	var existing int
	cm.db.QueryRow(`SELECT COUNT(*) FROM resource_nodes
                    WHERE chunk_x = ? AND chunk_z = ? AND local_x = ? AND local_z = ? AND is_active = 1`,
		chunkX, chunkZ, localX, localZ).Scan(&existing)

	if existing > 0 {
		return // Position occupied
	}

	// Generate yield
	yield := template.MinYield + rand.Intn(template.MaxYield-template.MinYield+1)

	cm.db.Exec(`INSERT INTO resource_nodes
                (chunk_x, chunk_z, local_x, local_z, node_type, node_subtype, max_yield, current_yield, regeneration_rate, spawn_type)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chunkX, chunkZ, localX, localZ, template.NodeType, template.NodeSubtype,
		yield, yield, template.RegenerationRate, template.SpawnType)
}

func (cm *ChunkManager) getSpawnTemplates(spawnType int) []SpawnTemplate {
	rows, err := cm.db.Query(`SELECT template_id, node_type, node_subtype, spawn_type, min_yield, max_yield, regeneration_rate, respawn_delay_hours, spawn_weight
                              FROM node_spawn_templates WHERE spawn_type = ?`, spawnType)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var templates []SpawnTemplate
	for rows.Next() {
		var t SpawnTemplate
		rows.Scan(&t.TemplateID, &t.NodeType, &t.NodeSubtype, &t.SpawnType,
			&t.MinYield, &t.MaxYield, &t.RegenerationRate, &t.RespawnDelayHours, &t.SpawnWeight)
		templates = append(templates, t)
	}

	return templates
}

func (cm *ChunkManager) respawnNodes(chunkX, chunkZ int) {
	now := time.Now()

	// Find nodes ready to respawn
	rows, err := cm.db.Query(`SELECT node_id, max_yield FROM resource_nodes
                              WHERE chunk_x = ? AND chunk_z = ? AND is_active = 0 AND respawn_timer IS NOT NULL AND respawn_timer <= ?`,
		chunkX, chunkZ, now)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var nodeID, maxYield int
		rows.Scan(&nodeID, &maxYield)

		// Respawn the node
		cm.db.Exec(`UPDATE resource_nodes SET current_yield = ?, is_active = 1, respawn_timer = NULL, spawned_at = CURRENT_TIMESTAMP
                    WHERE node_id = ?`, maxYield, nodeID)
	}
}

// StartHarvest begins a harvest session for a player
func (cm *ChunkManager) StartHarvest(nodeID, playerID int) (*HarvestSession, error) {
	tx, err := cm.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Check if node exists and is active
	var currentYield int
	err = tx.QueryRow(`SELECT current_yield FROM resource_nodes WHERE node_id = ? AND is_active = 1`,
		nodeID).Scan(&currentYield)
	if err != nil {
		return nil, fmt.Errorf("node not found or inactive")
	}

	if currentYield <= 0 {
		return nil, fmt.Errorf("node is depleted")
	}

	// Check if player already has an active session
	var existingSession int
	tx.QueryRow(`SELECT COUNT(*) FROM harvest_sessions WHERE player_id = ? AND last_activity > ?`,
		playerID, time.Now().Add(-SESSION_TIMEOUT*time.Minute)).Scan(&existingSession)

	if existingSession > 0 {
		return nil, fmt.Errorf("player already has active harvest session")
	}

	// Create harvest session
	result, err := tx.Exec(`INSERT INTO harvest_sessions (node_id, player_id) VALUES (?, ?)`,
		nodeID, playerID)
	if err != nil {
		return nil, err
	}

	sessionID, _ := result.LastInsertId()

	session := &HarvestSession{
		SessionID:         int(sessionID),
		NodeID:            nodeID,
		PlayerID:          playerID,
		StartedAt:         time.Now(),
		LastActivity:      time.Now(),
		ResourcesGathered: 0,
	}

	return session, tx.Commit()
}

// HarvestResource performs a harvest action
func (cm *ChunkManager) HarvestResource(sessionID int, harvestAmount int) error {
	tx, err := cm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Validate session
	var nodeID, playerID int
	var lastActivity time.Time
	err = tx.QueryRow(`SELECT node_id, player_id, last_activity FROM harvest_sessions WHERE session_id = ?`,
		sessionID).Scan(&nodeID, &playerID, &lastActivity)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	if time.Since(lastActivity) > SESSION_TIMEOUT*time.Minute {
		return fmt.Errorf("session expired")
	}

	// Get node info
	var currentYield, maxYield int
	err = tx.QueryRow(`SELECT current_yield, max_yield FROM resource_nodes WHERE node_id = ?`,
		nodeID).Scan(&currentYield, &maxYield)
	if err != nil {
		return err
	}

	// Calculate actual harvest amount
	actualHarvest := harvestAmount
	if actualHarvest > currentYield {
		actualHarvest = currentYield
	}

	if actualHarvest <= 0 {
		return fmt.Errorf("node is depleted")
	}

	// Update node
	newYield := currentYield - actualHarvest
	_, err = tx.Exec(`UPDATE resource_nodes SET current_yield = ?, last_harvest = CURRENT_TIMESTAMP WHERE node_id = ?`,
		newYield, nodeID)
	if err != nil {
		return err
	}

	// If node is depleted, set respawn timer
	if newYield <= 0 {
		// Get respawn delay from template
		var respawnHours int
		tx.QueryRow(`SELECT nst.respawn_delay_hours FROM resource_nodes rn
                     JOIN node_spawn_templates nst ON rn.node_type = nst.node_type AND rn.node_subtype = nst.node_subtype
                     WHERE rn.node_id = ?`, nodeID).Scan(&respawnHours)

		respawnTime := time.Now().Add(time.Duration(respawnHours) * time.Hour)
		tx.Exec(`UPDATE resource_nodes SET is_active = 0, respawn_timer = ? WHERE node_id = ?`,
			respawnTime, nodeID)
	}

	// Update session
	_, err = tx.Exec(`UPDATE harvest_sessions SET last_activity = CURRENT_TIMESTAMP, resources_gathered = resources_gathered + ?
                     WHERE session_id = ?`, actualHarvest, sessionID)
	if err != nil {
		return err
	}

	// Log harvest
	_, err = tx.Exec(`INSERT INTO harvest_log (node_id, player_id, amount_harvested, node_yield_before, node_yield_after)
                     VALUES (?, ?, ?, ?, ?)`,
		nodeID, playerID, actualHarvest, currentYield, newYield)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RegenerateResources processes natural regeneration
func (cm *ChunkManager) RegenerateResources() error {
	// This would be called periodically (e.g., every hour)
	_, err := cm.db.Exec(`UPDATE resource_nodes
                         SET current_yield = MIN(current_yield + regeneration_rate, max_yield)
                         WHERE regeneration_rate > 0 AND is_active = 1 AND current_yield < max_yield`)
	return err
}

// CleanupExpiredSessions removes old harvest sessions
func (cm *ChunkManager) CleanupExpiredSessions() error {
	cutoff := time.Now().Add(-SESSION_TIMEOUT * time.Minute)
	_, err := cm.db.Exec(`DELETE FROM harvest_sessions WHERE last_activity < ?`, cutoff)
	return err
}

func (cm *ChunkManager) Close() error {
	return cm.db.Close()
}

// Example usage
func main() {
	cm, err := NewChunkManager("game.db")
	if err != nil {
		log.Fatal(err)
	}
	defer cm.Close()

	// Load chunk nodes
	nodes, err := cm.LoadChunk(0, 0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d resource nodes in chunk (0,0)\n", len(nodes))

	for _, node := range nodes {
		fmt.Printf("Node %d: Type %d, Yield %d/%d at (%d,%d)\n",
			node.NodeID, node.NodeType, node.CurrentYield, node.MaxYield,
			node.LocalX, node.LocalZ)
	}

	// Example harvest
	if len(nodes) > 0 {
		node := nodes[0]
		session, err := cm.StartHarvest(node.NodeID, 1)
		if err != nil {
			fmt.Printf("Failed to start harvest: %v\n", err)
		} else {
			fmt.Printf("Started harvest session %d\n", session.SessionID)

			// Harvest 10 resources
			err = cm.HarvestResource(session.SessionID, 10)
			if err != nil {
				fmt.Printf("Failed to harvest: %v\n", err)
			} else {
				fmt.Printf("Successfully harvested 10 resources\n")
			}
		}
	}
}
