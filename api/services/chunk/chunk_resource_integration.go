package chunk

import (
	"context"
	"fmt"
	"os"

	"github.com/VoidMesh/api/api/db"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceV1 "github.com/VoidMesh/api/api/proto/resource/v1"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/resource"
	"github.com/charmbracelet/log"
)

// ResourceGeneratorIntegration provides integration between the chunk and resource services
type ResourceGeneratorIntegration struct {
	resourceService *resource.Service
	db              db.DBTX
	logger          *log.Logger
}

// NewResourceGeneratorIntegration creates a new resource generator integration
func NewResourceGeneratorIntegration(db db.DBTX, noiseGen *noise.Generator) *ResourceGeneratorIntegration {
	resourceService := resource.NewService(db, noiseGen)

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Prefix:          "resource-generator",
	})

	return &ResourceGeneratorIntegration{
		resourceService: resourceService,
		db:              db,
		logger:          logger,
	}
}

// GenerateAndAttachResources generates resources for a chunk and attaches them
func (rgi *ResourceGeneratorIntegration) GenerateAndAttachResources(ctx context.Context, chunk *chunkV1.ChunkData) error {
	if chunk == nil {
		rgi.logger.Error("Cannot generate resources for nil chunk")
		return fmt.Errorf("nil chunk provided")
	}

	rgi.logger.Info("Generating resources for chunk",
		"chunk_x", chunk.ChunkX,
		"chunk_y", chunk.ChunkY,
	)

	// Generate resources for this chunk
	resources, err := rgi.resourceService.GenerateResourcesForChunk(ctx, chunk)
	if err != nil {
		rgi.logger.Error("Failed to generate resources", "error", err)
		return err
	}

	rgi.logger.Debug("Generated resources for chunk", "count", len(resources))

	// Store resources in the database
	err = rgi.resourceService.StoreResourceNodes(ctx, chunk.ChunkX, chunk.ChunkY, resources)
	if err != nil {
		rgi.logger.Error("Failed to store resource nodes", "error", err)
		return err
	}

	// Attach resources to the chunk data
	chunk.Resources = resources

	return nil
}

// AttachResourcesToChunk attaches existing resources to a chunk
func (rgi *ResourceGeneratorIntegration) AttachResourcesToChunk(ctx context.Context, chunk *chunkV1.ChunkData) error {
	if chunk == nil {
		rgi.logger.Error("Cannot attach resources to nil chunk")
		return fmt.Errorf("nil chunk provided")
	}

	rgi.logger.Debug("Attaching resources to chunk",
		"chunk_x", chunk.ChunkX,
		"chunk_y", chunk.ChunkY,
	)

	// Get resources for this chunk from the database
	resources, err := rgi.resourceService.GetResourcesForChunk(ctx, chunk.ChunkX, chunk.ChunkY)
	if err != nil {
		rgi.logger.Error("Failed to get resources for chunk", "error", err)
		return err
	}

	rgi.logger.Debug("Attached resources to chunk", "count", len(resources))

	// Attach resources to the chunk data
	chunk.Resources = resources

	return nil
}

// AttachResourcesToChunks attaches existing resources to multiple chunks
func (rgi *ResourceGeneratorIntegration) AttachResourcesToChunks(ctx context.Context, chunks []*chunkV1.ChunkData) error {
	if len(chunks) == 0 {
		rgi.logger.Debug("No chunks provided, skipping resource attachment")
		return nil
	}

	rgi.logger.Debug("Attaching resources to multiple chunks", "chunk_count", len(chunks))

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

		// Get resources for this batch of chunks
		resources, err := rgi.resourceService.GetResourcesForChunks(ctx, batch)
		if err != nil {
			rgi.logger.Error("Failed to get resources for chunk batch", "error", err)
			return err
		}

		// Group resources by chunk
		resourcesByChunk := make(map[string][]*resourceV1.ResourceNode)
		for _, resource := range resources {
			key := coordKey(resource.ChunkX, resource.ChunkY)
			resourcesByChunk[key] = append(resourcesByChunk[key], resource)
		}

		// Attach resources to their respective chunks
		for key, chunkResources := range resourcesByChunk {
			if chunk, ok := chunkMap[key]; ok {
				chunk.Resources = chunkResources
			}
		}
	}

	rgi.logger.Debug("Finished attaching resources to chunks")

	return nil
}

// coordKey generates a string key from chunk coordinates
func coordKey(x, y int32) string {
	return fmt.Sprintf("%d:%d", x, y)
}

// GetResourceService returns the resource service
func (rgi *ResourceGeneratorIntegration) GetResourceService() *resource.Service {
	return rgi.resourceService
}
