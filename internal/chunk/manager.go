package chunk

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/VoidMesh/api/internal/db"
	"github.com/aquilax/go-perlin"
	"github.com/charmbracelet/log"
)

type Manager struct {
	db        *sql.DB
	queries   *db.Queries
	worldSeed int64
	noiseGens map[int64]*perlin.Perlin
}

func NewManager(database *sql.DB) *Manager {
	m := &Manager{
		db:        database,
		queries:   db.New(database),
		noiseGens: make(map[int64]*perlin.Perlin),
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

			// Check if position is occupied
			existing, err := m.queries.CheckNodePosition(ctx, db.CheckNodePositionParams{
				ChunkX: chunkX,
				ChunkZ: chunkZ,
				LocalX: localX,
				LocalZ: localZ,
			})
			if err != nil {
				log.Error("failed to check node position in cluster", "error", err, "local_x", localX, "local_z", localZ)
				continue
			}

			if existing > 0 {
				continue // Position occupied, try next node
			}

			// Create the node
			err = m.createNodeAtPosition(ctx, chunkX, chunkZ, localX, localZ, template)
			if err != nil {
				log.Error("failed to create node in cluster", "error", err, "local_x", localX, "local_z", localZ)
				continue
			}
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

func (m *Manager) StartHarvest(ctx context.Context, nodeID, playerID int64) (*HarvestSession, error) {
	log.Debug("Starting harvest session", "node_id", nodeID, "player_id", playerID)
	start := time.Now()

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	txQueries := m.queries.WithTx(tx)

	// Check if node exists and is active
	node, err := txQueries.GetNode(ctx, nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("node not found")
		}
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if !node.IsActive.Valid || node.IsActive.Int64 != 1 {
		return nil, fmt.Errorf("node is not active")
	}

	if node.CurrentYield <= 0 {
		return nil, fmt.Errorf("node is depleted")
	}

	// Check if player already has an active session
	cutoff := time.Now().Add(-SessionTimeout * time.Minute)
	_, err = txQueries.GetPlayerActiveSession(ctx, db.GetPlayerActiveSessionParams{
		PlayerID:     playerID,
		LastActivity: sql.NullTime{Time: cutoff, Valid: true},
	})
	if err == nil {
		return nil, fmt.Errorf("player already has active harvest session")
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check active session: %w", err)
	}

	// Create harvest session
	session, err := txQueries.CreateHarvestSession(ctx, db.CreateHarvestSessionParams{
		NodeID:   nodeID,
		PlayerID: playerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create harvest session: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug("Harvest session created successfully", "node_id", nodeID, "player_id", playerID, "session_id", session.SessionID, "duration", time.Since(start))
	startedAt := time.Now()
	if session.StartedAt.Valid {
		startedAt = session.StartedAt.Time
	}

	lastActivity := time.Now()
	if session.LastActivity.Valid {
		lastActivity = session.LastActivity.Time
	}

	resourcesGathered := int64(0)
	if session.ResourcesGathered.Valid {
		resourcesGathered = session.ResourcesGathered.Int64
	}

	return &HarvestSession{
		SessionID:         session.SessionID,
		NodeID:            session.NodeID,
		PlayerID:          session.PlayerID,
		StartedAt:         startedAt,
		LastActivity:      lastActivity,
		ResourcesGathered: resourcesGathered,
	}, nil
}

func (m *Manager) HarvestResource(ctx context.Context, sessionID int64, harvestAmount int64) (*HarvestResponse, error) {
	log.Debug("Processing harvest request", "session_id", sessionID, "harvest_amount", harvestAmount)
	start := time.Now()

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	txQueries := m.queries.WithTx(tx)

	// Validate session
	session, err := txQueries.GetHarvestSession(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	lastActivity := time.Now()
	if session.LastActivity.Valid {
		lastActivity = session.LastActivity.Time
	}

	if time.Since(lastActivity) > SessionTimeout*time.Minute {
		return nil, fmt.Errorf("session expired")
	}

	// Get node info
	node, err := txQueries.GetNode(ctx, session.NodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Calculate actual harvest amount
	actualHarvest := harvestAmount
	if actualHarvest > node.CurrentYield {
		actualHarvest = node.CurrentYield
	}

	log.Debug("Harvest amount calculated", "session_id", sessionID, "requested", harvestAmount, "actual", actualHarvest, "node_yield", node.CurrentYield)

	if actualHarvest <= 0 {
		return nil, fmt.Errorf("node is depleted")
	}

	// Update node yield
	newYield := node.CurrentYield - actualHarvest
	err = txQueries.UpdateNodeYield(ctx, db.UpdateNodeYieldParams{
		CurrentYield: newYield,
		NodeID:       session.NodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update node yield: %w", err)
	}

	// If node is depleted, set respawn timer
	if newYield <= 0 {
		log.Debug("Node depleted, setting respawn timer", "node_id", session.NodeID, "node_type", node.NodeType)
		nodeSubtype := int64(0)
		if node.NodeSubtype.Valid {
			nodeSubtype = node.NodeSubtype.Int64
		}

		respawnHours, err := txQueries.GetRespawnDelay(ctx, db.GetRespawnDelayParams{
			NodeType:    node.NodeType,
			NodeSubtype: sql.NullInt64{Int64: nodeSubtype, Valid: true},
		})
		if err != nil {
			log.Error("failed to get respawn delay", "error", err, "node_id", session.NodeID)
			respawnHours = sql.NullInt64{Int64: 24, Valid: true} // Default to 24 hours
		}

		hours := int64(24)
		if respawnHours.Valid {
			hours = respawnHours.Int64
		}

		respawnTime := time.Now().Add(time.Duration(hours) * time.Hour)
		log.Debug("Node respawn scheduled", "node_id", session.NodeID, "respawn_hours", hours, "respawn_time", respawnTime)
		err = txQueries.DeactivateNode(ctx, db.DeactivateNodeParams{
			RespawnTimer: sql.NullTime{Time: respawnTime, Valid: true},
			NodeID:       session.NodeID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to deactivate node: %w", err)
		}
	}

	// Update session
	err = txQueries.UpdateSessionActivity(ctx, db.UpdateSessionActivityParams{
		ResourcesGathered: sql.NullInt64{Int64: actualHarvest, Valid: true},
		SessionID:         sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Log harvest
	err = txQueries.CreateHarvestLog(ctx, db.CreateHarvestLogParams{
		NodeID:          session.NodeID,
		PlayerID:        session.PlayerID,
		AmountHarvested: actualHarvest,
		NodeYieldBefore: node.CurrentYield,
		NodeYieldAfter:  newYield,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create harvest log: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug("Harvest transaction completed", "session_id", sessionID, "harvested", actualHarvest, "new_yield", newYield, "duration", time.Since(start))
	currentGathered := int64(0)
	if session.ResourcesGathered.Valid {
		currentGathered = session.ResourcesGathered.Int64
	}

	return &HarvestResponse{
		Success:           true,
		AmountHarvested:   actualHarvest,
		NodeYieldAfter:    newYield,
		ResourcesGathered: currentGathered + actualHarvest,
	}, nil
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

func (m *Manager) CleanupExpiredSessions(ctx context.Context) error {
	log.Debug("Starting session cleanup")
	start := time.Now()
	cutoff := time.Now().Add(-SessionTimeout * time.Minute)
	log.Debug("Session cleanup cutoff calculated", "cutoff", cutoff, "timeout_minutes", SessionTimeout)

	err := m.queries.CleanupExpiredSessions(ctx, sql.NullTime{Time: cutoff, Valid: true})
	if err != nil {
		log.Error("Failed to cleanup expired sessions", "error", err, "duration", time.Since(start))
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	log.Debug("Session cleanup completed", "duration", time.Since(start))
	return nil
}

func (m *Manager) GetPlayerSessions(ctx context.Context, playerID int64) ([]HarvestSession, error) {
	log.Debug("Getting player sessions", "player_id", playerID)
	start := time.Now()

	dbSessions, err := m.queries.GetPlayerSessions(ctx, playerID)
	if err != nil {
		log.Error("Failed to get player sessions", "error", err, "player_id", playerID, "duration", time.Since(start))
		return nil, fmt.Errorf("failed to get player sessions: %w", err)
	}

	log.Debug("Retrieved player sessions", "player_id", playerID, "session_count", len(dbSessions), "duration", time.Since(start))

	sessions := make([]HarvestSession, len(dbSessions))
	for i, session := range dbSessions {
		startedAt := time.Now()
		if session.StartedAt.Valid {
			startedAt = session.StartedAt.Time
		}

		lastActivity := time.Now()
		if session.LastActivity.Valid {
			lastActivity = session.LastActivity.Time
		}

		resourcesGathered := int64(0)
		if session.ResourcesGathered.Valid {
			resourcesGathered = session.ResourcesGathered.Int64
		}

		sessions[i] = HarvestSession{
			SessionID:         session.SessionID,
			NodeID:            session.NodeID,
			PlayerID:          session.PlayerID,
			StartedAt:         startedAt,
			LastActivity:      lastActivity,
			ResourcesGathered: resourcesGathered,
		}
	}

	log.Debug("Player sessions processed", "player_id", playerID, "total_sessions", len(sessions))
	return sessions, nil
}
