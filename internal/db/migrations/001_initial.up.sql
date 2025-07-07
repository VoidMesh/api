-- Chunks table - stores metadata about each chunk
CREATE TABLE chunks (
    chunk_x INTEGER NOT NULL,
    chunk_z INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chunk_x, chunk_z)
);

-- Resource nodes table - these are the harvestable objects
CREATE TABLE resource_nodes (
    node_id INTEGER PRIMARY KEY AUTOINCREMENT,
    chunk_x INTEGER NOT NULL,
    chunk_z INTEGER NOT NULL,
    local_x INTEGER NOT NULL,  -- 0-15 for 16x16 chunks
    local_z INTEGER NOT NULL,  -- 0-15 for 16x16 chunks
    node_type INTEGER NOT NULL,     -- 1=iron_ore, 2=gold_ore, 3=wood, etc.
    node_subtype INTEGER DEFAULT 0, -- Rich/Poor quality, tree size, etc.
    
    -- Resource mechanics
    max_yield INTEGER NOT NULL,     -- Total resources this node can provide
    current_yield INTEGER NOT NULL, -- Remaining resources
    regeneration_rate INTEGER DEFAULT 0, -- Resources per hour (0 = no regen)
    
    -- Timing
    spawned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_harvest TIMESTAMP,
    respawn_timer TIMESTAMP,        -- When this node will respawn (if depleted)
    
    -- Node behavior
    spawn_type INTEGER NOT NULL,    -- 0=random, 1=static_daily, 2=static_permanent
    is_active INTEGER DEFAULT 1,    -- 1=active, 0=depleted
    
    FOREIGN KEY (chunk_x, chunk_z) REFERENCES chunks(chunk_x, chunk_z) ON DELETE CASCADE
);

-- Harvest sessions - tracks who is harvesting what
CREATE TABLE harvest_sessions (
    session_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resources_gathered INTEGER DEFAULT 0,
    
    FOREIGN KEY (node_id) REFERENCES resource_nodes(node_id) ON DELETE CASCADE
);

-- Harvest log - permanent record of all harvests
CREATE TABLE harvest_log (
    log_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    amount_harvested INTEGER NOT NULL,
    harvested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    node_yield_before INTEGER NOT NULL,
    node_yield_after INTEGER NOT NULL
);

-- Node spawn templates - defines what can spawn where
CREATE TABLE node_spawn_templates (
    template_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_type INTEGER NOT NULL,
    node_subtype INTEGER DEFAULT 0,
    spawn_type INTEGER NOT NULL,
    min_yield INTEGER NOT NULL,
    max_yield INTEGER NOT NULL,
    regeneration_rate INTEGER DEFAULT 0,
    respawn_delay_hours INTEGER DEFAULT 24,
    spawn_weight INTEGER DEFAULT 1,  -- Higher = more likely to spawn
    biome_restriction TEXT,          -- JSON array of allowed biomes
    -- Cluster parameters
    cluster_size_min INTEGER DEFAULT 1,    -- Minimum nodes per cluster
    cluster_size_max INTEGER DEFAULT 1,    -- Maximum nodes per cluster
    cluster_spread_min INTEGER DEFAULT 0,  -- Minimum spread radius from cluster center
    cluster_spread_max INTEGER DEFAULT 0,  -- Maximum spread radius from cluster center
    clusters_per_chunk INTEGER DEFAULT 1   -- Number of clusters per chunk
);

-- Indexes for performance
CREATE INDEX idx_resource_nodes_chunk ON resource_nodes(chunk_x, chunk_z);
CREATE INDEX idx_resource_nodes_active ON resource_nodes(is_active, spawn_type);
CREATE INDEX idx_resource_nodes_respawn ON resource_nodes(respawn_timer) WHERE respawn_timer IS NOT NULL;
CREATE INDEX idx_harvest_sessions_node ON harvest_sessions(node_id);
CREATE INDEX idx_harvest_sessions_player ON harvest_sessions(player_id, last_activity);
CREATE INDEX idx_harvest_log_node ON harvest_log(node_id);
CREATE INDEX idx_harvest_log_player ON harvest_log(player_id);

-- Insert initial spawn templates with cluster parameters
INSERT INTO node_spawn_templates (node_type, node_subtype, spawn_type, min_yield, max_yield, regeneration_rate, respawn_delay_hours, spawn_weight, cluster_size_min, cluster_size_max, cluster_spread_min, cluster_spread_max, clusters_per_chunk) VALUES
-- Static daily iron ore nodes (medium clusters)
(1, 1, 1, 100, 200, 5, 24, 3, 2, 4, 1, 3, 2),
(1, 2, 1, 300, 500, 10, 24, 1, 1, 2, 1, 2, 1),
-- Random gold ore nodes (small clusters)
(2, 1, 0, 50, 100, 2, 12, 2, 1, 3, 1, 2, 1),
(2, 2, 0, 150, 300, 5, 12, 1, 1, 2, 1, 2, 1),
-- Permanent wood nodes (large clusters like forests)
(3, 1, 2, 50, 100, 1, 6, 4, 3, 5, 2, 4, 1);