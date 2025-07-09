# Harvest UI Implementation

This document describes the implementation of the harvest feedback UI in the VoidMesh debug tool.

## Features Implemented

### 1. Current Harvest Target Display
- Shows the name of the currently targeted node (e.g., "Rich Iron Ore")
- Displays an HP bar with visual progress indicator using Unicode characters
- Shows current/max yield values (e.g., "5/10")
- Only appears when actively harvesting a node

### 2. Harvest Feedback Messages
- Displays the last `harvestMsg` from the harvesting system
- Shows loot obtained (e.g., "+1 Iron Ore, +1 Stone")
- Shows status messages like "Started harvesting" or "No node at cursor"
- Appears in the harvest status section

### 3. Harvest Status Indicators
- Shows "Harvesting in progress" when actively harvesting
- Shows "Harvest finished - Node depleted" when node is fully harvested
- Indicates harvest completion status

### 4. Visual Node Highlighting
- Target nodes are highlighted with a bright green background and black text
- Adds a white rounded border around the target node sprite
- Makes the harvest target clearly visible in the viewport
- Distinct from regular cursor highlighting

## UI Layout

The harvest feedback appears as a new section between the main content and status bar:

```
┌─────────────────────────────────────────────────────┐
│                  Chunk Explorer                     │
├─────────────────────────────────────────────────────┤
│  [Grid View]              [Info Panel]             │
├─────────────────────────────────────────────────────┤
│  Current Target: Rich Iron Ore                      │
│  Health: [████████████░░░░░░░░] 12/20               │
│  Feedback: +1 Iron Ore, +1 Stone                    │
│  Status: Harvesting in progress                     │
├─────────────────────────────────────────────────────┤
│  Status Bar                                         │
└─────────────────────────────────────────────────────┘
```

## Code Changes

### Components (`cmd/debug/components/styles.go`)
- Added `ProgressBarStyle` for HP bar display
- Added `HarvestStatusStyle` for the harvest feedback section

### Chunk Explorer (`cmd/debug/models/chunk_explorer.go`)
- Added `renderHarvestStatus()` function to render the harvest feedback UI
- Enhanced node highlighting in `renderGrid()` with green background and border
- Updated `View()` to include harvest status section
- Updated legend to document the new highlighting

## Usage

1. Navigate to a resource node using arrow keys
2. Press `H` to start harvesting
3. Press `Enter` or `Space` to perform harvest ticks
4. The UI will show:
   - Current target name and HP bar
   - Feedback messages about loot obtained
   - Status of the harvest operation
   - Visual highlighting of the target node

## Visual Styling

- **Harvest Status Panel**: Golden border with dark background
- **Target Node Highlighting**: Bright green background with white border
- **Progress Bar**: Visual bar with filled/empty indicators
- **Typography**: Bold text for important information

The implementation provides clear visual feedback to players about their harvesting activities, making the harvest system more engaging and informative.
