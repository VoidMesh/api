# Chunk-Based Resource System Design

## Project Overview

This project implements a chunk-based resource system for an API-only video game, inspired by successful MMO resource mechanics from EVE Online (asteroid belts) and Guild Wars 2 (resource nodes).

## Key Design Principles

### Resource Philosophy
- **Shared Resources**: Multiple players can harvest from the same resource node
- **Depletion Mechanics**: Nodes have finite yield but regenerate over time
- **Respawn System**: Depleted nodes return after configurable timers
- **Quality Tiers**: Different node subtypes with varying yields and rarity

### System Architecture
- **SQLite Database**: Lightweight, file-based storage suitable for game servers
- **Go Implementation**: High-performance backend with proper transaction handling
- **Chunk System**: 16x16 coordinate-based world organization (Minecraft-style)
- **Session Management**: Prevents exploitation while allowing concurrent harvesting

## Resource Node Types

### 1. Static Daily Nodes (GW2 Style)
- Spawn at predictable locations
- Reset every 24 hours
- Higher quality resources
- Example: Iron ore veins, gold deposits

### 2. Random Spawn Nodes
- Appear at random locations within chunks
- Respawn at different locations after depletion
- Maintains resource scarcity and exploration incentive
- Example: Rare mineral deposits

### 3. Static Permanent Nodes
- Always exist at the same locations
- Regenerate continuously
- Basic resource gathering
- Example: Trees, stone quarries

## Technical Features

### Database Design
- **Normalized schema** with separate tables for chunks, nodes, sessions, and logs
- **Efficient indexing** for spatial queries and time-based operations
- **Audit trail** with complete harvest logging
- **Template system** for configurable node spawning

### Concurrency Handling
- **Transaction-based harvesting** prevents race conditions
- **Session timeouts** prevent node camping
- **Yield tracking** ensures consistent resource depletion
- **Multiple player support** on same nodes

### Performance Considerations
- **Sparse storage** - only active nodes stored in database
- **Chunk-based loading** - efficient spatial queries
- **Background processes** for regeneration and cleanup
- **Configurable parameters** via spawn templates

## Implementation Status

### Completed Components
- ✅ Database schema with full relationship mapping
- ✅ Core Go implementation with transaction safety
- ✅ Spawn template system for configurable resources
- ✅ Harvest session management
- ✅ Node regeneration and respawn mechanics
- ✅ Complete audit logging

### Integration Points
- **API Endpoints**: RESTful API for game client integration
- **Background Services**: Periodic regeneration and cleanup tasks
- **Configuration**: Template-based resource balancing
- **Monitoring**: Harvest analytics and resource economics

## File Structure
```
project/
├── docs/
│   ├── design-discussion.md
│   ├── database-schema.md
│   └── api-specification.md
├── database/
│   ├── schema.sql
│   └── sample-data.sql
├── src/
│   ├── chunk_manager.go
│   ├── api_handlers.go (to be implemented)
│   └── background_jobs.go (to be implemented)
└── README.md
```

## Next Steps

1. **API Layer**: Implement HTTP handlers for client integration
2. **Background Services**: Set up periodic tasks for resource management
3. **Configuration System**: Environment-based resource balancing
4. **Testing**: Unit tests and load testing for concurrent harvesting
5. **Monitoring**: Resource economics dashboard and analytics

## Key Design Decisions Rationale

### Why This Approach?
- **Scalability**: Chunk-based system scales to large worlds
- **Player Interaction**: Shared resources create meaningful player interactions
- **Economic Balance**: Regeneration and respawn timers control resource flow
- **Flexibility**: Template system allows easy game balancing
- **Data Integrity**: Transaction-based operations prevent duplication bugs

### Alternative Approaches Considered
- **Single-use resources**: Rejected due to lack of player interaction
- **JSON blob storage**: Rejected due to query inflexibility
- **In-memory only**: Rejected due to persistence requirements
- **NoSQL solutions**: Rejected due to complex relational data needs