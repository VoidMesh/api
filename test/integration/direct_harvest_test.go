package integration

import (
	"context"
	"testing"

	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/chunk/testutils"
)

func TestDirectHarvestNonExistentNodeFails(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	nonExistentNodeID := int64(999999)

	// Try to harvest non-existent node
	harvestCtx := chunk.HarvestContext{
		PlayerID: 1,
		NodeID:   nonExistentNodeID,
	}

	result, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err == nil {
		t.Error("Expected error when harvesting non-existent node, but got nil")
	}
	if result != nil && result.Success {
		t.Error("Expected failed harvest result when harvesting non-existent node")
	}
}

func TestDirectHarvestInactiveNodeFails(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create an inactive/depleted node
	node := world.CreateInactiveTestNode(t, 0, 0, 5, 5, chunk.IronOre)

	// Try to harvest inactive node
	harvestCtx := chunk.HarvestContext{
		PlayerID: playerID,
		NodeID:   node.NodeID,
	}

	result, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err == nil {
		t.Error("Expected error when harvesting inactive node, but got nil")
	}
	if result != nil && result.Success {
		t.Error("Expected failed harvest result when harvesting inactive node")
	}
	if err.Error() != "node is not active" {
		t.Errorf("Expected 'node is not active' error, got: %v", err)
	}
}

func TestDirectHarvestDepletedNodeFails(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with 0 yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.IronOre, 0)

	// Try to harvest depleted node
	harvestCtx := chunk.HarvestContext{
		PlayerID: playerID,
		NodeID:   node.NodeID,
	}

	result, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err == nil {
		t.Error("Expected error when harvesting depleted node, but got nil")
	}
	if result != nil && result.Success {
		t.Error("Expected failed harvest result when harvesting depleted node")
	}
	if err.Error() != "node is depleted" {
		t.Errorf("Expected 'node is depleted' error, got: %v", err)
	}
}

func TestDirectHarvestSuccessfulHarvest(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with some yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.IronOre, 10)

	// Perform direct harvest
	harvestCtx := chunk.HarvestContext{
		PlayerID: playerID,
		NodeID:   node.NodeID,
	}

	result, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err != nil {
		t.Fatalf("Failed to harvest node: %v", err)
	}
	if result == nil {
		t.Fatal("Expected harvest result, but got nil")
	}
	if !result.Success {
		t.Error("Expected successful harvest, but got failure")
	}

	// Verify harvest result structure
	if len(result.PrimaryLoot) == 0 {
		t.Error("Expected primary loot, but got none")
	}
	if len(result.PrimaryLoot) > 0 {
		loot := result.PrimaryLoot[0]
		if loot.ItemType != chunk.IronOre {
			t.Errorf("Expected loot type %d, got %d", chunk.IronOre, loot.ItemType)
		}
		if loot.Quantity != 1 {
			t.Errorf("Expected loot quantity 1, got %d", loot.Quantity)
		}
		if loot.Source != "primary" {
			t.Errorf("Expected loot source 'primary', got %s", loot.Source)
		}
	}

	// Verify node state
	if result.NodeState.CurrentYield != 9 { // 10 - 1 = 9
		t.Errorf("Expected node yield after harvest to be 9, got %d", result.NodeState.CurrentYield)
	}
	if !result.NodeState.IsActive {
		t.Error("Expected node to still be active")
	}

	// Verify harvest details
	if result.HarvestDetails.BaseYield != 1 {
		t.Errorf("Expected base yield 1, got %d", result.HarvestDetails.BaseYield)
	}
	if result.HarvestDetails.TotalYield != 1 {
		t.Errorf("Expected total yield 1, got %d", result.HarvestDetails.TotalYield)
	}

	// Verify inventory was updated
	inventory := world.PlayerManager.GetInventory(playerID)
	if inventory == nil {
		t.Fatal("Expected player inventory, but got nil")
	}
	if inventory[chunk.IronOre] != 1 {
		t.Errorf("Expected 1 iron ore in inventory, got %d", inventory[chunk.IronOre])
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
		if stat.AmountHarvested != 1 {
			t.Errorf("Expected harvested amount 1, got %d", stat.AmountHarvested)
		}
		if stat.NodeID != node.NodeID {
			t.Errorf("Expected node ID %d, got %d", node.NodeID, stat.NodeID)
		}
	}
}

func TestDirectHarvestDailyLimit(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with some yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.GoldOre, 10)

	// Perform first harvest (should succeed)
	harvestCtx := chunk.HarvestContext{
		PlayerID: playerID,
		NodeID:   node.NodeID,
	}

	result1, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err != nil {
		t.Fatalf("Failed to harvest node first time: %v", err)
	}
	if result1 == nil || !result1.Success {
		t.Fatal("Expected successful first harvest")
	}

	// Try to harvest the same node again (should fail due to daily limit)
	result2, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err == nil {
		t.Error("Expected error when harvesting same node twice in a day, but got nil")
	}
	if result2 != nil && result2.Success {
		t.Error("Expected failed harvest result for second daily harvest")
	}
	if err.Error() != "already harvested this node today" {
		t.Errorf("Expected 'already harvested this node today' error, got: %v", err)
	}

	// Verify inventory only has one harvest worth
	inventory := world.PlayerManager.GetInventory(playerID)
	if inventory[chunk.GoldOre] != 1 {
		t.Errorf("Expected 1 gold ore in inventory after daily limit hit, got %d", inventory[chunk.GoldOre])
	}
}

func TestDirectHarvestNodeDepletion(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with minimal yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.Wood, 1)

	// Perform harvest to deplete the node
	harvestCtx := chunk.HarvestContext{
		PlayerID: playerID,
		NodeID:   node.NodeID,
	}

	result, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err != nil {
		t.Fatalf("Failed to harvest node: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatal("Expected successful harvest")
	}

	// Verify node is depleted and inactive
	if result.NodeState.CurrentYield != 0 {
		t.Errorf("Expected node yield to be 0 after depletion, got %d", result.NodeState.CurrentYield)
	}
	if result.NodeState.IsActive {
		t.Error("Expected node to be inactive after depletion")
	}
	if result.NodeState.RespawnTimer == nil {
		t.Error("Expected node to have respawn timer after depletion")
	}

	// Try to harvest depleted node (should fail)
	result2, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err == nil {
		t.Error("Expected error when harvesting depleted node, but got nil")
	}
	if result2 != nil && result2.Success {
		t.Error("Expected failed harvest result when harvesting depleted node")
	}
}

func TestDirectHarvestMultipleResourceTypes(t *testing.T) {
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
		harvestCtx := chunk.HarvestContext{
			PlayerID: playerID,
			NodeID:   node.NodeID,
		}

		result, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
		if err != nil {
			t.Fatalf("Failed to harvest node %d: %v", i, err)
		}
		if result == nil || !result.Success {
			t.Fatalf("Expected successful harvest from node %d", i)
		}
		if len(result.PrimaryLoot) != 1 {
			t.Errorf("Expected 1 loot item from node %d, got %d", i, len(result.PrimaryLoot))
		}
		if len(result.PrimaryLoot) > 0 && result.PrimaryLoot[0].ItemType != expectedTypes[i] {
			t.Errorf("Expected loot type %d from node %d, got %d", expectedTypes[i], i, result.PrimaryLoot[0].ItemType)
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

func TestDirectHarvestMultiplePlayers(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	player1ID := int64(1)
	player2ID := int64(2)

	// Create a node with some yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.Stone, 10)

	// Player 1 harvests
	harvestCtx1 := chunk.HarvestContext{
		PlayerID: player1ID,
		NodeID:   node.NodeID,
	}

	result1, err := world.ChunkManager.HarvestNode(ctx, harvestCtx1)
	if err != nil {
		t.Fatalf("Failed player 1 harvest: %v", err)
	}
	if result1 == nil || !result1.Success {
		t.Fatal("Expected successful player 1 harvest")
	}

	// Player 2 harvests (should fail due to daily limit)
	harvestCtx2 := chunk.HarvestContext{
		PlayerID: player2ID,
		NodeID:   node.NodeID,
	}

	result2, err := world.ChunkManager.HarvestNode(ctx, harvestCtx2)
	if err != nil {
		t.Fatalf("Failed player 2 harvest: %v", err)
	}
	if result2 == nil || !result2.Success {
		t.Fatal("Expected successful player 2 harvest")
	}

	// Verify both players have resources
	inventory1 := world.PlayerManager.GetInventory(player1ID)
	inventory2 := world.PlayerManager.GetInventory(player2ID)

	if inventory1[chunk.Stone] != 1 {
		t.Errorf("Expected 1 stone in player 1 inventory, got %d", inventory1[chunk.Stone])
	}
	if inventory2[chunk.Stone] != 1 {
		t.Errorf("Expected 1 stone in player 2 inventory, got %d", inventory2[chunk.Stone])
	}

	// Verify node yield decreased appropriately
	if result2.NodeState.CurrentYield != 8 { // 10 - 1 - 1 = 8
		t.Errorf("Expected node yield to be 8 after both harvests, got %d", result2.NodeState.CurrentYield)
	}
}

func TestDirectHarvestWithFutureFeatures(t *testing.T) {
	// Create test world
	world := testutils.CreateTestWorld(t)
	defer world.Cleanup()

	ctx := context.Background()
	playerID := int64(1)

	// Create a node with some yield
	node := world.CreateTestNode(t, 0, 0, 5, 5, chunk.IronOre, 10)

	// Perform harvest with future features (should handle gracefully)
	harvestCtx := chunk.HarvestContext{
		PlayerID: playerID,
		NodeID:   node.NodeID,
		// Future features - should be handled gracefully even if nil
		CharacterStats: &chunk.CharacterStats{
			PlayerID:    playerID,
			MiningLevel: 5,
			MiningBonus: 0.1,  // 10% bonus (currently unused)
			LuckBonus:   0.05, // 5% luck (currently unused)
		},
		ToolID:    nil,                    // No tool equipped
		ToolStats: nil,                    // No tool stats
		Bonuses:   []chunk.HarvestBonus{}, // No bonuses
	}

	result, err := world.ChunkManager.HarvestNode(ctx, harvestCtx)
	if err != nil {
		t.Fatalf("Failed to harvest node with future features: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatal("Expected successful harvest with future features")
	}

	// Verify current behavior (future features are not yet implemented)
	if result.HarvestDetails.StatBonus != 0 {
		t.Errorf("Expected stat bonus to be 0 (not yet implemented), got %d", result.HarvestDetails.StatBonus)
	}
	if result.HarvestDetails.ToolBonus != 0 {
		t.Errorf("Expected tool bonus to be 0 (not yet implemented), got %d", result.HarvestDetails.ToolBonus)
	}
	if result.HarvestDetails.TotalYield != 1 {
		t.Errorf("Expected total yield to be 1 (base amount), got %d", result.HarvestDetails.TotalYield)
	}

	// Verify future fields are properly structured
	if result.ExperienceGained != nil {
		t.Error("Expected experience gained to be nil (not yet implemented)")
	}
	if result.ToolWear != nil {
		t.Error("Expected tool wear to be nil (not yet implemented)")
	}
	if len(result.BonusLoot) != 0 {
		t.Errorf("Expected no bonus loot (not yet implemented), got %d items", len(result.BonusLoot))
	}
}
