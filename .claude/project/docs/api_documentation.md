# VoidMesh API Documentation

## Overview

VoidMesh API is a RESTful service implementing a chunk-based resource system for multiplayer harvesting mechanics. The API provides endpoints for chunk management, resource harvesting, and player session tracking.

**Base URL:** `http://localhost:8080/api/v1`

## Authentication

Currently, the API does not implement authentication. Player identification is handled through `player_id` parameters in requests.

## Data Models

### ResourceNode

Represents a harvestable resource in the game world.

```json
{
  "node_id": 123,
  "chunk_x": 10,
  "chunk_z": 5,
  "local_x": 8,
  "local_z": 12,
  "node_type": 1,
  "node_subtype": 1,
  "max_yield": 150,
  "current_yield": 120,
  "regeneration_rate": 5,
  "spawned_at": "2024-01-15T10:30:00Z",
  "last_harvest": "2024-01-15T14:20:00Z",
  "respawn_timer": null,
  "spawn_type": 1,
  "is_active": true
}
```

**Field Descriptions:**
- `node_id`: Unique identifier for the resource node
- `chunk_x`, `chunk_z`: Chunk coordinates in the world
- `local_x`, `local_z`: Position within the chunk (0-15)
- `node_type`: Resource type (1=Iron, 2=Gold, 3=Wood, 4=Stone)
- `node_subtype`: Quality tier (0=Poor, 1=Normal, 2=Rich)
- `max_yield`: Maximum resources this node can provide
- `current_yield`: Resources currently available
- `regeneration_rate`: Resources restored per hour
- `spawned_at`: When the node was created
- `last_harvest`: Last time someone harvested from this node
- `respawn_timer`: When depleted node will respawn (null if active)
- `spawn_type`: Spawn behavior (0=Random, 1=Static Daily, 2=Static Permanent)
- `is_active`: Whether the node can be harvested

### HarvestSession

Tracks active harvesting sessions to prevent exploitation.

```json
{
  "session_id": 456,
  "node_id": 123,
  "player_id": 789,
  "started_at": "2024-01-15T14:15:00Z",
  "last_activity": "2024-01-15T14:20:00Z",
  "resources_gathered": 25
}
```

**Field Descriptions:**
- `session_id`: Unique identifier for the harvest session
- `node_id`: Resource node being harvested
- `player_id`: Player conducting the harvest
- `started_at`: When the session began
- `last_activity`: Last time the player performed a harvest action
- `resources_gathered`: Total resources collected in this session

### ErrorResponse

Standard error response format.

```json
{
  "error": "node is depleted",
  "code": 400,
  "message": "node is depleted"
}
```

## API Endpoints

### Health Check

**GET** `/health`

Returns the API health status.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": 1642248600,
  "service": "voidmesh-api",
  "version": "1.0.0"
}
```

### Chunk Management

#### Get Chunk Nodes

**GET** `/chunks/{x}/{z}/nodes`

Retrieves all active resource nodes in a specific chunk.

**Parameters:**
- `x` (path, integer): Chunk X coordinate
- `z` (path, integer): Chunk Z coordinate

**Response:**
```json
{
  "chunk_x": 10,
  "chunk_z": 5,
  "nodes": [
    {
      "node_id": 123,
      "chunk_x": 10,
      "chunk_z": 5,
      "local_x": 8,
      "local_z": 12,
      "node_type": 1,
      "node_subtype": 1,
      "max_yield": 150,
      "current_yield": 120,
      "regeneration_rate": 5,
      "spawned_at": "2024-01-15T10:30:00Z",
      "last_harvest": "2024-01-15T14:20:00Z",
      "respawn_timer": null,
      "spawn_type": 1,
      "is_active": true
    }
  ]
}
```

**Status Codes:**
- `200`: Success
- `400`: Invalid chunk coordinates
- `500`: Internal server error

**Example:**
```bash
curl http://localhost:8080/api/v1/chunks/10/5/nodes
```

### Harvest Management

#### Start Harvest Session

**POST** `/harvest/start`

Initiates a new harvest session for a player at a specific resource node.

**Request Body:**
```json
{
  "node_id": 123,
  "player_id": 789
}
```

**Response:**
```json
{
  "session_id": 456,
  "node_id": 123,
  "player_id": 789,
  "started_at": "2024-01-15T14:15:00Z",
  "last_activity": "2024-01-15T14:15:00Z",
  "resources_gathered": 0
}
```

**Status Codes:**
- `201`: Session created successfully
- `400`: Invalid request (node not found, player already has session, node depleted)
- `500`: Internal server error

**Business Rules:**
- Players can only have one active session at a time
- Nodes must be active and have available yield
- Sessions automatically expire after 5 minutes of inactivity

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/harvest/start \
  -H "Content-Type: application/json" \
  -d '{"node_id": 123, "player_id": 789}'
```

#### Harvest Resources

**PUT** `/harvest/sessions/{sessionId}`

Performs a harvest action within an active session.

**Parameters:**
- `sessionId` (path, integer): Active harvest session ID

**Request Body:**
```json
{
  "harvest_amount": 10
}
```

**Response:**
```json
{
  "success": true,
  "amount_harvested": 10,
  "node_yield_after": 110,
  "resources_gathered": 35
}
```

**Status Codes:**
- `200`: Harvest successful
- `400`: Invalid session, session expired, or invalid harvest amount
- `500`: Internal server error

**Business Rules:**
- Sessions expire after 5 minutes of inactivity
- Harvest amount cannot exceed available node yield
- Node automatically becomes inactive when yield reaches 0
- All harvest actions are logged for audit purposes

**Example:**
```bash
curl -X PUT http://localhost:8080/api/v1/harvest/sessions/456 \
  -H "Content-Type: application/json" \
  -d '{"harvest_amount": 10}'
```

### Player Management

#### Get Player Sessions

**GET** `/players/{playerId}/sessions`

Retrieves all active harvest sessions for a specific player.

**Parameters:**
- `playerId` (path, integer): Player identifier

**Response:**
```json
{
  "player_id": 789,
  "sessions": [
    {
      "session_id": 456,
      "node_id": 123,
      "player_id": 789,
      "started_at": "2024-01-15T14:15:00Z",
      "last_activity": "2024-01-15T14:20:00Z",
      "resources_gathered": 25
    }
  ]
}
```

**Status Codes:**
- `200`: Success
- `400`: Invalid player ID
- `500`: Internal server error

**Example:**
```bash
curl http://localhost:8080/api/v1/players/789/sessions
```

## Resource Types

### Node Types

| Type | ID | Description |
|------|----|-----------| 
| Iron Ore | 1 | Basic mining resource |
| Gold Ore | 2 | Valuable mining resource |
| Wood | 3 | Renewable resource from trees |
| Stone | 4 | Construction material |

### Node Subtypes (Quality Tiers)

| Subtype | ID | Description |
|---------|----|-----------| 
| Poor Quality | 0 | Lower yield resources |
| Normal Quality | 1 | Standard yield resources |
| Rich Quality | 2 | High yield resources |

### Spawn Types

| Type | ID | Description |
|------|----|-----------| 
| Random Spawn | 0 | Appears randomly, respawns elsewhere |
| Static Daily | 1 | Fixed location, resets every 24 hours |
| Static Permanent | 2 | Always exists, regenerates continuously |

## Error Handling

The API uses standard HTTP status codes and returns consistent error responses:

### Common Error Responses

**400 Bad Request**
```json
{
  "error": "invalid request body",
  "code": 400,
  "message": "invalid request body"
}
```

**404 Not Found**
```json
{
  "error": "node not found",
  "code": 404,
  "message": "node not found"
}
```

**500 Internal Server Error**
```json
{
  "error": "Internal server error",
  "code": 500,
  "message": "Internal server error"
}
```

Note: Internal server errors do not expose detailed error information to clients for security reasons.

## Rate Limiting

Currently, no rate limiting is implemented. Consider implementing rate limiting in production environments.

## CORS Policy

The API includes CORS headers to support web clients. All origins are currently allowed in development.

## Background Processes

The API includes several background processes that affect resource availability:

### Resource Regeneration
- Runs hourly
- Restores node yield based on `regeneration_rate`
- Only affects active nodes with regeneration > 0

### Session Cleanup
- Runs every 5 minutes
- Removes expired sessions (no activity for 5+ minutes)
- Automatically frees up "claimed" nodes

### Node Respawning
- Runs hourly
- Reactivates depleted nodes whose `respawn_timer` has expired
- Resets node yield to `max_yield`

## Integration Examples

### Client Session Management

```javascript
// Start harvest session
const startHarvest = async (nodeId, playerId) => {
  const response = await fetch('/api/v1/harvest/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ node_id: nodeId, player_id: playerId })
  });
  return response.json();
};

// Perform harvest
const harvestResource = async (sessionId, amount) => {
  const response = await fetch(`/api/v1/harvest/sessions/${sessionId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ harvest_amount: amount })
  });
  return response.json();
};
```

### Chunk Loading

```javascript
// Load chunk data
const loadChunk = async (chunkX, chunkZ) => {
  const response = await fetch(`/api/v1/chunks/${chunkX}/${chunkZ}/nodes`);
  return response.json();
};
```

## Performance Considerations

### Optimization Strategies

1. **Chunk-based Loading**: Only load nodes for visible chunks
2. **Session Validation**: Check session expiry client-side before API calls
3. **Batch Operations**: Group multiple harvest actions when possible
4. **Caching**: Cache chunk data client-side with invalidation

### Database Indexes

The API uses optimized database indexes for:
- Spatial queries (chunk coordinates)
- Active node filtering
- Session management
- Time-based operations

## Security Considerations

### Current Limitations

1. **No Authentication**: API accepts any player ID
2. **No Rate Limiting**: Susceptible to abuse
3. **No Input Validation**: Basic validation only

### Recommended Improvements

1. Implement JWT-based authentication
2. Add rate limiting per player
3. Validate all input parameters
4. Add request logging and monitoring
5. Implement proper session management with secure tokens

## Versioning

The API uses URL versioning (e.g., `/api/v1/`). Future versions will maintain backward compatibility where possible.

## Support

For issues, questions, or feature requests, please refer to the project's GitHub repository.