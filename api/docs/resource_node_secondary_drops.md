# Resource Node Secondary Drops System

## Overview

This document describes the secondary drops system for resource nodes in VoidMesh. When a player harvests a resource node, they receive not only the primary resource but may also obtain secondary materials related to that resource.

## Implementation

Each resource type has a `secondary_drops` array in its properties JSON object that defines what additional resources can be obtained when harvesting that node. The system is designed to be realistic, with secondary drops that make sense for each resource type.

### Secondary Drops Structure

The `secondary_drops` property uses the following format:

```json
"secondary_drops": [
  {
    "name": "Resource Name",
    "chance": 0.7,       // 70% chance to drop
    "min": 1,            // Minimum quantity
    "max": 3             // Maximum quantity
  },
  {
    "name": "Another Resource",
    "chance": 0.3,
    "min": 1,
    "max": 2
  }
]
```

When a player harvests a resource node, the system:
1. Always provides the primary resource based on `yield_min` and `yield_max`
2. For each entry in `secondary_drops`:
   - Generates a random number between 0 and 1
   - If the number is less than `chance`, the secondary resource drops
   - Determines a random quantity between `min` and `max` inclusive

## Secondary Resources

### Grass Terrain Resources

**Herb Patch**
- Primary: Medicinal Herbs (1-3)
- Secondary:
  - Common Grass (70% chance, 1-2 units)
  - Seeds (30% chance, 1 unit)

**Berry Bush**
- Primary: Berries (2-5)
- Secondary:
  - Twigs (50% chance, 1-2 units)
  - Leaves (60% chance, 1-3 units)

**Mineral Outcropping**
- Primary: Minerals (1-3)
- Secondary:
  - Stone (80% chance, 1-3 units)
  - Dirt (40% chance, 1-2 units)

### Water Terrain Resources

**Fishing Spot**
- Primary: Fish (1-3)
- Secondary:
  - Algae (40% chance, 1-2 units)
  - Shells (20% chance, 1 unit)

**Kelp Bed**
- Primary: Kelp (2-4)
- Secondary:
  - Salt (30% chance, 1 unit)
  - Tiny Fish (25% chance, 1 unit)

**Pearl Formation**
- Primary: Pearls (1-2)
- Secondary:
  - Shells (90% chance, 2-4 units)
  - Sand (50% chance, 1-2 units)

### Sand Terrain Resources

**Crystal Formation**
- Primary: Crystals (1-3)
- Secondary:
  - Sand (80% chance, 2-4 units)
  - Stone Fragments (40% chance, 1-2 units)

**Clay Deposit**
- Primary: Clay (2-5)
- Secondary:
  - Sand (70% chance, 1-3 units)
  - Silt (40% chance, 1-2 units)

**Desert Plant**
- Primary: Desert Herbs (1-2)
- Secondary:
  - Sand (60% chance, 1-2 units)
  - Seeds (20% chance, 1 unit)

### Wood/Dirt Terrain Resources

**Harvestable Tree**
- Primary: Wood (3-8)
- Secondary:
  - Sticks (80% chance, 2-4 units)
  - Leaves (90% chance, 3-6 units)
  - Bark (40% chance, 1-2 units)

**Mushroom Circle**
- Primary: Mushrooms (2-6)
- Secondary:
  - Spores (30% chance, 1 unit)
  - Dirt (60% chance, 1-2 units)

**Wild Honey Hive**
- Primary: Honey (1-4)
- Secondary:
  - Beeswax (70% chance, 1-2 units)
  - Bark (30% chance, 1 unit)

### Stone Terrain Resources

**Stone Vein**
- Primary: Stone Blocks (3-8)
- Secondary:
  - Gravel (60% chance, 2-4 units)
  - Dust (40% chance, 1-2 units)

**Gem Deposit**
- Primary: Gems (1-3)
- Secondary:
  - Stone (90% chance, 2-4 units)
  - Crystal Fragments (40% chance, 1-2 units)

**Metal Ore**
- Primary: Metal Ore (2-5)
- Secondary:
  - Stone (90% chance, 2-5 units)
  - Pyrite (30% chance, 1-2 units)
  - Sulfur (20% chance, 1 unit)

## Geological Accuracy

The secondary drops for metal ores have been designed with geological accuracy in mind:

- Metal ores are commonly found embedded in stone, hence the high chance of stone drops
- Pyrite (fool's gold) is a common mineral association with many metal deposits
- Sulfur is frequently found in metal sulfide deposits, a common form of metal ores

This system provides realistic resource gathering that reflects natural associations between materials.

## Future Enhancements

Future versions of the secondary drops system could include:

1. **Skill-based modifiers**: Player skills could increase the chance or quantity of secondary drops
2. **Harvesting tool effects**: Different tools could yield different secondary materials
3. **Special rare drops**: Very low chance for valuable rare materials
4. **Seasonal variations**: Different seasons affecting drop rates or types