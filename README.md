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

### 🎮 Player Management & Authentication
- Token-based authentication system
- Player registration and login
- Real-time player tracking and online status
- Player inventory and statistics
- Position tracking and updates

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
┌───────────────────────────────────────────────────────────┐
│                      HTTP API Layer                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │  Handlers   │  │ Middleware  │  │   Routes    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└───────────────────────────────────────────────────────────┘
                              │
┌───────────────────────────────────────────────────────────┐
│                   Business Logic Layer                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ChunkManager │  │ Resource    │  │ Harvest     │        │
│  │             │  │ Generation  │  │ Sessions    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└───────────────────────────────────────────────────────────┘
                              │
┌───────────────────────────────────────────────────────────┐
│                     Data Access Layer                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   SQLite    │  │    SQLC     │  │ Migrations  │        │
│  │  Database   │  │  Generated  │  │             │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└───────────────────────────────────────────────────────────┘
```

## API Endpoints

### Public Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/api/v1/chunks/{x}/{z}/nodes` | Load chunk nodes |
| `POST` | `/api/v1/players/register` | Register new player |
| `POST` | `/api/v1/players/login` | Login and get token |
| `GET` | `/api/v1/players/online` | List online players |
| `GET` | `/api/v1/players/{id}/profile` | Get player profile |

### Protected Endpoints (Require Authentication)
| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/nodes/{nodeId}/harvest` | Harvest resources from node |
| `POST` | `/api/v1/players/logout` | Logout and invalidate token |
| `GET` | `/api/v1/players/me` | Get current player info |
| `PUT` | `/api/v1/players/me/position` | Update player position |
| `GET` | `/api/v1/players/me/inventory` | Get player inventory |
| `GET` | `/api/v1/players/me/stats` | Get player statistics |

## Resource Types

| Type | ID | Description | Yield Range | Regen Rate |
|------|----|-----------|-----------| -----------|
| Iron Ore | 1 | Basic mining resource | 100-500 | 5-10/hour |
| Gold Ore | 2 | Valuable mining resource | 50-300 | 2-5/hour |
| Wood | 3 | Renewable tree resource | 50-100 | 1/hour |
| Stone | 4 | Construction material | 75-150 | 2/hour |

## Documentation

### 📖 Complete Documentation Set

- **[API Reference](API_REFERENCE.md)** - Complete REST API reference with examples
- **[Claude Code Integration](CLAUDE.md)** - Instructions for Claude Code development
- **[Debug Tool Guide](cmd/debug/README.md)** - TUI debugging tool documentation
- **[Change Log](CHANGELOG.md)** - Version history and feature updates

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
│   ├── chunk/               # Chunk and resource management
│   ├── player/              # Player management and authentication
│   ├── config/              # Configuration management  
│   └── db/                  # Database layer (SQLC generated)
├── test/                    # Integration tests
├── game.db                  # SQLite database
├── API_REFERENCE.md         # Complete API documentation
├── CLAUDE.md               # Claude Code integration guide
└── main.go                  # Application entry point
```

## Example Usage

### Authentication and Harvesting

```bash
# Register a new player
curl -X POST http://localhost:8080/api/v1/players/register \
  -H "Content-Type: application/json" \
  -d '{"username": "player1", "password": "securepassword"}'

# Login to get session token
curl -X POST http://localhost:8080/api/v1/players/login \
  -H "Content-Type: application/json" \
  -d '{"username": "player1", "password": "securepassword"}'

# Load chunk nodes
curl http://localhost:8080/api/v1/chunks/0/0/nodes

# Harvest resources (requires authentication)
curl -X POST http://localhost:8080/api/v1/nodes/1/harvest \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{"harvest_amount": 10}'
```

### JavaScript Client

```javascript
const client = new VoidMeshClient('http://localhost:8080/api/v1');

// Register and login
await client.register('player1', 'securepassword');
const { session_token } = await client.login('player1', 'securepassword');

// Set authentication token
client.setToken(session_token);

// Load chunk
const chunk = await client.loadChunk(0, 0);

// Harvest resources directly
const result = await client.harvestNode(nodeId, 10);

// Get player inventory
const inventory = await client.getInventory();
```

## Background Services

The API includes automatic background processes:

- **Resource Regeneration** (hourly): Restores node yield based on regeneration rates
- **Node Respawning** (hourly): Reactivates depleted nodes after respawn timers
- **Player Session Management**: Tracks player login/logout status
- **Statistics Updates**: Maintains player gameplay statistics

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
- Bearer token authentication system
- Password hashing with salt and SHA-256
- Input validation on all endpoints
- SQL injection prevention via parameterized queries
- Transaction-based data integrity
- Protected routes with authentication middleware

**Production Recommendations:**
- Upgrade to bcrypt or Argon2 for password hashing
- Implement JWT with proper expiration
- Add rate limiting per player and IP
- Enable request logging and monitoring
- Use HTTPS in production
- Restrict CORS origins from wildcard (*)

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
