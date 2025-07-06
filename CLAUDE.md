# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VoidMesh API is a Go-based backend service implementing a chunk-based resource system for an API-only video game. The system draws inspiration from EVE Online's asteroid belts and Guild Wars 2's resource nodes, featuring shared harvesting mechanics, node depletion, and regeneration systems.

## Architecture

### Core Components
- **ChunkManager**: Primary interface for chunk and resource operations
- **SQLite Database**: Lightweight storage with proper transaction handling
- **Resource Nodes**: Harvestable objects with yield, regeneration, and respawn mechanics
- **Harvest Sessions**: Prevents exploitation while allowing concurrent harvesting
- **Spawn Templates**: Configurable system for resource node generation

### Key Design Patterns
- **Chunk-based World**: 16x16 coordinate system for spatial organization
- **Transaction-based Operations**: Ensures data integrity during concurrent access
- **Template System**: Allows game balancing without code changes
- **Audit Trail**: Complete logging of all harvest activities

## Development Commands

### Database Setup
```bash
# Initialize database with schema
sqlite3 game.db < .claude/project/code/chunk_db_schema.sql

# Verify tables
sqlite3 game.db ".tables"
```

### Go Development
```bash
# Build the project
go build -o voidmesh-api

# Run with database initialization
go run . 

# Run tests (when implemented)
go test ./...

# Get dependencies
go mod tidy
```

## Project Structure

```
/
├── go.mod                    # Go module definition
├── main.go                   # Entry point (when implemented)
├── internal/                 # Private application code
│   ├── chunk/               # Chunk management
│   ├── resource/            # Resource node logic
│   └── harvest/             # Harvest session handling
├── api/                     # HTTP handlers (when implemented)
├── config/                  # Configuration management
└── .claude/project/         # Project documentation and prototypes
    ├── code/
    │   ├── chunk_manager.go      # Core implementation reference
    │   ├── chunk_db_schema.sql   # Database schema
    │   └── go.mod               # Dependencies reference
    └── docs/                    # Technical documentation
```

## Database Schema

### Core Tables
- **chunks**: Chunk metadata and timestamps
- **resource_nodes**: Harvestable nodes with yield and timing data
- **harvest_sessions**: Active player harvesting sessions
- **harvest_log**: Permanent audit trail
- **node_spawn_templates**: Configurable node generation rules

### Key Relationships
- Chunks contain multiple resource nodes
- Resource nodes can have multiple concurrent harvest sessions
- All harvesting actions are logged for analytics

## Node Types and Mechanics

### Resource Types
- `IRON_ORE = 1`: Basic mining resource
- `GOLD_ORE = 2`: Valuable mining resource  
- `WOOD = 3`: Renewable resource from trees
- `STONE = 4`: Construction material

### Quality Subtypes
- `POOR_QUALITY = 0`: Lower yield
- `NORMAL_QUALITY = 1`: Standard yield
- `RICH_QUALITY = 2`: High yield

### Spawn Behaviors
- `RANDOM_SPAWN = 0`: Appears randomly, respawns elsewhere
- `STATIC_DAILY = 1`: Fixed location, resets every 24 hours
- `STATIC_PERMANENT = 2`: Always exists, regenerates continuously

## API Integration Points

### Expected REST Endpoints
- `GET /chunks/{x}/{z}/nodes` - Load chunk resource nodes
- `POST /nodes/{nodeId}/harvest` - Start harvest session
- `PUT /sessions/{sessionId}/harvest` - Perform harvest action
- `GET /players/{playerId}/sessions` - Get player's active sessions

### Background Services
- Hourly resource regeneration
- Session cleanup (5-minute intervals)
- Daily node respawning
- Analytics data processing

## Development Guidelines

### Transaction Safety
- All harvest operations must use database transactions
- Implement retry logic for SQLite busy errors
- Validate session timeouts before processing

### Performance Considerations
- Use prepared statements for repeated queries
- Implement connection pooling for production
- Index all spatial and temporal queries
- Batch background operations

### Error Handling
- Differentiate between user errors and system errors
- Log all transaction failures with context
- Implement graceful degradation for database issues
- Return appropriate HTTP status codes

## Testing Strategy

### Unit Tests
- Test concurrent harvesting scenarios
- Validate node depletion and regeneration
- Test session timeout mechanics
- Verify spawn template logic

### Integration Tests
- Test full harvest workflows
- Validate database constraints
- Test API endpoint integration
- Load test concurrent players

## Configuration

### Environment Variables
- `DB_PATH`: Database file location
- `SESSION_TIMEOUT`: Harvest session timeout (minutes)
- `REGEN_INTERVAL`: Resource regeneration frequency
- `LOG_LEVEL`: Application logging level

### Template Configuration
Spawn templates can be modified in the database to adjust:
- Resource yield ranges
- Regeneration rates
- Respawn delays
- Spawn probabilities

## Key Implementation Notes

### Session Management
- Sessions expire after 5 minutes of inactivity
- Players can only have one active session at a time
- Multiple players can harvest from the same node
- Session cleanup runs automatically

### Resource Economics
- Nodes have finite yield but regenerate over time
- Depleted nodes respawn after configured delays
- Harvest amounts are validated against available yield
- All resource flows are logged for analysis

### Concurrency Handling
- Database transactions prevent race conditions
- Proper error handling for SQLite busy states
- Session validation prevents exploitation
- Cleanup processes maintain system health