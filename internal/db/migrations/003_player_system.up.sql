-- Players table - core player information
CREATE TABLE players (
    player_id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(32) UNIQUE NOT NULL,
    password_hash VARCHAR(256) NOT NULL,
    salt VARCHAR(64) NOT NULL,
    email VARCHAR(255) UNIQUE,
    
    -- Player state
    world_x REAL DEFAULT 0.0,
    world_y REAL DEFAULT 0.0,
    world_z REAL DEFAULT 0.0,
    current_chunk_x INTEGER DEFAULT 0,
    current_chunk_z INTEGER DEFAULT 0,
    
    -- Player status
    is_online INTEGER DEFAULT 0,
    last_login TIMESTAMP,
    last_logout TIMESTAMP,
    
    -- Account management
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CHECK (length(username) >= 3 AND length(username) <= 32),
    CHECK (length(password_hash) > 0),
    CHECK (length(salt) > 0)
);

-- Player inventories table - tracks resources per player
CREATE TABLE player_inventories (
    inventory_id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    resource_type INTEGER NOT NULL,     -- 1=iron_ore, 2=gold_ore, 3=wood, 4=stone
    resource_subtype INTEGER DEFAULT 0, -- Quality level
    quantity INTEGER DEFAULT 0,
    
    -- Metadata
    first_obtained TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (player_id) REFERENCES players(player_id) ON DELETE CASCADE,
    UNIQUE(player_id, resource_type, resource_subtype)
);

-- Player statistics table - gameplay metrics
CREATE TABLE player_stats (
    stat_id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    
    -- Resource gathering stats
    total_resources_harvested INTEGER DEFAULT 0,
    total_harvest_sessions INTEGER DEFAULT 0,
    iron_ore_harvested INTEGER DEFAULT 0,
    gold_ore_harvested INTEGER DEFAULT 0,
    wood_harvested INTEGER DEFAULT 0,
    stone_harvested INTEGER DEFAULT 0,
    
    -- Node interaction stats
    unique_nodes_discovered INTEGER DEFAULT 0,
    total_nodes_harvested INTEGER DEFAULT 0,
    
    -- Session stats
    total_playtime_minutes INTEGER DEFAULT 0,
    sessions_count INTEGER DEFAULT 0,
    
    -- Timestamps
    first_harvest TIMESTAMP,
    last_harvest TIMESTAMP,
    stats_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (player_id) REFERENCES players(player_id) ON DELETE CASCADE,
    UNIQUE(player_id)
);

-- Player sessions table - tracks active game sessions
CREATE TABLE player_sessions (
    session_id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    session_token VARCHAR(256) UNIQUE NOT NULL,
    
    -- Session data
    ip_address VARCHAR(45),
    user_agent TEXT,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (player_id) REFERENCES players(player_id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_players_username ON players(username);
CREATE INDEX idx_players_email ON players(email);
CREATE INDEX idx_players_online ON players(is_online);
CREATE INDEX idx_players_position ON players(current_chunk_x, current_chunk_z);

CREATE INDEX idx_player_inventories_player ON player_inventories(player_id);
CREATE INDEX idx_player_inventories_resource ON player_inventories(resource_type, resource_subtype);
CREATE INDEX idx_player_inventories_updated ON player_inventories(last_updated);

CREATE INDEX idx_player_stats_player ON player_stats(player_id);
CREATE INDEX idx_player_stats_updated ON player_stats(stats_updated);

CREATE INDEX idx_player_sessions_player ON player_sessions(player_id);
CREATE INDEX idx_player_sessions_token ON player_sessions(session_token);
CREATE INDEX idx_player_sessions_expires ON player_sessions(expires_at);
CREATE INDEX idx_player_sessions_activity ON player_sessions(last_activity);

-- Update existing harvest_sessions and harvest_log tables to have foreign key constraints
-- Note: SQLite doesn't support adding foreign key constraints to existing tables
-- So we'll create a view to validate player_id exists for now

-- Create a view to validate all player references
CREATE VIEW player_validation AS
SELECT 
    'harvest_sessions' as table_name,
    hs.player_id,
    CASE WHEN p.player_id IS NULL THEN 'INVALID' ELSE 'VALID' END as status
FROM harvest_sessions hs
LEFT JOIN players p ON hs.player_id = p.player_id
UNION ALL
SELECT 
    'harvest_log' as table_name,
    hl.player_id,
    CASE WHEN p.player_id IS NULL THEN 'INVALID' ELSE 'VALID' END as status
FROM harvest_log hl
LEFT JOIN players p ON hl.player_id = p.player_id;