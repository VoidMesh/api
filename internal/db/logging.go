package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/charmbracelet/log"
)

// LoggingQueries wraps the generated Queries struct to add debug logging
type LoggingQueries struct {
	*Queries
}

// NewLoggingQueries creates a new LoggingQueries instance
func NewLoggingQueries(db DBTX) *LoggingQueries {
	return &LoggingQueries{
		Queries: New(db),
	}
}

// WithTx creates a new LoggingQueries with a transaction
func (lq *LoggingQueries) WithTx(tx *sql.Tx) *LoggingQueries {
	return &LoggingQueries{
		Queries: lq.Queries.WithTx(tx),
	}
}

// Helper function to log query execution
func (lq *LoggingQueries) logQuery(ctx context.Context, queryName string, start time.Time, err error, args ...interface{}) {
	duration := time.Since(start)
	
	if err != nil {
		log.Debug("Database query failed", 
			"query", queryName, 
			"duration", duration, 
			"error", err,
			"args", args,
		)
	} else {
		log.Debug("Database query executed", 
			"query", queryName, 
			"duration", duration,
			"args", args,
		)
	}
}

// CreateChunk with logging
func (lq *LoggingQueries) CreateChunk(ctx context.Context, arg CreateChunkParams) error {
	start := time.Now()
	log.Debug("Executing CreateChunk", "chunk_x", arg.ChunkX, "chunk_z", arg.ChunkZ)
	
	err := lq.Queries.CreateChunk(ctx, arg)
	lq.logQuery(ctx, "CreateChunk", start, err, arg)
	return err
}

// GetChunkNodes with logging
func (lq *LoggingQueries) GetChunkNodes(ctx context.Context, arg GetChunkNodesParams) ([]ResourceNode, error) {
	start := time.Now()
	log.Debug("Executing GetChunkNodes", "chunk_x", arg.ChunkX, "chunk_z", arg.ChunkZ)
	
	result, err := lq.Queries.GetChunkNodes(ctx, arg)
	lq.logQuery(ctx, "GetChunkNodes", start, err, arg)
	
	if err == nil {
		log.Debug("GetChunkNodes result", "node_count", len(result), "chunk_x", arg.ChunkX, "chunk_z", arg.ChunkZ)
	}
	
	return result, err
}

// CreateNode with logging
func (lq *LoggingQueries) CreateNode(ctx context.Context, arg CreateNodeParams) (int64, error) {
	start := time.Now()
	log.Debug("Executing CreateNode", 
		"chunk_x", arg.ChunkX, 
		"chunk_z", arg.ChunkZ, 
		"local_x", arg.LocalX, 
		"local_z", arg.LocalZ, 
		"node_type", arg.NodeType,
		"max_yield", arg.MaxYield,
		"spawn_type", arg.SpawnType,
	)
	
	result, err := lq.Queries.CreateNode(ctx, arg)
	lq.logQuery(ctx, "CreateNode", start, err, arg)
	
	if err == nil {
		log.Debug("CreateNode result", "node_id", result)
	}
	
	return result, err
}

// GetSpawnTemplates with logging
func (lq *LoggingQueries) GetSpawnTemplates(ctx context.Context) ([]NodeSpawnTemplate, error) {
	start := time.Now()
	log.Debug("Executing GetSpawnTemplates")
	
	result, err := lq.Queries.GetSpawnTemplates(ctx)
	lq.logQuery(ctx, "GetSpawnTemplates", start, err)
	
	if err == nil {
		log.Debug("GetSpawnTemplates result", "template_count", len(result))
	}
	
	return result, err
}

// GetChunkNodeCount with logging
func (lq *LoggingQueries) GetChunkNodeCount(ctx context.Context, arg GetChunkNodeCountParams) (int64, error) {
	start := time.Now()
	log.Debug("Executing GetChunkNodeCount", 
		"chunk_x", arg.ChunkX, 
		"chunk_z", arg.ChunkZ, 
		"node_type", arg.NodeType,
	)
	
	result, err := lq.Queries.GetChunkNodeCount(ctx, arg)
	lq.logQuery(ctx, "GetChunkNodeCount", start, err, arg)
	
	if err == nil {
		log.Debug("GetChunkNodeCount result", "count", result)
	}
	
	return result, err
}

// CheckNodePosition with logging
func (lq *LoggingQueries) CheckNodePosition(ctx context.Context, arg CheckNodePositionParams) (int64, error) {
	start := time.Now()
	log.Debug("Executing CheckNodePosition", 
		"chunk_x", arg.ChunkX, 
		"chunk_z", arg.ChunkZ, 
		"local_x", arg.LocalX, 
		"local_z", arg.LocalZ,
	)
	
	result, err := lq.Queries.CheckNodePosition(ctx, arg)
	lq.logQuery(ctx, "CheckNodePosition", start, err, arg)
	
	if err == nil {
		log.Debug("CheckNodePosition result", "count", result)
	}
	
	return result, err
}

// GetNode with logging
func (lq *LoggingQueries) GetNode(ctx context.Context, nodeID int64) (ResourceNode, error) {
	start := time.Now()
	log.Debug("Executing GetNode", "node_id", nodeID)
	
	result, err := lq.Queries.GetNode(ctx, nodeID)
	lq.logQuery(ctx, "GetNode", start, err, nodeID)
	
	if err == nil {
		log.Debug("GetNode result", 
			"node_id", result.NodeID, 
			"node_type", result.NodeType, 
			"current_yield", result.CurrentYield,
			"is_active", result.IsActive,
		)
	}
	
	return result, err
}

// CreateHarvestSession with logging
func (lq *LoggingQueries) CreateHarvestSession(ctx context.Context, arg CreateHarvestSessionParams) (HarvestSession, error) {
	start := time.Now()
	log.Debug("Executing CreateHarvestSession", "node_id", arg.NodeID, "player_id", arg.PlayerID)
	
	result, err := lq.Queries.CreateHarvestSession(ctx, arg)
	lq.logQuery(ctx, "CreateHarvestSession", start, err, arg)
	
	if err == nil {
		log.Debug("CreateHarvestSession result", "session_id", result.SessionID)
	}
	
	return result, err
}

// GetHarvestSession with logging
func (lq *LoggingQueries) GetHarvestSession(ctx context.Context, sessionID int64) (HarvestSession, error) {
	start := time.Now()
	log.Debug("Executing GetHarvestSession", "session_id", sessionID)
	
	result, err := lq.Queries.GetHarvestSession(ctx, sessionID)
	lq.logQuery(ctx, "GetHarvestSession", start, err, sessionID)
	
	if err == nil {
		log.Debug("GetHarvestSession result", 
			"session_id", result.SessionID, 
			"node_id", result.NodeID, 
			"player_id", result.PlayerID,
		)
	}
	
	return result, err
}

// UpdateNodeYield with logging
func (lq *LoggingQueries) UpdateNodeYield(ctx context.Context, arg UpdateNodeYieldParams) error {
	start := time.Now()
	log.Debug("Executing UpdateNodeYield", "node_id", arg.NodeID, "current_yield", arg.CurrentYield)
	
	err := lq.Queries.UpdateNodeYield(ctx, arg)
	lq.logQuery(ctx, "UpdateNodeYield", start, err, arg)
	
	return err
}

// RegenerateNodeYield with logging
func (lq *LoggingQueries) RegenerateNodeYield(ctx context.Context) error {
	start := time.Now()
	log.Debug("Executing RegenerateNodeYield")
	
	err := lq.Queries.RegenerateNodeYield(ctx)
	lq.logQuery(ctx, "RegenerateNodeYield", start, err)
	
	return err
}

// CleanupExpiredSessions with logging
func (lq *LoggingQueries) CleanupExpiredSessions(ctx context.Context, cutoff sql.NullTime) error {
	start := time.Now()
	log.Debug("Executing CleanupExpiredSessions", "cutoff", cutoff.Time)
	
	err := lq.Queries.CleanupExpiredSessions(ctx, cutoff)
	lq.logQuery(ctx, "CleanupExpiredSessions", start, err, cutoff)
	
	return err
}

// GetPlayerSessions with logging
func (lq *LoggingQueries) GetPlayerSessions(ctx context.Context, playerID int64) ([]HarvestSession, error) {
	start := time.Now()
	log.Debug("Executing GetPlayerSessions", "player_id", playerID)
	
	result, err := lq.Queries.GetPlayerSessions(ctx, playerID)
	lq.logQuery(ctx, "GetPlayerSessions", start, err, playerID)
	
	if err == nil {
		log.Debug("GetPlayerSessions result", "session_count", len(result), "player_id", playerID)
	}
	
	return result, err
}

// Add other missing methods that are used in the chunk manager

// GetWorldConfig with logging
func (lq *LoggingQueries) GetWorldConfig(ctx context.Context, configKey string) (string, error) {
	start := time.Now()
	log.Debug("Executing GetWorldConfig", "config_key", configKey)
	
	result, err := lq.Queries.GetWorldConfig(ctx, configKey)
	lq.logQuery(ctx, "GetWorldConfig", start, err, configKey)
	
	return result, err
}

// SetWorldConfig with logging
func (lq *LoggingQueries) SetWorldConfig(ctx context.Context, arg SetWorldConfigParams) error {
	start := time.Now()
	log.Debug("Executing SetWorldConfig", "config_key", arg.ConfigKey, "config_value", arg.ConfigValue)
	
	err := lq.Queries.SetWorldConfig(ctx, arg)
	lq.logQuery(ctx, "SetWorldConfig", start, err, arg)
	
	return err
}

// GetChunkOccupiedPositions with logging
func (lq *LoggingQueries) GetChunkOccupiedPositions(ctx context.Context, arg GetChunkOccupiedPositionsParams) ([]GetChunkOccupiedPositionsRow, error) {
	start := time.Now()
	log.Debug("Executing GetChunkOccupiedPositions", "chunk_x", arg.ChunkX, "chunk_z", arg.ChunkZ)
	
	result, err := lq.Queries.GetChunkOccupiedPositions(ctx, arg)
	lq.logQuery(ctx, "GetChunkOccupiedPositions", start, err, arg)
	
	if err == nil {
		log.Debug("GetChunkOccupiedPositions result", "position_count", len(result))
	}
	
	return result, err
}

// GetNodesToRespawn with logging
func (lq *LoggingQueries) GetNodesToRespawn(ctx context.Context, arg GetNodesToRespawnParams) ([]GetNodesToRespawnRow, error) {
	start := time.Now()
	log.Debug("Executing GetNodesToRespawn", "chunk_x", arg.ChunkX, "chunk_z", arg.ChunkZ)
	
	result, err := lq.Queries.GetNodesToRespawn(ctx, arg)
	lq.logQuery(ctx, "GetNodesToRespawn", start, err, arg)
	
	if err == nil {
		log.Debug("GetNodesToRespawn result", "node_count", len(result))
	}
	
	return result, err
}

// ReactivateNode with logging
func (lq *LoggingQueries) ReactivateNode(ctx context.Context, arg ReactivateNodeParams) error {
	start := time.Now()
	log.Debug("Executing ReactivateNode", "node_id", arg.NodeID, "current_yield", arg.CurrentYield)
	
	err := lq.Queries.ReactivateNode(ctx, arg)
	lq.logQuery(ctx, "ReactivateNode", start, err, arg)
	
	return err
}

// GetPlayerActiveSession with logging
func (lq *LoggingQueries) GetPlayerActiveSession(ctx context.Context, arg GetPlayerActiveSessionParams) (HarvestSession, error) {
	start := time.Now()
	log.Debug("Executing GetPlayerActiveSession", "player_id", arg.PlayerID)
	
	result, err := lq.Queries.GetPlayerActiveSession(ctx, arg)
	lq.logQuery(ctx, "GetPlayerActiveSession", start, err, arg)
	
	return result, err
}

// GetRespawnDelay with logging
func (lq *LoggingQueries) GetRespawnDelay(ctx context.Context, arg GetRespawnDelayParams) (sql.NullInt64, error) {
	start := time.Now()
	log.Debug("Executing GetRespawnDelay", "node_type", arg.NodeType)
	
	result, err := lq.Queries.GetRespawnDelay(ctx, arg)
	lq.logQuery(ctx, "GetRespawnDelay", start, err, arg)
	
	if err == nil {
		log.Debug("GetRespawnDelay result", "delay_hours", result)
	}
	
	return result, err
}

// DeactivateNode with logging
func (lq *LoggingQueries) DeactivateNode(ctx context.Context, arg DeactivateNodeParams) error {
	start := time.Now()
	log.Debug("Executing DeactivateNode", "node_id", arg.NodeID)
	
	err := lq.Queries.DeactivateNode(ctx, arg)
	lq.logQuery(ctx, "DeactivateNode", start, err, arg)
	
	return err
}

// UpdateSessionActivity with logging
func (lq *LoggingQueries) UpdateSessionActivity(ctx context.Context, arg UpdateSessionActivityParams) error {
	start := time.Now()
	log.Debug("Executing UpdateSessionActivity", "session_id", arg.SessionID, "resources_gathered", arg.ResourcesGathered)
	
	err := lq.Queries.UpdateSessionActivity(ctx, arg)
	lq.logQuery(ctx, "UpdateSessionActivity", start, err, arg)
	
	return err
}

// CreateHarvestLog with logging
func (lq *LoggingQueries) CreateHarvestLog(ctx context.Context, arg CreateHarvestLogParams) error {
	start := time.Now()
	log.Debug("Executing CreateHarvestLog", 
		"node_id", arg.NodeID, 
		"player_id", arg.PlayerID, 
		"amount_harvested", arg.AmountHarvested,
		"node_yield_before", arg.NodeYieldBefore,
		"node_yield_after", arg.NodeYieldAfter,
	)
	
	err := lq.Queries.CreateHarvestLog(ctx, arg)
	lq.logQuery(ctx, "CreateHarvestLog", start, err, arg)
	
	return err
}