package server

import (
	"context"

	"github.com/VoidMesh/api/api/internal/logging"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	worldV1 "github.com/VoidMesh/api/api/proto/world/v1"
	"github.com/VoidMesh/api/api/server/handlers"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/VoidMesh/api/api/services/terrain"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

// RegisterServices registers all service handlers with the gRPC server
func RegisterServices(server *grpc.Server, database *pgxpool.Pool) {
	logger := logging.GetLogger()

	// Create world service first using the new constructor pattern
	worldLogger := world.NewDefaultLoggerWrapper()
	worldService := world.NewServiceWithPool(database, worldLogger)

	// Get default world or create if it doesn't exist
	defaultWorld, err := worldService.GetDefaultWorld(context.Background())
	if err != nil {
		logger.Error("Failed to get default world", "error", err)
		return
	}

	// Create shared components
	noiseGen := noise.NewGenerator(defaultWorld.Seed)

	// Create and register chunk service with shared noise generator using new constructor
	chunkHandler := handlers.NewChunkServer(database, worldService, noiseGen.(*noise.Generator))
	chunkV1.RegisterChunkServiceServer(server, chunkHandler)

	// Create and register resource node service with shared noise generator using new constructor
	resourceNodeService := resource_node.NewNodeServiceWithPool(database, noiseGen.(*noise.Generator), worldService)
	resourceNodeHandler := handlers.NewResourceNodeHandler(resourceNodeService, worldService)
	resourceNodeV1.RegisterResourceNodeServiceServer(server, resourceNodeHandler)

	// Create and register terrain service using new constructor
	terrainLogger := terrain.NewDefaultLoggerWrapper()
	terrainService := terrain.NewService(terrainLogger)
	terrainHandler := handlers.NewTerrainHandler(terrainService)
	terrainV1.RegisterTerrainServiceServer(server, terrainHandler)

	// Register world service
	worldHandler := handlers.NewWorldHandler(worldService)
	worldV1.RegisterWorldServiceServer(server, worldHandler)

	logger.Info("Registered all gRPC services")
}
