package handlers

import (
	"context"
	"os"

	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceV1 "github.com/VoidMesh/api/api/proto/resource/v1"
	"github.com/VoidMesh/api/api/services/resource"
	"github.com/charmbracelet/log"
)

// ResourceHandler implements the ResourceService gRPC service
type ResourceHandler struct {
	resourceV1.UnimplementedResourceServiceServer
	resourceService *resource.Service
	logger          *log.Logger
}

// NewResourceHandler creates a new resource handler
func NewResourceHandler(resourceService *resource.Service) *ResourceHandler {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		Prefix:          "resource-handler",
	})

	return &ResourceHandler{
		resourceService: resourceService,
		logger:          logger,
	}
}

// GetResourcesInChunk retrieves all resources in a specific chunk
func (h *ResourceHandler) GetResourcesInChunk(ctx context.Context, req *resourceV1.GetResourcesInChunkRequest) (*resourceV1.GetResourcesInChunkResponse, error) {
	h.logger.Debug("Getting resources in chunk",
		"chunk_x", req.ChunkX,
		"chunk_y", req.ChunkY,
	)

	resources, err := h.resourceService.GetResourcesForChunk(ctx, req.ChunkX, req.ChunkY)
	if err != nil {
		h.logger.Error("Failed to get resources for chunk", "error", err)
		return nil, err
	}

	h.logger.Debug("Retrieved resources for chunk", "count", len(resources))
	return &resourceV1.GetResourcesInChunkResponse{
		Resources: resources,
	}, nil
}

// GetResourcesInChunks retrieves resources in multiple chunks
func (h *ResourceHandler) GetResourcesInChunks(ctx context.Context, req *resourceV1.GetResourcesInChunksRequest) (*resourceV1.GetResourcesInChunksResponse, error) {
	h.logger.Debug("Getting resources in multiple chunks",
		"chunk_count", len(req.Coordinates),
	)

	// Convert resource coordinates to chunk coordinates
	chunkCoords := make([]*chunkV1.ChunkCoordinate, len(req.Coordinates))
	for i, coord := range req.Coordinates {
		chunkCoords[i] = &chunkV1.ChunkCoordinate{
			ChunkX: coord.ChunkX,
			ChunkY: coord.ChunkY,
		}
	}

	resources, err := h.resourceService.GetResourcesForChunks(ctx, chunkCoords)
	if err != nil {
		h.logger.Error("Failed to get resources for chunks", "error", err)
		return nil, err
	}

	h.logger.Debug("Retrieved resources for chunks", "count", len(resources))
	return &resourceV1.GetResourcesInChunksResponse{
		Resources: resources,
	}, nil
}
