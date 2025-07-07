# VoidMesh API

A Go-based backend service implementing a chunk-based resource system for multiplayer harvesting mechanics, inspired by EVE Online's asteroid belts and Guild Wars 2's resource nodes.

[![Go Version](https://img.shields.io/badge/Go-1.24.3+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![SQLite](https://img.shields.io/badge/Database-SQLite-blue.svg)](https://sqlite.org)

## Quick Start

```bash
# Clone repository
git clone https://github.com/VoidMesh/api.git
cd api

# Install dependencies
go mod tidy

# Initialize database
sqlite3 game.db < internal/db/migrations/001_initial.up.sql

# Run server
go run ./cmd/server
```

Server runs on `http://localhost:8080`

## Features

### 🌍 Chunk-Based World System
- 16x16 coordinate-based spatial organization
- Infinite world support with efficient sparse storage
- Dynamic chunk loading and node generation

### ⛏️ Resource Harvesting Mechanics
- **Shared Resources**: Multiple players can harvest from the same node
- **Depletion & Regeneration**: Nodes have finite yield but restore over time
- **Quality Tiers**: Poor, Normal, and Rich resource variants
- **Multiple Spawn Types**: Random, Static Daily, and Static Permanent nodes

### 🎮 Multiplayer Session Management
- Session-based harvesting prevents exploitation
- 5-minute activity timeouts
- Concurrent harvesting support
- Complete audit trail

### 🗄️ Robust Data Layer
- SQLite database with transaction safety
- SQLC-generated type-safe queries
- Database migrations with rollback support
- Comprehensive indexing for performance

### 🎛️ Template-Driven Configuration
- Configurable spawn templates for game balancing
- Noise-based procedural generation
- Cluster spawning for realistic resource distribution
- No-code resource tuning

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP API Layer                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │  Handlers   │  │ Middleware  │  │   Routes    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                   Business Logic Layer                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ChunkManager │  │ Resource    │  │ Harvest     │        │
│  │             │  │ Generation  │  │ Sessions    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                     Data Access Layer                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   SQLite    │  │    SQLC     │  │ Migrations  │        │
│  │  Database   │  │  Generated  │  │             │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/api/v1/chunks/{x}/{z}/nodes` | Load chunk nodes |
| `POST` | `/api/v1/harvest/start` | Start harvest session |
| `PUT` | `/api/v1/harvest/sessions/{id}` | Harvest resources |
| `GET` | `/api/v1/players/{id}/sessions` | Get player sessions |

## Resource Types

| Type | ID | Description | Yield Range | Regen Rate |
|------|----|-----------|-----------| -----------|
| Iron Ore | 1 | Basic mining resource | 100-500 | 5-10/hour |
| Gold Ore | 2 | Valuable mining resource | 50-300 | 2-5/hour |
| Wood | 3 | Renewable tree resource | 50-100 | 1/hour |
| Stone | 4 | Construction material | 75-150 | 2/hour |

## Documentation

### 📖 Complete Documentation Set

- **[API Documentation](.claude/project/docs/api_documentation.md)** - Complete REST API reference with examples
- **[Developer Guide](.claude/project/docs/developer_guide.md)** - Architecture, setup, and contribution guidelines  
- **[User Guide](.claude/project/docs/user_guide.md)** - Integration examples for various platforms
- **[Database Schema](.claude/project/docs/database_schema_docs.md)** - Database design and relationships

### 🏗️ Design Documents

- **[Project Overview](.claude/project/docs/project_overview.md)** - High-level system design
- **[Game Design Summary](.claude/project/docs/voidmesh_game_design_summary.md)** - Game mechanics inspiration
- **[Implementation Guide](.claude/project/docs/implementation_guide.md)** - Step-by-step implementation

## Development

### Prerequisites

- Go 1.24.3+
- SQLite 3.x
- Optional: Docker, make

### Development Commands

```bash
# Database operations
sqlite3 game.db < internal/db/migrations/001_initial.up.sql
sqlc generate  # Regenerate database code

# Build and run
go build -o voidmesh-api
./voidmesh-api

# Testing
go test ./...
go test -race ./...

# Code quality
gofmt -w .
golangci-lint run
```

### Project Structure

```
voidmesh-api/
├── cmd/                      # Application entry points
│   ├── debug/               # Debug CLI tool with TUI
│   └── server/              # Main API server
├── internal/                # Private application code
│   ├── api/                 # HTTP handlers and routes
│   ├── chunk/               # Core business logic
│   ├── config/              # Configuration management  
│   └── db/                  # Database layer (SQLC generated)
├── test/                    # Integration tests
├── .claude/project/docs/    # Complete documentation
├── game.db                  # SQLite database
└── main.go                  # Application entry point
```

## Example Usage

### Load Chunk and Start Harvesting

```bash
# Load chunk nodes
curl http://localhost:8080/api/v1/chunks/0/0/nodes

# Start harvest session
curl -X POST http://localhost:8080/api/v1/harvest/start \
  -H "Content-Type: application/json" \
  -d '{"node_id": 1, "player_id": 123}'

# Harvest resources
curl -X PUT http://localhost:8080/api/v1/harvest/sessions/1 \
  -H "Content-Type: application/json" \
  -d '{"harvest_amount": 10}'
```

### JavaScript Client

```javascript
const client = new VoidMeshClient('http://localhost:8080/api/v1');

// Load chunk
const chunk = await client.loadChunk(0, 0);

// Start harvesting
const session = await client.startHarvest(nodeId, playerId);

// Harvest resources  
const result = await client.harvestResource(session.session_id, 10);
```

## Background Services

The API includes automatic background processes:

- **Resource Regeneration** (hourly): Restores node yield based on regeneration rates
- **Session Cleanup** (5 minutes): Removes expired harvest sessions
- **Node Respawning** (hourly): Reactivates depleted nodes after respawn timers

## Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o voidmesh-api

FROM alpine:latest
RUN apk add sqlite
COPY --from=builder /app/voidmesh-api .
EXPOSE 8080
CMD ["./voidmesh-api"]
```

## Performance Features

- **Optimized Indexes**: Spatial and temporal query optimization
- **Transaction Safety**: All critical operations use database transactions
- **Caching Layer**: Occupied position caching with TTL
- **Batch Operations**: Efficient background processing
- **Connection Pooling**: Configurable database connection management

## Security Considerations

**Current Implementation:**
- Input validation on all endpoints
- SQL injection prevention via parameterized queries
- Transaction-based data integrity

**Production Recommendations:**
- Implement JWT authentication
- Add rate limiting per player
- Enable request logging and monitoring
- Use HTTPS in production

## Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/new-feature`
3. Make changes and add tests
4. Run tests: `go test ./...`
5. Submit pull request

### Commit Convention

```
feat: add new feature
fix: resolve bug
docs: update documentation  
refactor: improve code structure
test: add tests
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📋 **Issues**: [GitHub Issues](https://github.com/VoidMesh/api/issues)
- 📖 **Documentation**: [Complete Docs](.claude/project/docs/)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/VoidMesh/api/discussions)

---

Built with ❤️ for multiplayer game developers