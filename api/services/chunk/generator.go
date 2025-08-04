package chunk

import (
	"context"
	"fmt"
	"time"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/logging"
	chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
	"github.com/VoidMesh/api/api/services/noise"
	"github.com/VoidMesh/api/api/services/world"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ChunkSize = 32 // 32x32 cells per chunk
)

type Service struct {
	db                      *pgxpool.Pool
	noiseGen                *noise.Generator
	worldService            *world.Service
	defaultWorldID          pgtype.UUID
	chunkSize               int32
	resourceNodeIntegration *ResourceNodeGeneratorIntegration
}

func NewService(db *pgxpool.Pool, worldService *world.Service, noiseGen *noise.Generator) *Service {
	logger := logging.GetLogger()
	logger.Debug("Creating new chunk service", "chunk_size", ChunkSize)
	resourceNodeIntegration := NewResourceNodeGeneratorIntegration(db, noiseGen, worldService)

	// Get default world or create if it doesn't exist
	defaultWorld, err := worldService.GetDefaultWorld(context.Background())
	if err != nil {
		logger.Error("Failed to get default world", "error", err)
	}

	return &Service{
		db:                      db,
		noiseGen:                noiseGen,
		worldService:            worldService,
		defaultWorldID:          defaultWorld.ID,
		chunkSize:               ChunkSize,
		resourceNodeIntegration: resourceNodeIntegration,
	}
}

// GenerateChunk creates a new chunk using procedural generation
func (s *Service) GenerateChunk(chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
	logger := logging.WithChunkCoords(chunkX, chunkY)
	logger.Debug("Starting chunk generation")

	start := time.Now()
	cells := make([]*chunkV1.TerrainCell, ChunkSize*ChunkSize)
	logger.Debug("Allocated terrain cells array", "total_cells", ChunkSize*ChunkSize)

	// Generate terrain for each cell in the chunk
	logger.Debug("Generating terrain cells using noise")
	terrainCounts := make(map[chunkV1.TerrainType]int)
	for y := int32(0); y < ChunkSize; y++ {
		for x := int32(0); x < ChunkSize; x++ {
			// Calculate world coordinates
			worldX := chunkX*ChunkSize + x
			worldY := chunkY*ChunkSize + y

			// Generate terrain type based on noise
			terrainType := s.getTerrainType(worldX, worldY)
			terrainCounts[terrainType]++

			// Store in row-major order
			index := y*ChunkSize + x
			cells[index] = &chunkV1.TerrainCell{
				TerrainType: terrainType,
			}
		}
	}

	logger.Debug("Terrain generation completed", "terrain_distribution", terrainCounts)

	duration := time.Since(start)
	logger.Info("Chunk generation completed", "duration", duration, "cells_generated", len(cells))

	// Get world seed for the default world
	world, err := s.worldService.GetWorldByID(context.Background(), s.defaultWorldID)
	if err != nil {
		return nil, fmt.Errorf("failed to get world: %w", err)
	}

	chunk := &chunkV1.ChunkData{
		ChunkX:      chunkX,
		ChunkY:      chunkY,
		Cells:       cells,
		Seed:        world.Seed,
		GeneratedAt: timestamppb.New(time.Now()),
	}

	return chunk, nil
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
	logger := logging.WithChunkCoords(chunkX, chunkY)

	// Try to get chunk from database first
	chunk, err := s.getChunkFromDB(ctx, chunkX, chunkY)
	if err == nil {
		// Attach resources to existing chunk
		err = s.resourceNodeIntegration.AttachResourceNodesToChunk(ctx, chunk)
		if err != nil {
			// Don't fail chunk retrieval if resource attachment fails
			logger.Error("Failed to attach resources to existing chunk", "error", err)
		}
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

	// Generate and attach resources after chunk is saved
	logger.Debug("Generating resources for new chunk")
	err = s.resourceNodeIntegration.GenerateAndAttachResourceNodes(ctx, generatedChunk)
	if err != nil {
		logger.Error("Failed to generate resources for chunk", "error", err)
		// Don't fail chunk generation if resource generation fails
	}

	return generatedChunk, nil
}

// getChunkFromDB retrieves a chunk from the database
func (s *Service) getChunkFromDB(ctx context.Context, chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
	dbChunk, err := db.New(s.db).GetChunk(ctx, db.GetChunkParams{
		WorldID: s.defaultWorldID,
		ChunkX:  chunkX,
		ChunkY:  chunkY,
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
		WorldID:   s.defaultWorldID,
		ChunkX:    chunk.ChunkX,
		ChunkY:    chunk.ChunkY,
		ChunkData: data,
	})

	return err
}

// GetChunksInRange retrieves multiple chunks in a rectangular area
func (s *Service) GetChunksInRange(ctx context.Context, minX, maxX, minY, maxY int32) ([]*chunkV1.ChunkData, error) {
	// Create a list of coordinates to process
	var coordinates [][2]int32
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			coordinates = append(coordinates, [2]int32{x, y})
		}
	}

	return s.getChunksParallel(ctx, coordinates)
}

// GetChunksInRadius retrieves chunks in a circular area around a center point
func (s *Service) GetChunksInRadius(ctx context.Context, centerX, centerY, radius int32) ([]*chunkV1.ChunkData, error) {
	// Create a list of coordinates to process
	var coordinates [][2]int32
	for x := centerX - radius; x <= centerX+radius; x++ {
		for y := centerY - radius; y <= centerY+radius; y++ {
			// Check if chunk is within radius (Manhattan distance for simplicity)
			distance := abs(x-centerX) + abs(y-centerY)
			if distance <= radius {
				coordinates = append(coordinates, [2]int32{x, y})
			}
		}
	}

	return s.getChunksParallel(ctx, coordinates)
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// getChunksParallel processes a list of chunk coordinates in parallel
func (s *Service) getChunksParallel(ctx context.Context, coordinates [][2]int32) ([]*chunkV1.ChunkData, error) {
	if len(coordinates) == 0 {
		return []*chunkV1.ChunkData{}, nil
	}

	// Determine optimal worker count based on number of coordinates
	workerCount := 4
	if len(coordinates) < workerCount {
		workerCount = len(coordinates)
	}

	// Create channels for work distribution and result collection
	coordChan := make(chan [2]int32, len(coordinates))
	resultChan := make(chan *chunkResult, len(coordinates))
	errChan := make(chan error, workerCount)
	doneChan := make(chan struct{})

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		go s.chunkWorker(ctx, coordChan, resultChan, errChan, doneChan)
	}

	// Send coordinates to workers
	go func() {
		defer close(coordChan)
		for _, coord := range coordinates {
			coordChan <- coord
		}
	}()

	// Collect results
	var chunks []*chunkV1.ChunkData
	resultsCount := 0
	workersDone := 0

	for {
		select {
		case result := <-resultChan:
			if result != nil {
				chunks = append(chunks, result.chunk)
				resultsCount++
			}
			if resCount := len(coordinates); resultsCount >= resCount {
				close(doneChan) // Signal workers to stop
				return chunks, nil
			}
		case err := <-errChan:
			close(doneChan) // Signal all workers to stop on error
			return nil, err
		case <-doneChan:
			workersDone++
			if workersDone >= workerCount {
				return chunks, nil
			}
		case <-ctx.Done():
			close(doneChan) // Signal workers to stop on context cancellation
			return nil, ctx.Err()
		}
	}
}

type chunkResult struct {
	chunk *chunkV1.ChunkData
	x, y  int32
}

// chunkWorker processes chunk coordinates from a channel
func (s *Service) chunkWorker(ctx context.Context, coordChan <-chan [2]int32, resultChan chan<- *chunkResult, errChan chan<- error, doneChan <-chan struct{}) {
	for {
		select {
		case coord, ok := <-coordChan:
			if !ok {
				return // Channel closed, no more work
			}
			
			chunk, err := s.GetOrCreateChunk(ctx, coord[0], coord[1])
			if err != nil {
				errChan <- fmt.Errorf("failed to get chunk (%d,%d): %w", coord[0], coord[1], err)
				return
			}
			
			resultChan <- &chunkResult{
				chunk: chunk,
				x:     coord[0],
				y:     coord[1],
			}
		case <-doneChan:
			return // Worker was signaled to stop
		case <-ctx.Done():
			return // Context was canceled
		}
	}
}
