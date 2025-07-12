package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	pbChunkV1 "github.com/VoidMesh/platform/api/proto/chunk/v1"
	pbMeowV1 "github.com/VoidMesh/platform/api/proto/meow/v1"
	pbUserV1 "github.com/VoidMesh/platform/api/proto/user/v1"
	pbWorldV1 "github.com/VoidMesh/platform/api/proto/world/v1"
	"github.com/VoidMesh/platform/api/server/handlers"
	"github.com/VoidMesh/platform/api/server/middleware" // Uncomment to enable JWT middleware
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func Serve() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a listener on TCP port for gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}
	defer lis.Close()

	// Create a new gRPC server
	// To enable JWT authentication, uncomment the following lines:
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET environment variable is required for production")
	}
	g := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.JWTAuthInterceptor(jwtSecret)),
	)

	defer g.GracefulStop()

	// Register reflection service
	reflection.Register(g)

	// Register health check service
	grpc_health_v1.RegisterHealthServer(g, health.NewServer())

	// Create a new PostgreSQL connection pool
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
	}

	// Register V1 services
	pbMeowV1.RegisterMeowServiceServer(g, handlers.NewMeowerServer(db))
	pbUserV1.RegisterUserServiceServer(g, handlers.NewUserServer(db))
	pbWorldV1.RegisterWorldServiceServer(g, handlers.NewWorldServer(db))
	pbChunkV1.RegisterChunkServiceServer(g, handlers.NewChunkServer(db))

	// Serve the gRPC server
	log.Printf("API server listening at %v", lis.Addr())
	if err := g.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
