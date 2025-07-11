package chunk

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/VoidMesh/api/internal/db"
	"github.com/aquilax/go-perlin"
	"github.com/charmbracelet/log"
)

type occupiedCacheEntry struct {
	positions map[int64]struct{} // key = localX<<8 | localZ
	expires   time.Time
}

type Manager struct {
	db            *sql.DB
	queries       *db.LoggingQueries
	worldSeed     int64
	noiseGens     map[int64]*perlin.Perlin
	occupiedCache map[[2]int64]occupiedCacheEntry // key = [2]int64{chunkX,chunkZ}
	cacheMutex    sync.RWMutex
	playerManager PlayerManager // Interface for player operations
}

// PlayerManager interface for player operations
type PlayerManager interface {
	AddToInventory(ctx context.Context, playerID int64, resourceType, resourceSubtype, quantity int64) error
	UpdateHarvestStats(ctx context.Context, playerID int64, update HarvestStatsUpdate) error
}


func NewManager(database *sql.DB, playerMgr PlayerManager) *Manager {
	m := &Manager{
		db:            database,
		queries:       db.NewLoggingQueries(database),
		noiseGens:     make(map[int64]*perlin.Perlin),
		occupiedCache: make(map[[2]int64]occupiedCacheEntry),
		playerManager: playerMgr,
	}

	// Initialize world seed and noise generators
	ctx := context.Background()
	err := m.initializeWorldSeed(ctx)
	if err != nil {
		log.Error("failed to initialize world seed", "error", err)
	}

	return m
}

func (m *Manager) initializeWorldSeed(ctx context.Context) error {
	// Try to get existing world seed
	seedStr, err := m.queries.GetWorldConfig(ctx, "world_seed")

	if err == sql.ErrNoRows {
		// Generate new world seed
		m.worldSeed = time.Now().UnixNano()
		err = m.queries.SetWorldConfig(ctx, db.SetWorldConfigParams{
			ConfigKey:   "world_seed",
			ConfigValue: strconv.FormatInt(m.worldSeed, 10),
		})
		if err != nil {
			return fmt.Errorf("failed to set world seed: %w", err)
		}
		log.Info("generated new world seed", "seed", m.worldSeed)
	} else if err != nil {
		return fmt.Errorf("failed to get world seed: %w", err)
	} else {
		// Use existing seed
		m.worldSeed, err = strconv.ParseInt(seedStr, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse world seed: %w", err)
		}
		log.Info("loaded existing world seed", "seed", m.worldSeed)
	}

	// Initialize noise generators for each resource type
	resourceTypes := []int64{IronOre, GoldOre, Wood, Stone}
	for _, resourceType := range resourceTypes {
		// Use different seeds for each resource type
		resourceSeed := m.worldSeed + resourceType*1000
		randSource := rand.NewSource(resourceSeed)
		m.noiseGens[resourceType] = perlin.NewPerlinRandSource(0.1, 0.1, 3, randSource)
	}

	return nil
}

func (m *Manager) evaluateChunkNoise(chunkX, chunkZ int64, template db.NodeSpawnTemplate) float64 {
	noiseGen, exists := m.noiseGens[template.NodeType]
	if !exists {
		return 0.0
	}

	noiseScale := 0.1
	if template.NoiseScale.Valid {
		noiseScale = template.NoiseScale.Float64
	}

	x := float64(chunkX) * noiseScale
	z := float64(chunkZ) * noiseScale

	return noiseGen.Noise2D(x, z)
}

// determineBehaviorFromSeed calculates spawn behavior deterministically from position and world seed
func (m *Manager) determineBehaviorFromSeed(chunkX, chunkZ, localX, localZ int64) int64 {
	// Create deterministic hash from position and world seed
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("%d:%d:%d:%d:%d", m.worldSeed, chunkX, chunkZ, localX, localZ)))
	hash := h.Sum64()

	// Use hash to determine spawn behavior
	// 30% chance for Static Daily (spawn_type = 1), 70% chance for Random Spawn (spawn_type = 0)
	if hash%100 < 30 {
		return 1 // Static Daily
}
return 0 // Random Spawn
}

func (m *Manager) getOccupiedPositions(ctx context.Context, chunkX, chunkZ int64) (map[int64]struct{}, error) {
	key := [2]int64{chunkX, chunkZ}
	m.cacheMutex.RLock()
	entry, exists := m.occupiedCache[key]
	m.cacheMutex.RUnlock()

	if exists && time.Now().Before(entry.expires) {
		return entry.positions, nil
	}

	rows, err := m.queries.GetChunkOccupiedPositions(ctx, db.GetChunkOccupiedPositionsParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get occupied positions: %w", err)
	}

	positions := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		posKey := (row.LocalX << 8) | row.LocalZ
		positions[posKey] = struct{}{}
	}

	m.cacheMutex.Lock()
	m.occupiedCache[key] = occupiedCacheEntry{
		positions: positions,
		expires:   time.Now().Add(30 * time.Second),
	}
	m.cacheMutex.Unlock()

	return positions, nil
}

func (m *Manager) LoadChunk(ctx context.Context, chunkX, chunkZ int64) (*ChunkResponse, error) {
	log.Debug("Loading chunk", "chunk_x", chunkX, "chunk_z", chunkZ)
	start := time.Now()

	// Ensure chunk exists
	log.Debug("Ensuring chunk exists in database", "chunk_x", chunkX, "chunk_z", chunkZ)
	err := m.queries.CreateChunk(ctx, db.CreateChunkParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
	})
	if err != nil {
		log.Error("failed to create chunk", "error", err, "chunk_x", chunkX, "chunk_z", chunkZ)
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}
	log.Debug("Chunk ensured in database", "chunk_x", chunkX, "chunk_z", chunkZ)

	// Generate nodes if needed
	log.Debug("Starting node generation", "chunk_x", chunkX, "chunk_z", chunkZ)
	err = m.generateNodes(ctx, chunkX, chunkZ)
	if err != nil {
		log.Error("failed to generate nodes", "error", err, "chunk_x", chunkX, "chunk_z", chunkZ)
		return nil, fmt.Errorf("failed to generate nodes: %w", err)
	}
	log.Debug("Node generation completed", "chunk_x", chunkX, "chunk_z", chunkZ)

	// Load active nodes
	log.Debug("Loading active nodes from database", "chunk_x", chunkX, "chunk_z", chunkZ)
	dbNodes, err := m.queries.GetChunkNodes(ctx, db.GetChunkNodesParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
	})
	if err != nil {
		log.Error("failed to get chunk nodes", "error", err, "chunk_x", chunkX, "chunk_z", chunkZ)
		return nil, fmt.Errorf("failed to get chunk nodes: %w", err)
	}
	log.Debug("Loaded nodes from database", "chunk_x", chunkX, "chunk_z", chunkZ, "node_count", len(dbNodes))

	// Convert to response format
	nodes := make([]ResourceNode, len(dbNodes))
	for i, node := range dbNodes {
		nodes[i] = ResourceNode{
			NodeID:           node.NodeID,
			ChunkX:           node.ChunkX,
			ChunkZ:           node.ChunkZ,
			LocalX:           node.LocalX,
			LocalZ:           node.LocalZ,
			NodeType:         node.NodeType,
			NodeSubtype:      node.NodeSubtype.Int64,
			MaxYield:         node.MaxYield,
			CurrentYield:     node.CurrentYield,
			RegenerationRate: node.RegenerationRate.Int64,
			SpawnedAt:        node.SpawnedAt.Time,
			SpawnType:        node.SpawnType,
			IsActive:         node.IsActive.Int64 == 1,
		}
		if node.LastHarvest.Valid {
			nodes[i].LastHarvest = &node.LastHarvest.Time
		}
		if node.RespawnTimer.Valid {
			nodes[i].RespawnTimer = &node.RespawnTimer.Time
		}
	}

	log.Debug("Chunk loading completed", "chunk_x", chunkX, "chunk_z", chunkZ, "total_nodes", len(nodes), "duration", time.Since(start))
	return &ChunkResponse{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
		Nodes:  nodes,
	}, nil
}

func (m *Manager) generateNodes(ctx context.Context, chunkX, chunkZ int64) error {
	log.Debug("starting unified node generation", "chunk_x", chunkX, "chunk_z", chunkZ)

	// Get ALL templates - no filtering by spawn_type
	templates, err := m.queries.GetSpawnTemplates(ctx)
	if err != nil {
		return fmt.Errorf("failed to get spawn templates: %w", err)
	}

	log.Debug("evaluating all templates", "chunk_x", chunkX, "chunk_z", chunkZ, "template_count", len(templates))

	// Process each template with noise evaluation
	for _, template := range templates {
		// Evaluate noise for this chunk and template
		noiseValue := m.evaluateChunkNoise(chunkX, chunkZ, template)

		// Check if this resource should spawn based on noise threshold
		noiseThreshold := 0.5
		if template.NoiseThreshold.Valid {
			noiseThreshold = template.NoiseThreshold.Float64
		}

		log.Debug("evaluating template", "chunk_x", chunkX, "chunk_z", chunkZ, "template_id", template.TemplateID, "node_type", template.NodeType, "noise_value", noiseValue, "threshold", noiseThreshold)

		if noiseValue > noiseThreshold {
			// Calculate max nodes based on noise intensity
			maxNodes := int64((noiseValue - noiseThreshold) * 8) // 0-8 nodes based on noise
			if maxNodes < 1 {
				maxNodes = 1
			}
			if maxNodes > 3 {
				maxNodes = 3
			}

			// Count existing nodes for this template in this chunk
			existingCount, err := m.queries.GetChunkNodeCount(ctx, db.GetChunkNodeCountParams{
				ChunkX:      chunkX,
				ChunkZ:      chunkZ,
				NodeType:    template.NodeType,
				NodeSubtype: template.NodeSubtype,
			})
			if err != nil {
				log.Error("failed to get existing node count", "error", err)
				continue
			}

			log.Debug("node count check", "chunk_x", chunkX, "chunk_z", chunkZ, "template_id", template.TemplateID, "existing_count", existingCount, "max_nodes", maxNodes)

			// Spawn nodes if below threshold
			if existingCount < maxNodes {
				nodesToSpawn := maxNodes - existingCount
				for i := int64(0); i < nodesToSpawn; i++ {
					err := m.spawnNode(ctx, chunkX, chunkZ, template)
					if err != nil {
						log.Error("failed to spawn node", "error", err, "template_id", template.TemplateID, "node_type", template.NodeType, "attempt", i)
						continue
					}
					log.Debug("successfully spawned node", "chunk_x", chunkX, "chunk_z", chunkZ, "template_id", template.TemplateID, "node_type", template.NodeType, "attempt", i)
				}
			}
		} else {
			log.Debug("skipping template - below noise threshold", "chunk_x", chunkX, "chunk_z", chunkZ, "template_id", template.TemplateID, "noise_value", noiseValue, "threshold", noiseThreshold)
		}
	}

	// Respawn depleted nodes whose timer has expired
	err = m.respawnNodes(ctx, chunkX, chunkZ)
	if err != nil {
		return fmt.Errorf("failed to respawn nodes: %w", err)
	}

	log.Debug("completed unified node generation", "chunk_x", chunkX, "chunk_z", chunkZ)
	return nil
}

func (m *Manager) spawnNode(ctx context.Context, chunkX, chunkZ int64, template db.NodeSpawnTemplate) error {
	// Use cluster spawning if cluster parameters are defined
	clusterSizeMin := int64(1)
	clusterSizeMax := int64(1)
	clusterSpreadMin := int64(0)
	clusterSpreadMax := int64(0)
	clustersPerChunk := int64(1)

	if template.ClusterSizeMin.Valid {
		clusterSizeMin = template.ClusterSizeMin.Int64
	}
	if template.ClusterSizeMax.Valid {
		clusterSizeMax = template.ClusterSizeMax.Int64
	}
	if template.ClusterSpreadMin.Valid {
		clusterSpreadMin = template.ClusterSpreadMin.Int64
	}
	if template.ClusterSpreadMax.Valid {
		clusterSpreadMax = template.ClusterSpreadMax.Int64
	}
	if template.ClustersPerChunk.Valid {
		clustersPerChunk = template.ClustersPerChunk.Int64
	}

	// If cluster parameters indicate clustering, use cluster spawning
	if clusterSizeMax > 1 || clusterSpreadMax > 0 {
		return m.spawnNodeCluster(ctx, chunkX, chunkZ, template, clusterSizeMin, clusterSizeMax, clusterSpreadMin, clusterSpreadMax, clustersPerChunk)
	}

	// Otherwise, use single node spawning
	return m.spawnSingleNode(ctx, chunkX, chunkZ, template)
}

func (m *Manager) spawnSingleNode(ctx context.Context, chunkX, chunkZ int64, template db.NodeSpawnTemplate) error {
	// Find random position
	localX := int64(rand.Intn(ChunkSize))
	localZ := int64(rand.Intn(ChunkSize))

	// Check if position is occupied
	existing, err := m.queries.CheckNodePosition(ctx, db.CheckNodePositionParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
		LocalX: localX,
		LocalZ: localZ,
	})
	if err != nil {
		return fmt.Errorf("failed to check node position: %w", err)
	}

	if existing > 0 {
		return nil // Position occupied
	}

	return m.createNodeAtPosition(ctx, chunkX, chunkZ, localX, localZ, template)
}

func (m *Manager) spawnNodeCluster(ctx context.Context, chunkX, chunkZ int64, template db.NodeSpawnTemplate, clusterSizeMin, clusterSizeMax, clusterSpreadMin, clusterSpreadMax, clustersPerChunk int64) error {
	// Get occupied positions for this chunk (cached)
	occupiedPositions, err := m.getOccupiedPositions(ctx, chunkX, chunkZ)
	if err != nil {
		return fmt.Errorf("failed to get occupied positions: %w", err)
	}

	// Generate clusters for this template
	for i := int64(0); i < clustersPerChunk; i++ {
		// Find cluster center
		centerX := int64(rand.Intn(ChunkSize))
		centerZ := int64(rand.Intn(ChunkSize))

		// Determine cluster size
		clusterSize := clusterSizeMin
		if clusterSizeMax > clusterSizeMin {
			clusterSize = clusterSizeMin + int64(rand.Intn(int(clusterSizeMax-clusterSizeMin+1)))
		}

		log.Debug("spawning cluster", "chunk_x", chunkX, "chunk_z", chunkZ, "center_x", centerX, "center_z", centerZ, "cluster_size", clusterSize, "template_id", template.TemplateID)

		// Spawn nodes in cluster
		for j := int64(0); j < clusterSize; j++ {
			// Find position within cluster spread
			localX, localZ := m.findClusterPosition(centerX, centerZ, clusterSpreadMin, clusterSpreadMax)

			// Check if position is occupied using cached data
			// Position encoding: combine (localX, localZ) into single int64 key
			// localX << 8 shifts X coordinate to upper bits (multiply by 256)
			// | localZ adds Z coordinate to lower 8 bits
			// Since ChunkSize=16, coordinates range 0-15 (4 bits each)
			// This encoding supports up to 256x256 positions per chunk
			posKey := (localX << 8) | localZ
			if _, occupied := occupiedPositions[posKey]; occupied {
				continue // Position occupied, try next node
			}

			// Create the node
			err = m.createNodeAtPosition(ctx, chunkX, chunkZ, localX, localZ, template)
			if err != nil {
				log.Error("failed to create node in cluster", "error", err, "local_x", localX, "local_z", localZ)
				continue
			}

			// Update occupied positions cache to include the new node
			occupiedPositions[posKey] = struct{}{}
		}
	}

	return nil
}

func (m *Manager) findClusterPosition(centerX, centerZ, spreadMin, spreadMax int64) (int64, int64) {
	// Calculate spread distance
	spreadDistance := spreadMin
	if spreadMax > spreadMin {
		spreadDistance = spreadMin + int64(rand.Intn(int(spreadMax-spreadMin+1)))
	}

	// If no spread, return center
	if spreadDistance == 0 {
		return centerX, centerZ
	}

	// Generate random angle
	angle := rand.Float64() * 2.0 * 3.14159265359 // 2π

	// Calculate offset
	offsetX := int64(float64(spreadDistance) * math.Cos(angle))
	offsetZ := int64(float64(spreadDistance) * math.Sin(angle))

	// Calculate new position
	newX := centerX + offsetX
	newZ := centerZ + offsetZ

	// Clamp to chunk boundaries
	if newX < 0 {
		newX = 0
	}
	if newX >= ChunkSize {
		newX = ChunkSize - 1
	}
	if newZ < 0 {
		newZ = 0
	}
	if newZ >= ChunkSize {
		newZ = ChunkSize - 1
	}

	return newX, newZ
}

func (m *Manager) createNodeAtPosition(ctx context.Context, chunkX, chunkZ, localX, localZ int64, template db.NodeSpawnTemplate) error {
	// Generate yield
	yield := template.MinYield + int64(rand.Intn(int(template.MaxYield-template.MinYield+1)))

	nodeSubtype := int64(0)
	if template.NodeSubtype.Valid {
		nodeSubtype = template.NodeSubtype.Int64
	}

	regenRate := int64(0)
	if template.RegenerationRate.Valid {
		regenRate = template.RegenerationRate.Int64
	}

	// Calculate spawn behavior deterministically from position and seed
	determinedSpawnType := m.determineBehaviorFromSeed(chunkX, chunkZ, localX, localZ)

	_, err := m.queries.CreateNode(ctx, db.CreateNodeParams{
		ChunkX:           chunkX,
		ChunkZ:           chunkZ,
		LocalX:           localX,
		LocalZ:           localZ,
		NodeType:         template.NodeType,
		NodeSubtype:      sql.NullInt64{Int64: nodeSubtype, Valid: true},
		MaxYield:         yield,
		CurrentYield:     yield,
		RegenerationRate: sql.NullInt64{Int64: regenRate, Valid: true},
		SpawnType:        determinedSpawnType,
	})
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	log.Debug("spawned new node", "chunk_x", chunkX, "chunk_z", chunkZ, "local_x", localX, "local_z", localZ, "node_type", template.NodeType, "yield", yield)

	return nil
}

func (m *Manager) respawnNodes(ctx context.Context, chunkX, chunkZ int64) error {
	now := time.Now()

	nodesToRespawn, err := m.queries.GetNodesToRespawn(ctx, db.GetNodesToRespawnParams{
		ChunkX:       chunkX,
		ChunkZ:       chunkZ,
		RespawnTimer: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to get nodes to respawn: %w", err)
	}

	for _, node := range nodesToRespawn {
		err := m.queries.ReactivateNode(ctx, db.ReactivateNodeParams{
			CurrentYield: node.MaxYield,
			NodeID:       node.NodeID,
		})
		if err != nil {
			log.Error("failed to reactivate node", "error", err, "node_id", node.NodeID)
			continue
		}

		log.Debug("respawned node", "node_id", node.NodeID)
	}

	return nil
}



func (m *Manager) RegenerateResources(ctx context.Context) error {
	log.Debug("Starting resource regeneration")
	start := time.Now()

	err := m.queries.RegenerateNodeYield(ctx)
	if err != nil {
		log.Error("Failed to regenerate node yield", "error", err, "duration", time.Since(start))
		return fmt.Errorf("failed to regenerate node yield: %w", err)
	}

	log.Debug("Resource regeneration completed", "duration", time.Since(start))
	return nil
}



// Save persists any pending chunk manager state to the database
// This method ensures all chunk-related data is properly persisted
func (m *Manager) Save(ctx context.Context) error {
	log.Debug("Persisting chunk manager state")
	start := time.Now()

	// Force a database sync to ensure all transactions are committed
	if err := m.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database during chunk save: %w", err)
	}

	// Clear any stale cache entries to ensure fresh data on next load
	m.cacheMutex.Lock()
	for key, entry := range m.occupiedCache {
		if time.Now().After(entry.expires) {
			delete(m.occupiedCache, key)
		}
	}
	m.cacheMutex.Unlock()

	log.Debug("Chunk manager state persisted successfully", "duration", time.Since(start))
	return nil
}

// HarvestNode performs a single harvest action on a node with daily limits
// Returns detailed harvest results including loot, node state, and future-ready extensions
func (m *Manager) HarvestNode(ctx context.Context, harvestCtx HarvestContext) (*HarvestResult, error) {
	log.Debug("Harvesting node", "node_id", harvestCtx.NodeID, "player_id", harvestCtx.PlayerID)
	start := time.Now()

	// Check daily harvest limit
	dailyCount, err := m.queries.GetPlayerDailyHarvest(ctx, db.GetPlayerDailyHarvestParams{
		PlayerID: harvestCtx.PlayerID,
		NodeID:   harvestCtx.NodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check daily harvest limit: %w", err)
	}

	if dailyCount > 0 {
		return &HarvestResult{
			Success: false,
		}, fmt.Errorf("already harvested this node today")
	}

	// Begin database transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	txQueries := m.queries.WithTx(tx)

	// Get fresh node data to avoid race conditions
	dbNode, err := txQueries.GetNode(ctx, harvestCtx.NodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Check if node is active and harvestable
	if !dbNode.IsActive.Valid || dbNode.IsActive.Int64 != 1 {
		return &HarvestResult{
			Success: false,
		}, fmt.Errorf("node is not active")
	}

	if dbNode.CurrentYield <= 0 {
		return &HarvestResult{
			Success: false,
		}, fmt.Errorf("node is depleted")
	}

	// Calculate harvest amount - future: apply character stats and tool bonuses
	baseYield := int64(1) // Base harvest amount
	statBonus := int64(0) // Future: from character stats
	toolBonus := int64(0) // Future: from tool bonuses

	// Apply future bonuses (currently unused)
	if harvestCtx.CharacterStats != nil {
		statBonus = int64(harvestCtx.CharacterStats.MiningBonus)
	}
	if harvestCtx.ToolStats != nil {
		toolBonus = int64(harvestCtx.ToolStats.YieldMultiplier)
	}

	totalYield := baseYield + statBonus + toolBonus
	if totalYield > dbNode.CurrentYield {
		totalYield = dbNode.CurrentYield
	}

	// Update node yield
	newYield := dbNode.CurrentYield - totalYield
	err = txQueries.UpdateNodeYield(ctx, db.UpdateNodeYieldParams{
		CurrentYield: newYield,
		NodeID:       harvestCtx.NodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update node yield: %w", err)
	}

	// Determine if harvesting is finished
	finished := newYield <= 0

	// If node is depleted, set respawn timer
	if finished {
		log.Debug("Node depleted, setting respawn timer", "node_id", harvestCtx.NodeID, "node_type", dbNode.NodeType)
		nodeSubtype := int64(0)
		if dbNode.NodeSubtype.Valid {
			nodeSubtype = dbNode.NodeSubtype.Int64
		}

		respawnHours, err := txQueries.GetRespawnDelay(ctx, db.GetRespawnDelayParams{
			NodeType:    dbNode.NodeType,
			NodeSubtype: sql.NullInt64{Int64: nodeSubtype, Valid: true},
		})
		if err != nil {
			log.Error("failed to get respawn delay", "error", err, "node_id", harvestCtx.NodeID)
			respawnHours = sql.NullInt64{Int64: 24, Valid: true} // Default to 24 hours
		}

		hours := int64(24)
		if respawnHours.Valid {
			hours = respawnHours.Int64
		}

		respawnTime := time.Now().Add(time.Duration(hours) * time.Hour)
		log.Debug("Node respawn scheduled", "node_id", harvestCtx.NodeID, "respawn_hours", hours, "respawn_time", respawnTime)
		err = txQueries.DeactivateNode(ctx, db.DeactivateNodeParams{
			RespawnTimer: sql.NullTime{Time: respawnTime, Valid: true},
			NodeID:       harvestCtx.NodeID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to deactivate node: %w", err)
		}
	}

	// Log harvest
	err = txQueries.CreateHarvestLog(ctx, db.CreateHarvestLogParams{
		NodeID:          harvestCtx.NodeID,
		PlayerID:        harvestCtx.PlayerID,
		AmountHarvested: totalYield,
		NodeYieldBefore: dbNode.CurrentYield,
		NodeYieldAfter:  newYield,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create harvest log: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update player inventory and stats (outside of transaction to avoid deadlocks)
	if m.playerManager != nil {
		nodeSubtype := int64(0)
		if dbNode.NodeSubtype.Valid {
			nodeSubtype = dbNode.NodeSubtype.Int64
		}

		// Add resources to player inventory
		err = m.playerManager.AddToInventory(ctx, harvestCtx.PlayerID, dbNode.NodeType, nodeSubtype, totalYield)
		if err != nil {
			log.Error("Failed to add resources to player inventory", "error", err, "player_id", harvestCtx.PlayerID, "resource_type", dbNode.NodeType, "amount", totalYield)
			// Don't fail the harvest if inventory update fails
		}

		// Update player harvest stats
		statsUpdate := HarvestStatsUpdate{
			ResourceType:    dbNode.NodeType,
			AmountHarvested: totalYield,
			NodeID:          harvestCtx.NodeID,
			IsNewNode:       false, // TODO: Track if this is a new node for the player
		}
		err = m.playerManager.UpdateHarvestStats(ctx, harvestCtx.PlayerID, statsUpdate)
		if err != nil {
			log.Error("Failed to update player harvest stats", "error", err, "player_id", harvestCtx.PlayerID)
			// Don't fail the harvest if stats update fails
		}
	}

	// Generate primary loot
	nodeSubtype := int64(0)
	if dbNode.NodeSubtype.Valid {
		nodeSubtype = dbNode.NodeSubtype.Int64
	}

	primaryLoot := []LootItem{
		{
			ItemType:    dbNode.NodeType,
			ItemSubtype: nodeSubtype,
			Quantity:    totalYield,
			Quality:     1.0, // Future: variable quality
			Source:      "primary",
		},
	}

	// Future: Generate bonus loot based on stats, tools, etc.
	bonusLoot := []LootItem{} // Empty for now

	// Create node state response
	respawnTimer := (*time.Time)(nil)
	if finished {
		respawnTime := time.Now().Add(24 * time.Hour) // Default respawn time
		respawnTimer = &respawnTime
	}

	nodeState := NodeState{
		CurrentYield: newYield,
		IsActive:     !finished,
		RespawnTimer: respawnTimer,
		LastHarvest:  &[]time.Time{time.Now()}[0],
	}

	harvestDetails := HarvestDetails{
		BaseYield:  baseYield,
		StatBonus:  statBonus,
		ToolBonus:  toolBonus,
		TotalYield: totalYield,
		BonusRolls: 0,    // Future: bonus material rolls
		LuckFactor: 1.0,  // Future: luck calculations
	}

	result := &HarvestResult{
		Success:        true,
		PrimaryLoot:    primaryLoot,
		BonusLoot:      bonusLoot,
		NodeState:      nodeState,
		HarvestDetails: harvestDetails,
		// Future fields remain nil for now
	}

	log.Debug("Harvest completed", "node_id", harvestCtx.NodeID, "player_id", harvestCtx.PlayerID, "harvested", totalYield, "new_yield", newYield, "finished", finished, "duration", time.Since(start))
	return result, nil
}

// HarvestNodeLegacy maintains backward compatibility with the old signature
// This can be removed once all callers are updated
func (m *Manager) HarvestNodeLegacy(ctx context.Context, node *ResourceNode, playerID int64) ([]int64, bool, error) {
	harvestCtx := HarvestContext{
		PlayerID: playerID,
		NodeID:   node.NodeID,
	}

	result, err := m.HarvestNode(ctx, harvestCtx)
	if err != nil {
		return nil, false, err
	}

	if !result.Success {
		return nil, false, fmt.Errorf("harvest failed")
	}

	// Convert new format back to legacy format
	loot := make([]int64, 0, len(result.PrimaryLoot))
	for _, item := range result.PrimaryLoot {
		loot = append(loot, item.ItemType)
	}

	finished := !result.NodeState.IsActive
	return loot, finished, nil
}
