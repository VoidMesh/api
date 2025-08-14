package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/internal/logging"
	worldV1 "github.com/VoidMesh/api/api/proto/world/v1"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type worldServiceServer struct {
	worldV1.UnimplementedWorldServiceServer
	worldService WorldService
	logger       *log.Logger
}

func NewWorldServer(
	worldService WorldService,
) worldV1.WorldServiceServer {
	logger := logging.WithComponent("world-handler")
	logger.Debug("Creating new WorldService server instance")
	return &worldServiceServer{
		worldService: worldService,
		logger:       logger,
	}
}

// NewWorldServerWithPool creates a world server with all dependencies wired up
// This function maintains backward compatibility while providing dependency injection
func NewWorldServerWithPool(dbPool *pgxpool.Pool) (worldV1.WorldServiceServer, error) {
	logger := logging.WithComponent("world-handler")
	logger.Debug("Creating WorldService server with dependency injection")

	// Create world service with all dependencies
	worldService, err := NewWorldServiceWithPool(dbPool)
	if err != nil {
		logger.Error("Failed to create world service", "error", err)
		return nil, err
	}

	return NewWorldServer(worldService), nil
}

// NewWorldHandler creates a new world handler (legacy compatibility)
// This function is provided for backward compatibility with existing code
func NewWorldHandler(worldService *world.Service) worldV1.WorldServiceServer {
	// Wrap the concrete service to match our interface
	wrappedService := NewWorldServiceFromConcreteService(worldService)
	return NewWorldServer(wrappedService)
}

// GetWorld retrieves a world by ID
func (s *worldServiceServer) GetWorld(ctx context.Context, req *worldV1.GetWorldRequest) (*worldV1.GetWorldResponse, error) {
	logger := s.logger.With("operation", "GetWorld", "world_id_request", req.WorldId)
	logger.Debug("Received GetWorld request")

	var worldID pgtype.UUID
	err := worldID.Scan(string(req.WorldId))
	if err != nil {
		logger.Warn("Invalid world ID format", "world_id_request", req.WorldId, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
	}

	world, err := s.worldService.GetWorldByID(ctx, worldID)
	if err != nil {
		logger.Warn("World not found", "world_id", req.WorldId, "error", err)
		return nil, status.Errorf(codes.NotFound, "World not found: %v", err)
	}

	logger.Info("World retrieved successfully", "world_id", req.WorldId, "world_name", world.Name)
	return &worldV1.GetWorldResponse{
		World: &worldV1.World{
			Id:        world.ID.Bytes[:],
			Name:      world.Name,
			Seed:      world.Seed,
			CreatedAt: timestamppb.New(world.CreatedAt.Time),
		},
	}, nil
}

// GetDefaultWorld retrieves the default world
func (s *worldServiceServer) GetDefaultWorld(ctx context.Context, req *worldV1.GetDefaultWorldRequest) (*worldV1.GetDefaultWorldResponse, error) {
	logger := s.logger.With("operation", "GetDefaultWorld")
	logger.Debug("Received GetDefaultWorld request")

	world, err := s.worldService.GetDefaultWorld(ctx)
	if err != nil {
		logger.Error("Failed to get default world", "error", err)
		return nil, status.Errorf(codes.Internal, "Failed to get default world: %v", err)
	}

	logger.Info("Default world retrieved successfully", "world_id", world.ID.Bytes[:], "world_name", world.Name)
	return &worldV1.GetDefaultWorldResponse{
		World: &worldV1.World{
			Id:        world.ID.Bytes[:],
			Name:      world.Name,
			Seed:      world.Seed,
			CreatedAt: timestamppb.New(world.CreatedAt.Time),
		},
	}, nil
}

// ListWorlds retrieves all worlds
func (s *worldServiceServer) ListWorlds(ctx context.Context, req *worldV1.ListWorldsRequest) (*worldV1.ListWorldsResponse, error) {
	logger := s.logger.With("operation", "ListWorlds")
	logger.Debug("Received ListWorlds request")

	worlds, err := s.worldService.ListWorlds(ctx)
	if err != nil {
		logger.Error("Failed to list worlds", "error", err)
		return nil, status.Errorf(codes.Internal, "Failed to list worlds: %v", err)
	}

	protoWorlds := make([]*worldV1.World, 0, len(worlds))
	for _, world := range worlds {
		protoWorlds = append(protoWorlds, &worldV1.World{
			Id:        world.ID.Bytes[:],
			Name:      world.Name,
			Seed:      world.Seed,
			CreatedAt: timestamppb.New(world.CreatedAt.Time),
		})
	}

	logger.Info("Successfully listed worlds", "count", len(protoWorlds))
	return &worldV1.ListWorldsResponse{
		Worlds: protoWorlds,
	}, nil
}

// UpdateWorldName updates a world's name
func (s *worldServiceServer) UpdateWorldName(ctx context.Context, req *worldV1.UpdateWorldNameRequest) (*worldV1.UpdateWorldNameResponse, error) {
	logger := s.logger.With("operation", "UpdateWorldName", "world_id_request", req.WorldId, "new_name", req.Name)
	logger.Debug("Received UpdateWorldName request")

	var worldID pgtype.UUID
	err := worldID.Scan(string(req.WorldId))
	if err != nil {
		logger.Warn("Invalid world ID format", "world_id_request", req.WorldId, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
	}

	world, err := s.worldService.UpdateWorld(ctx, worldID, req.Name)
	if err != nil {
		logger.Error("Failed to update world name", "world_id", req.WorldId, "new_name", req.Name, "error", err)
		return nil, status.Errorf(codes.Internal, "Failed to update world: %v", err)
	}

	logger.Info("World name updated successfully", "world_id", world.ID.Bytes[:], "old_name", world.Name, "new_name", req.Name)
	return &worldV1.UpdateWorldNameResponse{
		World: &worldV1.World{
			Id:        world.ID.Bytes[:],
			Name:      world.Name,
			Seed:      world.Seed,
			CreatedAt: timestamppb.New(world.CreatedAt.Time),
		},
	}, nil
}

// DeleteWorld deletes a world
func (s *worldServiceServer) DeleteWorld(ctx context.Context, req *worldV1.DeleteWorldRequest) (*worldV1.DeleteWorldResponse, error) {
	logger := s.logger.With("operation", "DeleteWorld", "world_id_request", req.WorldId)
	logger.Debug("Received DeleteWorld request")

	var worldID pgtype.UUID
	err := worldID.Scan(string(req.WorldId))
	if err != nil {
		logger.Warn("Invalid world ID format", "world_id_request", req.WorldId, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid world ID: %v", err)
	}

	logger.Debug("Deleting world from database")
	err = s.worldService.DeleteWorld(ctx, worldID)
	if err != nil {
		logger.Error("Failed to delete world", "world_id", req.WorldId, "error", err)
		return nil, status.Errorf(codes.Internal, "Failed to delete world: %v", err)
	}

	logger.Info("World deleted successfully", "world_id", req.WorldId)
	return &worldV1.DeleteWorldResponse{}, nil
}
