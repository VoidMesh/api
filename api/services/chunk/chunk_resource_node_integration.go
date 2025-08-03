package chunk

import (
	"context"
	"fmt"
	"os"

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

	rgi.logger.Info("Generating resource nodes for chunk",
		"chunk_x", chunk.ChunkX,
		"chunk_y", chunk.ChunkY,
	)

	// Generate resource nodes for this chunk
	resources, err := rgi.resourceNodeService.GenerateResourcesForChunk(ctx, chunk)
	if err != nil {
		rgi.logger.Error("Failed to generate resource nodes", "error", err)
		return err
	}
	print("Generated ", len(resources), " resource nodes for chunk (", chunk.ChunkX, ",", chunk.ChunkY, ")\n")

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
	print("Retrieved ", len(resources), " resource nodes for chunk (", chunk.ChunkX, ",", chunk.ChunkY, ") from database\n")

	rgi.logger.Debug("Attached resource nodes to chunk", "count", len(resources))

	// Attach resource nodes to the chunk data
	chunk.ResourceNodes = resources

	return nil
}

// AttachResourceNodesToChunks attaches existing resource nodes to multiple chunks
func (rgi *ResourceNodeGeneratorIntegration) AttachResourceNodesToChunks(ctx context.Context, chunks []*chunkV1.ChunkData) error {
	if len(chunks) == 0 {
		rgi.logger.Debug("No chunks provided, skipping resource node attachment")
		return nil
	}

	rgi.logger.Debug("Attaching resource nodes to multiple chunks", "chunk_count", len(chunks))

	// Extract chunk coordinates
	coordinates := make([]*chunkV1.ChunkCoordinate, len(chunks))
	chunkMap := make(map[string]*chunkV1.ChunkData)

	for i, chunk := range chunks {
		coordinates[i] = &chunkV1.ChunkCoordinate{
			ChunkX: chunk.ChunkX,
			ChunkY: chunk.ChunkY,
		}

		// Create a map key for each chunk
		key := coordKey(chunk.ChunkX, chunk.ChunkY)
		chunkMap[key] = chunk
	}

	// Process chunks in batches of 5 (limitation of our SQL query)
	for i := 0; i < len(coordinates); i += 5 {
		end := i + 5
		if end > len(coordinates) {
			end = len(coordinates)
		}

		batch := coordinates[i:end]

		// Get resource nodes for this batch of chunks
		resources, err := rgi.resourceNodeService.GetResourcesForChunks(ctx, batch)
		if err != nil {
			rgi.logger.Error("Failed to get resource nodes for chunk batch", "error", err)
			return err
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
	}

	rgi.logger.Debug("Finished attaching resource nodes to chunks")

	return nil
}

// coordKey generates a string key from chunk coordinates
func coordKey(x, y int32) string {
	return fmt.Sprintf("%d:%d", x, y)
}

// GetResourceNodeService returns the resource node service
func (rgi *ResourceNodeGeneratorIntegration) GetResourceNodeService() *resource_node.NodeService {
	return rgi.resourceNodeService
}
