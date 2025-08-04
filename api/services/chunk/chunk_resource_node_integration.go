package chunk

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ResourceNodeGeneratorIntegration provides integration between the chunk and resource node services
type ResourceNodeGeneratorIntegration struct {
	resourceNodeService *resource_node.NodeService
	db                  *pgxpool.Pool
	logger              *log.Logger
}

// NewResourceNodeGeneratorIntegration creates a new resource generator integration
func NewResourceNodeGeneratorIntegration(db *pgxpool.Pool, noiseGen *noise.Generator, worldService *world.Service) *ResourceNodeGeneratorIntegration {
	resourceNodeService := resource_node.NewNodeService(db, noiseGen, worldService)

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Prefix:          "resource-generator",
	})

	return &ResourceNodeGeneratorIntegration{
		resourceNodeService: resourceNodeService,
		db:                  db,
		logger:              logger,
	}
}

// GenerateAndAttachResourceNodes generates resource nodes for a chunk and attaches them
func (rgi *ResourceNodeGeneratorIntegration) GenerateAndAttachResourceNodes(ctx context.Context, chunk *chunkV1.ChunkData) error {
	if chunk == nil {
		rgi.logger.Error("Cannot generate resource nodes for nil chunk")
		return fmt.Errorf("nil chunk provided")
	}

	rgi.logger.Debug("Generating resource nodes for chunk",
		"chunk_x", chunk.ChunkX,
		"chunk_y", chunk.ChunkY,
	)

	// Generate resource nodes for this chunk
	resources, err := rgi.resourceNodeService.GenerateResourcesForChunk(ctx, chunk)
	if err != nil {
		rgi.logger.Error("Failed to generate resource nodes", "error", err)
		return err
	}
	rgi.logger.Debug("Generated resource nodes for chunk", "count", len(resources))

	// Store resource nodes in the database
	err = rgi.resourceNodeService.StoreResourceNodes(ctx, chunk.ChunkX, chunk.ChunkY, resources)
	if err != nil {
		rgi.logger.Error("Failed to store resource nodes", "error", err)
		return err
	}

	// Attach resource nodes to the chunk data
	chunk.ResourceNodes = resources

	return nil
}

// AttachResourceNodesToChunk attaches existing resource nodes to a chunk
func (rgi *ResourceNodeGeneratorIntegration) AttachResourceNodesToChunk(ctx context.Context, chunk *chunkV1.ChunkData) error {
	if chunk == nil {
		rgi.logger.Error("Cannot attach resource nodes to nil chunk")
		return fmt.Errorf("nil chunk provided")
	}

	rgi.logger.Debug("Attaching resource nodes to chunk",
		"chunk_x", chunk.ChunkX,
		"chunk_y", chunk.ChunkY,
	)

	// Get resource nodes for this chunk from the database
	resources, err := rgi.resourceNodeService.GetResourcesForChunk(ctx, chunk.ChunkX, chunk.ChunkY)
	if err != nil {
		rgi.logger.Error("Failed to get resource nodes for chunk", "error", err)
		return err
	}
	rgi.logger.Debug("Retrieved and attached resource nodes", "count", len(resources))

	// Attach resource nodes to the chunk data
	chunk.ResourceNodes = resources

	return nil
}

// AttachResourceNodesToChunks attaches existing resource nodes to multiple chunks
func (rgi *ResourceNodeGeneratorIntegration) AttachResourceNodesToChunks(ctx context.Context, chunks []*chunkV1.ChunkData) error {
	// Skip processing if no chunks
	if len(chunks) == 0 {
		return nil
	}

	// Pre-allocate arrays and maps to reduce allocations
	chunkCount := len(chunks)
	rgi.logger.Debug("Attaching resource nodes to chunks", "chunk_count", chunkCount)
	
	// Build chunk map for lookups
	chunkMap := make(map[string]*chunkV1.ChunkData, chunkCount)
	
	// Track min/max coordinates to determine if we should use range query
	var minX, maxX, minY, maxY int32
	
	// Initialize with first chunk's coordinates
	if len(chunks) > 0 {
		minX, maxX = chunks[0].ChunkX, chunks[0].ChunkX
		minY, maxY = chunks[0].ChunkY, chunks[0].ChunkY
	}
	
	// Build chunk map and find min/max coordinates
	for _, chunk := range chunks {
		// Update min/max coordinates
		if chunk.ChunkX < minX {
			minX = chunk.ChunkX
		}
		if chunk.ChunkX > maxX {
			maxX = chunk.ChunkX
		}
		if chunk.ChunkY < minY {
			minY = chunk.ChunkY
		}
		if chunk.ChunkY > maxY {
			maxY = chunk.ChunkY
		}
		
		// Use a more efficient key generation
		key := coordKey(chunk.ChunkX, chunk.ChunkY)
		chunkMap[key] = chunk
	}

	// Calculate area metrics
	width := maxX - minX + 1
	height := maxY - minY + 1
	area := width * height
	
	// Determine if we should use range query or batch query
	// Use range query if:
	// 1. There are more than 10 chunks OR
	// 2. The chunks form a compact area (more than 75% of the bounding box is filled)
	useRangeQuery := len(chunks) > 10 || (area > 0 && float32(len(chunks))/float32(area) > 0.75)
	
	var resources []*resourceNodeV1.ResourceNode
	var err error
	
	if useRangeQuery {
		// Use range query for efficiency with many chunks
		rgi.logger.Debug("Using range query for resource nodes", 
			"chunk_count", len(chunks),
			"bounds", fmt.Sprintf("%d,%d to %d,%d", minX, minY, maxX, maxY))
			
		resources, err = rgi.resourceNodeService.GetResourcesInChunkRange(ctx, minX, maxX, minY, maxY)
		if err != nil {
			rgi.logger.Error("Failed to get resource nodes in chunk range", "error", err)
			return err
		}
	} else {
		// For fewer chunks, process in batches using the existing method
		// Extract chunk coordinates for batch processing
		coordinates := make([]*chunkV1.ChunkCoordinate, len(chunks))
		for i, chunk := range chunks {
			coordinates[i] = &chunkV1.ChunkCoordinate{
				ChunkX: chunk.ChunkX,
				ChunkY: chunk.ChunkY,
			}
		}
		
		// Process chunks in batches of 5 (limitation of our SQL query)
		for i := 0; i < len(coordinates); i += 5 {
			end := min(i + 5, len(coordinates))

			batch := coordinates[i:end]

			// Get resource nodes for this batch of chunks
			batchResources, err := rgi.resourceNodeService.GetResourcesForChunks(ctx, batch)
			if err != nil {
				rgi.logger.Error("Failed to get resource nodes for chunk batch", "error", err)
				return err
			}
			
			// Append batch results to overall resources
			resources = append(resources, batchResources...)
		}
	}

	// Group resource nodes by chunk
	resourcesByChunk := make(map[string][]*resourceNodeV1.ResourceNode)
	for _, resource := range resources {
		key := coordKey(resource.ChunkX, resource.ChunkY)
		resourcesByChunk[key] = append(resourcesByChunk[key], resource)
	}

	// Attach resource nodes to their respective chunks
	for key, chunkResources := range resourcesByChunk {
		if chunk, ok := chunkMap[key]; ok {
			chunk.ResourceNodes = chunkResources
		}
	}

	rgi.logger.Debug("Finished attaching resource nodes to chunks", "total_resources", len(resources))

	return nil
}

// coordKey generates a string key from chunk coordinates
// Uses strconv for more efficient string conversion than fmt.Sprintf
func coordKey(x, y int32) string {
	// Use a simple string builder to avoid allocation and formatting overhead
	var sb strings.Builder
	
	// Pre-allocate enough space for most coordinate strings
	sb.Grow(16)
	
	// Convert and concatenate directly
	sb.WriteString(strconv.FormatInt(int64(x), 10))
	sb.WriteByte(':')
	sb.WriteString(strconv.FormatInt(int64(y), 10))
	
	return sb.String()
}

// GetResourceNodeService returns the resource node service
func (rgi *ResourceNodeGeneratorIntegration) GetResourceNodeService() *resource_node.NodeService {
	return rgi.resourceNodeService
}
