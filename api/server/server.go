package server

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/logging"
	pbCharacterActionsV1 "github.com/VoidMesh/api/api/proto/character_actions/v1"
	pbCharacterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	pbChunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	pbInventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	pbResourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	pbTerrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	pbUserV1 "github.com/VoidMesh/api/api/proto/user/v1"
	pbWorldV1 "github.com/VoidMesh/api/api/proto/world/v1"
	"github.com/VoidMesh/api/api/server/handlers"
	"github.com/VoidMesh/api/api/server/middleware" // Uncomment to enable JWT middleware
	"github.com/VoidMesh/api/api/services/character"
	"github.com/VoidMesh/api/api/services/character_actions"
	"github.com/VoidMesh/api/api/services/inventory"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func Serve() {
	logger := logging.GetLogger()
	logger.Info("Starting VoidMesh gRPC server initialization")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Debug("Context created for server lifecycle management")

	// Create a listener on TCP port for gRPC server
	logger.Debug("Creating TCP listener on port 50051")
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Fatal("Failed to create TCP listener", "error", err, "port", 50051)
	}
	defer func() {
		if err := lis.Close(); err != nil {
			logger.Error("Failed to close listener", "error", err)
		}
	}()
	logger.Info("TCP listener created successfully", "address", lis.Addr().String())

	// Create a new gRPC server
	logger.Debug("Initializing gRPC server with JWT authentication")

	// Configure JWT authentication
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		logger.Fatal("JWT_SECRET environment variable is required for production")
	}
	logger.Debug("JWT secret loaded", "length", len(jwtSecret))
	g := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.JWTAuthInterceptor(jwtSecret)),
	)
	logger.Info("gRPC server created with JWT authentication interceptor")

	defer func() {
		logger.Info("Initiating graceful server shutdown")
		g.GracefulStop()
		logger.Info("Server shutdown completed")
	}()

	// Register reflection service
	logger.Debug("Registering gRPC reflection service")
	reflection.Register(g)
	logger.Debug("gRPC reflection service registered successfully")

	// Register health check service
	logger.Debug("Registering health check service")
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(g, healthServer)
	logger.Debug("Health check service registered successfully")

	// Create a new PostgreSQL connection pool
	databaseURL := os.Getenv("DATABASE_URL")
	logger.Debug("Connecting to PostgreSQL database", "url_length", len(databaseURL))

	start := time.Now()
	dbPool, err := pgxpool.New(ctx, databaseURL)
	connectionDuration := time.Since(start)

	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err, "duration", connectionDuration)
	}
	logger.Info("Database connection pool created successfully", "duration", connectionDuration)

	// Create world service using new constructor
	worldLogger := world.NewDefaultLoggerWrapper()
	worldService := world.NewServiceWithPool(dbPool, worldLogger)

	// Get default world or create if it doesn't exist
	defaultWorld, err := worldService.GetDefaultWorld(ctx)
	if err != nil {
		logger.Error("Failed to get default world", "error", err)
		return
	}
	logger.Debug("Default world loaded", "world_id", defaultWorld.ID, "seed", defaultWorld.Seed)

	// Create shared noise generator
	noiseGen := noise.NewGenerator(defaultWorld.Seed)

	// Register V1 services
	logger.Debug("Registering gRPC service handlers")

	logger.Debug("Registering UserService")
	userServer, err := handlers.NewUserServerWithPool(dbPool)
	if err != nil {
		logger.Fatal("Failed to create user server", "error", err)
	}
	pbUserV1.RegisterUserServiceServer(g, userServer)

	logger.Debug("Registering WorldService")
	pbWorldV1.RegisterWorldServiceServer(g, handlers.NewWorldHandler(worldService))

	logger.Debug("Registering CharacterService")
	characterServer, err := handlers.NewCharacterServerWithPool(dbPool)
	if err != nil {
		logger.Fatal("Failed to create character server", "error", err)
		return
	}
	pbCharacterV1.RegisterCharacterServiceServer(g, characterServer)

	// Create underlying character service for other services
	// We need to create the chunk service as well
	chunkService, err := handlers.NewChunkServiceWithPool(dbPool)
	if err != nil {
		logger.Fatal("Failed to create chunk service for character service", "error", err)
	}
	characterRealService := character.NewServiceWithPool(dbPool, chunkService)

	logger.Debug("Registering TerrainService")
	terrainService := handlers.NewTerrainServiceWithDefaultLogger()
	terrainLogger := &handlers.LoggerWrapper{Logger: logging.WithComponent("terrain-handler")}
	terrainServer := handlers.NewTerrainServer(terrainService, terrainLogger)
	pbTerrainV1.RegisterTerrainServiceServer(g, terrainServer)

	logger.Debug("Registering ResourceNodeService")
	resourceNodeService := resource_node.NewNodeServiceWithPool(dbPool, noiseGen.(*noise.Generator), worldService)
	pbResourceNodeV1.RegisterResourceNodeServiceServer(g, handlers.NewResourceNodeHandler(resourceNodeService, worldService))

	logger.Debug("Registering ChunkService")
	chunkServer, err := handlers.NewChunkServerWithPool(dbPool)
	if err != nil {
		logger.Fatal("Failed to create chunk server", "error", err)
	}
	pbChunkV1.RegisterChunkServiceServer(g, chunkServer)

	logger.Debug("Registering InventoryService")
	inventoryService := inventory.NewServiceWithPool(dbPool, characterRealService, resourceNodeService)
	pbInventoryV1.RegisterInventoryServiceServer(g, handlers.NewInventoryHandler(inventoryService))

	logger.Debug("Registering CharacterActionsService")
	characterActionsService := character_actions.NewService(
		character_actions.NewDatabaseWrapper(db.New(dbPool)),
		character_actions.NewInventoryServiceAdapter(inventoryService),
		character_actions.NewCharacterServiceAdapter(characterRealService),
		character_actions.NewResourceNodeServiceAdapter(resourceNodeService),
		character_actions.NewDefaultLoggerWrapper(),
	)
	characterActionsHandler := handlers.NewCharacterActionsServiceWithPool(characterActionsService)
	pbCharacterActionsV1.RegisterCharacterActionsServiceServer(g, handlers.NewCharacterActionsServer(characterActionsHandler))

	logger.Info("All gRPC services registered successfully")

	// Serve the gRPC server
	logger.Info("🚀 VoidMesh API server ready to accept connections",
		"address", lis.Addr().String(),
		"services", []string{"User", "World", "Character", "Chunk", "ResourceNode", "Terrain", "Inventory", "CharacterActions"},
		"features", []string{"JWT Auth", "Health Check", "Reflection"})

	logger.Debug("Starting to serve gRPC requests")
	if err := g.Serve(lis); err != nil {
		logger.Fatal("gRPC server failed to serve", "error", err)
	}
}
