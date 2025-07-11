package testutils

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/db"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// MockPlayerManager implements chunk.PlayerManager for testing
type MockPlayerManager struct {
	inventory map[int64]map[int64]int64 // playerID -> resourceType -> quantity
	stats     map[int64][]chunk.HarvestStatsUpdate
}

func NewMockPlayerManager() *MockPlayerManager {
	return &MockPlayerManager{
		inventory: make(map[int64]map[int64]int64),
		stats:     make(map[int64][]chunk.HarvestStatsUpdate),
	}
}

func (m *MockPlayerManager) AddToInventory(ctx context.Context, playerID int64, resourceType, resourceSubtype, quantity int64) error {
	if m.inventory[playerID] == nil {
		m.inventory[playerID] = make(map[int64]int64)
	}
	m.inventory[playerID][resourceType] += quantity
	return nil
}

func (m *MockPlayerManager) UpdateHarvestStats(ctx context.Context, playerID int64, update chunk.HarvestStatsUpdate) error {
	m.stats[playerID] = append(m.stats[playerID], update)
	return nil
}

func (m *MockPlayerManager) GetInventory(playerID int64) map[int64]int64 {
	return m.inventory[playerID]
}

func (m *MockPlayerManager) GetStats(playerID int64) []chunk.HarvestStatsUpdate {
	return m.stats[playerID]
}

// insertDefaultSpawnTemplates inserts default spawn template data for testing
func insertDefaultSpawnTemplates(db *sql.DB) error {
	ctx := context.Background()

	// Insert default spawn templates for each resource type
	templates := []struct {
		nodeType     int64
		nodeSubtype  int64
		respawnHours int64
	}{
		{chunk.IronOre, 0, 24},
		{chunk.GoldOre, 0, 48},
		{chunk.Wood, 0, 12},
		{chunk.Stone, 0, 6},
	}

	for _, template := range templates {
		_, err := db.ExecContext(ctx, `
			INSERT OR REPLACE INTO node_spawn_templates (
				node_type, node_subtype, spawn_type, min_yield, max_yield, 
				regeneration_rate, respawn_delay_hours, spawn_weight
			) VALUES (?, ?, 0, 1, 10, 0, ?, 1.0)
		`, template.nodeType, template.nodeSubtype, template.respawnHours)
		if err != nil {
			return err
		}
	}

	return nil
}

// TestWorld represents a test world with database and chunk manager
type TestWorld struct {
	DB            *sql.DB
	ChunkManager  *chunk.Manager
	PlayerManager *MockPlayerManager
	tempDBPath    string
}

// CreateTestWorld creates a new test world with a temporary database
func CreateTestWorld(t *testing.T) *TestWorld {
	// Create temporary database for testing
	tempDB := "test_" + t.Name() + ".db"

	// Open database
	db, err := sql.Open("sqlite3", tempDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Run migrations
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://../../internal/db/migrations", "sqlite3", driver)
	if err != nil {
		t.Fatalf("Failed to create migrate instance: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create mock player manager
	playerManager := NewMockPlayerManager()

	// Create chunk manager
	chunkManager := chunk.NewManager(db, playerManager)

	// Insert default spawn template data
	err = insertDefaultSpawnTemplates(db)
	if err != nil {
		t.Fatalf("Failed to insert default spawn templates: %v", err)
	}

	return &TestWorld{
		DB:            db,
		ChunkManager:  chunkManager,
		PlayerManager: playerManager,
		tempDBPath:    tempDB,
	}
}

// Cleanup closes the database and removes the temporary file
func (tw *TestWorld) Cleanup() {
	tw.DB.Close()
	os.Remove(tw.tempDBPath)
}

// CreateTestNode creates a test node in the database
func (tw *TestWorld) CreateTestNode(t *testing.T, chunkX, chunkZ, localX, localZ, nodeType, yield int64) *chunk.ResourceNode {
	ctx := context.Background()

	// First ensure chunk exists
	_, err := tw.ChunkManager.LoadChunk(ctx, chunkX, chunkZ)
	if err != nil {
		t.Fatalf("Failed to load chunk: %v", err)
	}

	// Create node directly in database
	queries := db.NewLoggingQueries(tw.DB)
	_, err = queries.CreateNode(ctx, db.CreateNodeParams{
		ChunkX:           chunkX,
		ChunkZ:           chunkZ,
		LocalX:           localX,
		LocalZ:           localZ,
		NodeType:         nodeType,
		NodeSubtype:      sql.NullInt64{Int64: 0, Valid: true},
		MaxYield:         yield,
		CurrentYield:     yield,
		RegenerationRate: sql.NullInt64{Int64: 0, Valid: true},
		SpawnType:        0,
	})
	if err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	// Get the created node
	nodes, err := queries.GetChunkNodes(ctx, db.GetChunkNodesParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
	})
	if err != nil {
		t.Fatalf("Failed to get chunk nodes: %v", err)
	}

	// Find our node
	for _, node := range nodes {
		if node.LocalX == localX && node.LocalZ == localZ && node.NodeType == nodeType {
			return &chunk.ResourceNode{
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
		}
	}

	t.Fatalf("Failed to find created test node")
	return nil
}

// CreateInactiveTestNode creates a test node that is inactive/depleted
func (tw *TestWorld) CreateInactiveTestNode(t *testing.T, chunkX, chunkZ, localX, localZ, nodeType int64) *chunk.ResourceNode {
	ctx := context.Background()

	// First ensure chunk exists
	_, err := tw.ChunkManager.LoadChunk(ctx, chunkX, chunkZ)
	if err != nil {
		t.Fatalf("Failed to load chunk: %v", err)
	}

	// Create node directly in database
	queries := db.NewLoggingQueries(tw.DB)
	_, err = queries.CreateNode(ctx, db.CreateNodeParams{
		ChunkX:           chunkX,
		ChunkZ:           chunkZ,
		LocalX:           localX,
		LocalZ:           localZ,
		NodeType:         nodeType,
		NodeSubtype:      sql.NullInt64{Int64: 0, Valid: true},
		MaxYield:         10,
		CurrentYield:     0, // Depleted
		RegenerationRate: sql.NullInt64{Int64: 0, Valid: true},
		SpawnType:        0,
	})
	if err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	// Get the created node
	nodes, err := queries.GetChunkNodes(ctx, db.GetChunkNodesParams{
		ChunkX: chunkX,
		ChunkZ: chunkZ,
	})
	if err != nil {
		t.Fatalf("Failed to get chunk nodes: %v", err)
	}

	// Find our node and make it inactive
	for _, node := range nodes {
		if node.LocalX == localX && node.LocalZ == localZ && node.NodeType == nodeType {
			// Deactivate the node
			err = queries.DeactivateNode(ctx, db.DeactivateNodeParams{
				RespawnTimer: sql.NullTime{},
				NodeID:       node.NodeID,
			})
			if err != nil {
				t.Fatalf("Failed to deactivate node: %v", err)
			}

			return &chunk.ResourceNode{
				NodeID:           node.NodeID,
				ChunkX:           node.ChunkX,
				ChunkZ:           node.ChunkZ,
				LocalX:           node.LocalX,
				LocalZ:           node.LocalZ,
				NodeType:         node.NodeType,
				NodeSubtype:      node.NodeSubtype.Int64,
				MaxYield:         node.MaxYield,
				CurrentYield:     0,
				RegenerationRate: node.RegenerationRate.Int64,
				SpawnedAt:        node.SpawnedAt.Time,
				SpawnType:        node.SpawnType,
				IsActive:         false,
			}
		}
	}

	t.Fatalf("Failed to find created test node")
	return nil
}
