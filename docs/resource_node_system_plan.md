# Resource Node Spawning System Implementation Plan

## Overview

This document outlines the implementation plan for adding resource nodes to the VoidMesh world. Resource nodes will spawn in clusters on appropriate terrain types with varying rarity.

## Resource Node Types

### Grass Terrain

- **Herb Patches**: Medicinal plants and cooking ingredients
- **Berry Bushes**: Food and crafting materials
- **Mineral Outcroppings**: Stone and metal deposits

### Water Terrain

- **Fishing Spots**: Different fish varieties
- **Kelp/Seaweed Beds**: Crafting materials and food
- **Pearl Formations**: Rare crafting components

### Sand Terrain

- **Crystals/Gems**: Crafting and magical components
- **Clay Deposits**: Building materials and pottery
- **Desert Plants**: Special ingredients and rare materials

### Wood/Forest Terrain

- **Harvestable Trees**: Different wood types
- **Mushroom Circles**: Alchemy ingredients and food
- **Wild Honey Hives**: Food and crafting materials

## Implementation Phases

### Phase 1: Data Models & Database

1.  **Resource Node Types as Enums**: Resource node types are now defined as strongly-typed enums in the Protocol Buffer definitions (`resource_node.proto`). This provides compile-time safety and better integration with game clients. The definitions are hardcoded in `api/services/resource_node/types.go`.

2.  **Create `resource_nodes` table**:
    -   ID, `resource_node_type_id` (references the enum, not a DB table)
    -   Position data (chunk_x, chunk_y, pos_x, pos_y)
    -   `cluster_id` to group related nodes
    -   Size/quantity information
    -   Creation timestamp

3.  **Update database schema and migrations**:
    -   Add the new `resource_nodes` table.
    -   Remove the old `resource_types` table.
    -   Create necessary indexes for efficient queries.

4.  **Generate SQLC code for database operations**:
    -   CRUD operations for resource nodes.
    -   Query functions for finding resource nodes in chunks.

### Phase 2: Core Spawning Algorithm

1.  **Resource Distribution Layer**:
    -   Implement separate Perlin noise maps for resource distribution.
    -   Use different frequency/octaves than terrain generation.
    -   Create configurable thresholds for resource type rarity.

2.  **Terrain Validation System**:
    -   Check terrain type for each potential spawn point.
    -   Implement compatibility matrix for resource nodes and terrain.
    -   Create buffer zones (1-2 cells) from terrain transitions.

3.  **Cluster Generation**:
    -   Algorithm to determine number of nodes per cluster (1-6).
    -   Local distribution around initial spawn point.
    -   Terrain validation for each node in the cluster.
    -   Minimum distance requirements between clusters.

4.  **Resource Placement Configuration**:
    -   Easily adjustable spawn rates.
    -   Configurable cluster sizes and distribution.
    -   Rarity tiers with corresponding Perlin noise thresholds.

5.  **Integration with Chunk Generation**:
    -   Hook into existing chunk generation process.
    -   Generate resource nodes after terrain is established.
    -   Store resource nodes in the database.

### Phase 3: API Integration

1.  **Update Protobuf Definitions**:
    -   Create new message types for `ResourceNode` and `ResourceNodeType`.
    -   Define a `ResourceNodeTypeId` enum for all possible resource node types.
    -   Update `ChunkData` message to include `repeated ResourceNode resource_nodes`.
    -   Define a `ResourceNodeService` with methods for resource interaction.
    -   Define a `TerrainService` to expose terrain type information.

2.  **Extend Chunk API**:
    -   Include resource node information in chunk retrieval.
    -   Add filtering options for resource node types.
    -   Optimize for efficient data transfer.

3.  **Resource Discovery**:
    -   API for discovering resource nodes within visible chunks.
    -   Pagination and filtering for resource node queries.

## Database Schema

```sql
-- Resource Nodes Table
CREATE TABLE resource_nodes (
    id SERIAL PRIMARY KEY,
    resource_node_type_id INTEGER NOT NULL, -- Resource node type ID (defined in code as enum)
    chunk_x INTEGER NOT NULL,
    chunk_y INTEGER NOT NULL,
    cluster_id TEXT NOT NULL,
    pos_x INTEGER NOT NULL,
    pos_y INTEGER NOT NULL,
    size INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(chunk_x, chunk_y, pos_x, pos_y)
);

-- Indexes
CREATE INDEX idx_resource_nodes_chunk ON resource_nodes(chunk_x, chunk_y);
CREATE INDEX idx_resource_nodes_type ON resource_nodes(resource_node_type_id);
CREATE INDEX idx_resource_nodes_cluster ON resource_nodes(cluster_id);
```

## Protobuf Definitions

```protobuf
// From: resource_node/v1/resource_node.proto

// Enum for all resource types
enum ResourceNodeTypeId {
  RESOURCE_NODE_TYPE_ID_UNSPECIFIED = 0;
  RESOURCE_NODE_TYPE_ID_HERB_PATCH = 1;
  // ... and so on for all resource types
}

// Resource Node Type
message ResourceNodeType {
  int32 id = 1;
  string name = 2;
  string description = 3;
  string terrain_type = 4;
  ResourceRarity rarity = 5;
  ResourceVisual visual_data = 6;
  ResourceProperties properties = 7;
}

// Resource Node
message ResourceNode {
  int32 id = 1;
  ResourceNodeTypeId resource_node_type_id = 2;
  ResourceNodeType resource_node_type = 3;
  int32 chunk_x = 4;
  int32 chunk_y = 5;
  int32 pos_x = 6;
  int32 pos_y = 7;
  string cluster_id = 8;
  int32 size = 9;
  google.protobuf.Timestamp created_at = 10;
}

// From: chunk/v1/chunk.proto

// Updated Chunk message
message ChunkData {
  int32 chunk_x = 1;
  int32 chunk_y = 2;
  repeated TerrainCell cells = 3;
  int64 seed = 4;
  google.protobuf.Timestamp generated_at = 5;
  repeated resource_node.v1.ResourceNode resource_nodes = 6; // Resource nodes in this chunk
}
```

## Spawn Algorithm Pseudocode

```
function GenerateResourcesForChunk(chunk):
    resourceMap = {}

    // For each resource node type
    for resourcenodetype in allResourceNodeTypes:
        // Generate Perlin noise map for this resource type
        noiseMap = GeneratePerlinNoise(
            chunk.posX, chunk.posY,
            resourcenodetype.id, // Use ID as part of seed
            resourcenodetype.frequency
        )

        // Find potential spawn points
        spawnPoints = []
        for x, y in chunkCoordinates:
            terrainType = GetTerrainType(chunk, x, y)

            // Check if terrain is compatible with resource
            if IsCompatibleTerrain(resourcenodetype, terrainType):
                // Check if not too close to terrain transition
                if IsNotNearTerrainTransition(chunk, x, y):
                    noiseValue = noiseMap[x][y]

                    // Check if noise value exceeds spawn threshold
                    if noiseValue > GetSpawnThreshold(resourcenodetype.rarity):
                        spawnPoints.append((x, y, noiseValue))

        // Sort by noise value (highest first for rarest spawns)
        spawnPoints.sort(byNoiseValueDescending)

        // Generate clusters
        while spawnPoints and len(resourceMap) < MAX_RESOURCES_PER_CHUNK:
            startPoint = spawnPoints.pop()

            // Determine cluster size (1-6 nodes)
            clusterSize = DetermineClusterSize(resourcenodetype.rarity)
            clusterId = GenerateClusterId()

            // Create initial node
            resourceNode = CreateResourceNode(
                resourcenodetype.id,
                chunk.id,
                startPoint.x,
                startPoint.y,
                clusterId
            )
            resourceMap[startPoint.x + "," + startPoint.y] = resourceNode

            // Generate additional nodes in cluster
            for i in range(clusterSize - 1):
                // Find nearby point within cluster radius
                nearbyPoint = FindNearbyPoint(startPoint, CLUSTER_RADIUS)

                // Validate terrain compatibility
                terrainType = GetTerrainType(chunk, nearbyPoint.x, nearbyPoint.y)
                if IsCompatibleTerrain(resourcenodetype, terrainType) and
                   IsNotNearTerrainTransition(chunk, nearbyPoint.x, nearbyPoint.y) and
                   (nearbyPoint.x + "," + nearbyPoint.y) not in resourceMap:

                    // Create additional node
                    resourceNode = CreateResourceNode(
                        resourcenodetype.id,
                        chunk.id,
                        nearbyPoint.x,
                        nearbyPoint.y,
                        clusterId
                    )
                    resourceMap[nearbyPoint.x + "," + nearbyPoint.y] = resourceNode

    return resourceMap.values()
```

## Next Steps After Implementation

1.  **Visualization**: Implement client-side rendering of resource nodes
2.  **Resource Harvesting**: Add mechanics for players to collect resources
3.  **Resource Respawning**: Implement time-based respawn system for harvested resources
4.  **Inventory System**: Create inventory to store collected resources
5.  **Crafting System**: Allow players to use resources for crafting items
