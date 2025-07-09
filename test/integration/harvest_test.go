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
	nonExistentNodeID := int64(999999)

	// Try to start harvest on non-existent node
	session, err := world.ChunkManager.StartHarvest(ctx, nonExistentNodeID, playerID)
	if err == nil {
		t.Error("Expected error when starting harvest on non-existent node, but got nil")
	}
	if session != nil {
		t.Error("Expected nil session when starting harvest on non-existent node, but got session")
	}
	if err.Error() != "node not found" {
		t.Errorf("Expected 'node not found' error, got: %v", err)
	}
}

func TestHarvestInactiveNodeFails(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create an inactive/depleted node
	node := world.CreateInactiveTestNode(t, 0, 0, 5, 5, chunk.IronOre)

	// Try to start harvest on inactive node
	session, err := world.ChunkManager.StartHarvest(ctx, node.NodeID, playerID)
	if err == nil {
		t.Error("Expected error when starting harvest on inactive node, but got nil")
	}
	if session != nil {
		t.Error("Expected nil session when starting harvest on inactive node, but got session")
	}
	if err.Error() != "node is not active" {
		t.Errorf("Expected 'node is not active' error, got: %v", err)
	}
}

func TestHarvestDepletedNodeFails(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with 0 yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.IronOre, 0)

	// Try to start harvest on depleted node
	session, err := world.ChunkManager.StartHarvest(ctx, node.NodeID, playerID)
	if err == nil {
		t.Error("Expected error when starting harvest on depleted node, but got nil")
	}
	if session != nil {
		t.Error("Expected nil session when starting harvest on depleted node, but got session")
	}
	if err.Error() != "node is depleted" {
		t.Errorf("Expected 'node is depleted' error, got: %v", err)
	}
}

func TestSuccessfulHarvestAddsItemsToInventory(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with some yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.IronOre, 10)

	// Start harvest session
	session, err := world.ChunkManager.StartHarvest(ctx, node.NodeID, playerID)
	if err != nil {
		t.Fatalf("Failed to start harvest session: %v", err)
	}
	if session == nil {
		t.Fatal("Expected harvest session, but got nil")
	}

	// Verify session details
	if session.NodeID != node.NodeID {
		t.Errorf("Expected session node ID %d, got %d", node.NodeID, session.NodeID)
	}
	if session.PlayerID != playerID {
		t.Errorf("Expected session player ID %d, got %d", playerID, session.PlayerID)
	}

	// Perform harvest
	harvestAmount := int64(3)
	response, err := world.ChunkManager.HarvestResource(ctx, session.SessionID, harvestAmount)
	if err != nil {
		t.Fatalf("Failed to harvest resource: %v", err)
	}
	if response == nil {
		t.Fatal("Expected harvest response, but got nil")
	}

	// Verify harvest response
	if !response.Success {
		t.Error("Expected successful harvest, but got failure")
	}
	if response.AmountHarvested != harvestAmount {
		t.Errorf("Expected harvested amount %d, got %d", harvestAmount, response.AmountHarvested)
	}
	if response.NodeYieldAfter != 7 { // 10 - 3 = 7
		t.Errorf("Expected node yield after harvest to be 7, got %d", response.NodeYieldAfter)
	}
	if response.ResourcesGathered != harvestAmount {
		t.Errorf("Expected resources gathered %d, got %d", harvestAmount, response.ResourcesGathered)
	}

	// Verify inventory was updated
	inventory := world.PlayerManager.GetInventory(playerID)
	if inventory == nil {
		t.Fatal("Expected player inventory, but got nil")
	}
	if inventory[chunk.IronOre] != harvestAmount {
		t.Errorf("Expected %d iron ore in inventory, got %d", harvestAmount, inventory[chunk.IronOre])
	}

	// Verify harvest stats were updated
	stats := world.PlayerManager.GetStats(playerID)
	if len(stats) != 1 {
		t.Errorf("Expected 1 harvest stat update, got %d", len(stats))
	}
	if len(stats) > 0 {
		stat := stats[0]
		if stat.ResourceType != chunk.IronOre {
			t.Errorf("Expected resource type %d, got %d", chunk.IronOre, stat.ResourceType)
		}
		if stat.AmountHarvested != harvestAmount {
			t.Errorf("Expected harvested amount %d, got %d", harvestAmount, stat.AmountHarvested)
		}
		if stat.NodeID != node.NodeID {
			t.Errorf("Expected node ID %d, got %d", node.NodeID, stat.NodeID)
		}
	}
}

func TestNodeMarkedDepletedWhenFinished(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with low yield
	initialYield := int64(2)
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.IronOre, initialYield)

	// Start harvest session
	session, err := world.ChunkManager.StartHarvest(ctx, node.NodeID, playerID)
	if err != nil {
		t.Fatalf("Failed to start harvest session: %v", err)
	}

	// Harvest all resources to deplete the node
	response, err := world.ChunkManager.HarvestResource(ctx, session.SessionID, initialYield)
	if err != nil {
		t.Fatalf("Failed to harvest resource: %v", err)
	}

	// Verify harvest response
	if !response.Success {
		t.Error("Expected successful harvest, but got failure")
	}
	if response.AmountHarvested != initialYield {
		t.Errorf("Expected harvested amount %d, got %d", initialYield, response.AmountHarvested)
	}
	if response.NodeYieldAfter != 0 {
		t.Errorf("Expected node yield after harvest to be 0, got %d", response.NodeYieldAfter)
	}

	// Verify inventory was updated
	inventory := world.PlayerManager.GetInventory(playerID)
	if inventory[chunk.IronOre] != initialYield {
		t.Errorf("Expected %d iron ore in inventory, got %d", initialYield, inventory[chunk.IronOre])
	}

	// Try to start another harvest session on the same node (should fail)
	_, err = world.ChunkManager.StartHarvest(ctx, node.NodeID, playerID)
	if err == nil {
		t.Error("Expected error when starting harvest on depleted node, but got nil")
	}
	if err.Error() != "node is not active" {
		t.Errorf("Expected 'node is not active' error, got: %v", err)
	}
}

func TestHarvestNodeMethod(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with some yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.Wood, 5)

	// Test harvesting using HarvestNode method
	loot, finished, err := world.ChunkManager.HarvestNode(ctx, node, playerID)
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

	// Verify node state was updated
	if node.CurrentYield != 4 { // 5 - 1 = 4
		t.Errorf("Expected node yield to be 4, got %d", node.CurrentYield)
	}
	if !node.IsActive {
		t.Error("Expected node to still be active")
	}

	// Harvest until depletion
	for i := 0; i < 4; i++ {
		loot, finished, err = world.ChunkManager.HarvestNode(ctx, node, playerID)
		if err != nil {
			t.Fatalf("Failed to harvest node on iteration %d: %v", i, err)
		}
		if i == 3 && !finished {
			t.Error("Expected node to be finished after final harvest")
		}
		if i < 3 && finished {
			t.Errorf("Expected node not to be finished on iteration %d", i)
		}
	}

	// Verify node is depleted and inactive
	if node.CurrentYield != 0 {
		t.Errorf("Expected node yield to be 0, got %d", node.CurrentYield)
	}
	if node.IsActive {
		t.Error("Expected node to be inactive after depletion")
	}
	if node.RespawnTimer == nil {
		t.Error("Expected node to have respawn timer after depletion")
	}

	// Try to harvest depleted node
	_, _, err = world.ChunkManager.HarvestNode(ctx, node, playerID)
	if err == nil {
		t.Error("Expected error when harvesting depleted node, but got nil")
	}
	if err.Error() != "node is not active" {
		t.Errorf("Expected 'node is not active' error, got: %v", err)
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

	// Harvest from each node
	nodes := []*chunk.ResourceNode{ironNode, goldNode, woodNode, stoneNode}
	expectedTypes := []int64{chunk.IronOre, chunk.GoldOre, chunk.Wood, chunk.Stone}

	for i, node := range nodes {
		loot, _, err := world.ChunkManager.HarvestNode(ctx, node, playerID)
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

func TestPlayerAlreadyHasActiveSession(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create two nodes
	node1 := world.CreateTestNode(t, 0, 0, 1, 1, chunk.IronOre, 5)
	node2 := world.CreateTestNode(t, 0, 0, 2, 2, chunk.GoldOre, 5)

	// Start harvest session on first node
	session1, err := world.ChunkManager.StartHarvest(ctx, node1.NodeID, playerID)
	if err != nil {
		t.Fatalf("Failed to start first harvest session: %v", err)
	}
	if session1 == nil {
		t.Fatal("Expected harvest session, but got nil")
	}

	// Try to start harvest session on second node (should fail)
	session2, err := world.ChunkManager.StartHarvest(ctx, node2.NodeID, playerID)
	if err == nil {
		t.Error("Expected error when starting second harvest session, but got nil")
	}
	if session2 != nil {
		t.Error("Expected nil session when starting second harvest session, but got session")
	}
	if err.Error() != "player already has active harvest session" {
		t.Errorf("Expected 'player already has active harvest session' error, got: %v", err)
	}
}
