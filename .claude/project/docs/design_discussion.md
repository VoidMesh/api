# Design Discussion Summary

## Initial Requirements
- **Chunk-based system** similar to Minecraft for world organization
- **SQLite storage** for lightweight, file-based persistence
- **Square chunks** with standard size (16x16 blocks)
- **Single layer** implementation (2D world)
- **Resource gathering** mechanics for players
- **Exclusive harvesting** - initially requested single-use resources

## Design Evolution

### First Iteration: Single-Use Resources
**Initial Approach:**
- Each resource could only be harvested by one player
- Simple ownership model with `harvested_by` field
- Individual resource tracking with unique IDs

**Problems Identified:**
- Limited player interaction
- Poor scalability for multiplayer environments
- Unrealistic resource scarcity

### Second Iteration: MMO-Inspired Resource Nodes
**Key Insight:** Real MMOs use shared resource mechanics for better gameplay

**EVE Online Model:**
- Asteroid belts with finite but substantial resources
- Multiple players can mine the same asteroid
- Asteroids deplete over time and respawn

**Guild Wars 2 Model:**
- Static daily nodes that reset every 24 hours
- Random resource nodes that appear and disappear
- Multiple players can harvest the same node

### Final Design: Hybrid System

**Three Node Types:**
1. **Static Daily**: Predictable locations, 24-hour reset cycle
2. **Random Spawn**: Unpredictable locations, respawn after depletion
3. **Static Permanent**: Always present, continuous regeneration

**Key Mechanics:**
- **Shared Harvesting**: Multiple players per node
- **Yield System**: Finite resources per node with depletion
- **Regeneration**: Natural resource restoration over time
- **Session Management**: Prevents exploitation while allowing concurrent access

## Technical Decisions

### Database Schema
**Choice:** Separate tables for chunks, nodes, sessions, and logs
**Rationale:** 
- Normalized design for efficient queries
- Separate concerns (spatial vs temporal data)
- Audit trail capabilities
- Template-driven spawning system

### Concurrency Model
**Choice:** Transaction-based harvesting with session timeouts
**Rationale:**
- Prevents race conditions during harvesting
- Stops players from indefinitely "claiming" nodes
- Allows multiple simultaneous harvesters
- Maintains data consistency

### Storage Format
**Choice:** Relational tables vs JSON blobs
**Rationale:**
- Complex queries needed for spatial and temporal operations
- Relationships between players, nodes, and harvest events
- Indexing requirements for performance
- Audit and analytics capabilities

## Key Insights from Discussion

### Resource Economics
- **Scarcity vs Availability**: Balance between resource competition and player frustration
- **Regeneration Rates**: Control resource flow and economic inflation
- **Quality Tiers**: Create value hierarchies and exploration incentives

### Player Experience
- **Concurrent Harvesting**: Enables cooperation and social interaction
- **Predictable Resources**: Static nodes provide reliable gathering spots
- **Discovery Elements**: Random spawns encourage exploration
- **Session Mechanics**: Prevent griefing while maintaining fairness

### System Scalability
- **Chunk-based Loading**: Only load active areas into memory
- **Background Processing**: Regeneration and cleanup as separate processes
- **Template System**: Easy game balancing without code changes
- **Sparse Storage**: Only store non-default data

## Implementation Highlights

### Transaction Safety
```go
// Example: Atomic harvesting operation
tx, err := cm.db.Begin()
defer tx.Rollback()
// ... validate session and node state
// ... update node yield
// ... log harvest event
return tx.Commit()
```

### Flexible Spawning
```sql
-- Template-driven node generation
INSERT INTO resource_nodes (...)
SELECT template_id, calculated_yield, spawn_location
FROM node_spawn_templates 
WHERE spawn_conditions_met
```

### Session Management
```go
// Prevent exploitation while allowing concurrency
if time.Since(lastActivity) > SESSION_TIMEOUT {
    return fmt.Errorf("session expired")
}
```

## Lessons Learned

### Design Process
- **Start simple** but be prepared to evolve based on real-world game mechanics
- **Study successful systems** rather than inventing from scratch
- **Consider player psychology** not just technical requirements
- **Plan for concurrency** from the beginning

### Technical Architecture
- **Transactions are critical** for multiplayer resource systems
- **Separate spatial and temporal concerns** in database design
- **Template systems** provide flexibility without complexity
- **Audit trails** are essential for debugging and analytics

### Game Design
- **Shared resources** create more interesting gameplay than exclusive ones
- **Multiple node types** cater to different player preferences
- **Regeneration mechanics** prevent permanent resource depletion
- **Quality tiers** add depth without complexity