package main

import (
	"github.com/VoidMesh/platform/api/server"
)

const (
	apiEndpoint = "localhost:50051"
)

func main() {
	server.Serve()
}
