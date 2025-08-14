// Package mockservices provides mock implementations of service interfaces for testing
package mockservices

import (
    "context"
)

//go:generate mockgen -source=interfaces.go -destination=mock_interfaces.go -package=mockservices

// Simple service interfaces for mocking without proto dependencies
// These will be refined as we develop the actual service interfaces

// CharacterServiceInterface represents a basic character service interface
type CharacterServiceInterface interface {
    CreateCharacter(ctx context.Context, userID, name string) (string, error)
    GetCharacter(ctx context.Context, id string) (interface{}, error)
    MoveCharacter(ctx context.Context, characterID string, x, y int32) error
    ListCharacters(ctx context.Context, userID string) ([]interface{}, error)
}

// ChunkServiceInterface represents a basic chunk service interface
type ChunkServiceInterface interface {
    GetChunk(ctx context.Context, worldID string, x, y int32) (interface{}, error)
    GenerateChunk(ctx context.Context, worldID string, x, y int32) (interface{}, error)
}

// WorldServiceInterface represents a basic world service interface  
type WorldServiceInterface interface {
    CreateWorld(ctx context.Context, name string, seed int64) (string, error)
    GetWorld(ctx context.Context, id string) (interface{}, error)
}
