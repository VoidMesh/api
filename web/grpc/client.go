package grpc

import (
	"context"
	"os"

	characterV1 "github.com/VoidMesh/platform/api/proto/character/v1"
	chunkV1 "github.com/VoidMesh/platform/api/proto/chunk/v1"
	userV1 "github.com/VoidMesh/platform/api/proto/user/v1"
	worldV1 "github.com/VoidMesh/platform/api/proto/world/v1"
	"github.com/charmbracelet/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	apiEndpoint = "localhost:50051"
)

type Client struct {
	UserService      userV1.UserServiceClient
	CharacterService characterV1.CharacterServiceClient
	WorldService     worldV1.WorldServiceClient
	ChunkService     chunkV1.ChunkServiceClient
	conn             *grpc.ClientConn
}

// NewClient initializes and returns a new gRPC client for our services API.
func NewClient() *Client {
	conn, err := grpc.NewClient(getApiEndpoint(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}

	client := &Client{
		CharacterService: characterV1.NewCharacterServiceClient(conn),
		ChunkService:     chunkV1.NewChunkServiceClient(conn),
		UserService:      userV1.NewUserServiceClient(conn),
		WorldService:     worldV1.NewWorldServiceClient(conn),
		conn:             conn,
	}

	return client
}

func getApiEndpoint() string {
	if os.Getenv("API_ENDPOINT") != "" {
		return os.Getenv("API_ENDPOINT")
	}
	return apiEndpoint
}

// WithAuth adds JWT authorization to the context
func WithAuth(ctx context.Context, token string) context.Context {
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	return metadata.NewOutgoingContext(ctx, md)
}
