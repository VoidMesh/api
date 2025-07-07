package chunk

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/VoidMesh/api/internal/db"
	"github.com/charmbracelet/log"
)

type Manager struct {
	db      *sql.DB
	queries *db.Queries
}

func NewManager(database *sql.DB) *Manager {
	return &Manager{
		db:      database,
		queries: db.New(database),
	}
}

func (m *Manager) LoadChunk(ctx context.Context, chunkX, chunkZ int64) (*ChunkResponse, error) {
	// Ensure chunk exists
	err := m.queries.CreateChunk(ctx, db.CreateChunkParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
	})
	if err != nil {
		log.Error("failed to create chunk", "error", err, "chunk_x", chunkX, "chunk_z", chunkZ)
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}

	// Generate nodes if needed
	err = m.generateNodes(ctx, chunkX, chunkZ)
	if err != nil {
		log.Error("failed to generate nodes", "error", err, "chunk_x", chunkX, "chunk_z", chunkZ)
		return nil, fmt.Errorf("failed to generate nodes: %w", err)
	}

	// Load active nodes
	dbNodes, err := m.queries.GetChunkNodes(ctx, db.GetChunkNodesParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
	})
	if err != nil {
		log.Error("failed to get chunk nodes", "error", err, "chunk_x", chunkX, "chunk_z", chunkZ)
		return nil, fmt.Errorf("failed to get chunk nodes: %w", err)
	}

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

	return &ChunkResponse{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
		Nodes:  nodes,
	}, nil
}

func (m *Manager) generateNodes(ctx context.Context, chunkX, chunkZ int64) error {
	log.Debug("starting node generation", "chunk_x", chunkX, "chunk_z", chunkZ)

	// Check if we need to spawn daily nodes
	err := m.spawnDailyNodes(ctx, chunkX, chunkZ)
	if err != nil {
		return fmt.Errorf("failed to spawn daily nodes: %w", err)
	}

	// Check if we need to spawn random nodes
	err = m.spawnRandomNodes(ctx, chunkX, chunkZ)
	if err != nil {
		return fmt.Errorf("failed to spawn random nodes: %w", err)
	}

	// Respawn depleted nodes whose timer has expired
	err = m.respawnNodes(ctx, chunkX, chunkZ)
	if err != nil {
		return fmt.Errorf("failed to respawn nodes: %w", err)
	}

	log.Debug("completed node generation", "chunk_x", chunkX, "chunk_z", chunkZ)
	return nil
}

func (m *Manager) spawnDailyNodes(ctx context.Context, chunkX, chunkZ int64) error {
	// Get today's date as a string for the SQL DATE() comparison
	todayStr := time.Now().Format("2006-01-02")

	log.Debug("checking daily node spawn", "chunk_x", chunkX, "chunk_z", chunkZ, "date", todayStr)

	count, err := m.queries.GetDailyNodeCount(ctx, db.GetDailyNodeCountParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
		Date:   time.Now(), // SQL will extract date part
	})
	if err != nil {
		return fmt.Errorf("failed to get daily node count: %w", err)
	}

	log.Debug("daily node count check", "chunk_x", chunkX, "chunk_z", chunkZ, "existing_count", count)

	if count > 0 {
		log.Debug("skipping daily spawn - already spawned today", "chunk_x", chunkX, "chunk_z", chunkZ)
		return nil // Already spawned today
	}

	// Spawn daily nodes
	templates, err := m.queries.GetSpawnTemplates(ctx, StaticDaily)
	if err != nil {
		return fmt.Errorf("failed to get spawn templates: %w", err)
	}

	log.Debug("spawning daily nodes", "chunk_x", chunkX, "chunk_z", chunkZ, "template_count", len(templates))

	for _, template := range templates {
		spawnWeight := int64(1)
		if template.SpawnWeight.Valid {
			spawnWeight = template.SpawnWeight.Int64
		}

		log.Debug("spawning daily nodes from template", "chunk_x", chunkX, "chunk_z", chunkZ, "template_id", template.TemplateID, "node_type", template.NodeType, "spawn_weight", spawnWeight)

		// For daily spawning, spawn weight determines how many times we attempt to spawn this template
		for i := int64(0); i < spawnWeight; i++ {
			err := m.spawnNode(ctx, chunkX, chunkZ, template)
			if err != nil {
				log.Error("failed to spawn daily node cluster", "error", err, "template_id", template.TemplateID, "attempt", i)
				// Continue with other nodes
			} else {
				log.Debug("successfully spawned daily node cluster", "chunk_x", chunkX, "chunk_z", chunkZ, "template_id", template.TemplateID, "node_type", template.NodeType, "attempt", i)
			}
		}
	}

	return nil
}

func (m *Manager) spawnRandomNodes(ctx context.Context, chunkX, chunkZ int64) error {
	activeCount, err := m.queries.GetRandomNodeCount(ctx, db.GetRandomNodeCountParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
	})
	if err != nil {
		return fmt.Errorf("failed to get random node count: %w", err)
	}

	log.Debug("random node count check", "chunk_x", chunkX, "chunk_z", chunkZ, "active_count", activeCount)

	// Maintain 2-5 random nodes per chunk - only spawn if below minimum
	minRandomNodes := int64(2)

	if activeCount < minRandomNodes {
		templates, err := m.queries.GetSpawnTemplates(ctx, RandomSpawn)
		if err != nil {
			return fmt.Errorf("failed to get spawn templates: %w", err)
		}

		// Spawn enough nodes to reach minimum
		nodesToSpawn := minRandomNodes - activeCount
		log.Debug("spawning random nodes", "chunk_x", chunkX, "chunk_z", chunkZ, "current_count", activeCount, "min", minRandomNodes, "to_spawn", nodesToSpawn)

		if len(templates) > 0 {
			// Group templates by cluster requirements to avoid spawning too many clusters
			for _, template := range templates {
				// Check if we should spawn this template based on spawn weight
				spawnWeight := int64(1)
				if template.SpawnWeight.Valid {
					spawnWeight = template.SpawnWeight.Int64
				}

				// Simple spawn probability based on weight
				if rand.Int63n(100) < spawnWeight*10 {
					err := m.spawnNode(ctx, chunkX, chunkZ, template)
					if err != nil {
						log.Error("failed to spawn random node cluster", "error", err, "template_id", template.TemplateID, "node_type", template.NodeType)
					} else {
						log.Debug("successfully spawned random node cluster", "chunk_x", chunkX, "chunk_z", chunkZ, "template_id", template.TemplateID, "node_type", template.NodeType)
					}
				}
			}
		}
	} else {
		log.Debug("skipping random spawn - minimum nodes exist", "chunk_x", chunkX, "chunk_z", chunkZ, "active_count", activeCount, "min", minRandomNodes)
	}

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
		SpawnType:        template.SpawnType,
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
	err := m.queries.RegenerateNodeYield(ctx)
	if err != nil {
		return fmt.Errorf("failed to regenerate node yield: %w", err)
	}
	return nil
}

func (m *Manager) CleanupExpiredSessions(ctx context.Context) error {
	cutoff := time.Now().Add(-SessionTimeout * time.Minute)
	err := m.queries.CleanupExpiredSessions(ctx, sql.NullTime{Time: cutoff, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return nil
}

func (m *Manager) GetPlayerSessions(ctx context.Context, playerID int64) ([]HarvestSession, error) {
	dbSessions, err := m.queries.GetPlayerSessions(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player sessions: %w", err)
	}

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

	return sessions, nil
}
