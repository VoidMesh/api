-- Recreate harvest_sessions table if needed (for rollback)
CREATE TABLE harvest_sessions (
    session_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resources_gathered INTEGER DEFAULT 0,
    
    FOREIGN KEY (node_id) REFERENCES resource_nodes(node_id) ON DELETE CASCADE
);

-- Recreate indexes
CREATE INDEX idx_harvest_sessions_node ON harvest_sessions(node_id);
CREATE INDEX idx_harvest_sessions_player ON harvest_sessions(player_id, last_activity);