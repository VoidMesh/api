package resource_node

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/VoidMesh/api/api/db"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Resource generation constants
	ResourceNoiseScale   = 150.0 // Larger scale for more spread out resources
	ResourceDetailScale  = 30.0  // Smaller scale for fine details
	ResourceBufferZone   = 1     // Cells away from terrain transitions to avoid spawning
	MinClusterDistance   = 4     // Minimum distance between cluster centers
	MaxResourcesPerChunk = 24    // Maximum resources per chunk
	ChunkSize            = 32    // Size of a chunk in cells

	// Spawn thresholds by rarity
	CommonThreshold   = 0.3  // Common resources (easily found)
	UncommonThreshold = 0.5  // Uncommon resources (moderately rare)
	RareThreshold     = 0.7  // Rare resources (hard to find)
	VeryRareThreshold = 0.85 // Very rare resources (extremely rare)

	// Cluster size probability weights
	// Format: [min, max, weight]
	// Higher weight = more likely
)

// Cluster size probabilities by rarity
// Maps rarity to [min size, max size, weight for sizes 1-6]
var ClusterSizes = map[string]map[int]int{
	"common": {
		1: 10, // Size 1: weight 10
		2: 30, // Size 2: weight 30
		3: 40, // Size 3: weight 40
		4: 15, // Size 4: weight 15
		5: 5,  // Size 5: weight 5
		6: 0,  // Size 6: weight 0
	},
	"uncommon": {
		1: 15, // Size 1: weight 15
		2: 35, // Size 2: weight 35
		3: 30, // Size 3: weight 30
		4: 15, // Size 4: weight 15
		5: 5,  // Size 5: weight 5
		6: 0,  // Size 6: weight 0
	},
	"rare": {
		1: 25, // Size 1: weight 25
		2: 40, // Size 2: weight 40
		3: 25, // Size 3: weight 25
		4: 10, // Size 4: weight 10
		5: 0,  // Size 5: weight 0
		6: 0,  // Size 6: weight 0
	},
	"very_rare": {
		1: 60, // Size 1: weight 60
		2: 30, // Size 2: weight 30
		3: 10, // Size 3: weight 10
		4: 0,  // Size 4: weight 0
		5: 0,  // Size 5: weight 0
		6: 0,  // Size 6: weight 0
	},
}

// NodeService provides resource node generation functionality
type NodeService struct {
	db             DatabaseInterface
	noiseGen       NoiseGeneratorInterface
	worldService   WorldServiceInterface
	rnd            RandomGeneratorInterface
	logger         LoggerInterface
	// Cache of hardcoded resource types to avoid rebuilding on each request
	resourceTypes []*resourceNodeV1.ResourceNodeType
	// Map of resource types by terrain for faster lookups
	resourceTypesByTerrain map[string][]*resourceNodeV1.ResourceNodeType
	// Map of resource types by ID for faster lookups
	resourceTypesByID map[int32]*resourceNodeV1.ResourceNodeType
}

// NewNodeService creates a new resource node service with dependency injection.
func NewNodeService(
	db DatabaseInterface,
	noiseGen NoiseGeneratorInterface,
	worldService WorldServiceInterface,
	rnd RandomGeneratorInterface,
	logger LoggerInterface,
) *NodeService {
	componentLogger := logger.With("component", "resource-node-service")
	componentLogger.Debug("Creating new resource node service")

	// Initialize resource type caches
	service := &NodeService{
		db:                     db,
		noiseGen:               noiseGen,
		worldService:           worldService,
		rnd:                    rnd,
		logger:                 componentLogger,
		resourceTypesByTerrain: make(map[string][]*resourceNodeV1.ResourceNodeType),
		resourceTypesByID:      make(map[int32]*resourceNodeV1.ResourceNodeType),
	}

	// Preload resource types
	service.resourceTypes = service.getHardcodedResourceTypes()

	// Group resource types by terrain for faster lookup
	for _, r := range service.resourceTypes {
		terrainType := r.TerrainType
		service.resourceTypesByTerrain[terrainType] = append(service.resourceTypesByTerrain[terrainType], r)
		service.resourceTypesByID[r.Id] = r
	}

	return service
}

// NewNodeServiceWithPool creates a service with concrete implementations (convenience constructor for production use).
func NewNodeServiceWithPool(
	pool *pgxpool.Pool,
	noiseGen *noise.Generator,
	worldService *world.Service,
) *NodeService {
	// Create a deterministic random source based on the noise generator's seed
	rnd := NewRandomGenerator(noiseGen.GetSeed())
	logger := NewDefaultLoggerWrapper()

	return NewNodeService(
		NewDatabaseWrapper(pool),
		NewNoiseGeneratorAdapter(noiseGen),
		NewWorldServiceAdapter(worldService),
		rnd,
		logger,
	)
}

// GenerateResourcesForChunk generates resource nodes for a chunk
func (s *NodeService) GenerateResourcesForChunk(ctx context.Context, chunk *chunkV1.ChunkData) ([]*resourceNodeV1.ResourceNode, error) {
	s.logger.Debug("Generating resource nodes for chunk", "chunk_x", chunk.ChunkX, "chunk_y", chunk.ChunkY)

	// Use the pre-grouped resource types by terrain from service initialization
	s.logger.Debug("Using cached resource types", "count", len(s.resourceTypes))

	// Using the cached resourceTypesByTerrain instead of building it each time

	s.logger.Debug("Grouped resources by terrain", "terrain_types", len(s.resourceTypesByTerrain))

	// Map to track occupied positions
	occupiedPositions := make(map[string]bool)

	// Map to track cluster centers for minimum distance check
	clusterCenters := make([]struct{ x, y int32 }, 0)

	// List to collect all generated resources
	var resourceNodes []*resourceNodeV1.ResourceNode

	// Process each resource type for this chunk
	for terrainType, resourceNodeTypes := range s.resourceTypesByTerrain {
		s.logger.Debug("Processing terrain type", "terrain_type", terrainType, "resource_types_count", len(resourceNodeTypes))
		for _, resourceNodeType := range resourceNodeTypes {
			// Create a separate noise map for this resource type
			// Use resource ID as additional seed to make different resources spawn in different patterns
			resourceSeed := s.noiseGen.GetSeed() + int64(resourceNodeType.Id)
			resourceRng := rand.New(rand.NewSource(resourceSeed))

			// Generate potential spawn points
			spawnPoints := s.findPotentialSpawnPoints(
				chunk,
				terrainType,
				s.getRarityThresholdFromEnum(resourceNodeType.Rarity),
				resourceSeed,
			)

			s.logger.Debug("Found spawn points", "resource_name", resourceNodeType.Name, "spawn_points_count", len(spawnPoints))

			// Shuffle spawn points to avoid patterns
			resourceRng.Shuffle(len(spawnPoints), func(i, j int) {
				spawnPoints[i], spawnPoints[j] = spawnPoints[j], spawnPoints[i]
			})

			// Try to create clusters from the spawn points
			for _, point := range spawnPoints {
				// Check if we've reached the max resources per chunk
				if len(resourceNodes) >= MaxResourcesPerChunk {
					break
				}

				// Check minimum distance from other clusters
				tooClose := false
				for _, center := range clusterCenters {
					dist := distance(point.x, point.y, center.x, center.y)
					if dist < MinClusterDistance {
						tooClose = true
						break
					}
				}
				if tooClose {
					continue
				}

				// This point becomes a cluster center
				clusterCenters = append(clusterCenters, struct{ x, y int32 }{point.x, point.y})

				// Generate a unique cluster ID
				clusterID := generateClusterID(chunk.ChunkX, chunk.ChunkY, point.x, point.y, resourceNodeType.Id)

				// Determine cluster size
				clusterSize := s.determineClusterSizeFromEnum(resourceNodeType.Rarity)

				// Create the first resource node at the center point
				posKey := fmt.Sprintf("%d,%d", point.x, point.y)
				if !occupiedPositions[posKey] {
					// Create the resource node
					resourceNode := &resourceNodeV1.ResourceNode{
						ResourceNodeType:   resourceNodeType,
						ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId(resourceNodeType.Id),
						ChunkX:             chunk.ChunkX,
						ChunkY:             chunk.ChunkY,
						PosX:               point.x,
						PosY:               point.y,
						ClusterId:          clusterID,
						Size:               1,
						CreatedAt:          timestamppb.Now(),
					}
					resourceNodes = append(resourceNodes, resourceNode)
					occupiedPositions[posKey] = true

					// Generate additional nodes in the cluster
					s.generateClusterNodes(
						chunk,
						resourceNode,
						clusterSize-1, // Subtract 1 since we already created the center node
						occupiedPositions,
						resourceNodeType.TerrainType,
						&resourceNodes,
					)
				}
			}
		}
	}

	return resourceNodes, nil
}

// findPotentialSpawnPoints finds potential resource spawn points in a chunk
func (s *NodeService) findPotentialSpawnPoints(
	chunk *chunkV1.ChunkData,
	terrainType string,
	threshold float64,
	resourceSeed int64,
) []struct {
	x, y       int32
	noiseValue float64
} {
	// Create a specialized noise generator for this resource
	resourceNoiseGen := noise.NewGenerator(resourceSeed)

	var spawnPoints []struct {
		x, y       int32
		noiseValue float64
	}

	// Scan the entire chunk
	for y := int32(0); y < ChunkSize; y++ {
		for x := int32(0); x < ChunkSize; x++ {
			// Get cell's terrain type
			cellIndex := y*ChunkSize + x
			if cellIndex >= int32(len(chunk.Cells)) {
				continue
			}
			cell := chunk.Cells[cellIndex]

			// Check if terrain type matches the resource's terrain type
			cellTerrainType := s.terrainTypeToString(cell.TerrainType)
			if cellTerrainType != terrainType {
				continue
			}

			// Check if cell is in a buffer zone near terrain transitions
			if s.isNearTerrainTransition(chunk, x, y) {
				continue
			}

			// Get the world coordinates
			worldX := chunk.ChunkX*ChunkSize + x
			worldY := chunk.ChunkY*ChunkSize + y

			// Calculate noise value for this position
			// Combine a large-scale noise for overall distribution with a small-scale noise for detail
			largeScaleNoise := resourceNoiseGen.GetTerrainNoise(int(worldX), int(worldY), ResourceNoiseScale)
			detailNoise := resourceNoiseGen.GetTerrainNoise(int(worldX), int(worldY), ResourceDetailScale)

			// Weight the large scale more heavily for better clustering
			combinedNoise := largeScaleNoise*0.8 + detailNoise*0.2

			// Normalize to 0-1 range
			normalizedNoise := (combinedNoise + 1.0) / 2.0

			// Check if noise exceeds the threshold for this resource's rarity
			if normalizedNoise > threshold {
				spawnPoints = append(spawnPoints, struct {
					x, y       int32
					noiseValue float64
				}{
					x:          x,
					y:          y,
					noiseValue: normalizedNoise,
				})
			}
		}
	}

	return spawnPoints
}

// generateClusterNodes generates additional nodes around a cluster center
func (s *NodeService) generateClusterNodes(
	chunk *chunkV1.ChunkData,
	centerNode *resourceNodeV1.ResourceNode,
	numNodes int,
	occupiedPositions map[string]bool,
	terrainType string,
	resources *[]*resourceNodeV1.ResourceNode,
) {
	// No additional nodes needed
	if numNodes <= 0 {
		return
	}

	// Define possible directions for cluster expansion
	directions := []struct{ dx, dy int32 }{
		{-1, -1},
		{0, -1},
		{1, -1},
		{-1, 0},
		{1, 0},
		{-1, 1},
		{0, 1},
		{1, 1},
	}

	// Try to create the requested number of nodes
	nodesCreated := 0
	maxAttempts := numNodes * 3 // Allow multiple attempts

	for attempt := 0; attempt < maxAttempts && nodesCreated < numNodes; attempt++ {
		// Start at the center
		baseX := centerNode.PosX
		baseY := centerNode.PosY

		// Choose a random direction and distance
		dir := directions[s.rnd.Intn(len(directions))]
		distance := int32(1 + s.rnd.Intn(2)) // 1-2 cells away

		// Calculate new position
		newX := baseX + dir.dx*distance
		newY := baseY + dir.dy*distance

		// Check bounds
		if newX < 0 || newX >= ChunkSize || newY < 0 || newY >= ChunkSize {
			continue
		}

		// Check if position is occupied
		posKey := fmt.Sprintf("%d,%d", newX, newY)
		if occupiedPositions[posKey] {
			continue
		}

		// Check terrain type
		cellIndex := newY*ChunkSize + newX
		if cellIndex >= int32(len(chunk.Cells)) {
			continue
		}
		cell := chunk.Cells[cellIndex]
		cellTerrainType := s.terrainTypeToString(cell.TerrainType)

		if cellTerrainType != terrainType {
			continue
		}

		// Check buffer zone
		if s.isNearTerrainTransition(chunk, newX, newY) {
			continue
		}

		// Create a new node
		resourceNode := &resourceNodeV1.ResourceNode{
			ResourceNodeType:   centerNode.ResourceNodeType,
			ResourceNodeTypeId: centerNode.ResourceNodeTypeId,
			ChunkX:             centerNode.ChunkX,
			ChunkY:             centerNode.ChunkY,
			PosX:               newX,
			PosY:               newY,
			ClusterId:          centerNode.ClusterId,
			Size:               1,
			CreatedAt:          timestamppb.Now(),
		}

		// Add to the resources slice
		*resources = append(*resources, resourceNode)

		// Mark position as occupied
		occupiedPositions[posKey] = true
		nodesCreated++
	}
}

// isNearTerrainTransition checks if a cell is near a terrain transition
func (s *NodeService) isNearTerrainTransition(chunk *chunkV1.ChunkData, x, y int32) bool {
	// Get the terrain type of the current cell
	cellIndex := y*ChunkSize + x
	if cellIndex >= int32(len(chunk.Cells)) {
		return true // Out of bounds, consider it a transition
	}
	currentType := chunk.Cells[cellIndex].TerrainType

	// Check cells in a radius of ResourceBufferZone
	for dy := -ResourceBufferZone; dy <= ResourceBufferZone; dy++ {
		for dx := -ResourceBufferZone; dx <= ResourceBufferZone; dx++ {
			// Skip the center cell
			if dx == 0 && dy == 0 {
				continue
			}

			// Calculate position to check
			checkX := x + int32(dx)
			checkY := y + int32(dy)

			// Check bounds
			if checkX < 0 || checkX >= ChunkSize || checkY < 0 || checkY >= ChunkSize {
				return true // Out of bounds, consider it a transition
			}

			// Check terrain type
			checkIndex := checkY*ChunkSize + checkX
			if checkIndex >= int32(len(chunk.Cells)) {
				return true
			}
			checkType := chunk.Cells[checkIndex].TerrainType

			// If we find a different terrain type, this is near a transition
			if checkType != currentType {
				return true
			}
		}
	}

	return false
}

// determineClusterSize determines the size of a resource cluster based on rarity
func (s *NodeService) determineClusterSize(rarity string) int {
	rarityLower := strings.ToLower(rarity)

	// Default to common if rarity not found
	clusterSizeWeights, ok := ClusterSizes[rarityLower]
	if !ok {
		clusterSizeWeights = ClusterSizes["common"]
	}

	// Calculate total weight
	totalWeight := 0
	for _, weight := range clusterSizeWeights {
		totalWeight += weight
	}

	// If all weights are 0, return 1
	if totalWeight == 0 {
		return 1
	}

	// Roll a random number between 0 and total weight
	roll := s.rnd.Intn(totalWeight)

	// Find which size range this roll falls into
	cumWeight := 0
	for size, weight := range clusterSizeWeights {
		cumWeight += weight
		if roll < cumWeight {
			return size
		}
	}

	// Fallback to 1
	return 1
}

// getHardcodedResourceTypes returns all resource types defined in the proto
// This is now only called during service initialization
func (s *NodeService) getHardcodedResourceTypes() []*resourceNodeV1.ResourceNodeType {
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

// terrainTypeToString converts a terrain type enum to a string
func (s *NodeService) terrainTypeToString(terrainType chunkV1.TerrainType) string {
	// Convert terrain type enum to simplified string that matches resource definitions
	switch terrainType {
	case chunkV1.TerrainType_TERRAIN_TYPE_GRASS:
		return "grass"
	case chunkV1.TerrainType_TERRAIN_TYPE_WATER:
		return "water"
	case chunkV1.TerrainType_TERRAIN_TYPE_SAND:
		return "sand"
	case chunkV1.TerrainType_TERRAIN_TYPE_STONE:
		return "stone"
	case chunkV1.TerrainType_TERRAIN_TYPE_DIRT:
		return "dirt"
	default:
		return "unknown"
	}
}

// getRarityThresholdFromEnum returns the noise threshold for a given rarity enum
func (s *NodeService) getRarityThresholdFromEnum(rarity resourceNodeV1.ResourceRarity) float64 {
	switch rarity {
	case resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON:
		return CommonThreshold
	case resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON:
		return UncommonThreshold
	case resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE:
		return RareThreshold
	case resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE:
		return VeryRareThreshold
	default:
		return CommonThreshold
	}
}

// determineClusterSizeFromEnum returns the cluster size based on rarity enum
func (s *NodeService) determineClusterSizeFromEnum(rarity resourceNodeV1.ResourceRarity) int {
	var rarityString string
	switch rarity {
	case resourceNodeV1.ResourceRarity_RESOURCE_RARITY_COMMON:
		rarityString = "common"
	case resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON:
		rarityString = "uncommon"
	case resourceNodeV1.ResourceRarity_RESOURCE_RARITY_RARE:
		rarityString = "rare"
	case resourceNodeV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE:
		rarityString = "very_rare"
	default:
		rarityString = "common"
	}

	return s.determineClusterSize(rarityString)
}

// generateClusterID generates a unique ID for a resource cluster
func generateClusterID(chunkX, chunkY, posX, posY int32, resourceNodeTypeID int32) string {
	// Create a unique string based on position and resource type
	input := fmt.Sprintf("%d:%d:%d:%d:%d:%d", chunkX, chunkY, posX, posY, resourceNodeTypeID, time.Now().UnixNano())

	// Generate MD5 hash
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 characters of the hash
}

// distance calculates the Euclidean distance between two points
func distance(x1, y1, x2, y2 int32) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return (dx*dx + dy*dy)
}

// StoreResourceNodes stores generated resource nodes in the database
func (s *NodeService) StoreResourceNodes(ctx context.Context, chunkX, chunkY int32, resources []*resourceNodeV1.ResourceNode) error {
	s.logger.Debug("Storing resource nodes", "chunk_x", chunkX, "chunk_y", chunkY, "count", len(resources))

	// Get default world to get the WorldID
	defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default world: %w", err)
	}

	// Delete any existing resources for this chunk
	err = s.db.DeleteResourceNodesInChunk(ctx, db.DeleteResourceNodesInChunkParams{
		WorldID: defaultWorld.ID,
		ChunkX:  chunkX,
		ChunkY:  chunkY,
	})
	if err != nil {
		return fmt.Errorf("failed to delete existing resources: %w", err)
	}

	// Insert all new resources
	for _, resource := range resources {
		_, err := s.db.CreateResourceNode(ctx, db.CreateResourceNodeParams{
			ResourceNodeTypeID: int32(resource.ResourceNodeType.Id),
			WorldID:            defaultWorld.ID,
			ChunkX:             resource.ChunkX,
			ChunkY:             resource.ChunkY,
			ClusterID:          resource.ClusterId,
			PosX:               resource.PosX,
			PosY:               resource.PosY,
			Size:               resource.Size,
		})
		if err != nil {
			return fmt.Errorf("failed to create resource node: %w", err)
		}
	}

	return nil
}

// GetResourcesForChunk retrieves all resources in a chunk, generating them if they don't exist
func (s *NodeService) GetResourcesForChunk(ctx context.Context, chunkX, chunkY int32) ([]*resourceNodeV1.ResourceNode, error) {
	// Single debug log instead of both info and debug
	s.logger.Debug("Getting resource nodes for chunk", "chunk_x", chunkX, "chunk_y", chunkY)

	// Get default world to get the WorldID
	defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default world: %w", err)
	}

	// First, try to get existing resources from database
	dbResources, err := s.db.GetResourceNodesInChunk(ctx, db.GetResourceNodesInChunkParams{
		WorldID: defaultWorld.ID,
		ChunkX:  chunkX,
		ChunkY:  chunkY,
	})
	if err != nil {
		s.logger.Error("Failed to get resource nodes for chunk", "error", err, "chunk_x", chunkX, "chunk_y", chunkY)
		return nil, fmt.Errorf("failed to get resource nodes for chunk: %w", err)
	}

	// If resources exist, return them
	if len(dbResources) > 0 {
		s.logger.Debug("Found existing resource nodes in database", "count", len(dbResources))
		return s.convertDBResourcesToProto(dbResources), nil
	}

	// Check if chunk exists in database
	chunkExists, err := s.db.ChunkExists(ctx, db.ChunkExistsParams{
		WorldID: defaultWorld.ID,
		ChunkX:  chunkX,
		ChunkY:  chunkY,
	})
	if err != nil {
		s.logger.Error("Failed to check if chunk exists", "error", err)
		return nil, fmt.Errorf("failed to check if chunk exists: %w", err)
	}

	s.logger.Debug("Chunk exists check result", "chunk_exists", chunkExists)

	// If chunk doesn't exist, we can't generate resources without terrain data
	if !chunkExists {
		s.logger.Debug("Chunk does not exist, cannot generate resources without terrain data")
		return []*resourceNodeV1.ResourceNode{}, nil
	}

	// Chunk exists but has no resources - generate them
	s.logger.Debug("Generating resources for existing chunk")

	// Get chunk data to generate resources
	chunkData, err := s.getChunkDataForResourceGeneration(ctx, chunkX, chunkY)
	if err != nil {
		s.logger.Error("Failed to get chunk data for resource generation", "error", err)
		return nil, fmt.Errorf("failed to get chunk data: %w", err)
	}

	// Generate resources for this chunk
	// Logging already happens inside GenerateResourcesForChunk
	resources, err := s.GenerateResourcesForChunk(ctx, chunkData)
	if err != nil {
		s.logger.Error("Failed to generate resources for chunk", "error", err)
		return nil, fmt.Errorf("failed to generate resources: %w", err)
	}

	s.logger.Debug("Resource generation completed", "resource_count", len(resources))

	// Store the generated resources in database
	if len(resources) > 0 {
		err = s.StoreResourceNodes(ctx, chunkX, chunkY, resources)
		if err != nil {
			s.logger.Error("Failed to store generated resources", "error", err)
			// Don't fail the request if storage fails, just return the generated resources
		}
	}

	s.logger.Debug("Generated and stored resource nodes", "count", len(resources))
	return resources, nil
}

// GetResourcesForChunks retrieves all resources in multiple chunks
func (s *NodeService) GetResourcesForChunks(ctx context.Context, chunks []*chunkV1.ChunkCoordinate) ([]*resourceNodeV1.ResourceNode, error) {
	s.logger.Debug("Getting resource nodes for multiple chunks", "chunk_count", len(chunks))

	// Maximum of 5 chunks at a time with our current query
	if len(chunks) > 5 {
		s.logger.Warn("Too many chunks requested, limiting to 5", "requested", len(chunks))
		chunks = chunks[:5]
	}

	// Get default world to get the WorldID
	defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default world: %w", err)
	}

	// Fill in the parameters
	params := db.GetResourceNodesInChunksParams{
		WorldID: defaultWorld.ID,
	}

	// Add parameters for each chunk
	for i, chunk := range chunks {
		switch i {
		case 0:
			params.ChunkX = chunk.ChunkX
			params.ChunkY = chunk.ChunkY
		case 1:
			params.ChunkX_2 = chunk.ChunkX
			params.ChunkY_2 = chunk.ChunkY
		case 2:
			params.ChunkX_3 = chunk.ChunkX
			params.ChunkY_3 = chunk.ChunkY
		case 3:
			params.ChunkX_4 = chunk.ChunkX
			params.ChunkY_4 = chunk.ChunkY
		case 4:
			params.ChunkX_5 = chunk.ChunkX
			params.ChunkY_5 = chunk.ChunkY
		}
	}

	// If we have less than 5 chunks, fill the rest with invalid values
	for i := len(chunks); i < 5; i++ {
		switch i {
		case 0:
			params.ChunkX = -99999
			params.ChunkY = -99999
		case 1:
			params.ChunkX_2 = -99999
			params.ChunkY_2 = -99999
		case 2:
			params.ChunkX_3 = -99999
			params.ChunkY_3 = -99999
		case 3:
			params.ChunkX_4 = -99999
			params.ChunkY_4 = -99999
		case 4:
			params.ChunkX_5 = -99999
			params.ChunkY_5 = -99999
		}
	}

	dbResources, err := s.db.GetResourceNodesInChunks(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource nodes for chunks: %w", err)
	}

	return s.convertDBResourcesToProtoFromChunks(dbResources), nil
}

// GetResourcesInChunkRange retrieves all resources in a range of chunks
func (s *NodeService) GetResourcesInChunkRange(ctx context.Context, minX, maxX, minY, maxY int32) ([]*resourceNodeV1.ResourceNode, error) {
	// Get default world to get the WorldID
	defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default world: %w", err)
	}

	dbResources, err := s.db.GetResourceNodesInChunkRange(ctx, db.GetResourceNodesInChunkRangeParams{
		WorldID:  defaultWorld.ID,
		ChunkX:   minX,
		ChunkX_2: maxX,
		ChunkY:   minY,
		ChunkY_2: maxY,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get resource nodes in chunk range: %w", err)
	}

	return s.convertDBResourcesToProtoFromChunkRange(dbResources), nil
}

// convertDBResourcesToProto converts database resource nodes to protobuf format
func (s *NodeService) convertDBResourcesToProto(dbResources []db.ResourceNode) []*resourceNodeV1.ResourceNode {
	return s.convertResourceRows(dbResources)
}

// convertDBResourcesToProtoFromChunks converts database resource nodes from chunks query to protobuf format
func (s *NodeService) convertDBResourcesToProtoFromChunks(dbResources []db.ResourceNode) []*resourceNodeV1.ResourceNode {
	return s.convertResourceRows(dbResources)
}

// convertDBResourcesToProtoFromChunkRange converts database resource nodes from chunk range query to protobuf format
func (s *NodeService) convertDBResourcesToProtoFromChunkRange(dbResources []db.ResourceNode) []*resourceNodeV1.ResourceNode {
	return s.convertResourceRows(dbResources)
}

// convertResourceRows is the common conversion logic
func (s *NodeService) convertResourceRows(dbResources []db.ResourceNode) []*resourceNodeV1.ResourceNode {
	result := make([]*resourceNodeV1.ResourceNode, 0, len(dbResources))

	// Use the cached resource types map

	for _, r := range dbResources {
		// Get the hardcoded resource type from the cached map
		resourceNodeType, ok := s.resourceTypesByID[r.ResourceNodeTypeID]
		if !ok {
			// If type is not found, create a generic placeholder
			resourceNodeType = &resourceNodeV1.ResourceNodeType{
				Id:          r.ResourceNodeTypeID,
				Name:        fmt.Sprintf("Unknown Type %d", r.ResourceNodeTypeID),
				Description: "Unknown resource type",
				TerrainType: "unknown",
				Rarity:      resourceNodeV1.ResourceRarity_RESOURCE_RARITY_UNSPECIFIED,
				VisualData:  &resourceNodeV1.ResourceVisual{},
				Properties:  &resourceNodeV1.ResourceProperties{},
			}
		}

		// Convert timestamp
		var createdAt *timestamppb.Timestamp
		if r.CreatedAt.Valid {
			createdAt = timestamppb.New(r.CreatedAt.Time)
		}

		// Create the resource node
		node := &resourceNodeV1.ResourceNode{
			Id:                 r.ID,
			ResourceNodeTypeId: resourceNodeV1.ResourceNodeTypeId(resourceNodeType.Id),
			ResourceNodeType:   resourceNodeType,
			ChunkX:             r.ChunkX,
			ChunkY:             r.ChunkY,
			PosX:               r.PosX,
			PosY:               r.PosY,
			ClusterId:          r.ClusterID,
			Size:               r.Size,
			CreatedAt:          createdAt,
		}

		result = append(result, node)
	}

	return result
}

// getChunkDataForResourceGeneration retrieves chunk data from database for resource generation
func (s *NodeService) getChunkDataForResourceGeneration(ctx context.Context, chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
	// Get default world to get the WorldID
	defaultWorld, err := s.worldService.GetDefaultWorld(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default world: %w", err)
	}

	// Get chunk from database
	dbChunk, err := s.db.GetChunk(ctx, db.GetChunkParams{
		WorldID: defaultWorld.ID,
		ChunkX:  chunkX,
		ChunkY:  chunkY,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk from database: %w", err)
	}

	// Deserialize protobuf data
	var chunkData chunkV1.ChunkData
	err = proto.Unmarshal(dbChunk.ChunkData, &chunkData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize chunk data: %w", err)
	}

	return &chunkData, nil
}


// GetResourceNode retrieves a single resource node by ID
func (s *NodeService) GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error) {
	return s.db.GetResourceNode(ctx, id)
}
