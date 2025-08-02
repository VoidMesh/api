package main

import (
	"github.com/VoidMesh/api/api/internal/logging"
	"github.com/VoidMesh/api/api/server"
)

func main() {
	// Initialize logging system
	logging.InitLogger()

	logger := logging.GetLogger()
	logger.Info("VoidMesh API server starting up")
	logger.Debug("Debug logging enabled for maximum visibility")

	// Start the server
	server.Serve()
}
