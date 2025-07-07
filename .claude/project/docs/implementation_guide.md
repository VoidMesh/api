# Implementation Guide

## Getting Started

### Prerequisites
- Go 1.19+ with SQLite driver (`github.com/mattn/go-sqlite3`)
- SQLite 3.x
- Basic understanding of chunk-based world systems

### Setup Steps

1. **Initialize Database:**
```bash
# Create database file
sqlite3 game.db < schema.sql

# Verify tables created
sqlite3 game.db ".tables"
```

2. **Install Go Dependencies:**
```bash
go mod init chunk-resource-system
go get github.com/mattn/go-sqlite3
```

3. **Run Initial Setup:**
```go
cm, err := NewChunkManager("game.db")
if err != nil {
    log.Fatal(err)
}
defer cm.Close()
```

## Core Components

### ChunkManager
**Primary interface for all chunk and resource operations**

```go
type ChunkManager struct {
    db *sql.DB
}

// Key methods:
func (cm *ChunkManager) LoadChunk(chunkX, chunkZ int) ([]ResourceNode, error)
func (cm *ChunkManager) StartHarvest(nodeID, playerID int) (*HarvestSession, error)
func (cm *ChunkManager) HarvestResource(sessionID int, harvestAmount int) error
```

### Resource Node Types
```go
const (
    // Node types
    IRON_ORE = 1
    GOLD_ORE = 2
    WOOD = 3
    STONE = 4
    
    // Quality subtypes
    POOR_QUALITY = 0
    NORMAL_QUALITY = 1
    RICH_QUALITY = 2
    
    // Spawn behaviors
    RANDOM_SPAWN = 0
    STATIC_DAILY = 1
    STATIC_PERMANENT = 2
)
```

## Usage Patterns

### Basic Chunk Loading
```go
// Load all active nodes in a chunk
nodes, err := chunkManager.LoadChunk(chunkX, chunkZ)
if err != nil {
    return err
}

// Process nodes for client response
for _, node := range nodes {
    // Send node data to game client
    sendNodeToClient(node)
}
```

### Player Harvesting Flow
```go
// 1. Start harvest session
session, err := cm.StartHarvest(nodeID, playerID)
if err != nil {
    return fmt.Errorf("cannot start harvest: %v", err)
}

// 2. Perform harvest action
harvestAmount := calculateHarvestAmount(playerSkill, nodeType)
err = cm.HarvestResource(session.SessionID, harvestAmount)
if err != nil {
    return fmt.Errorf("harvest failed: %v", err)
}

// 3. Session automatically expires after timeout
```

### Background Processes
```go
// Regenerate resources (run hourly)
func scheduleResourceRegeneration(cm *ChunkManager) {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            cm.RegenerateResources()
        }
    }()
}

// Cleanup expired sessions (run every 5 minutes)
func scheduleSessionCleanup(cm *ChunkManager) {
    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for range ticker.C {
            cm.CleanupExpiredSessions()
        }
    }()
}
```

## API Integration

### Recommended REST Endpoints

```go
// GET /chunks/{x}/{z}/nodes
func getChunkNodes(w http.ResponseWriter, r *http.Request) {
    chunkX, _ := strconv.Atoi(mux.Vars(r)["x"])
    chunkZ, _ := strconv.Atoi(mux.Vars(r)["z"])
    
    nodes, err := chunkManager.LoadChunk(chunkX, chunkZ)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(nodes)
}

// POST /nodes/{nodeId}/harvest
func startHarvest(w http.ResponseWriter, r *http.Request) {
    nodeID, _ := strconv.Atoi(mux.Vars(r)["nodeId"])
    playerID := getPlayerIDFromAuth(r)
    
    session, err := chunkManager.StartHarvest(nodeID, playerID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    json.NewEncoder(w).Encode(session)
}

// PUT /sessions/{sessionId}/harvest
func performHarvest(w http.ResponseWriter, r *http.Request) {
    sessionID, _ := strconv.Atoi(mux.Vars(r)["sessionId"])
    
    var req struct {
        Amount int `json:"amount"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    err := chunkManager.HarvestResource(sessionID, req.Amount)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}
```

## Configuration Management

### Spawn Template Configuration
```go
type GameConfig struct {
    SpawnTemplates []SpawnTemplate `json:"spawn_templates"`
    ChunkSize      int             `json:"chunk_size"`
    SessionTimeout int             `json:"session_timeout_minutes"`
}

func loadGameConfig(configPath string) (*GameConfig, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }
    
    var config GameConfig
    err = json.Unmarshal(data, &config)
    return &config, err
}
```

### Example Configuration File
```json
{
    "chunk_size": 16,
    "session_timeout_minutes": 5,
    "spawn_templates": [
        {
            "node_type": 1,
            "node_subtype": 1,
            "spawn_type": 1,
            "min_yield": 100,
            "max_yield": 200,
            "regeneration_rate": 5,
            "respawn_delay_hours": 24,
            "spawn_weight": 3
        }
    ]
}
```

## Error Handling

### Common Error Scenarios
```go
// Node not found or inactive
if err == sql.ErrNoRows {
    return fmt.Errorf("node %d not found or inactive", nodeID)
}

// Player already harvesting elsewhere
if strings.Contains(err.Error(), "active harvest session") {
    return fmt.Errorf("player %d already has active session", playerID)
}

// Node depleted
if strings.Contains(err.Error(), "depleted") {
    return fmt.Errorf("node %d has no resources remaining", nodeID)
}

// Session expired
if strings.Contains(err.Error(), "session expired") {
    return fmt.Errorf("harvest session %d has expired", sessionID)
}
```

### Transaction Retry Logic
```go
func withRetry(operation func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        err := operation()
        if err == nil {
            return nil
        }
        
        // Check if error is retryable (e.g., SQLITE_BUSY)
        if isRetryableError(err) {
            time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
            continue
        }
        
        return err
    }
    return fmt.Errorf("operation failed after %d retries", maxRetries)
}
```

## Testing Strategies

### Unit Tests
```go
func TestHarvestResource(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    cm := &ChunkManager{db: db}
    
    // Create test node
    nodeID := createTestNode(cm, 100) // 100 yield
    
    // Start harvest session
    session, err := cm.StartHarvest(nodeID, 1)
    assert.NoError(t, err)
    
    // Perform harvest
    err = cm.HarvestResource(session.SessionID, 30)
    assert.NoError(t, err)
    
    // Verify node yield decreased
    node := getNode(cm, nodeID)
    assert.Equal(t, 70, node.CurrentYield)
}
```

### Load Testing
```go
func TestConcurrentHarvesting(t *testing.T) {
    cm := setupTestChunkManager(t)
    nodeID := createTestNode(cm, 1000)
    
    var wg sync.WaitGroup
    numPlayers := 10
    
    for i := 0; i < numPlayers; i++ {
        wg.Add(1)
        go func(playerID int) {
            defer wg.Done()
            
            session, err := cm.StartHarvest(nodeID, playerID)
            if err != nil {
                t.Errorf("Player %d failed to start harvest: %v", playerID, err)
                return
            }
            
            err = cm.HarvestResource(session.SessionID, 50)
            if err != nil {
                t.Errorf("Player %d failed to harvest: %v", playerID, err)
            }
        }(i)
    }
    
    wg.Wait()
    
    // Verify total yield decreased correctly
    node := getNode(cm, nodeID)
    assert.Equal(t, 500, node.CurrentYield) // 1000 - (10 * 50)
}
```

## Performance Optimization

### Database Tuning
```sql
-- Enable WAL mode for better concurrency
PRAGMA journal_mode = WAL;

-- Optimize for read-heavy workload
PRAGMA cache_size = 10000;

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;
```

### Connection Pooling
```go
func setupDatabase(dbPath string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_timeout=5000")
    if err != nil {
        return nil, err
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    return db, nil
}
```

## Monitoring and Analytics

### Key Metrics to Track
- **Resource Generation Rate**: Nodes spawned per hour
- **Harvest Success Rate**: Successful vs failed harvest attempts
- **Node Utilization**: Average yield extracted per node
- **Player Activity**: Active harvest sessions over time
- **Resource Economics**: Total resources in circulation

### Analytics Queries
```sql
-- Top harvesting players
SELECT player_id, SUM(amount_harvested) as total_harvested
FROM harvest_log 
WHERE harvested_at > datetime('now', '-7 days')
GROUP BY player_id 
ORDER BY total_harvested DESC;

-- Resource depletion rates
SELECT node_type, AVG(max_yield - current_yield) as avg_depleted
FROM resource_nodes 
WHERE is_active = 1
GROUP BY node_type;

-- Daily harvest volume
SELECT DATE(harvested_at) as harvest_date, 
       SUM(amount_harvested) as daily_total
FROM harvest_log 
GROUP BY DATE(harvested_at)
ORDER BY harvest_date DESC;
```