# VoidMesh Game Client Development Guide

This guide explains how to develop game clients that connect to the VoidMesh API, with a focus on leveraging the strongly-typed nature of Go and Protocol Buffers.

## Architecture Overview

VoidMesh uses a client-server architecture where:

1. The API server provides services via gRPC
2. Game clients consume these services to render and interact with the game world
3. All data is strongly typed via Protocol Buffers

## Core Services

VoidMesh provides these main services:

- **ChunkService**: Access to the game world map data
- **ResourceNodeService**: Information about resource nodes in the world
- **TerrainService**: Details about different terrain types
- **CharacterService**: Character creation and movement

## Strongly-Typed Development

One of the key advantages of VoidMesh is the strongly-typed nature of all game data:

1. All terrain types, resource node types, and other game elements are defined as static enums in Protocol Buffers
2. Client developers can access these definitions at compile time
3. This enables type-safe code with IDE autocomplete and compile-time validation

## Getting Started

### 1. Import the Protocol Buffer Definitions

```go
import (
    chunkV1 "github.com/VoidMesh/api/api/proto/chunk/v1"
    resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
    terrainV1 "github.com/VoidMesh/api/api/proto/terrain/v1"
)
```

### 2. Connect to the VoidMesh API

```go
conn, err := grpc.Dial("api.voidmesh.example:50051", grpc.WithInsecure())
if err != nil {
    log.Fatalf("Failed to connect: %v", err)
}
defer conn.Close()

// Create service clients
chunkClient := chunkV1.NewChunkServiceClient(conn)
resourceClient := resourceNodeV1.NewResourceNodeServiceClient(conn)
terrainClient := terrainV1.NewTerrainServiceClient(conn)
```

### 3. Access Static Type Definitions

All terrain types and resource nodetypes are available as enums:

```go
// Access terrain types
grassTerrain := terrainV1.TerrainType_TERRAIN_TYPE_GRASS
waterTerrain := terrainV1.TerrainType_TERRAIN_TYPE_WATER

// Access resource node types
herbResource := resourceNodeV1.ResourcenodetypeId_RESOURCE_TYPE_HERB_PATCH
fishingSpot := resourceNodeV1.ResourcenodetypeId_RESOURCE_TYPE_FISHING_SPOT
```

## Rendering a Chunk: Tutorial

This tutorial shows how to fetch and render a chunk of the game world.

### 1. Fetch a Chunk

```go
response, err := chunkClient.GetChunk(context.Background(), &chunkV1.GetChunkRequest{
    ChunkX: 0,
    ChunkY: 0,
})
if err != nil {
    log.Fatalf("Failed to get chunk: %v", err)
}
chunk := response.Chunk
```

### 2. Render Terrain

```go
for y := 0; y < 32; y++ {
    for x := 0; x < 32; x++ {
        cell := chunk.Cells[y*32+x]

        // Get the terrain type and render accordingly
        switch cell.TerrainType {
        case chunkV1.TerrainType_TERRAIN_TYPE_GRASS:
            renderGrass(x, y)
        case chunkV1.TerrainType_TERRAIN_TYPE_WATER:
            renderWater(x, y)
        case chunkV1.TerrainType_TERRAIN_TYPE_STONE:
            renderStone(x, y)
        // Handle other terrain types...
        }
    }
}
```

### 3. Render Resource Nodes

```go
for _, resourceNode := range chunk.Resources {
    // The resource node type is available in the resource_node_type_id field
    switch resourceNode.ResourcenodetypeId {
    case resourceNodeV1.ResourcenodetypeId_RESOURCE_TYPE_HERB_PATCH:
        renderHerb(resourceNode.PosX, resourceNode.PosY)
    case resourceNodeV1.ResourcenodetypeId_RESOURCE_TYPE_FISHING_SPOT:
        renderFishingSpot(resourceNode.PosX, resourceNode.PosY)
    // Handle other resource node types...
    }
}
```

## Discovering Available Types During Development

During development, you may want to explore all available terrain and resource node types:

### Getting All Terrain Types

```go
response, err := terrainClient.GetTerrainTypes(context.Background(), &terrainV1.GetTerrainTypesRequest{})
if err != nil {
    log.Fatalf("Failed to get terrain types: %v", err)
}

for _, terrain := range response.TerrainTypes {
    fmt.Printf("Terrain: %s\n", terrain.Name)
    fmt.Printf("  Description: %s\n", terrain.Description)
    fmt.Printf("  Color: %s\n", terrain.Visual.BaseColor)
    fmt.Printf("  Movement Speed: %f\n", terrain.Properties.MovementSpeedMultiplier)
}
```

### Getting All Resource Node Types

```go
response, err := resourceClient.GetResourceNodeTypes(context.Background(), &resourceNodeV1.GetResourceNodeTypesRequest{})
if err != nil {
    log.Fatalf("Failed to get resource node types: %v", err)
}

for _, resourceNodeType := range response.ResourceNodeTypes {
    fmt.Printf("Resource Node Type: %s\n", resourceNodeType.Name)
    fmt.Printf("  Description: %s\n", resourceNodeType.Description)
    fmt.Printf("  Terrain: %s\n", resourceNodeType.TerrainType)
    fmt.Printf("  Sprite: %s\n", resourceNodeType.VisualData.Sprite)
}
```

## Benefits of the Strongly-Typed Approach

1. **Compile-Time Safety**: Errors in resource node or terrain references are caught at compile time
2. **IDE Integration**: Autocomplete for terrain and resource node types
3. **Discoverability**: Easy to find all available types during development
4. **Performance**: Efficient binary encoding with Protocol Buffers
5. **Language Neutrality**: Client can be written in any language that supports gRPC

## Best Practices

1. Use the static enum definitions (TerrainType, ResourceNodeTypeId) when possible
2. Fetch the full type information only when needed (for first-time setup or detailed displays)
3. Cache the results of GetTerrainTypes() and GetResourceNodeTypes() calls
4. Use terrain and resource node properties to adjust game mechanics (e.g., movement speed on different terrain)

## Further Resources

- [VoidMesh API Reference](https://docs.example.com/voidmesh/api) - Full API documentation
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers/docs/overview) - Learn more about Protocol Buffers
- [gRPC Documentation](https://grpc.io/docs/) - More information about gRPC
