package world

import (
	"context"
	"strconv"

	"github.com/VoidMesh/api/api/db"
	worldV1 "github.com/VoidMesh/api/api/proto/world/v1"
	"github.com/VoidMesh/api/api/services/chunk"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db           *pgxpool.Pool
	chunkService *chunk.Service
	worldSeed    int64
	chunkSize    int32
}

func NewService(db *pgxpool.Pool, chunkService *chunk.Service, worldSeed int64) *Service {
	return &Service{
		db:           db,
		chunkService: chunkService,
		worldSeed:    worldSeed,
		chunkSize:    chunk.ChunkSize,
	}
}

// GetWorldInfo returns information about the world
func (s *Service) GetWorldInfo(ctx context.Context, req *worldV1.GetWorldInfoRequest) (*worldV1.GetWorldInfoResponse, error) {
	// Get world settings from database
	nameSetting, err := db.New(s.db).GetWorldSetting(ctx, "world_name")
	if err != nil {
		nameSetting.Value = "VoidMesh World" // Default name
	}

	seedSetting, err := db.New(s.db).GetWorldSetting(ctx, "seed")
	if err != nil {
		seedSetting.Value = strconv.FormatInt(s.worldSeed, 10)
	}

	chunkSizeSetting, err := db.New(s.db).GetWorldSetting(ctx, "chunk_size")
	if err != nil {
		chunkSizeSetting.Value = "32"
	}

	seed, _ := strconv.ParseInt(seedSetting.Value, 10, 64)
	chunkSizeInt, _ := strconv.ParseInt(chunkSizeSetting.Value, 10, 32)

	return &worldV1.GetWorldInfoResponse{
		WorldInfo: &worldV1.WorldInfo{
			Name:      nameSetting.Value,
			Seed:      seed,
			ChunkSize: int32(chunkSizeInt),
		},
	}, nil
}
