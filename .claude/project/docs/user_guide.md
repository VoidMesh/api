# VoidMesh API User Guide

## Overview

VoidMesh API is a backend service for multiplayer resource harvesting games. This guide explains how to integrate with the API and implement client-side functionality for chunk-based resource management.

## Quick Start

### Basic Integration

1. **Start the server:**
```bash
./voidmesh-api
# Server runs on http://localhost:8080
```

2. **Test the connection:**
```bash
curl http://localhost:8080/health
```

3. **Load your first chunk:**
```bash
curl http://localhost:8080/api/v1/chunks/0/0/nodes
```

### Basic Workflow

1. **Load Chunk** → Get available resource nodes
2. **Start Harvest** → Begin harvesting a resource
3. **Harvest Resources** → Extract resources from nodes
4. **Monitor Sessions** → Track active harvesting sessions

## Core Concepts

### Chunks

The world is divided into 16x16 chunks. Each chunk can contain multiple resource nodes.

- **Chunk Coordinates**: Integer coordinates (can be negative)
- **Local Coordinates**: Position within chunk (0-15)
- **Dynamic Loading**: Chunks are loaded on-demand

### Resource Nodes

Harvestable objects in the world with finite resources that regenerate over time.

**Node Types:**
- **Iron Ore** (Type 1): Basic mining resource
- **Gold Ore** (Type 2): Valuable mining resource  
- **Wood** (Type 3): Renewable resource from trees
- **Stone** (Type 4): Construction material

**Quality Levels:**
- **Poor** (0): Lower yield
- **Normal** (1): Standard yield
- **Rich** (2): Higher yield

**Spawn Behaviors:**
- **Random Spawn** (0): Appears randomly, respawns elsewhere when depleted
- **Static Daily** (1): Fixed location, resets every 24 hours
- **Static Permanent** (2): Always exists, regenerates continuously

### Harvest Sessions

Prevents players from indefinitely claiming nodes while allowing concurrent harvesting.

- **5-minute timeout**: Sessions expire after 5 minutes of inactivity
- **One session per player**: Players can only have one active session
- **Concurrent harvesting**: Multiple players can harvest the same node

## API Usage Examples

### JavaScript/TypeScript

```javascript
class VoidMeshClient {
    constructor(baseUrl = 'http://localhost:8080/api/v1') {
        this.baseUrl = baseUrl;
    }

    async loadChunk(chunkX, chunkZ) {
        const response = await fetch(`${this.baseUrl}/chunks/${chunkX}/${chunkZ}/nodes`);
        return response.json();
    }

    async startHarvest(nodeId, playerId) {
        const response = await fetch(`${this.baseUrl}/harvest/start`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ node_id: nodeId, player_id: playerId })
        });
        return response.json();
    }

    async harvestResource(sessionId, amount) {
        const response = await fetch(`${this.baseUrl}/harvest/sessions/${sessionId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ harvest_amount: amount })
        });
        return response.json();
    }

    async getPlayerSessions(playerId) {
        const response = await fetch(`${this.baseUrl}/players/${playerId}/sessions`);
        return response.json();
    }
}

// Usage example
const client = new VoidMeshClient();

async function gameLoop() {
    const playerId = 123;
    
    // Load chunk around player
    const chunk = await client.loadChunk(0, 0);
    console.log(`Loaded chunk with ${chunk.nodes.length} nodes`);
    
    // Find an active node
    const activeNode = chunk.nodes.find(node => node.is_active && node.current_yield > 0);
    if (!activeNode) {
        console.log('No active nodes found');
        return;
    }
    
    // Start harvesting
    const session = await client.startHarvest(activeNode.node_id, playerId);
    console.log(`Started harvest session ${session.session_id}`);
    
    // Harvest resources
    const result = await client.harvestResource(session.session_id, 10);
    console.log(`Harvested ${result.amount_harvested} resources`);
    
    // Check all player sessions
    const sessions = await client.getPlayerSessions(playerId);
    console.log(`Player has ${sessions.sessions.length} active sessions`);
}
```

### Python

```python
import requests
import json

class VoidMeshClient:
    def __init__(self, base_url="http://localhost:8080/api/v1"):
        self.base_url = base_url
        self.session = requests.Session()
    
    def load_chunk(self, chunk_x, chunk_z):
        response = self.session.get(f"{self.base_url}/chunks/{chunk_x}/{chunk_z}/nodes")
        return response.json()
    
    def start_harvest(self, node_id, player_id):
        data = {"node_id": node_id, "player_id": player_id}
        response = self.session.post(f"{self.base_url}/harvest/start", json=data)
        return response.json()
    
    def harvest_resource(self, session_id, amount):
        data = {"harvest_amount": amount}
        response = self.session.put(f"{self.base_url}/harvest/sessions/{session_id}", json=data)
        return response.json()
    
    def get_player_sessions(self, player_id):
        response = self.session.get(f"{self.base_url}/players/{player_id}/sessions")
        return response.json()

# Usage example
client = VoidMeshClient()
player_id = 123

# Load chunk
chunk = client.load_chunk(0, 0)
print(f"Loaded chunk with {len(chunk['nodes'])} nodes")

# Find active node
active_nodes = [node for node in chunk['nodes'] if node['is_active'] and node['current_yield'] > 0]
if active_nodes:
    node = active_nodes[0]
    
    # Start harvest
    session = client.start_harvest(node['node_id'], player_id)
    print(f"Started session {session['session_id']}")
    
    # Harvest resources
    result = client.harvest_resource(session['session_id'], 10)
    print(f"Harvested {result['amount_harvested']} resources")
```

### Unity C#

```csharp
using System;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.Networking;
using System.Collections;

public class VoidMeshClient : MonoBehaviour
{
    private string baseUrl = "http://localhost:8080/api/v1";
    
    [System.Serializable]
    public class ChunkResponse
    {
        public int chunk_x;
        public int chunk_z;
        public List<ResourceNode> nodes;
    }
    
    [System.Serializable]
    public class ResourceNode
    {
        public int node_id;
        public int chunk_x;
        public int chunk_z;
        public int local_x;
        public int local_z;
        public int node_type;
        public int node_subtype;
        public int max_yield;
        public int current_yield;
        public int regeneration_rate;
        public string spawned_at;
        public string last_harvest;
        public string respawn_timer;
        public int spawn_type;
        public bool is_active;
    }
    
    [System.Serializable]
    public class HarvestSession
    {
        public int session_id;
        public int node_id;
        public int player_id;
        public string started_at;
        public string last_activity;
        public int resources_gathered;
    }
    
    [System.Serializable]
    public class StartHarvestRequest
    {
        public int node_id;
        public int player_id;
    }
    
    [System.Serializable]
    public class HarvestRequest
    {
        public int harvest_amount;
    }
    
    [System.Serializable]
    public class HarvestResponse
    {
        public bool success;
        public int amount_harvested;
        public int node_yield_after;
        public int resources_gathered;
    }
    
    public IEnumerator LoadChunk(int chunkX, int chunkZ, System.Action<ChunkResponse> callback)
    {
        string url = $"{baseUrl}/chunks/{chunkX}/{chunkZ}/nodes";
        
        using (UnityWebRequest request = UnityWebRequest.Get(url))
        {
            yield return request.SendWebRequest();
            
            if (request.result == UnityWebRequest.Result.Success)
            {
                ChunkResponse chunk = JsonUtility.FromJson<ChunkResponse>(request.downloadHandler.text);
                callback(chunk);
            }
            else
            {
                Debug.LogError($"Failed to load chunk: {request.error}");
            }
        }
    }
    
    public IEnumerator StartHarvest(int nodeId, int playerId, System.Action<HarvestSession> callback)
    {
        string url = $"{baseUrl}/harvest/start";
        StartHarvestRequest requestData = new StartHarvestRequest { node_id = nodeId, player_id = playerId };
        string json = JsonUtility.ToJson(requestData);
        
        using (UnityWebRequest request = UnityWebRequest.Put(url, json))
        {
            request.method = "POST";
            request.SetRequestHeader("Content-Type", "application/json");
            
            yield return request.SendWebRequest();
            
            if (request.result == UnityWebRequest.Result.Success)
            {
                HarvestSession session = JsonUtility.FromJson<HarvestSession>(request.downloadHandler.text);
                callback(session);
            }
            else
            {
                Debug.LogError($"Failed to start harvest: {request.error}");
            }
        }
    }
    
    public IEnumerator HarvestResource(int sessionId, int amount, System.Action<HarvestResponse> callback)
    {
        string url = $"{baseUrl}/harvest/sessions/{sessionId}";
        HarvestRequest requestData = new HarvestRequest { harvest_amount = amount };
        string json = JsonUtility.ToJson(requestData);
        
        using (UnityWebRequest request = UnityWebRequest.Put(url, json))
        {
            request.SetRequestHeader("Content-Type", "application/json");
            
            yield return request.SendWebRequest();
            
            if (request.result == UnityWebRequest.Result.Success)
            {
                HarvestResponse response = JsonUtility.FromJson<HarvestResponse>(request.downloadHandler.text);
                callback(response);
            }
            else
            {
                Debug.LogError($"Failed to harvest resource: {request.error}");
            }
        }
    }
}

// Usage in a MonoBehaviour
public class GameController : MonoBehaviour
{
    public VoidMeshClient client;
    public int playerId = 123;
    
    void Start()
    {
        // Load chunk around player
        StartCoroutine(client.LoadChunk(0, 0, OnChunkLoaded));
    }
    
    void OnChunkLoaded(VoidMeshClient.ChunkResponse chunk)
    {
        Debug.Log($"Loaded chunk with {chunk.nodes.Count} nodes");
        
        // Find active node
        var activeNode = chunk.nodes.Find(node => node.is_active && node.current_yield > 0);
        if (activeNode != null)
        {
            // Start harvesting
            StartCoroutine(client.StartHarvest(activeNode.node_id, playerId, OnHarvestStarted));
        }
    }
    
    void OnHarvestStarted(VoidMeshClient.HarvestSession session)
    {
        Debug.Log($"Started harvest session {session.session_id}");
        
        // Harvest some resources
        StartCoroutine(client.HarvestResource(session.session_id, 10, OnResourceHarvested));
    }
    
    void OnResourceHarvested(VoidMeshClient.HarvestResponse response)
    {
        Debug.Log($"Harvested {response.amount_harvested} resources");
    }
}
```

## Game Design Considerations

### Resource Economics

1. **Scarcity Management:**
   - Nodes have limited yield
   - Regeneration rates control supply
   - Respawn timers prevent over-harvesting

2. **Player Interaction:**
   - Multiple players can harvest the same node
   - Session timeouts prevent node hoarding
   - Shared resource pools create competition

3. **Exploration Incentives:**
   - Random spawn nodes encourage exploration
   - Different chunks have different resources
   - Noise-based distribution creates natural patterns

### Balancing Parameters

**Yield Values:**
- Iron Ore: 100-500 (basic resource)
- Gold Ore: 50-300 (valuable but scarce)
- Wood: 50-100 (renewable)
- Stone: 75-150 (construction material)

**Regeneration Rates:**
- Iron: 5-10 per hour
- Gold: 2-5 per hour
- Wood: 1 per hour (trees grow slowly)
- Stone: 2 per hour

**Respawn Timers:**
- Iron: 24 hours
- Gold: 12 hours
- Wood: 6 hours
- Stone: 8 hours

### Client-Side Optimization

1. **Chunk Loading:**
   - Load chunks as players move
   - Cache chunk data client-side
   - Unload distant chunks to save memory

2. **Session Management:**
   - Track session timeouts locally
   - Automatically retry failed requests
   - Show session status in UI

3. **Resource Updates:**
   - Poll for resource changes
   - Update UI when nodes are depleted
   - Show regeneration progress

## Common Integration Patterns

### Real-Time Updates

```javascript
// Poll for chunk updates
setInterval(async () => {
    const chunk = await client.loadChunk(currentChunkX, currentChunkZ);
    updateGameWorld(chunk);
}, 30000); // Every 30 seconds

// Check session status
setInterval(async () => {
    const sessions = await client.getPlayerSessions(playerId);
    updateSessionUI(sessions);
}, 5000); // Every 5 seconds
```

### Error Handling

```javascript
async function safeApiCall(apiFunction, ...args) {
    try {
        return await apiFunction(...args);
    } catch (error) {
        if (error.status === 400) {
            // Handle client errors (bad requests)
            console.warn('Invalid request:', error.message);
        } else if (error.status >= 500) {
            // Handle server errors
            console.error('Server error:', error.message);
            // Retry after delay
            setTimeout(() => safeApiCall(apiFunction, ...args), 5000);
        }
        throw error;
    }
}
```

### Progressive Loading

```javascript
// Load chunks in a spiral pattern around player
function loadChunksAroundPlayer(playerChunkX, playerChunkZ, radius) {
    const chunks = [];
    for (let x = -radius; x <= radius; x++) {
        for (let z = -radius; z <= radius; z++) {
            chunks.push({
                x: playerChunkX + x,
                z: playerChunkZ + z,
                distance: Math.abs(x) + Math.abs(z)
            });
        }
    }
    
    // Sort by distance and load closest first
    chunks.sort((a, b) => a.distance - b.distance);
    
    for (const chunk of chunks) {
        client.loadChunk(chunk.x, chunk.z).then(data => {
            updateChunkInGame(chunk.x, chunk.z, data);
        });
    }
}
```

## Troubleshooting

### Common Issues

1. **"Session expired" errors:**
   - Sessions timeout after 5 minutes
   - Check session status before harvesting
   - Restart session if needed

2. **"Node not found" errors:**
   - Node may have been depleted
   - Check if node is still active
   - Reload chunk data

3. **"Player already has active session" errors:**
   - Each player can only have one session
   - Check existing sessions with `/players/{id}/sessions`
   - Wait for session to expire or complete

### Performance Tips

1. **Batch chunk loading:**
   - Load multiple chunks in parallel
   - Use Promise.all() for concurrent requests

2. **Cache management:**
   - Cache chunk data for 30-60 seconds
   - Update cache when nodes are modified
   - Clear cache when moving to new areas

3. **Session monitoring:**
   - Track session timeouts locally
   - Warn players before sessions expire
   - Automatically restart sessions when needed

## Advanced Features

### Custom Resource Types

Add new resource types by inserting into `node_spawn_templates`:

```sql
INSERT INTO node_spawn_templates (
    node_type, node_subtype, spawn_type, 
    min_yield, max_yield, regeneration_rate,
    respawn_delay_hours, spawn_weight
) VALUES (
    5, 1, 1,  -- New resource type 5
    200, 400, 8,  -- Yield range and regen
    48, 2  -- Respawn delay and weight
);
```

### Biome-Based Spawning

Configure biome restrictions in spawn templates:

```sql
UPDATE node_spawn_templates 
SET biome_restriction = '["mountain", "hill"]' 
WHERE node_type = 1; -- Iron ore only in mountains
```

### Analytics Integration

Track player behavior:

```javascript
// Track harvest events
analytics.track('resource_harvested', {
    player_id: playerId,
    node_type: nodeType,
    amount: harvestAmount,
    chunk_x: chunkX,
    chunk_z: chunkZ
});

// Track session durations
analytics.track('harvest_session_ended', {
    player_id: playerId,
    session_duration: sessionDuration,
    resources_gathered: totalGathered
});
```

This user guide provides everything needed to integrate with the VoidMesh API and build compelling multiplayer resource harvesting experiences. For technical implementation details, refer to the API Documentation and Developer Guide.