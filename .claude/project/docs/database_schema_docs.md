# Database Schema Documentation

## Overview
The database schema is designed to support a chunk-based resource system with shared harvesting mechanics. The schema prioritizes data integrity, query performance, and audit capabilities.

## Table Relationships
```
chunks (1) ──── (many) resource_nodes
resource_nodes (1) ──── (many) harvest_sessions
resource_nodes (1) ──── (many) harvest_log
node_spawn_templates (1) ──── (many) resource_nodes (logical)
```

## Table Specifications

### chunks
**Purpose:** Tracks chunk metadata and creation timestamps
```sql
CREATE TABLE chunks (
    chunk_x INTEGER NOT NULL,           -- X coordinate of chunk
    chunk_z INTEGER NOT NULL,           -- Z coordinate of chunk  
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chunk_x, chunk_z)
);
```

**Key Points:**
- Composite primary key for efficient spatial indexing
- Tracks when chunks are first loaded and last modified
- Foundation for all spatial queries

### resource_nodes
**Purpose:** Core table storing all harvestable resource nodes
```sql
CREATE TABLE resource_nodes (
    node_id INTEGER PRIMARY KEY AUTOINCREMENT,
    chunk_x INTEGER NOT NULL,           -- Parent chunk coordinates
    chunk_z INTEGER NOT NULL,
    local_x INTEGER NOT NULL,           -- Position within chunk (0-15)
    local_z INTEGER NOT NULL,
    node_type INTEGER NOT NULL,         -- Resource type (iron, gold, wood)
    node_subtype INTEGER DEFAULT 0,     -- Quality tier (poor, normal, rich)
    
    -- Resource mechanics
    max_yield INTEGER NOT NULL,         -- Total harvestable resources
    current_yield INTEGER NOT NULL,     -- Remaining resources
    regeneration_rate INTEGER DEFAULT 0, -- Resources restored per hour
    
    -- Timing
    spawned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_harvest TIMESTAMP,             -- Last time someone harvested
    respawn_timer TIMESTAMP,            -- When depleted node will respawn
    
    -- Node behavior
    spawn_type INTEGER NOT NULL,        -- 0=random, 1=daily, 2=permanent
    is_active INTEGER DEFAULT 1,        -- 1=harvestable, 0=depleted
    
    FOREIGN KEY (chunk_x, chunk_z) REFERENCES chunks(chunk_x, chunk_z)
);
```

**Key Points:**
- `node_id` provides unique identification across all chunks
- Local coordinates (0-15) allow for 16x16 chunk size
- Yield system supports both depletion and regeneration
- `spawn_type` controls node behavior and respawn mechanics
- `is_active` flag enables soft deletion for depleted nodes

### harvest_sessions
**Purpose:** Tracks active harvesting to prevent exploitation
```sql
CREATE TABLE harvest_sessions (
    session_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resources_gathered INTEGER DEFAULT 0,
    
    FOREIGN KEY (node_id) REFERENCES resource_nodes(node_id)
);
```

**Key Points:**
- Prevents players from indefinitely "claiming" nodes
- `last_activity` enables session timeout mechanisms
- Tracks total resources gathered per session
- Multiple sessions can exist for the same node (concurrent harvesting)

### harvest_log
**Purpose:** Permanent audit trail of all harvesting activity
```sql
CREATE TABLE harvest_log (
    log_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    amount_harvested INTEGER NOT NULL,
    harvested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    node_yield_before INTEGER NOT NULL,  -- Node state before harvest
    node_yield_after INTEGER NOT NULL,   -- Node state after harvest
);
```

**Key Points:**
- Immutable record of all resource gathering
- Before/after yield tracking for debugging and analytics
- Enables resource economics analysis
- No foreign key constraints to preserve historical data

### node_spawn_templates
**Purpose:** Configurable templates for node generation
```sql
CREATE TABLE node_spawn_templates (
    template_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_type INTEGER NOT NULL,         -- Resource type to spawn
    node_subtype INTEGER DEFAULT 0,     -- Quality tier
    spawn_type INTEGER NOT NULL,        -- Spawn behavior
    min_yield INTEGER NOT NULL,         -- Minimum resources
    max_yield INTEGER NOT NULL,         -- Maximum resources
    regeneration_rate INTEGER DEFAULT 0,
    respawn_delay_hours INTEGER DEFAULT 24,
    spawn_weight INTEGER DEFAULT 1,     -- Relative spawn probability
    biome_restriction TEXT              -- JSON array of allowed biomes
);
```

**Key Points:**
- Enables game balancing without code changes
- `spawn_weight` controls relative frequency of different node types
- Yield ranges allow for resource value variation
- `biome_restriction` supports future terrain-based spawning

## Index Strategy

### Spatial Queries
```sql
CREATE INDEX idx_resource_nodes_chunk ON resource_nodes(chunk_x, chunk_z);
```
**Purpose:** Fast chunk loading and spatial range queries

### State-Based Queries
```sql
CREATE INDEX idx_resource_nodes_active ON resource_nodes(is_active, spawn_type);
```
**Purpose:** Efficient filtering of active nodes by spawn type

### Time-Based Operations
```sql
CREATE INDEX idx_resource_nodes_respawn ON resource_nodes(respawn_timer) 
WHERE respawn_timer IS NOT NULL;
```
**Purpose:** Background processes for node respawning

### Session Management
```sql
CREATE INDEX idx_harvest_sessions_player ON harvest_sessions(player_id, last_activity);
```
**Purpose:** Player session validation and timeout cleanup

### Analytics
```sql
CREATE INDEX idx_harvest_log_node ON harvest_log(node_id);
CREATE INDEX idx_harvest_log_player ON harvest_log(player_id);
```
**Purpose:** Resource economics analysis and player activity tracking

## Query Patterns

### Common Operations

**Load Chunk Nodes:**
```sql
SELECT * FROM resource_nodes 
WHERE chunk_x = ? AND chunk_z = ? AND is_active = 1;
```

**Find Respawnable Nodes:**
```sql
SELECT node_id FROM resource_nodes 
WHERE is_active = 0 AND respawn_timer <= CURRENT_TIMESTAMP;
```

**Validate Harvest Session:**
```sql
SELECT node_id, player_id FROM harvest_sessions 
WHERE session_id = ? AND last_activity > ?;
```

**Get Player Harvest History:**
```sql
SELECT node_id, amount_harvested, harvested_at 
FROM harvest_log 
WHERE player_id = ? 
ORDER BY harvested_at DESC;
```

## Data Integrity Rules

### Constraints
- Chunk coordinates can be negative (infinite world support)
- Local coordinates must be 0-15 for standard chunk size
- Current yield cannot exceed max yield
- Session timeouts prevent indefinite node claiming

### Referential Integrity
- Nodes belong to existing chunks (foreign key)
- Sessions reference valid nodes (foreign key)
- Harvest logs preserve historical data (no foreign keys)

### Business Rules
- Only active nodes can be harvested
- Sessions expire after inactivity timeout
- Depleted nodes (yield = 0) become inactive
- Respawn timers are set when nodes become inactive

## Performance Considerations

### Read Optimization
- Chunk-based indexes for spatial queries
- Composite indexes for common filter combinations
- Partial indexes for time-based operations

### Write Optimization
- Minimal locking during harvest operations
- Batch updates for background regeneration
- Efficient session cleanup with indexed timestamps

### Storage Efficiency
- Only store non-air blocks/nodes (sparse representation)
- Use INTEGER types for coordinates and yields
- Timestamp fields for all temporal operations