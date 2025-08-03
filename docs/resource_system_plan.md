# Resource Spawning System Implementation Plan

## Overview
This document outlines the implementation plan for adding resource nodes to the VoidMesh world. Resources will spawn in clusters on appropriate terrain types with varying rarity.

## Resource Types

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

### Phase 1: Database & Data Models
1. Create `resource_types` table with:
   - ID, name, description
   - Compatible terrain type(s)
   - Rarity tier (common, uncommon, rare, very rare)
   - Visual representation data
   - Resource properties (hardness, yield, etc.)

2. Create `resource_nodes` table with:
   - ID, resource_type_id
   - Position data (x, y coordinates)
   - Chunk_id reference
   - Cluster_id to group related nodes
   - Size/quantity information
   - Creation timestamp

3. Update database schema and migrations:
   - Add new tables
   - Create necessary indexes for efficient queries
   - Add foreign key constraints

4. Generate SQLC code for database operations:
   - CRUD operations for resource types and nodes
   - Query functions for finding resources in chunks

### Phase 2: Core Spawning Algorithm

1. Resource Distribution Layer:
   - Implement separate Perlin noise maps for resource distribution
   - Use different frequency/octaves than terrain generation
   - Create configurable thresholds for resource type rarity

2. Terrain Validation System:
   - Check terrain type for each potential spawn point
   - Implement compatibility matrix for resources and terrain
   - Create buffer zones (1-2 cells) from terrain transitions

3. Cluster Generation:
   - Algorithm to determine number of nodes per cluster (1-6)
   - Local distribution around initial spawn point
   - Terrain validation for each node in the cluster
   - Minimum distance requirements between clusters

4. Resource Placement Configuration:
   - Easily adjustable spawn rates
   - Configurable cluster sizes and distribution
   - Rarity tiers with corresponding Perlin noise thresholds

5. Integration with Chunk Generation:
   - Hook into existing chunk generation process
   - Generate resources after terrain is established
   - Store resource nodes in database

### Phase 3: API Integration

1. Update Protobuf Definitions:
   - Create new message types for resources
   - Update chunk-related messages to include resources
   - Define service methods for resource interaction

2. Extend Chunk API:
   - Include resource information in chunk retrieval
   - Add filtering options for resource types
   - Optimize for efficient data transfer

3. Resource Discovery:
   - API for discovering resources within visible chunks
   - Pagination and filtering for resource queries

## Database Schema

```sql
-- Resource Types Table
CREATE TABLE resource_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    terrain_type VARCHAR(20) NOT NULL,
    rarity VARCHAR(20) NOT NULL,
    visual_data JSONB,
    properties JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Resource Nodes Table
CREATE TABLE resource_nodes (
    id SERIAL PRIMARY KEY,
    resource_type_id INTEGER NOT NULL REFERENCES resource_types(id),
    chunk_id INTEGER NOT NULL REFERENCES chunks(id),
    cluster_id VARCHAR(50) NOT NULL,
    pos_x INTEGER NOT NULL,
    pos_y INTEGER NOT NULL,
    size INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(chunk_id, pos_x, pos_y)
);

-- Indexes
CREATE INDEX idx_resource_nodes_chunk_id ON resource_nodes(chunk_id);
CREATE INDEX idx_resource_nodes_type_id ON resource_nodes(resource_type_id);
CREATE INDEX idx_resource_nodes_cluster_id ON resource_nodes(cluster_id);
```

## Protobuf Definitions

```protobuf
// Resource Type
message ResourceType {
  int32 id = 1;
  string name = 2;
  string description = 3;
  string terrain_type = 4;
  string rarity = 5;
  map<string, string> properties = 6;
}

// Resource Node
message ResourceNode {
  int32 id = 1;
  ResourceType resource_type = 2;
  int32 pos_x = 3;
  int32 pos_y = 4;
  string cluster_id = 5;
  int32 size = 6;
}

// Updated Chunk message
message Chunk {
  int32 id = 1;
  int32 pos_x = 2;
  int32 pos_y = 3;
  bytes terrain_data = 4;
  repeated ResourceNode resources = 5;
  // ... other existing fields
}
```

## Spawn Algorithm Pseudocode

```
function GenerateResourcesForChunk(chunk):
    resourceMap = {}
    
    // For each resource type
    for resourceType in resourceTypes:
        // Generate Perlin noise map for this resource type
        noiseMap = GeneratePerlinNoise(
            chunk.posX, chunk.posY, 
            resourceType.seed, 
            resourceType.frequency
        )
        
        // Find potential spawn points
        spawnPoints = []
        for x, y in chunkCoordinates:
            terrainType = GetTerrainType(chunk, x, y)
            
            // Check if terrain is compatible with resource
            if IsCompatibleTerrain(resourceType, terrainType):
                // Check if not too close to terrain transition
                if IsNotNearTerrainTransition(chunk, x, y):
                    noiseValue = noiseMap[x][y]
                    
                    // Check if noise value exceeds spawn threshold
                    if noiseValue > GetSpawnThreshold(resourceType.rarity):
                        spawnPoints.append((x, y, noiseValue))
        
        // Sort by noise value (highest first for rarest spawns)
        spawnPoints.sort(byNoiseValueDescending)
        
        // Generate clusters
        while spawnPoints and len(resourceMap) < MAX_RESOURCES_PER_CHUNK:
            startPoint = spawnPoints.pop()
            
            // Determine cluster size (1-6 nodes)
            clusterSize = DetermineClusterSize(resourceType.rarity)
            clusterId = GenerateClusterId()
            
            // Create initial node
            resourceNode = CreateResourceNode(
                resourceType.id,
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
                if IsCompatibleTerrain(resourceType, terrainType) and 
                   IsNotNearTerrainTransition(chunk, nearbyPoint.x, nearbyPoint.y) and
                   (nearbyPoint.x + "," + nearbyPoint.y) not in resourceMap:
                    
                    // Create additional node
                    resourceNode = CreateResourceNode(
                        resourceType.id,
                        chunk.id,
                        nearbyPoint.x,
                        nearbyPoint.y,
                        clusterId
                    )
                    resourceMap[nearbyPoint.x + "," + nearbyPoint.y] = resourceNode
    
    return resourceMap.values()
```

## Next Steps After Implementation

1. **Visualization**: Implement client-side rendering of resource nodes
2. **Resource Harvesting**: Add mechanics for players to collect resources
3. **Resource Respawning**: Implement time-based respawn system for harvested resources
4. **Inventory System**: Create inventory to store collected resources
5. **Crafting System**: Allow players to use resources for crafting items