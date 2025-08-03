package server

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/VoidMesh/api/api/internal/logging"
	pbCharacterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	pbChunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	pbResourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	pbTerrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
	pbUserV1 "github.com/VoidMesh/api/api/proto/user/v1"
	pbWorldV1 "github.com/VoidMesh/api/api/proto/world/v1"
	"github.com/VoidMesh/api/api/server/handlers"
	"github.com/VoidMesh/api/api/server/middleware" // Uncomment to enable JWT middleware
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/VoidMesh/api/api/services/terrain"
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
	defer lis.Close()
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

	// Create world service
	worldService := world.NewService(dbPool, logger)

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
	pbUserV1.RegisterUserServiceServer(g, handlers.NewUserServer(dbPool))

	logger.Debug("Registering WorldService")
	pbWorldV1.RegisterWorldServiceServer(g, handlers.NewWorldHandler(worldService))

	logger.Debug("Registering CharacterService")
	pbCharacterV1.RegisterCharacterServiceServer(g, handlers.NewCharacterServer(dbPool))

	logger.Debug("Registering TerrainService")
	pbTerrainV1.RegisterTerrainServiceServer(g, handlers.NewTerrainHandler(terrain.NewService()))

	logger.Debug("Registering ResourceNodeService")
	resourceNodeService := resource_node.NewNodeService(dbPool, noiseGen, worldService)
	pbResourceNodeV1.RegisterResourceNodeServiceServer(g, handlers.NewResourceNodeHandler(resourceNodeService, worldService))

	logger.Debug("Registering ChunkService")
	pbChunkV1.RegisterChunkServiceServer(g, handlers.NewChunkServer(dbPool, worldService, noiseGen))

	logger.Info("All gRPC services registered successfully")

	// Serve the gRPC server
	logger.Info("🚀 VoidMesh API server ready to accept connections",
		"address", lis.Addr().String(),
		"services", []string{"User", "World", "Character", "Chunk", "ResourceNode", "Terrain"},
		"features", []string{"JWT Auth", "Health Check", "Reflection"})

	logger.Debug("Starting to serve gRPC requests")
	if err := g.Serve(lis); err != nil {
		logger.Fatal("gRPC server failed to serve", "error", err)
	}
}
