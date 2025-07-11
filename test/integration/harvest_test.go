package integration

import (
	"context"
	"testing"

	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/chunk/testutils"
)

func TestHarvestUnreachableNodeFails(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a dummy node with non-existent ID for legacy test
	node := &chunk.ResourceNode{
		NodeID:       999999,
		IsActive:     true,
		CurrentYield: 5,
	}

	// Try to harvest non-existent node using legacy method
	_, _, err := world.ChunkManager.HarvestNodeLegacy(ctx, node, playerID)
	if err == nil {
		t.Error("Expected error when harvesting non-existent node, but got nil")
	}
}

// Removed session-based tests as they are now covered by direct_harvest_test.go

func TestHarvestNodeLegacyMethod(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with some yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.Wood, 5)

	// Test harvesting using HarvestNodeLegacy method (for backward compatibility)
	loot, finished, err := world.ChunkManager.HarvestNodeLegacy(ctx, node, playerID)
	if err != nil {
		t.Fatalf("Failed to harvest node: %v", err)
	}
	if finished {
		t.Error("Expected node not to be finished after first harvest")
	}
	if len(loot) != 1 {
		t.Errorf("Expected 1 loot item, got %d", len(loot))
	}
	if len(loot) > 0 && loot[0] != chunk.Wood {
		t.Errorf("Expected loot item to be Wood (%d), got %d", chunk.Wood, loot[0])
	}

	// Note: The legacy method no longer updates the passed-in node object
	// This is expected behavior with the new system
	// The actual node state is managed in the database

	// Note: Can't harvest the same node multiple times due to daily limit in new system
	// This test now validates the daily limit is working correctly

	// Try to harvest the same node again (should fail due to daily limit)
	_, _, err = world.ChunkManager.HarvestNodeLegacy(ctx, node, playerID)
	if err == nil {
		t.Error("Expected error when harvesting same node twice (daily limit), but got nil")
	}
	if err.Error() != "already harvested this node today" {
		t.Errorf("Expected 'already harvested this node today' error, got: %v", err)
	}
}

func TestHarvestMultipleResourceTypes(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create nodes of different types
	ironNode := world.CreateTestNode(t, 0, 0, 1, 1, chunk.IronOre, 3)
	goldNode := world.CreateTestNode(t, 0, 0, 2, 2, chunk.GoldOre, 2)
	woodNode := world.CreateTestNode(t, 0, 0, 3, 3, chunk.Wood, 4)
	stoneNode := world.CreateTestNode(t, 0, 0, 4, 4, chunk.Stone, 1)

	// Harvest from each node using legacy method
	nodes := []*chunk.ResourceNode{ironNode, goldNode, woodNode, stoneNode}
	expectedTypes := []int64{chunk.IronOre, chunk.GoldOre, chunk.Wood, chunk.Stone}

	for i, node := range nodes {
		loot, _, err := world.ChunkManager.HarvestNodeLegacy(ctx, node, playerID)
		if err != nil {
			t.Fatalf("Failed to harvest node %d: %v", i, err)
		}
		if len(loot) != 1 {
			t.Errorf("Expected 1 loot item from node %d, got %d", i, len(loot))
		}
		if len(loot) > 0 && loot[0] != expectedTypes[i] {
			t.Errorf("Expected loot type %d from node %d, got %d", expectedTypes[i], i, loot[0])
		}
	}

	// Verify all resource types are in inventory
	inventory := world.PlayerManager.GetInventory(playerID)
	for _, resourceType := range expectedTypes {
		if inventory[resourceType] != 1 {
			t.Errorf("Expected 1 unit of resource type %d in inventory, got %d", resourceType, inventory[resourceType])
		}
	}

	// Verify harvest stats
	stats := world.PlayerManager.GetStats(playerID)
	if len(stats) != 4 {
		t.Errorf("Expected 4 harvest stat updates, got %d", len(stats))
	}
}

// Removed session-based test as sessions no longer exist
