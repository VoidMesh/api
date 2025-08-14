package handlers

import (
	"context"

	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgxpool"
)

// resourceNodeServiceWrapper implements the ResourceNodeService interface using the real resource node service
type resourceNodeServiceWrapper struct {
	service *resource_node.NodeService
}

// NewResourceNodeService creates a new ResourceNodeService implementation
func NewResourceNodeService(service *resource_node.NodeService) ResourceNodeService {
	return &resourceNodeServiceWrapper{
		service: service,
	}
}

// NewResourceNodeServiceFromConcreteService creates a ResourceNodeService wrapper from a concrete resource node service
// This maintains backward compatibility with the old NewResourceNodeHandler function
func NewResourceNodeServiceFromConcreteService(service *resource_node.NodeService) ResourceNodeService {
	return NewResourceNodeService(service)
}

// NewResourceNodeServiceWithPool creates a resource node service with all dependencies wired up
// This function creates the necessary services and dependencies
func NewResourceNodeServiceWithPool(dbPool *pgxpool.Pool) (ResourceNodeService, error) {
	// Create world service
	worldLogger := world.NewDefaultLoggerWrapper()
	worldService := world.NewServiceWithPool(dbPool, worldLogger)

	// Get default world to create noise generator
	defaultWorld, err := worldService.GetDefaultWorld(context.Background())
	if err != nil {
		return nil, err
	}

	// Create noise generator
	noiseGen := noise.NewGenerator(defaultWorld.Seed)

	// Create resource node service
	resourceNodeService := resource_node.NewNodeServiceWithPool(dbPool, noiseGen.(*noise.Generator), worldService)

	return NewResourceNodeService(resourceNodeService), nil
}

// GetResourcesForChunk retrieves all resource nodes in a specific chunk
func (w *resourceNodeServiceWrapper) GetResourcesForChunk(ctx context.Context, chunkX, chunkY int32) ([]*resourceNodeV1.ResourceNode, error) {
	return w.service.GetResourcesForChunk(ctx, chunkX, chunkY)
}

// GetResourcesForChunks retrieves resource nodes in multiple chunks
func (w *resourceNodeServiceWrapper) GetResourcesForChunks(ctx context.Context, chunks []*chunkV1.ChunkCoordinate) ([]*resourceNodeV1.ResourceNode, error) {
	return w.service.GetResourcesForChunks(ctx, chunks)
}

// GetResourceNodeTypes returns all available resource node types
func (w *resourceNodeServiceWrapper) GetResourceNodeTypes(ctx context.Context) ([]*resourceNodeV1.ResourceNodeType, error) {
	// The service doesn't have this method exposed directly, so we'll use the hardcoded types
	// This is consistent with how the original handler worked
	return w.getHardcodedResourceTypes(), nil
}

// getHardcodedResourceTypes returns all resource types defined in the proto
// This mirrors the method from the resource node service
func (w *resourceNodeServiceWrapper) getHardcodedResourceTypes() []*resourceNodeV1.ResourceNodeType {
	return []*resourceNodeV1.ResourceNodeType{
		// Grass Terrain Resources
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HERB_PATCH),
			Name:        "Herb Patch",
			Description: "Medicinal plants and cooking ingredients",
			TerrainType: "grass",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "herb_patch", Color: "#4CAF50"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 3, RespawnTime: 300, YieldMin: 1, YieldMax: 3},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_BERRY_BUSH),
			Name:        "Berry Bush",
			Description: "Food and crafting materials",
			TerrainType: "grass",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "berry_bush", Color: "#8BC34A"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 4, RespawnTime: 450, YieldMin: 2, YieldMax: 4},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_MINERAL_OUTCROPPING),
			Name:        "Mineral Outcropping",
			Description: "Stone and metal deposits",
			TerrainType: "grass",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "mineral_outcropping", Color: "#795548"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 8, RespawnTime: 900, YieldMin: 1, YieldMax: 2},
		},
		// Water Terrain Resources
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_FISHING_SPOT),
			Name:        "Fishing Spot",
			Description: "Different fish varieties",
			TerrainType: "water",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "fishing_spot", Color: "#2196F3"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 10, RespawnTime: 600, YieldMin: 1, YieldMax: 2},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_KELP_BED),
			Name:        "Kelp Bed",
			Description: "Crafting materials and food",
			TerrainType: "water",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "kelp_bed", Color: "#009688"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 5, RespawnTime: 400, YieldMin: 2, YieldMax: 4},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_PEARL_FORMATION),
			Name:        "Pearl Formation",
			Description: "Rare crafting components",
			TerrainType: "water",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "pearl_formation", Color: "#E1F5FE"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 15, RespawnTime: 1800, YieldMin: 1, YieldMax: 1},
		},
		// Sand Terrain Resources
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_CRYSTAL_FORMATION),
			Name:        "Crystal Formation",
			Description: "Crafting and magical components",
			TerrainType: "sand",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "crystal_formation", Color: "#9C27B0"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 12, RespawnTime: 1200, YieldMin: 1, YieldMax: 2},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_CLAY_DEPOSIT),
			Name:        "Clay Deposit",
			Description: "Building materials and pottery",
			TerrainType: "sand",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "clay_deposit", Color: "#FF9800"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 6, RespawnTime: 450, YieldMin: 2, YieldMax: 5},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_DESERT_PLANT),
			Name:        "Desert Plant",
			Description: "Special ingredients and rare materials",
			TerrainType: "sand",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "desert_plant", Color: "#CDDC39"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 7, RespawnTime: 600, YieldMin: 1, YieldMax: 3},
		},
		// Dirt Terrain Resources
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_HARVESTABLE_TREE),
			Name:        "Harvestable Tree",
			Description: "Different wood types",
			TerrainType: "dirt",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "harvestable_tree", Color: "#8D6E63"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 10, RespawnTime: 900, YieldMin: 3, YieldMax: 6},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_MUSHROOM_CIRCLE),
			Name:        "Mushroom Circle",
			Description: "Alchemy ingredients and food",
			TerrainType: "dirt",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "mushroom_circle", Color: "#6D4C41"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 5, RespawnTime: 450, YieldMin: 2, YieldMax: 4},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_WILD_HONEY_HIVE),
			Name:        "Wild Honey Hive",
			Description: "Food and crafting materials",
			TerrainType: "dirt",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "wild_honey_hive", Color: "#FFC107"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 8, RespawnTime: 1200, YieldMin: 1, YieldMax: 2},
		},
		// Stone Terrain Resources
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_STONE_VEIN),
			Name:        "Stone Vein",
			Description: "Building materials",
			TerrainType: "stone",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "stone_vein", Color: "#607D8B"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 8, RespawnTime: 600, YieldMin: 2, YieldMax: 4},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_GEM_DEPOSIT),
			Name:        "Gem Deposit",
			Description: "Valuable gems",
			TerrainType: "stone",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "gem_deposit", Color: "#E91E63"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 15, RespawnTime: 1500, YieldMin: 1, YieldMax: 2},
		},
		{
			Id:          int32(resourceNodeV1.ResourceNodeTypeId_RESOURCE_NODE_TYPE_ID_METAL_ORE),
			Name:        "Metal Ore",
			Description: "Crafting materials",
			TerrainType: "stone",
			Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON,
			VisualData:  &resourceNodeV1.ResourceVisual{Sprite: "metal_ore", Color: "#9E9E9E"},
			Properties:  &resourceNodeV1.ResourceProperties{HarvestTime: 12, RespawnTime: 900, YieldMin: 1, YieldMax: 3},
		},
	}
}