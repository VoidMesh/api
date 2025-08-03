package resource_node

import (
	"context"

	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
)

// GetResourceNodeTypes returns all resource node types
func (s *NodeService) GetResourceNodeTypes(ctx context.Context) ([]*resourceNodeV1.ResourceNodeType, error) {
	s.logger.Debug("Getting all resource node types")

	// Create static resource type definitions based on the enum values
	resource_node_types := []*resourceNodeV1.ResourceNodeType{
		// Grass Terrain Resources
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
			Name:        "Herb Patch",
			Description: "A cluster of medicinal herbs with various healing properties.",
			TerrainType: "TERRAIN_TYPE_GRASS",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			VisualData: &resourceNodeV1.ResourceVisual{
				Sprite: "herb_patch",
				Color:  "#7CFC00",
			},
			Properties: &resourceNodeV1.ResourceProperties{
				HarvestTime: 2,
				RespawnTime: 300,
				YieldMin:    1,
				YieldMax:    3,
				SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
					{
						Name:      "Common Grass",
						Chance:    0.7,
						MinAmount: 1,
						MaxAmount: 2,
					},
					{
						Name:      "Seeds",
						Chance:    0.3,
						MinAmount: 1,
						MaxAmount: 1,
					},
				},
			},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_BERRY_BUSH),
			Name:        "Berry Bush",
			Description: "A bush full of sweet, edible berries.",
			TerrainType: "TERRAIN_TYPE_GRASS",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			VisualData: &resourceNodeV1.ResourceVisual{
				Sprite: "berry_bush",
				Color:  "#8B0000",
			},
			Properties: &resourceNodeV1.ResourceProperties{
				HarvestTime: 3,
				RespawnTime: 400,
				YieldMin:    2,
				YieldMax:    5,
				SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
					{
						Name:      "Twigs",
						Chance:    0.5,
						MinAmount: 1,
						MaxAmount: 2,
					},
					{
						Name:      "Leaves",
						Chance:    0.6,
						MinAmount: 1,
						MaxAmount: 3,
					},
				},
			},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_MINERAL_OUTCROPPING),
			Name:        "Mineral Outcropping",
			Description: "A small deposit of valuable minerals protruding from the ground.",
			TerrainType: "TERRAIN_TYPE_GRASS",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
			VisualData: &resourceNodeV1.ResourceVisual{
				Sprite: "mineral_outcrop",
				Color:  "#A9A9A9",
			},
			Properties: &resourceNodeV1.ResourceProperties{
				HarvestTime: 5,
				RespawnTime: 600,
				YieldMin:    1,
				YieldMax:    3,
				SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
					{
						Name:      "Stone",
						Chance:    0.8,
						MinAmount: 1,
						MaxAmount: 3,
					},
					{
						Name:      "Dirt",
						Chance:    0.4,
						MinAmount: 1,
						MaxAmount: 2,
					},
				},
			},
		},

		// Water Terrain Resources
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_FISHING_SPOT),
			Name:        "Fishing Spot",
			Description: "A location with an abundance of fish.",
			TerrainType: "TERRAIN_TYPE_WATER",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			VisualData: &resourceNodeV1.ResourceVisual{
				Sprite: "fishing_spot",
				Color:  "#1E90FF",
			},
			Properties: &resourceNodeV1.ResourceProperties{
				HarvestTime: 4,
				RespawnTime: 240,
				YieldMin:    1,
				YieldMax:    3,
				SecondaryDrops: []*resourceNodeV1.SecondaryDrop{
					{
						Name:      "Algae",
						Chance:    0.4,
						MinAmount: 1,
						MaxAmount: 2,
					},
					{
						Name:      "Shells",
						Chance:    0.2,
						MinAmount: 1,
						MaxAmount: 1,
					},
				},
			},
		},

		// Add other resource types here as needed...
	}

	return resource_node_types, nil
}
