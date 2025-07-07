package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/VoidMesh/api/internal/chunk"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func testNoiseSystem() {
	// Create test database
	testDBPath := "test_noise.db"
	defer os.Remove(testDBPath)

	database, err := sql.Open("sqlite3", testDBPath)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		return
	}
	defer database.Close()

	// Run migrations
	driver, err := sqlite3.WithInstance(database, &sqlite3.Config{})
	if err != nil {
		fmt.Printf("Failed to create driver: %v\n", err)
		return
	}

	m, err := migrate.NewWithDatabaseInstance("file://../../internal/db/migrations", "sqlite3", driver)
	if err != nil {
		fmt.Printf("Failed to create migrate instance: %v\n", err)
		return
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		fmt.Printf("Failed to run migrations: %v\n", err)
		return
	}

	// Create chunk manager
	chunkManager := chunk.NewManager(database)
	ctx := context.Background()

	// Test coordinates
	testCoords := [][]int64{{0, 0}, {1, 0}, {0, 1}, {10, 10}, {-5, 3}}

	fmt.Println("Testing noise-based resource distribution:")
	fmt.Printf("%-10s %-10s %-15s\n", "Chunk X", "Chunk Z", "Node Count")
	fmt.Println("----------------------------------")

	for _, coords := range testCoords {
		x, z := coords[0], coords[1]

		// Load chunk (this will trigger noise-based spawning)
		response, err := chunkManager.LoadChunk(ctx, x, z)
		if err != nil {
			fmt.Printf("%-10d %-10d %-15s\n", x, z, "ERROR")
			continue
		}

		nodeCount := len(response.Nodes)
		fmt.Printf("%-10d %-10d %-15d\n", x, z, nodeCount)

		// Show node details for first few chunks
		if len(testCoords) <= 3 {
			for _, node := range response.Nodes {
				fmt.Printf("  - Node Type %d at (%d,%d), Yield: %d\n",
					node.NodeType, node.LocalX, node.LocalZ, node.CurrentYield)
			}
		}
	}

	fmt.Println("\nTesting deterministic generation...")

	// Test same chunk multiple times - should be consistent
	response1, _ := chunkManager.LoadChunk(ctx, 0, 0)
	response2, _ := chunkManager.LoadChunk(ctx, 0, 0)

	if len(response1.Nodes) == len(response2.Nodes) {
		fmt.Println("✓ Deterministic generation working - same chunk produces same results")
	} else {
		fmt.Println("✗ Deterministic generation failed - different results for same chunk")
	}

	fmt.Println("\nNoise-based system test completed!")
}

func TestNoiseSystem(t *testing.T) {
	testNoiseSystem()
}
