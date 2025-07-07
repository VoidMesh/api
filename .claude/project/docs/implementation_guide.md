# Implementation Guide

## Getting Started

### Prerequisites
- Go 1.24.3+ with SQLite driver (`github.com/mattn/go-sqlite3`)
- SQLite 3.x
- SQLC for code generation
- Basic understanding of chunk-based world systems

### Setup Steps

1. **Clone and Setup:**
```bash
git clone https://github.com/VoidMesh/api.git
cd api
go mod tidy
```

2. **Initialize Database:**
```bash
# Apply initial schema
sqlite3 game.db < internal/db/migrations/001_initial.up.sql

# Apply performance updates
sqlite3 game.db < internal/db/migrations/002_performance_updates.up.sql

# Verify tables created
sqlite3 game.db ".tables"
```

3. **Generate Database Code:**
```bash
sqlc generate
```

4. **Run the Server:**
```bash
go run ./cmd/server
# or
go run main.go
```

## Core Components

### ChunkManager
**Primary interface for all chunk and resource operations**

Located in `internal/chunk/manager.go`, this is the core business logic component.

```go
type Manager struct {
    db            *sql.DB
    queries       *db.Queries
    worldSeed     int64
    noiseGens     map[int64]*perlin.Perlin
    occupiedCache map[[2]int64]occupiedCacheEntry
    cacheMutex    sync.RWMutex
}

// Key methods:
func (m *Manager) LoadChunk(ctx context.Context, chunkX, chunkZ int64) (*ChunkResponse, error)
func (m *Manager) StartHarvest(ctx context.Context, nodeID, playerID int64) (*HarvestSession, error)
func (m *Manager) HarvestResource(ctx context.Context, sessionID int64, harvestAmount int64) (*HarvestResponse, error)
func (m *Manager) RegenerateResources(ctx context.Context) error
func (m *Manager) CleanupExpiredSessions(ctx context.Context) error
```

**Key Features:**
- **Noise-based Generation**: Uses Perlin noise for realistic resource distribution
- **Cluster Spawning**: Resources can spawn in clusters based on templates
- **Caching Layer**: Occupied position caching with TTL for performance
- **Transaction Safety**: All critical operations use database transactions

### Resource Node Types
Located in `internal/chunk/types.go`:

```go
const (
    // Node types
    IronOre = 1
    GoldOre = 2
    Wood    = 3
    Stone   = 4

    // Node subtypes (quality)
    PoorQuality   = 0
    NormalQuality = 1
    RichQuality   = 2

    // Spawn types
    RandomSpawn     = 0
    StaticDaily     = 1
    StaticPermanent = 2

    // Harvest session timeout (minutes)
    SessionTimeout = 5
    
    // Chunk size
    ChunkSize = 16
)
```

## Usage Patterns

### Basic Chunk Loading
```go
// Load all active nodes in a chunk
ctx := context.Background()
chunkData, err := manager.LoadChunk(ctx, chunkX, chunkZ)
if err != nil {
    return err
}

// Process nodes for client response
for _, node := range chunkData.Nodes {
    // Send node data to game client
    sendNodeToClient(node)
}
```

### Player Harvesting Flow
```go
ctx := context.Background()

// 1. Start harvest session
session, err := manager.StartHarvest(ctx, nodeID, playerID)
if err != nil {
    return fmt.Errorf("cannot start harvest: %v", err)
}

// 2. Perform harvest action
harvestAmount := calculateHarvestAmount(playerSkill, nodeType)
result, err := manager.HarvestResource(ctx, session.SessionID, harvestAmount)
if err != nil {
    return fmt.Errorf("harvest failed: %v", err)
}

log.Printf("Harvested %d resources, node yield now %d", 
    result.AmountHarvested, result.NodeYieldAfter)

// 3. Session automatically expires after 5 minutes of inactivity
```

### Background Processes
```go
// Regenerate resources (run hourly)
func scheduleResourceRegeneration(manager *chunk.Manager) {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            ctx := context.Background()
            if err := manager.RegenerateResources(ctx); err != nil {
                log.Error("regeneration failed", "error", err)
            }
        }
    }()
}

// Cleanup expired sessions (run every 5 minutes)
func scheduleSessionCleanup(manager *chunk.Manager) {
    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for range ticker.C {
            ctx := context.Background()
            if err := manager.CleanupExpiredSessions(ctx); err != nil {
                log.Error("session cleanup failed", "error", err)
            }
        }
    }()
}
```

### Noise-Based Resource Generation
```go
// The system uses Perlin noise for natural resource distribution
func (m *Manager) evaluateChunkNoise(chunkX, chunkZ int64, template db.NodeSpawnTemplate) float64 {
    noiseGen, exists := m.noiseGens[template.NodeType]
    if !exists {
        return 0.0
    }

    noiseScale := 0.1
    if template.NoiseScale.Valid {
        noiseScale = template.NoiseScale.Float64
    }

    x := float64(chunkX) * noiseScale
    z := float64(chunkZ) * noiseScale

    return noiseGen.Noise2D(x, z)
}

// Resources spawn based on noise thresholds
if noiseValue > template.NoiseThreshold {
    // Calculate node count based on noise intensity
    maxNodes := int64((noiseValue - noiseThreshold) * 8)
    // Spawn nodes up to the calculated maximum
}
```

## API Integration

### Current REST API Implementation

The API is implemented in `internal/api/` with the following structure:

```go
// Handler struct (internal/api/handlers.go)
type Handler struct {
    chunkManager *chunk.Manager
}

// GET /api/v1/chunks/{x}/{z}/nodes
func (h *Handler) GetChunk(w http.ResponseWriter, r *http.Request) {
    chunkXStr := chi.URLParam(r, "x")
    chunkZStr := chi.URLParam(r, "z")

    chunkX, err := strconv.ParseInt(chunkXStr, 10, 64)
    if err != nil {
        h.renderError(w, r, http.StatusBadRequest, "invalid chunk x coordinate", err)
        return
    }

    chunkZ, err := strconv.ParseInt(chunkZStr, 10, 64)
    if err != nil {
        h.renderError(w, r, http.StatusBadRequest, "invalid chunk z coordinate", err)
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    chunkData, err := h.chunkManager.LoadChunk(ctx, chunkX, chunkZ)
    if err != nil {
        h.renderError(w, r, http.StatusInternalServerError, "failed to load chunk", err)
        return
    }

    render.JSON(w, r, chunkData)
}

// POST /api/v1/harvest/start
func (h *Handler) StartHarvest(w http.ResponseWriter, r *http.Request) {
    var req struct {
        NodeID   int64 `json:"node_id"`
        PlayerID int64 `json:"player_id"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.renderError(w, r, http.StatusBadRequest, "invalid request body", err)
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    session, err := h.chunkManager.StartHarvest(ctx, req.NodeID, req.PlayerID)
    if err != nil {
        h.renderError(w, r, http.StatusBadRequest, "failed to start harvest", err)
        return
    }

    render.Status(r, http.StatusCreated)
    render.JSON(w, r, session)
}

// PUT /api/v1/harvest/sessions/{sessionId}
func (h *Handler) HarvestResource(w http.ResponseWriter, r *http.Request) {
    sessionIDStr := chi.URLParam(r, "sessionId")
    sessionID, err := strconv.ParseInt(sessionIDStr, 10, 64)
    if err != nil {
        h.renderError(w, r, http.StatusBadRequest, "invalid session id", err)
        return
    }

    var req struct {
        HarvestAmount int64 `json:"harvest_amount"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.renderError(w, r, http.StatusBadRequest, "invalid request body", err)
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    harvestResult, err := h.chunkManager.HarvestResource(ctx, sessionID, req.HarvestAmount)
    if err != nil {
        h.renderError(w, r, http.StatusBadRequest, "failed to harvest resource", err)
        return
    }

    render.JSON(w, r, harvestResult)
}
```

**Router Setup (internal/api/routes.go):**
```go
func SetupRoutes(handler *Handler) *chi.Mux {
    r := chi.NewRouter()

    // Setup middleware
    for _, middleware := range SetupMiddleware() {
        r.Use(middleware)
    }

    r.Use(render.SetContentType(render.ContentTypeJSON))

    // Health check
    r.Get("/health", handler.HealthCheck)

    // API routes
    r.Route("/api/v1", func(r chi.Router) {
        r.Get("/chunks/{x}/{z}/nodes", handler.GetChunk)
        r.Post("/harvest/start", handler.StartHarvest)
        r.Put("/harvest/sessions/{sessionId}", handler.HarvestResource)
        r.Get("/players/{playerId}/sessions", handler.GetPlayerSessions)
    })

    return r
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