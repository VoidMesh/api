# Debug Environment Persistence

## Overview

The debug tool now includes explicit persistence hooks that ensure player and chunk updates are properly saved after every `ProcessHarvestTick` operation. This feature is controlled by environment variables and provides explicit `Save()` calls to both the player manager and chunk manager.

## Implementation Details

### New Methods Added

1. **chunk.Manager.Save(ctx context.Context) error**
   - Persists chunk manager state to database
   - Clears stale cache entries
   - Ensures database transaction commits

2. **player.Manager.Save(ctx context.Context) error**
   - Persists player manager state to database
   - Flushes pending player updates
   - Ensures database transaction commits

3. **ChunkExplorerModel.persistHarvestUpdates(ctx context.Context, playerID int64)**
   - Orchestrates persistence of both player and chunk updates
   - Called after every `ProcessHarvestTick` if debug environment requires it

### Environment Variables

The persistence feature can be enabled using any of these environment variables:

- `DEBUG_PERSIST=true` or `DEBUG_PERSIST=1` - Explicit persistence flag
- `DEBUG=true` or `DEBUG=1` - General debug flag that enables persistence

### Usage Examples

#### Enable persistence with DEBUG_PERSIST:
```bash
DEBUG_PERSIST=true go run main.go -view=chunks
```

#### Enable persistence with DEBUG:
```bash
DEBUG=1 go run main.go -view=chunks
```

#### Normal operation (no persistence):
```bash
go run main.go -view=chunks
```

## How It Works

1. **Normal Harvesting**: When a player performs a harvest tick, the `ProcessHarvestTick` method is called
2. **Persistence Check**: After the harvest operation, the method checks if persistence is required using environment variables
3. **Save Operations**: If persistence is enabled, it calls:
   - `chunkManager.Save(ctx)` - Persists chunk updates
   - `playerManager.Save(ctx)` - Persists player updates
4. **Error Handling**: Any persistence errors are logged but don't fail the harvest operation

## Code Flow

```
ProcessHarvestTick()
├── Perform harvest logic
├── Update harvest messages
├── Check shouldPersistInDebug()
├── If persistence required:
│   ├── Call persistChunkUpdates()
│   │   └── chunkManager.Save(ctx)
│   └── Call persistPlayerUpdates()
│       └── playerManager.Save(ctx)
└── Return load chunk command
```

## Testing

The persistence logic can be tested using the included test script:

```bash
go run test_persist_simple.go
```

This will test all environment variable combinations and verify the persistence logic works correctly.

## Benefits

1. **Explicit Control**: Debug environment can control when persistence occurs
2. **Performance**: Persistence only happens when explicitly requested
3. **Reliability**: Ensures data is properly committed in debug scenarios
4. **Logging**: Full logging of persistence operations for debugging
5. **Error Handling**: Graceful handling of persistence failures

## Production Considerations

- In production, persistence is handled automatically by the database layer
- The debug persistence hooks are specifically for development/debugging scenarios
- The feature does not impact normal operation when environment variables are not set
