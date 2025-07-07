# VoidMesh Game Design & Development Summary

## Project Overview
**VoidMesh** is an API-first complex simulation game being developed by a solo Go developer with no prior game development experience. The goal is to create a Dwarf Fortress-inspired game with EVE Online's player-driven economy, targeting a small but dedicated community initially (first milestone: 10 active unknown players).

## Core Game Concept

### Game Loop & Mechanics
- **Primary Career Path (MVP):** Industrialist - harvest → refine → craft → sell
- **5-10 minute gameplay cycle:**
  1. Create character via LLM-generated biography
  2. Spawn at starting location with cash (background-dependent)
  3. Complete tutorial missions teaching game mechanics
  4. Explore, harvest resources, interact with world/players
  5. Build/trade for profit

### World Design
- **Multi-planetary universe** with fewer planets than EVE Online
- **Shared world** where players can interact in same zones
- **Procedural generation** using seeds for terrain/biomes
- **Chunk-based system** (similar to Minecraft) with biomes crossing chunk boundaries
- **No Z-levels initially** (design with z=1 everywhere for future expansion)

### Economy Design
- **Player-driven economy** with minimal NPC involvement
- **No pay-to-win:** Powerful items are soulbound/accountbound
- **Multi-step production** with quality, efficiency, time constraints
- **Real-time crafting delays** (some recipes limited to once per 24h)
- **Market system:** Buy/sell orders, instant transactions, taxes/fees
- **Black market:** Tax-free but no insurance/protection

## Technical Architecture

### API-First Approach
- **No graphics/frontend** - pure backend with public API
- **Protobuf consideration** for type safety
- **WebSocket communication** for real-time updates
- **JWT authentication** planned
- **Rate limiting** implementation needed
- **Breaking change management** for API versioning

### Database & Infrastructure
- **SQLite for development**, PostgreSQL for production
- **SQLC for typed SQL queries**
- **Chunk-based data storage** with performance caching
- **Real-time multiplayer state sync** via WebSocket events

### Map System (Detailed Schema Provided)
- **Chunk-based world** (considering 32x32 or 64x64 chunks)
- **Resource node system** with spawn types:
  - Random spawns
  - Static daily respawns
  - Static permanent nodes
- **Harvest session tracking** for multiplayer conflict resolution
- **Node spawn templates** for biome-specific resource distribution

## MVP Feature Prioritization

### Core MVP Features (Agreed Upon)
1. **Harvest resources** in shared generated world
2. **Basic crafting/refining** (simple iron ore → iron bar → iron tool chain)
3. **Selling for profit** to NPC market at static prices

### Deferred Features (Post-MVP)
- Corporations and contracts
- Dynamic events
- Player-to-player trading
- Complex market mechanics (call/put orders)
- Multiple planets/solar systems
- Social interaction systems

### Character Creation
- **LLM-powered biography generation** using local models (LM Studio)
- **Fallback to static random generation** if LLM integration proves too complex
- **Possible payment requirement** for LLM character creation

## Development Strategy

### Technology Stack
- **Backend:** Go with standard libraries or lightweight framework
- **Database:** SQLite → PostgreSQL migration path
- **Real-time:** WebSocket (likely gorilla/websocket)
- **API:** REST endpoints returning JSON
- **Testing Client:** CLI/TUI before web interface

### Development Approach
- **Start simple:** Single planet, basic resource chain
- **Iterative expansion:** Prove core loop before adding complexity
- **Performance optimization:** Cache pre-calculated data, optimize database queries
- **No premature optimization:** Address performance issues as they arise

### Key Technical Challenges Identified
1. **Real-time multiplayer state synchronization**
2. **Procedural map generation that "makes sense"**
3. **Resource harvesting conflict resolution**
4. **Efficient chunk loading/caching system**

## Business Model

### Target Metrics
- **Initial goal:** 10 active players who aren't personally known
- **Revenue target:** €1+ monthly (very conservative)
- **Community-supported development** model

### Monetization Strategy
- **Quality of life improvements** (not pay-to-win)
- **Subscription tiers** for enhanced features
- **Character creation premium options**
- **No direct item purchases** that affect gameplay balance

## Game Design Philosophy

### Complexity Management
- **"Rudimentary compared to Dwarf Fortress"** - simpler but still deep
- **Horizontal progression** similar to Guild Wars 2
- **Time investment required** for powerful items (no shortcuts)
- **Dynamic events** affect economy (supply/demand disruption)

### Player Agency
- **Multiple career paths** planned (industrialist focus for MVP)
- **Player-owned bases** (private instances like GW2 Home Instance)
- **Freelance contract system** for player-to-player services
- **Meaningful economic decisions** with real consequences

## Immediate Next Steps

### Development Priorities
1. Choose Go framework/router for API
2. Implement basic chunk generation endpoint
3. Add simple resource node spawning
4. Build player movement system
5. Create harvest mechanics

### Decision Points Reached
- **No more feature planning** - start coding and iterate
- **Performance optimization** will be addressed as needed
- **Complex systems** (multiplayer conflicts, advanced economy) deferred to post-MVP
- **Focus on core gameplay loop** first

## Key Insights from Discussion

### Scope Management Success
- Successfully trimmed ambitious initial concept to manageable MVP
- Identified clear progression path from simple to complex features
- Maintained vision while being realistic about solo development constraints

### Technical Pragmatism
- Chose familiar technology stack (Go) over potentially "better" but unknown options
- Planned migration path for database without over-engineering initially
- Accepted "good enough" solutions for MVP (static pricing, simple conflict resolution)

### Innovation Aspects
- **API-first game development** is genuinely novel approach
- **LLM character creation** could be compelling differentiator
- **No graphics constraint** forces focus on gameplay mechanics
- **Community-driven client development** could create unique ecosystem

---

*This summary captures the essence of VoidMesh as a focused, technically achievable project that balances ambitious vision with practical development constraints. The next phase is implementation and iteration based on real user feedback.*