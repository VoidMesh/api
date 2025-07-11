# VoidMesh API Reference

Complete API documentation for the VoidMesh chunk-based resource harvesting system.

## Base URL

All API endpoints are prefixed with `/api/v1`:

```
http://localhost:8080/api/v1
```

## Authentication

The API uses **Bearer Token** authentication. Include the token in the `Authorization` header:

```
Authorization: Bearer <your-token-here>
```

### Authentication Flow

1. **Register** a new player account
2. **Login** to receive a session token
3. **Include token** in subsequent requests to protected endpoints
4. **Logout** to invalidate the token

## Endpoints Overview

### Public Endpoints
- `GET /health` - Health check
- `POST /players/register` - Register new player
- `POST /players/login` - Login and get token
- `GET /players/online` - List online players
- `GET /players/{playerID}/profile` - Get player profile
- `GET /chunks/{x}/{z}/nodes` - Get chunk nodes

### Protected Endpoints (Require Authentication)
- `POST /players/logout` - Logout and invalidate token
- `GET /players/me` - Get current player info
- `PUT /players/me/position` - Update player position
- `GET /players/me/inventory` - Get player inventory
- `GET /players/me/stats` - Get player statistics
- `POST /nodes/{nodeId}/harvest` - Harvest resources from node

---

## Player Management

### Register Player

Create a new player account.

**Endpoint:** `POST /players/register`

**Request Body:**
```json
{
  "username": "string",
  "password": "string",
  "email": "string" // optional
}
```

**Response:**
```json
{
  "success": true,
  "message": "Player registered successfully",
  "player": {
    "id": 123,
    "username": "player1",
    "email": "player@example.com",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

**Status Codes:**
- `201` - Player created successfully
- `400` - Invalid request data
- `409` - Username already exists

### Login Player

Authenticate a player and receive a session token.

**Endpoint:** `POST /players/login`

**Request Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Login successful",
  "session_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "player": {
    "id": 123,
    "username": "player1",
    "current_chunk_x": 0,
    "current_chunk_z": 0,
    "is_online": true
  }
}
```

**Status Codes:**
- `200` - Login successful
- `400` - Invalid request data
- `401` - Invalid credentials

### Logout Player

Invalidate the current session token.

**Endpoint:** `POST /players/logout`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "message": "Logout successful"
}
```

**Status Codes:**
- `200` - Logout successful
- `401` - Invalid or expired token

### Get Current Player

Get information about the currently authenticated player.

**Endpoint:** `GET /players/me`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": 123,
  "username": "player1",
  "email": "player@example.com",
  "current_chunk_x": 0,
  "current_chunk_z": 0,
  "current_world_x": 8.5,
  "current_world_y": 64.0,
  "current_world_z": 12.3,
  "is_online": true,
  "last_active": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-15T08:00:00Z"
}
```

**Status Codes:**
- `200` - Success
- `401` - Invalid or expired token

### Update Player Position

Update the player's current position in the world.

**Endpoint:** `PUT /players/me/position`

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "world_x": 8.5,
  "world_y": 64.0,
  "world_z": 12.3
}
```

**Response:**
```json
{
  "success": true,
  "message": "Position updated successfully",
  "position": {
    "world_x": 8.5,
    "world_y": 64.0,
    "world_z": 12.3,
    "chunk_x": 0,
    "chunk_z": 0
  }
}
```

**Status Codes:**
- `200` - Position updated
- `400` - Invalid coordinates
- `401` - Invalid or expired token

### Get Player Inventory

Get the player's current inventory.

**Endpoint:** `GET /players/me/inventory`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "inventory": [
    {
      "resource_type": 1,
      "resource_name": "Iron Ore",
      "quantity": 150
    },
    {
      "resource_type": 2,
      "resource_name": "Gold Ore",
      "quantity": 45
    }
  ]
}
```

**Status Codes:**
- `200` - Success
- `401` - Invalid or expired token

### Get Player Statistics

Get the player's gameplay statistics.

**Endpoint:** `GET /players/me/stats`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "total_resources_harvested": 2450,
  "total_playtime_minutes": 1200,
  "harvests_completed": 245,
  "chunks_visited": 12,
  "favorite_resource_type": 1,
  "last_harvest": "2024-01-15T10:25:00Z"
}
```

**Status Codes:**
- `200` - Success
- `401` - Invalid or expired token

### Get Online Players

List all currently online players.

**Endpoint:** `GET /players/online`

**Response:**
```json
{
  "players": [
    {
      "id": 123,
      "username": "player1",
      "current_chunk_x": 0,
      "current_chunk_z": 0,
      "last_active": "2024-01-15T10:30:00Z"
    },
    {
      "id": 124,
      "username": "player2",
      "current_chunk_x": 1,
      "current_chunk_z": 0,
      "last_active": "2024-01-15T10:29:00Z"
    }
  ]
}
```

**Status Codes:**
- `200` - Success

### Get Player Profile

Get public profile information for any player.

**Endpoint:** `GET /players/{playerID}/profile`

**Path Parameters:**
- `playerID` - The ID of the player

**Response:**
```json
{
  "id": 123,
  "username": "player1",
  "stats": {
    "total_resources_harvested": 2450,
    "total_playtime_minutes": 1200,
    "harvests_completed": 245,
    "chunks_visited": 12
  },
  "created_at": "2024-01-15T08:00:00Z"
}
```

**Status Codes:**
- `200` - Success
- `404` - Player not found

---

## Chunk Management

### Get Chunk Nodes

Load all resource nodes for a specific chunk.

**Endpoint:** `GET /chunks/{x}/{z}/nodes`

**Path Parameters:**
- `x` - Chunk X coordinate (integer)
- `z` - Chunk Z coordinate (integer)

**Response:**
```json
{
  "chunk": {
    "x": 0,
    "z": 0,
    "loaded_at": "2024-01-15T10:30:00Z"
  },
  "nodes": [
    {
      "id": 1,
      "chunk_x": 0,
      "chunk_z": 0,
      "local_x": 8,
      "local_z": 12,
      "resource_type": 1,
      "resource_subtype": 1,
      "current_yield": 450,
      "max_yield": 500,
      "quality_multiplier": 1.0,
      "regeneration_rate": 10,
      "spawn_behavior": 1,
      "respawn_timer": 0,
      "is_active": true,
      "last_harvest": "2024-01-15T10:25:00Z",
      "created_at": "2024-01-15T08:00:00Z"
    }
  ]
}
```

**Status Codes:**
- `200` - Success
- `400` - Invalid chunk coordinates

---

## Resource Harvesting

### Harvest Node

Harvest resources from a specific node.

**Endpoint:** `POST /nodes/{nodeId}/harvest`

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
- `nodeId` - The ID of the resource node

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
  "message": "Harvest successful",
  "harvest": {
    "resources_gained": 10,
    "resource_type": 1,
    "resource_name": "Iron Ore",
    "node_yield_remaining": 440,
    "quality_bonus": 0.0,
    "harvested_at": "2024-01-15T10:30:00Z"
  },
  "node": {
    "id": 1,
    "current_yield": 440,
    "max_yield": 500,
    "is_active": true,
    "respawn_timer": 0
  }
}
```

**Status Codes:**
- `200` - Harvest successful
- `400` - Invalid harvest amount or node depleted
- `401` - Invalid or expired token
- `404` - Node not found

---

## System Endpoints

### Health Check

Check if the API is running and healthy.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0"
}
```

**Status Codes:**
- `200` - System healthy

---

## Data Models

### Resource Types

| ID | Name | Description |
|----|------|-------------|
| 1 | Iron Ore | Basic mining resource |
| 2 | Gold Ore | Valuable mining resource |
| 3 | Wood | Renewable tree resource |
| 4 | Stone | Construction material |

### Resource Quality (Subtypes)

| ID | Name | Multiplier | Description |
|----|------|------------|-------------|
| 0 | Poor | 0.5x | Lower yield |
| 1 | Normal | 1.0x | Standard yield |
| 2 | Rich | 2.0x | Higher yield |

### Spawn Behaviors

| ID | Name | Description |
|----|------|-------------|
| 0 | Random | Appears randomly, respawns elsewhere |
| 1 | Static Daily | Fixed location, resets every 24 hours |
| 2 | Static Permanent | Always exists, regenerates continuously |

---

## Error Handling

All API endpoints return errors in a consistent format:

```json
{
  "success": false,
  "error": "Error message here",
  "code": "ERROR_CODE",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Common Error Codes

- `INVALID_REQUEST` - Malformed request data
- `UNAUTHORIZED` - Missing or invalid authentication
- `FORBIDDEN` - Insufficient permissions
- `NOT_FOUND` - Resource not found
- `CONFLICT` - Resource already exists
- `RATE_LIMITED` - Too many requests
- `INTERNAL_ERROR` - Server error

---

## Rate Limiting

The API implements rate limiting to prevent abuse:

- **Authentication endpoints**: 5 requests per minute per IP
- **Harvest endpoints**: 10 requests per minute per player
- **Other endpoints**: 100 requests per minute per IP

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642248600
```

---

## WebSocket Support (Future)

Real-time updates for:
- Player positions
- Resource node changes
- Harvest events
- Chat messages

*WebSocket implementation is planned for future releases.*