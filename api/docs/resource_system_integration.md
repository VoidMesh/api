# Resource System Integration Guide

This document explains how to integrate the new resource spawning system with your existing chunk generation system in VoidMesh.

## Overview

The resource system adds harvestable resource nodes to the world, spawning them in clusters on appropriate terrain types. Resources are tied to chunks and are generated procedurally using Perlin noise, similar to the terrain generation system.

## Integration Steps

### 1. Database Updates

The system adds two new tables to the database:
- `resource_types`: Defines different types of resources and their properties
- `resource_nodes`: Stores individual resource node instances in the world

All necessary database tables and seed data are already included in the `schema.sql` file. Simply run your database migrations as usual to apply these changes.

### 2. Protobuf Changes

The system adds new protobuf definitions:
- New `resource/v1/resource.proto` file defining resource messages and services
- Updated `chunk/v1/chunk.proto` to include resources in the `ChunkData` message

Regenerate the protobuf code:

```bash
./scripts/generate_protobuf.sh
```

### 3. Integration with Chunk Service

To integrate resources with your existing chunk generation system, we've provided a `ResourceGeneratorIntegration` component. Here's how to use it:

#### 3.1. In Your Chunk Service

Integrate the resource system with your chunk service by adding the following code to your chunk service:

```go
// In your chunk service implementation file

// Import the ResourceGeneratorIntegration
import (
    // ... existing imports
    "github.com/VoidMesh/api/api/services/chunk" // Contains the ResourceGeneratorIntegration
)

// Add the integration to your service struct
type Service struct {
    // ... existing fields
    resourceIntegration *chunk.ResourceGeneratorIntegration
}

// Initialize it in your constructor
func NewService(db db.DBTX, noiseGen *noise.Generator) *Service {
    // ... existing initialization
    
    // Create the resource integration
    resourceIntegration := chunk.NewResourceGeneratorIntegration(db, noiseGen)
    
    return &Service{
        // ... existing fields
        resourceIntegration: resourceIntegration,
    }
}
```

#### 3.2. Modify Your Chunk Methods

Add resource generation and attachment to your chunk methods:

```go
// When generating a new chunk
func (s *Service) GenerateChunk(ctx context.Context, chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
    // ... your existing terrain generation code
    
    // After terrain is generated
    chunk := &chunkV1.ChunkData{
        ChunkX: chunkX,
        ChunkY: chunkY,
        Cells:  cells,
        // ... other fields
    }
    
    // Generate and attach resources
    if err := s.resourceIntegration.GenerateAndAttachResources(ctx, chunk); err != nil {
        // Log the error but continue - resources are optional
        s.logger.Error("Failed to generate resources", "error", err)
    }
    
    // ... store chunk in database
    return chunk, nil
}

// When retrieving an existing chunk
func (s *Service) GetChunk(ctx context.Context, chunkX, chunkY int32) (*chunkV1.ChunkData, error) {
    // ... your existing chunk retrieval code
    
    // After retrieving the chunk
    if err := s.resourceIntegration.AttachResourcesToChunk(ctx, chunk); err != nil {
        // Log the error but continue - resources are optional
        s.logger.Error("Failed to attach resources", "error", err)
    }
    
    return chunk, nil
}

// When retrieving multiple chunks
func (s *Service) GetChunksInRange(ctx context.Context, minX, maxX, minY, maxY int32) ([]*chunkV1.ChunkData, error) {
    // ... your existing chunks retrieval code
    
    // After retrieving all chunks
    if err := s.resourceIntegration.AttachResourcesToChunks(ctx, chunks); err != nil {
        // Log the error but continue - resources are optional
        s.logger.Error("Failed to attach resources to chunks", "error", err)
    }
    
    return chunks, nil
}
```

### 4. Add Resource Service to gRPC Server

We've provided a service registration function that you can use. Simply import and call it in your server initialization code:

```go
// In your main.go or server setup file
import (
    // ... other imports
    "github.com/VoidMesh/api/api/server" // Contains service registration functions
)

func main() {
    // ... your existing code
    
    // Create gRPC server
    grpcServer := grpc.NewServer()
    
    // Register all services including the resource service
    server.RegisterServices(grpcServer, database, worldSeed)
    
    // ... continue with server startup
}
```

If you prefer to register services manually, here's how to register just the resource service:

```go
// Create noise generator with the same seed as your chunk service
noiseGen := noise.NewGenerator(worldSeed)

// Create resource service
resourceService := resource.NewService(database, noiseGen)

// Create and register resource handler
resourceHandler := handlers.NewResourceHandler(resourceService)
resourceV1.RegisterResourceServiceServer(grpcServer, resourceHandler)
```

## Testing

To test that resources are being properly generated and attached to chunks:

1. Start the server with the updated code
2. Request a chunk via the gRPC API
3. Verify that the response includes resource nodes in the chunk data
4. Check the database to ensure resource nodes are being stored

## Resource Types

The system comes pre-configured with several resource types for each terrain:

### Grass Terrain
- Herb Patches (medicinal plants, cooking ingredients)
- Berry Bushes (food, crafting materials)
- Mineral Outcroppings (stone, metals)

### Water Terrain
- Fishing Spots (different fish varieties)
- Kelp/Seaweed Beds (crafting materials, food)
- Pearl Formations (rare crafting components)

### Sand Terrain
- Crystals/Gems (crafting, magical components)
- Clay Deposits (building materials, pottery)
- Desert Plants (special ingredients, rare materials)

### Wood/Forest Terrain (Dirt Terrain)
- Harvestable Trees (different wood types)
- Mushroom Circles (alchemy ingredients, food)
- Wild Honey Hives (food, crafting materials)

### Stone Terrain
- Stone Veins (building materials)
- Gem Deposits (valuable gems)
- Metal Ores (crafting materials)

## Configuration

Resource generation can be tuned by modifying the constants in the `resource/generator.go` file:

- `ResourceNoiseScale`: Controls the large-scale distribution of resources
- `ResourceDetailScale`: Controls the fine detail of resource distribution
- `ResourceBufferZone`: Buffer cells from terrain transitions
- `MinClusterDistance`: Minimum distance between cluster centers
- `MaxResourcesPerChunk`: Maximum number of resources in a chunk
- Rarity thresholds: Control how rare each resource tier is
- Cluster size weights: Control how many resources appear in each cluster

## Next Steps

After integrating the resource system, you can:

1. Implement a client-side visualization for resource nodes
2. Add a harvesting system to allow players to collect resources
3. Implement an inventory system to store collected resources
4. Create a crafting system using the resources
5. Add resource respawning mechanics for harvested resources

## Troubleshooting

If you encounter issues with resource generation:

1. **Empty resources**: Check the database for resource types. If none exist, apply the `seed_resources.sql` file.
2. **Resources not showing in chunks**: Ensure the `AttachResourcesToChunk` method is being called when retrieving chunks.
3. **Database errors**: Check that the database schema has been updated with the new tables and indexes.
4. **No resources on certain terrain**: Verify that resource types are properly associated with terrain types in the database.