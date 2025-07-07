package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/VoidMesh/api/internal/chunk"
	_ "github.com/mattn/go-sqlite3"
)

func TestClusterSpawning(t *testing.T) {
	// Create temporary database for testing
	tempDB := "test_cluster_spawning.db"
	defer os.Remove(tempDB)

	// Open database
	db, err := sql.Open("sqlite3", tempDB)
	if err != nil {
		t.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Initialize database schema (you may need to run migrations here)
	// TODO: Add schema initialization

	// Create manager
	manager := chunk.NewManager(db)

	// Test chunk coordinates
	chunkX, chunkZ := int64(0), int64(0)

	t.Logf("Testing cluster spawning for chunk (%d, %d)", chunkX, chunkZ)

	// Load chunk (this will trigger node generation)
	chunkResponse, err := manager.LoadChunk(context.Background(), chunkX, chunkZ)
	if err != nil {
		t.Fatal("Failed to load chunk:", err)
	}

	t.Logf("Chunk loaded successfully with %d nodes:", len(chunkResponse.Nodes))
	for _, node := range chunkResponse.Nodes {
		t.Logf("  Node %d: Type=%d, Subtype=%d, Position=(%d,%d), Yield=%d/%d, SpawnType=%d",
			node.NodeID, node.NodeType, node.NodeSubtype, node.LocalX, node.LocalZ,
			node.CurrentYield, node.MaxYield, node.SpawnType)
	}

	// Test another chunk to see different clustering
	chunkX2, chunkZ2 := int64(1), int64(1)
	t.Logf("Testing cluster spawning for chunk (%d, %d)", chunkX2, chunkZ2)

	chunkResponse2, err := manager.LoadChunk(context.Background(), chunkX2, chunkZ2)
	if err != nil {
		t.Fatal("Failed to load chunk 2:", err)
	}

	t.Logf("Chunk 2 loaded successfully with %d nodes:", len(chunkResponse2.Nodes))
	for _, node := range chunkResponse2.Nodes {
		t.Logf("  Node %d: Type=%d, Subtype=%d, Position=(%d,%d), Yield=%d/%d, SpawnType=%d",
			node.NodeID, node.NodeType, node.NodeSubtype, node.LocalX, node.LocalZ,
			node.CurrentYield, node.MaxYield, node.SpawnType)
	}

	// Add some basic assertions
	if len(chunkResponse.Nodes) == 0 {
		t.Error("Expected at least some nodes in chunk 0,0")
	}

	if len(chunkResponse2.Nodes) == 0 {
		t.Error("Expected at least some nodes in chunk 1,1")
	}

	t.Log("Cluster spawning test completed successfully!")
}

// TestClusterSpawningVerbose can be run with go test -v to see detailed output
func TestClusterSpawningVerbose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping verbose cluster test in short mode")
	}

	// This is the same test but with more detailed output
	TestClusterSpawning(t)
}
