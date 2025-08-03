package server

import (
	"os"

	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	resourceV1 "github.com/VoidMesh/api/api/proto/resource/v1"
	"github.com/VoidMesh/api/api/server/handlers"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/resource"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

// RegisterServices registers all service handlers with the gRPC server
func RegisterServices(server *grpc.Server, database *pgxpool.Pool, worldSeed int64) {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
	})

	// Create shared components
	noiseGen := noise.NewGenerator(worldSeed)

	// Create and register chunk service with shared noise generator
	chunkHandler := handlers.NewChunkServer(database, noiseGen)
	chunkV1.RegisterChunkServiceServer(server, chunkHandler)

	// Create and register resource service with shared noise generator
	resourceService := resource.NewService(database, noiseGen)
	resourceHandler := handlers.NewResourceHandler(resourceService)
	resourceV1.RegisterResourceServiceServer(server, resourceHandler)

	logger.Info("Registered all gRPC services")
}
