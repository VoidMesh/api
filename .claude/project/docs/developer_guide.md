# VoidMesh API Developer Guide

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Development Environment Setup](#development-environment-setup)
3. [Project Structure](#project-structure)
4. [Core Components](#core-components)
5. [Database Operations](#database-operations)
6. [API Layer](#api-layer)
7. [Background Services](#background-services)
8. [Testing Strategy](#testing-strategy)
9. [Deployment Guide](#deployment-guide)
10. [Performance Optimization](#performance-optimization)
11. [Security Considerations](#security-considerations)
12. [Contributing Guidelines](#contributing-guidelines)

## Architecture Overview

VoidMesh API implements a chunk-based resource system using clean architecture principles:

```
┌─────────────────────────────────────────────────────────────┐
│                        API Layer                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │  Handlers   │  │ Middleware  │  │   Routes    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                     Business Logic                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ChunkManager │  │ Resource    │  │ Harvest     │        │
│  │             │  │ Generation  │  │ Sessions    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                     Data Layer                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   SQLite    │  │    SQLC     │  │ Migrations  │        │
│  │  Database   │  │  Generated  │  │             │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Principles

1. **Chunk-based World**: 16x16 coordinate system for efficient spatial organization
2. **Transaction Safety**: All harvesting operations use database transactions
3. **Concurrent Access**: Multiple players can harvest from the same node
4. **Resource Economics**: Nodes have finite yield but regenerate over time
5. **Audit Trail**: Complete logging of all harvest activities
6. **Template-driven**: Configurable spawn system for game balancing

## Development Environment Setup

### Prerequisites

- Go 1.24.3 or later
- SQLite 3.x
- Git
- Optional: make, docker

### Installation

1. **Clone the repository:**
```bash
git clone https://github.com/VoidMesh/api.git
cd api
```

2. **Install dependencies:**
```bash
go mod tidy
```

3. **Set up the database:**
```bash
# Initialize database with schema
sqlite3 game.db < internal/db/migrations/001_initial.up.sql

# Apply performance updates
sqlite3 game.db < internal/db/migrations/002_performance_updates.up.sql
```

4. **Verify installation:**
```bash
go build -o voidmesh-api
./voidmesh-api
```

### Development Tools

#### SQLC (SQL Compiler)

The project uses SQLC to generate type-safe Go code from SQL queries:

```bash
# Install SQLC
go install github.com/kyleconroy/sqlc/cmd/sqlc@latest

# Generate code (after modifying .sql files)
sqlc generate
```

#### Database Migrations

```bash
# Install golang-migrate
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create new migration
migrate create -ext sql -dir internal/db/migrations -seq descriptive_name

# Apply migrations
migrate -database "sqlite3://game.db" -path internal/db/migrations up
```

## Project Structure

```
voidmesh-api/
├── cmd/                          # Application entry points
│   ├── debug/                    # Debug CLI tool
│   │   ├── main.go
│   │   ├── components/           # UI components
│   │   └── models/              # Data models
│   └── server/                   # Main server
│       └── main.go
├── internal/                     # Private application code
│   ├── api/                      # HTTP layer
│   │   ├── handlers.go           # Request handlers
│   │   ├── middleware.go         # HTTP middleware
│   │   └── routes.go             # Route definitions
│   ├── chunk/                    # Core business logic
│   │   ├── manager.go            # Primary chunk manager
│   │   └── types.go              # Data types and constants
│   ├── config/                   # Configuration management
│   │   └── config.go
│   └── db/                       # Database layer
│       ├── migrations/           # Database migrations
│       ├── queries/              # SQL queries
│       ├── *.sql.go              # Generated SQLC code
│       ├── db.go                 # Database connection
│       ├── models.go             # Generated models
│       └── querier.go            # Generated interface
├── test/                         # Test files
│   └── integration/              # Integration tests
├── .claude/project/docs/         # Project documentation
├── game.db                       # SQLite database file
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── main.go                       # Main entry point
└── sqlc.yaml                     # SQLC configuration
```

## Core Components

### ChunkManager (`internal/chunk/manager.go`)

The heart of the system, responsible for:

- **Chunk Loading**: Loading and initializing chunks with resource nodes
- **Node Generation**: Creating nodes based on spawn templates and noise
- **Harvest Management**: Handling harvest sessions and resource extraction
- **Background Tasks**: Resource regeneration and session cleanup

Key methods:
```go
func (m *Manager) LoadChunk(ctx context.Context, chunkX, chunkZ int64) (*ChunkResponse, error)
func (m *Manager) StartHarvest(ctx context.Context, nodeID, playerID int64) (*HarvestSession, error)
func (m *Manager) HarvestResource(ctx context.Context, sessionID int64, harvestAmount int64) (*HarvestResponse, error)
func (m *Manager) RegenerateResources(ctx context.Context) error
func (m *Manager) CleanupExpiredSessions(ctx context.Context) error
```

### Resource Generation System

#### Noise-Based Spawning

The system uses Perlin noise to create natural resource distribution:

```go
// Each resource type has its own noise generator
resourceSeed := m.worldSeed + resourceType*1000
m.noiseGens[resourceType] = perlin.NewPerlinRandSource(0.1, 0.1, 3, randSource)

// Evaluate noise for spawning decisions
noiseValue := m.evaluateChunkNoise(chunkX, chunkZ, template)
if noiseValue > template.NoiseThreshold {
    // Spawn nodes based on noise intensity
}
```

#### Cluster Spawning

Resources can spawn in clusters for realistic distribution:

```go
// Template-driven cluster parameters
type ClusterParams struct {
    SizeMin      int64  // Minimum nodes per cluster
    SizeMax      int64  // Maximum nodes per cluster
    SpreadMin    int64  // Minimum spread radius
    SpreadMax    int64  // Maximum spread radius
    PerChunk     int64  // Clusters per chunk
}
```

### Session Management

Prevents exploitation while allowing concurrent harvesting:

```go
// Session timeout (5 minutes)
const SessionTimeout = 5

// Session validation
if time.Since(lastActivity) > SessionTimeout*time.Minute {
    return fmt.Errorf("session expired")
}
```

## Database Operations

### Query Organization

All database operations are organized in `/internal/db/queries/`:

- `chunks.sql`: Chunk management queries
- `resource_nodes.sql`: Node operations
- `harvest_sessions.sql`: Session management
- `harvest_log.sql`: Audit logging
- `spawn_templates.sql`: Template operations

### Transaction Patterns

Critical operations use database transactions:

```go
func (m *Manager) HarvestResource(ctx context.Context, sessionID int64, harvestAmount int64) (*HarvestResponse, error) {
    tx, err := m.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    txQueries := m.queries.WithTx(tx)
    
    // Perform operations with transaction
    // ...
    
    if err = tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return result, nil
}
```

### Common Query Patterns

**Spatial Queries:**
```sql
-- Get nodes in chunk
SELECT * FROM resource_nodes 
WHERE chunk_x = ? AND chunk_z = ? AND is_active = 1;

-- Check node position
SELECT COUNT(*) FROM resource_nodes 
WHERE chunk_x = ? AND chunk_z = ? AND local_x = ? AND local_z = ?;
```

**Time-based Queries:**
```sql
-- Find nodes to respawn
SELECT * FROM resource_nodes 
WHERE is_active = 0 AND respawn_timer <= CURRENT_TIMESTAMP;

-- Cleanup expired sessions
DELETE FROM harvest_sessions 
WHERE last_activity < ?;
```

## API Layer

### Handler Pattern

All HTTP handlers follow a consistent pattern:

```go
func (h *Handler) HandlerName(w http.ResponseWriter, r *http.Request) {
    // 1. Parse parameters
    param := chi.URLParam(r, "paramName")
    
    // 2. Validate input
    if param == "" {
        h.renderError(w, r, http.StatusBadRequest, "missing parameter", nil)
        return
    }
    
    // 3. Create context with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()
    
    // 4. Call business logic
    result, err := h.chunkManager.BusinessMethod(ctx, param)
    if err != nil {
        h.renderError(w, r, http.StatusInternalServerError, "operation failed", err)
        return
    }
    
    // 5. Return response
    render.Status(r, http.StatusOK)
    render.JSON(w, r, result)
}
```

### Middleware Stack

```go
func SetupMiddleware() []func(http.Handler) http.Handler {
    return []func(http.Handler) http.Handler{
        middleware.Logger,
        middleware.Recoverer,
        middleware.Timeout(60 * time.Second),
        cors.Handler(cors.Options{
            AllowedOrigins: []string{"*"},
            AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
            AllowedHeaders: []string{"*"},
        }),
    }
}
```

### Error Handling

Consistent error responses across all endpoints:

```go
func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
    errorResponse := chunk.ErrorResponse{
        Error:   message,
        Code:    status,
        Message: message,
    }
    
    if err != nil {
        log.Error("API error", "error", err, "message", message, "status", status)
        // Don't expose internal errors to clients
        if status >= 500 {
            errorResponse.Error = "Internal server error"
        }
    }
    
    render.Status(r, status)
    render.JSON(w, r, errorResponse)
}
```

## Background Services

### Resource Regeneration

Periodic task that restores node yield:

```go
func (m *Manager) RegenerateResources(ctx context.Context) error {
    // Uses database query to batch update all nodes
    err := m.queries.RegenerateNodeYield(ctx)
    if err != nil {
        return fmt.Errorf("failed to regenerate node yield: %w", err)
    }
    return nil
}
```

### Session Cleanup

Removes expired sessions:

```go
func (m *Manager) CleanupExpiredSessions(ctx context.Context) error {
    cutoff := time.Now().Add(-SessionTimeout * time.Minute)
    err := m.queries.CleanupExpiredSessions(ctx, sql.NullTime{Time: cutoff, Valid: true})
    if err != nil {
        return fmt.Errorf("failed to cleanup expired sessions: %w", err)
    }
    return nil
}
```

### Recommended Scheduler

```go
// Example background service runner
func startBackgroundServices(manager *chunk.Manager) {
    ticker := time.NewTicker(1 * time.Hour)
    sessionTicker := time.NewTicker(5 * time.Minute)
    
    go func() {
        for {
            select {
            case <-ticker.C:
                ctx := context.Background()
                if err := manager.RegenerateResources(ctx); err != nil {
                    log.Error("regeneration failed", "error", err)
                }
            case <-sessionTicker.C:
                ctx := context.Background()
                if err := manager.CleanupExpiredSessions(ctx); err != nil {
                    log.Error("session cleanup failed", "error", err)
                }
            }
        }
    }()
}
```

## Testing Strategy

### Unit Tests

Test individual components:

```go
func TestChunkManager_LoadChunk(t *testing.T) {
    // Set up test database
    db := setupTestDB(t)
    defer db.Close()
    
    manager := chunk.NewManager(db)
    
    // Test chunk loading
    chunk, err := manager.LoadChunk(context.Background(), 0, 0)
    assert.NoError(t, err)
    assert.NotNil(t, chunk)
    assert.Equal(t, int64(0), chunk.ChunkX)
    assert.Equal(t, int64(0), chunk.ChunkZ)
}
```

### Integration Tests

Test complete workflows:

```go
func TestHarvestWorkflow(t *testing.T) {
    // Set up test environment
    db := setupTestDB(t)
    defer db.Close()
    
    manager := chunk.NewManager(db)
    
    // Load chunk with nodes
    chunk, err := manager.LoadChunk(context.Background(), 0, 0)
    require.NoError(t, err)
    require.NotEmpty(t, chunk.Nodes)
    
    nodeID := chunk.Nodes[0].NodeID
    playerID := int64(1)
    
    // Start harvest session
    session, err := manager.StartHarvest(context.Background(), nodeID, playerID)
    require.NoError(t, err)
    
    // Harvest resources
    result, err := manager.HarvestResource(context.Background(), session.SessionID, 10)
    require.NoError(t, err)
    assert.Equal(t, int64(10), result.AmountHarvested)
}
```

### Database Testing

```go
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    
    // Run migrations
    migrationSQL := loadMigrationSQL(t)
    _, err = db.Exec(migrationSQL)
    require.NoError(t, err)
    
    return db
}
```

## Deployment Guide

### Binary Deployment

1. **Build the binary:**
```bash
go build -o voidmesh-api
```

2. **Run the server:**
```bash
./voidmesh-api
```

### Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o voidmesh-api

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/

COPY --from=builder /app/voidmesh-api .
COPY --from=builder /app/internal/db/migrations ./migrations

EXPOSE 8080
CMD ["./voidmesh-api"]
```

### Environment Variables

```bash
# Database configuration
DB_PATH=./game.db

# Server configuration
PORT=8080
HOST=0.0.0.0

# Game configuration
SESSION_TIMEOUT=5
REGEN_INTERVAL=3600
CLEANUP_INTERVAL=300

# Logging
LOG_LEVEL=info
```

## Performance Optimization

### Database Optimization

1. **Index Usage:**
```sql
-- Ensure proper indexes for common queries
CREATE INDEX idx_resource_nodes_chunk ON resource_nodes(chunk_x, chunk_z);
CREATE INDEX idx_harvest_sessions_player ON harvest_sessions(player_id, last_activity);
```

2. **Query Optimization:**
```go
// Use prepared statements for frequent queries
stmt, err := db.Prepare("SELECT * FROM resource_nodes WHERE chunk_x = ? AND chunk_z = ?")
```

3. **Connection Pooling:**
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Application Optimization

1. **Caching Strategy:**
```go
// Cache occupied positions for 30 seconds
type occupiedCacheEntry struct {
    positions map[int64]struct{}
    expires   time.Time
}
```

2. **Batch Operations:**
```go
// Batch regeneration updates
UPDATE resource_nodes 
SET current_yield = MIN(current_yield + regeneration_rate, max_yield)
WHERE is_active = 1 AND regeneration_rate > 0;
```

3. **Goroutine Management:**
```go
// Limit concurrent operations
semaphore := make(chan struct{}, 10)
```

## Security Considerations

### Current Security Measures

1. **Input Validation:**
```go
if req.NodeID <= 0 {
    return fmt.Errorf("node_id must be positive")
}
```

2. **SQL Injection Prevention:**
```go
// SQLC generates parameterized queries
func (q *Queries) GetNode(ctx context.Context, nodeID int64) (ResourceNode, error) {
    // Uses ? placeholders automatically
}
```

3. **Transaction Safety:**
```go
// All critical operations use transactions
tx, err := m.db.BeginTx(ctx, nil)
defer tx.Rollback()
```

### Security Improvements Needed

1. **Authentication:**
```go
// Implement JWT middleware
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if !validateToken(token) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

2. **Rate Limiting:**
```go
// Implement per-player rate limiting
func RateLimitMiddleware(next http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(10), 1)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

3. **Input Sanitization:**
```go
// Validate all input parameters
func validateCoordinate(coord int64) error {
    const maxCoord = 1000000
    if coord < -maxCoord || coord > maxCoord {
        return fmt.Errorf("coordinate out of bounds")
    }
    return nil
}
```

## Contributing Guidelines

### Code Style

1. **Go Formatting:**
```bash
gofmt -w .
goimports -w .
```

2. **Linting:**
```bash
golangci-lint run
```

3. **Testing:**
```bash
go test ./...
go test -race ./...
```

### Git Workflow

1. **Feature Branches:**
```bash
git checkout -b feature/new-feature
git commit -m "feat: add new feature"
git push origin feature/new-feature
```

2. **Commit Messages:**
```
feat: add new feature
fix: resolve harvesting bug
docs: update API documentation
refactor: improve chunk loading
test: add integration tests
```

### Pull Request Process

1. Ensure all tests pass
2. Add documentation for new features
3. Update API documentation if needed
4. Request code review
5. Address feedback
6. Merge after approval

### Development Tips

1. **Database Changes:**
   - Always create migrations
   - Test both up and down migrations
   - Update SQLC queries as needed

2. **API Changes:**
   - Update API documentation
   - Maintain backward compatibility
   - Add integration tests

3. **Performance:**
   - Profile before optimizing
   - Use benchmarks for critical paths
   - Monitor database query performance

This guide provides a comprehensive foundation for developing and maintaining the VoidMesh API. For specific implementation details, refer to the source code and inline documentation.