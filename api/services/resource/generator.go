package resource

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/VoidMesh/api/api/db"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceV1 "github.com/VoidMesh/api/api/proto/resource/v1"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/charmbracelet/log"
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

// Service provides resource generation functionality
type Service struct {
	db       db.DBTX
	noiseGen *noise.Generator
	rnd      *rand.Rand
	logger   *log.Logger
}

// NewService creates a new resource service
func NewService(db db.DBTX, noiseGen *noise.Generator) *Service {
	// Create a deterministic random source based on the noise generator's seed
	source := rand.NewSource(noiseGen.GetSeed())

	// Create logger
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Prefix:          "resource-service",
	})

	return &Service{
		db:       db,
		noiseGen: noiseGen,
		rnd:      rand.New(source),
		logger:   logger,
	}
}

// GenerateResourcesForChunk generates resource nodes for a chunk
func (s *Service) GenerateResourcesForChunk(ctx context.Context, chunk *chunkV1.ChunkData) ([]*resourceV1.ResourceNode, error) {
	s.logger.Debug("Generating resources for chunk", "chunk_x", chunk.ChunkX, "chunk_y", chunk.ChunkY)

	// Get all resource types from the database
	dbResources, err := db.New(s.db).ListResourceTypes(ctx)
	if err != nil {
		s.logger.Error("Failed to list resource types", "error", err)
		return nil, fmt.Errorf("failed to list resource types: %w", err)
	}

	// Group resource types by terrain
	resourcesByTerrain := make(map[string][]db.ResourceType)
	for _, r := range dbResources {
		terrainType := r.TerrainType
		resourcesByTerrain[terrainType] = append(resourcesByTerrain[terrainType], r)
	}

	// Map to track occupied positions
	occupiedPositions := make(map[string]bool)

	// Map to track cluster centers for minimum distance check
	clusterCenters := make([]struct{ x, y int32 }, 0)

	// List to collect all generated resources
	var resources []*resourceV1.ResourceNode

	// Process each resource type for this chunk
	for terrainType, resourceTypes := range resourcesByTerrain {
		for _, resourceType := range resourceTypes {
			// Create a separate noise map for this resource type
			// Use resource ID as additional seed to make different resources spawn in different patterns
			resourceSeed := s.noiseGen.GetSeed() + int64(resourceType.ID)
			resourceRng := rand.New(rand.NewSource(resourceSeed))

			// Generate potential spawn points
			spawnPoints := s.findPotentialSpawnPoints(
				chunk,
				terrainType,
				s.getRarityThreshold(resourceType.Rarity),
				resourceSeed,
			)

			// Shuffle spawn points to avoid patterns
			resourceRng.Shuffle(len(spawnPoints), func(i, j int) {
				spawnPoints[i], spawnPoints[j] = spawnPoints[j], spawnPoints[i]
			})

			// Try to create clusters from the spawn points
			for _, point := range spawnPoints {
				// Check if we've reached the max resources per chunk
				if len(resources) >= MaxResourcesPerChunk {
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
				clusterID := generateClusterID(chunk.ChunkX, chunk.ChunkY, point.x, point.y, resourceType.ID)

				// Determine cluster size
				clusterSize := s.determineClusterSize(resourceType.Rarity)

				// Create the first resource node at the center point
				posKey := fmt.Sprintf("%d,%d", point.x, point.y)
				if !occupiedPositions[posKey] {
					// Create the resource node
					resourceNode := &resourceV1.ResourceNode{
						ResourceType: s.convertResourceType(resourceType),
						ChunkX:       chunk.ChunkX,
						ChunkY:       chunk.ChunkY,
						PosX:         point.x,
						PosY:         point.y,
						ClusterId:    clusterID,
						Size:         1,
					}
					resources = append(resources, resourceNode)
					occupiedPositions[posKey] = true

					// Generate additional nodes in the cluster
					s.generateClusterNodes(
						chunk,
						resourceNode,
						clusterSize-1, // Subtract 1 since we already created the center node
						occupiedPositions,
						terrainType,
						&resources,
					)
				}
			}
		}
	}

	return resources, nil
}

// findPotentialSpawnPoints finds potential resource spawn points in a chunk
func (s *Service) findPotentialSpawnPoints(
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
func (s *Service) generateClusterNodes(
	chunk *chunkV1.ChunkData,
	centerNode *resourceV1.ResourceNode,
	numNodes int,
	occupiedPositions map[string]bool,
	terrainType string,
	resources *[]*resourceV1.ResourceNode,
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
		resourceNode := &resourceV1.ResourceNode{
			ResourceType: centerNode.ResourceType,
			ChunkX:       centerNode.ChunkX,
			ChunkY:       centerNode.ChunkY,
			PosX:         newX,
			PosY:         newY,
			ClusterId:    centerNode.ClusterId,
			Size:         1,
			CreatedAt:    timestamppb.Now(),
		}

		// Add to the resources slice
		*resources = append(*resources, resourceNode)

		// Mark position as occupied
		occupiedPositions[posKey] = true
		nodesCreated++
	}
}

// isNearTerrainTransition checks if a cell is near a terrain transition
func (s *Service) isNearTerrainTransition(chunk *chunkV1.ChunkData, x, y int32) bool {
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
func (s *Service) determineClusterSize(rarity string) int {
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

// convertResourceType converts a DB resource type to a protobuf resource type
func (s *Service) convertResourceType(dbResource db.ResourceType) *resourceV1.ResourceType {
	protoResource := &resourceV1.ResourceType{
		Id:          int32(dbResource.ID),
		Name:        dbResource.Name,
		Description: dbResource.Description.String,
		TerrainType: dbResource.TerrainType,
		Rarity:      s.stringToRarity(dbResource.Rarity),
		VisualData:  make(map[string]string),
		Properties:  make(map[string]string),
	}

	// Parse visual data JSON
	if len(dbResource.VisualData) > 0 {
		var visualData map[string]interface{}
		if err := json.Unmarshal(dbResource.VisualData, &visualData); err == nil {
			for k, v := range visualData {
				protoResource.VisualData[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Parse properties JSON
	if len(dbResource.Properties) > 0 {
		var properties map[string]interface{}
		if err := json.Unmarshal(dbResource.Properties, &properties); err == nil {
			for k, v := range properties {
				protoResource.Properties[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return protoResource
}

// stringToRarity converts a string rarity to the protobuf enum
func (s *Service) stringToRarity(rarity string) resourceV1.ResourceRarity {
	switch strings.ToLower(rarity) {
	case "common":
		return resourceV1.ResourceRarity_RESOURCE_RARITY_COMMON
	case "uncommon":
		return resourceV1.ResourceRarity_RESOURCE_RARITY_UNCOMMON
	case "rare":
		return resourceV1.ResourceRarity_RESOURCE_RARITY_RARE
	case "very_rare":
		return resourceV1.ResourceRarity_RESOURCE_RARITY_VERY_RARE
	default:
		return resourceV1.ResourceRarity_RESOURCE_RARITY_UNSPECIFIED
	}
}

// terrainTypeToString converts a terrain type enum to a string
func (s *Service) terrainTypeToString(terrainType chunkV1.TerrainType) string {
	// Just return the enum string name directly, which matches the database schema
	return terrainType.String()
}

// getRarityThreshold returns the noise threshold for a given rarity
func (s *Service) getRarityThreshold(rarity string) float64 {
	switch strings.ToLower(rarity) {
	case "common":
		return CommonThreshold
	case "uncommon":
		return UncommonThreshold
	case "rare":
		return RareThreshold
	case "very_rare":
		return VeryRareThreshold
	default:
		return CommonThreshold
	}
}

// generateClusterID generates a unique ID for a resource cluster
func generateClusterID(chunkX, chunkY, posX, posY int32, resourceTypeID int32) string {
	// Create a unique string based on position and resource type
	input := fmt.Sprintf("%d:%d:%d:%d:%d:%d", chunkX, chunkY, posX, posY, resourceTypeID, time.Now().UnixNano())

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
func (s *Service) StoreResourceNodes(ctx context.Context, chunkX, chunkY int32, resources []*resourceV1.ResourceNode) error {
	s.logger.Debug("Storing resource nodes", "chunk_x", chunkX, "chunk_y", chunkY, "count", len(resources))

	// Note: DBTX interface doesn't have Begin method
	// We'll work directly with the database connection
	queries := db.New(s.db)

	// Delete any existing resources for this chunk
	err := queries.DeleteResourceNodesInChunk(ctx, db.DeleteResourceNodesInChunkParams{
		ChunkX: chunkX,
		ChunkY: chunkY,
	})
	if err != nil {
		return fmt.Errorf("failed to delete existing resources: %w", err)
	}

	// Insert all new resources
	for _, resource := range resources {
		_, err := queries.CreateResourceNode(ctx, db.CreateResourceNodeParams{
			ResourceTypeID: int32(resource.ResourceType.Id),
			ChunkX:         resource.ChunkX,
			ChunkY:         resource.ChunkY,
			ClusterID:      resource.ClusterId,
			PosX:           resource.PosX,
			PosY:           resource.PosY,
			Size:           resource.Size,
		})
		if err != nil {
			return fmt.Errorf("failed to create resource node: %w", err)
		}
	}

	return nil
}

// GetResourcesForChunk retrieves all resources in a chunk
func (s *Service) GetResourcesForChunk(ctx context.Context, chunkX, chunkY int32) ([]*resourceV1.ResourceNode, error) {
	s.logger.Debug("Getting resources for chunk", "chunk_x", chunkX, "chunk_y", chunkY)

	dbResources, err := db.New(s.db).GetResourceNodesInChunk(ctx, db.GetResourceNodesInChunkParams{
		ChunkX: chunkX,
		ChunkY: chunkY,
	})
	if err != nil {
		s.logger.Error("Failed to get resources for chunk", "error", err, "chunk_x", chunkX, "chunk_y", chunkY)
		return nil, fmt.Errorf("failed to get resources for chunk: %w", err)
	}

	return s.convertDBResourcesToProto(dbResources), nil
}

// GetResourcesForChunks retrieves all resources in multiple chunks
func (s *Service) GetResourcesForChunks(ctx context.Context, chunks []*chunkV1.ChunkCoordinate) ([]*resourceV1.ResourceNode, error) {
	s.logger.Debug("Getting resources for multiple chunks", "chunk_count", len(chunks))

	// Maximum of 5 chunks at a time with our current query
	if len(chunks) > 5 {
		s.logger.Warn("Too many chunks requested, limiting to 5", "requested", len(chunks))
		chunks = chunks[:5]
	}

	// Fill in the parameters
	params := db.GetResourceNodesInChunksParams{}

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

	dbResources, err := db.New(s.db).GetResourceNodesInChunks(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources for chunks: %w", err)
	}

	return s.convertDBResourcesToProtoFromChunks(dbResources), nil
}

// GetResourcesInChunkRange retrieves all resources in a range of chunks
func (s *Service) GetResourcesInChunkRange(ctx context.Context, minX, maxX, minY, maxY int32) ([]*resourceV1.ResourceNode, error) {
	dbResources, err := db.New(s.db).GetResourceNodesInChunkRange(ctx, db.GetResourceNodesInChunkRangeParams{
		ChunkX:   minX,
		ChunkX_2: maxX,
		ChunkY:   minY,
		ChunkY_2: maxY,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get resources in chunk range: %w", err)
	}

	return s.convertDBResourcesToProtoFromChunkRange(dbResources), nil
}

// convertDBResourcesToProto converts database resource nodes to protobuf format
func (s *Service) convertDBResourcesToProto(dbResources []db.GetResourceNodesInChunkRow) []*resourceV1.ResourceNode {
	return s.convertResourceRows(dbResources)
}

// convertDBResourcesToProtoFromChunks converts database resource nodes from chunks query to protobuf format
func (s *Service) convertDBResourcesToProtoFromChunks(dbResources []db.GetResourceNodesInChunksRow) []*resourceV1.ResourceNode {
	// Convert to common format
	commonRows := make([]db.GetResourceNodesInChunkRow, len(dbResources))
	for i, r := range dbResources {
		commonRows[i] = db.GetResourceNodesInChunkRow{
			ID:             r.ID,
			ResourceTypeID: r.ResourceTypeID,
			ChunkX:         r.ChunkX,
			ChunkY:         r.ChunkY,
			ClusterID:      r.ClusterID,
			PosX:           r.PosX,
			PosY:           r.PosY,
			Size:           r.Size,
			CreatedAt:      r.CreatedAt,
			ResourceName:   r.ResourceName,
			TerrainType:    r.TerrainType,
			Rarity:         r.Rarity,
		}
	}
	return s.convertResourceRows(commonRows)
}

// convertDBResourcesToProtoFromChunkRange converts database resource nodes from chunk range query to protobuf format
func (s *Service) convertDBResourcesToProtoFromChunkRange(dbResources []db.GetResourceNodesInChunkRangeRow) []*resourceV1.ResourceNode {
	// Convert to common format
	commonRows := make([]db.GetResourceNodesInChunkRow, len(dbResources))
	for i, r := range dbResources {
		commonRows[i] = db.GetResourceNodesInChunkRow{
			ID:             r.ID,
			ResourceTypeID: r.ResourceTypeID,
			ChunkX:         r.ChunkX,
			ChunkY:         r.ChunkY,
			ClusterID:      r.ClusterID,
			PosX:           r.PosX,
			PosY:           r.PosY,
			Size:           r.Size,
			CreatedAt:      r.CreatedAt,
			ResourceName:   r.ResourceName,
			TerrainType:    r.TerrainType,
			Rarity:         r.Rarity,
		}
	}
	return s.convertResourceRows(commonRows)
}

// convertResourceRows is the common conversion logic
func (s *Service) convertResourceRows(dbResources []db.GetResourceNodesInChunkRow) []*resourceV1.ResourceNode {
	result := make([]*resourceV1.ResourceNode, 0, len(dbResources))

	for _, r := range dbResources {
		// Convert rarity string to enum
		rarityEnum := s.stringToRarity(r.Rarity)

		// Create the resource type
		resourceType := &resourceV1.ResourceType{
			Id:          r.ResourceTypeID,
			Name:        r.ResourceName,
			Description: "", // Not included in the join query
			TerrainType: r.TerrainType,
			Rarity:      rarityEnum,
			VisualData:  make(map[string]string),
			Properties:  make(map[string]string),
		}

		// Convert timestamp
		var createdAt *timestamppb.Timestamp
		if r.CreatedAt.Valid {
			createdAt = timestamppb.New(r.CreatedAt.Time)
		}

		// Create the resource node
		node := &resourceV1.ResourceNode{
			Id:           r.ID,
			ResourceType: resourceType,
			ChunkX:       r.ChunkX,
			ChunkY:       r.ChunkY,
			PosX:         r.PosX,
			PosY:         r.PosY,
			ClusterId:    r.ClusterID,
			Size:         r.Size,
			CreatedAt:    createdAt,
		}

		result = append(result, node)
	}

	return result
}
