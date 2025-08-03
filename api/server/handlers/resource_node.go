package handlers

import (
	"context"
	"os"

	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ResourceNodeHandler implements the ResourceNodeService gRPC service
type ResourceNodeHandler struct {
	resourceNodeV1.UnimplementedResourceNodeServiceServer
	resourceNodeService *resource_node.NodeService
	worldService        *world.Service
	logger              *log.Logger
}

// NewResourceNodeHandler creates a new resource node handler
func NewResourceNodeHandler(resourceNodeService *resource_node.NodeService, worldService *world.Service) *ResourceNodeHandler {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Prefix:          "resource-node-handler",
	})

	return &ResourceNodeHandler{
		resourceNodeService: resourceNodeService,
		worldService:        worldService,
		logger:              logger,
	}
}

// GetResourcesInChunk retrieves all resource nodes in a specific chunk
func (h *ResourceNodeHandler) GetResourcesInChunk(ctx context.Context, req *resourceNodeV1.GetResourcesInChunkRequest) (*resourceNodeV1.GetResourcesInChunkResponse, error) {
	h.logger.Info("RESOURCE NODE HANDLER: GetResourcesInChunk called",
		"world_id", req.WorldId,
		"chunk_x", req.ChunkX,
		"chunk_y", req.ChunkY,
	)

	// Convert world_id to pgtype.UUID
	var worldID pgtype.UUID
	if len(req.WorldId) == 0 {
		// If world_id is not provided, use default world
		defaultWorld, err := h.worldService.GetDefaultWorld(ctx)
		if err != nil {
			h.logger.Error("Failed to get default world", "error", err)
			return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	} else {
		err := worldID.Scan(req.WorldId)
		if err != nil {
			h.logger.Error("Invalid world ID", "error", err)
			return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	}

	h.logger.Debug("Getting resource nodes in chunk",
		"world_id", worldID,
		"chunk_x", req.ChunkX,
		"chunk_y", req.ChunkY,
	)

	resources, err := h.resourceNodeService.GetResourcesForChunk(ctx, req.ChunkX, req.ChunkY)
	if err != nil {
		h.logger.Error("Failed to get resource nodes for chunk", "error", err)
		return nil, err
	}

	h.logger.Debug("Retrieved resource nodes for chunk", "count", len(resources))
	return &resourceNodeV1.GetResourcesInChunkResponse{
		Resources: resources,
	}, nil
}

// GetResourcesInChunks retrieves resource nodes in multiple chunks
func (h *ResourceNodeHandler) GetResourcesInChunks(ctx context.Context, req *resourceNodeV1.GetResourcesInChunksRequest) (*resourceNodeV1.GetResourcesInChunksResponse, error) {
	h.logger.Debug("Getting resource nodes in multiple chunks",
		"chunk_count", len(req.Coordinates),
	)

	// Determine world ID to use
	var worldID pgtype.UUID

	// Get the worldID from the first coordinate (all should be the same world)
	if len(req.Coordinates) > 0 && len(req.Coordinates[0].WorldId) > 0 {
		err := worldID.Scan(req.Coordinates[0].WorldId)
		if err != nil {
			h.logger.Error("Invalid world ID", "error", err)
			return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
		}
	} else {
		// If world_id is not provided, use default world
		defaultWorld, err := h.worldService.GetDefaultWorld(ctx)
		if err != nil {
			h.logger.Error("Failed to get default world", "error", err)
			return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
		}
		worldID = defaultWorld.ID
	}

	// Convert resource coordinates to chunk coordinates
	chunkCoords := make([]*chunkV1.ChunkCoordinate, len(req.Coordinates))
	for i, coord := range req.Coordinates {
		chunkCoords[i] = &chunkV1.ChunkCoordinate{
			ChunkX: coord.ChunkX,
			ChunkY: coord.ChunkY,
		}
	}

	resources, err := h.resourceNodeService.GetResourcesForChunks(ctx, chunkCoords)
	if err != nil {
		h.logger.Error("Failed to get resource nodes for chunks", "error", err)
		return nil, err
	}

	h.logger.Debug("Retrieved resource nodes for chunks", "count", len(resources))
	return &resourceNodeV1.GetResourcesInChunksResponse{
		Resources: resources,
	}, nil
}

// GetResourceNodeTypes returns all available resource node types
func (h *ResourceNodeHandler) GetResourceNodeTypes(ctx context.Context, req *resourceNodeV1.GetResourceNodeTypesRequest) (*resourceNodeV1.GetResourceNodeTypesResponse, error) {
	h.logger.Debug("Getting all resource node types")

	resourceNodeTypes, err := h.resourceNodeService.GetResourceNodeTypes(ctx)
	if err != nil {
		h.logger.Error("Failed to get resource node types", "error", err)
		return nil, err
	}

	h.logger.Debug("Retrieved resource node types", "count", len(resourceNodeTypes))
	return &resourceNodeV1.GetResourceNodeTypesResponse{
		ResourceNodeTypes: resourceNodeTypes,
	}, nil
}
