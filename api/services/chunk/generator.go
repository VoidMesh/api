package chunk

import (
	"context"
	"fmt"
	"time"

	"github.com/VoidMesh/platform/api/db"
	chunkV1 "github.com/VoidMesh/platform/api/proto/chunk/v1"
	"github.com/VoidMesh/platform/api/services/noise"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ChunkSize = 32 // 32x32 cells per chunk
)

type Service struct {
	db        *pgxpool.Pool
	noiseGen  *noise.Generator
	worldSeed int64
	chunkSize int32
}

func NewService(db *pgxpool.Pool, worldSeed int64) *Service {
	return &Service{
		db:        db,
		noiseGen:  noise.NewGenerator(worldSeed),
		worldSeed: worldSeed,
		chunkSize: ChunkSize,
	}
}

// GenerateChunk creates a new chunk using procedural generation
func (s *Service) GenerateChunk(chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
	cells := make([]*chunkV1.TerrainCell, ChunkSize*ChunkSize)

	// Generate terrain for each cell in the chunk
	for y := int32(0); y < ChunkSize; y++ {
		for x := int32(0); x < ChunkSize; x++ {
			// Calculate world coordinates
			worldX := chunkX*ChunkSize + x
			worldY := chunkY*ChunkSize + y

			// Generate terrain type based on noise
			terrainType := s.getTerrainType(worldX, worldY)

			// Store in row-major order
			index := y*ChunkSize + x
			cells[index] = &chunkV1.TerrainCell{
				TerrainType: terrainType,
			}
		}
	}

	return &chunkV1.ChunkData{
		ChunkX:      chunkX,
		ChunkY:      chunkY,
		Cells:       cells,
		Seed:        s.worldSeed,
		GeneratedAt: timestamppb.New(time.Now()),
	}, nil
}

// getTerrainType determines terrain type based on noise values
func (s *Service) getTerrainType(x, y int32) chunkV1.TerrainType {
	// Use different scales for different terrain features
	elevation := s.noiseGen.GetTerrainNoise(int(x), int(y), 100.0) // Large scale elevation
	detail := s.noiseGen.GetTerrainNoise(int(x), int(y), 20.0)     // Fine detail

	// Combine noise values
	combined := elevation*0.7 + detail*0.3

	// Determine terrain type based on combined noise
	switch {
	case combined < -0.3:
		return chunkV1.TerrainType_TERRAIN_TYPE_WATER
	case combined < -0.1:
		return chunkV1.TerrainType_TERRAIN_TYPE_SAND
	case combined < 0.2:
		return chunkV1.TerrainType_TERRAIN_TYPE_GRASS
	case combined < 0.5:
		return chunkV1.TerrainType_TERRAIN_TYPE_DIRT
	default:
		return chunkV1.TerrainType_TERRAIN_TYPE_STONE
	}
}

// GetOrCreateChunk retrieves a chunk from database or generates it if it doesn't exist
func (s *Service) GetOrCreateChunk(ctx context.Context, chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
	// Try to get chunk from database first
	chunk, err := s.getChunkFromDB(ctx, chunkX, chunkY)
	if err == nil {
		return chunk, nil
	}

	// Chunk doesn't exist, generate it
	generatedChunk, err := s.GenerateChunk(chunkX, chunkY)
	if err != nil {
		return nil, fmt.Errorf("failed to generate chunk: %w", err)
	}

	// Store in database
	err = s.saveChunkToDB(ctx, generatedChunk)
	if err != nil {
		return nil, fmt.Errorf("failed to save chunk: %w", err)
	}

	return generatedChunk, nil
}

// getChunkFromDB retrieves a chunk from the database
func (s *Service) getChunkFromDB(ctx context.Context, chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
	dbChunk, err := db.New(s.db).GetChunk(ctx, db.GetChunkParams{
		ChunkX: chunkX,
		ChunkY: chunkY,
	})
	if err != nil {
		return nil, err
	}

	// Deserialize protobuf data
	var chunkData chunkV1.ChunkData
	err = proto.Unmarshal(dbChunk.ChunkData, &chunkData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize chunk data: %w", err)
	}

	return &chunkData, nil
}

// saveChunkToDB saves a chunk to the database
func (s *Service) saveChunkToDB(ctx context.Context, chunk *chunkV1.ChunkData) error {
	// Serialize protobuf data
	data, err := proto.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("failed to serialize chunk data: %w", err)
	}

	_, err = db.New(s.db).CreateChunk(ctx, db.CreateChunkParams{
		ChunkX:    chunk.ChunkX,
		ChunkY:    chunk.ChunkY,
		Seed:      chunk.Seed,
		ChunkData: data,
	})

	return err
}

// GetChunksInRange retrieves multiple chunks in a rectangular area
func (s *Service) GetChunksInRange(ctx context.Context, minX, maxX, minY, maxY int32) ([]*chunkV1.ChunkData, error) {
	var chunks []*chunkV1.ChunkData

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			chunk, err := s.GetOrCreateChunk(ctx, x, y)
			if err != nil {
				return nil, fmt.Errorf("failed to get chunk (%d,%d): %w", x, y, err)
			}
			chunks = append(chunks, chunk)
		}
	}

	return chunks, nil
}

// GetChunksInRadius retrieves chunks in a circular area around a center point
func (s *Service) GetChunksInRadius(ctx context.Context, centerX, centerY, radius int32) ([]*chunkV1.ChunkData, error) {
	var chunks []*chunkV1.ChunkData

	for x := centerX - radius; x <= centerX+radius; x++ {
		for y := centerY - radius; y <= centerY+radius; y++ {
			// Check if chunk is within radius (Manhattan distance for simplicity)
			distance := abs(x-centerX) + abs(y-centerY)
			if distance <= radius {
				chunk, err := s.GetOrCreateChunk(ctx, x, y)
				if err != nil {
					return nil, fmt.Errorf("failed to get chunk (%d,%d): %w", x, y, err)
				}
				chunks = append(chunks, chunk)
			}
		}
	}

	return chunks, nil
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
