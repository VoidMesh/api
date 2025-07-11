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

## Claude Code + Gemini Integration Strategy

### Overview

Claude Code can leverage Gemini Pro 2.5 as a powerful analysis tool while maintaining superior code quality. Gemini excels at data analysis and research but requires micro-management for code generation. Claude Code serves as the "brain" of the operation, using Gemini as a specialized tool for specific tasks.

### Gemini Capabilities

- **Large Context Window**: 2M+ tokens (vs Claude's smaller context)
- **Generous Free Tier**: Cost-effective for bulk operations
- **Parallel Execution**: Multiple instances can run simultaneously
- **Full Codebase Analysis**: `--all_files` flag for complete project context

### When to Use Gemini

**Ideal Use Cases:**
- **Research & Analysis**: Codebase exploration, pattern identification
- **Bulk Operations**: Generating boilerplate, test data, documentation
- **Cross-File Analysis**: Finding dependencies, unused code, security issues
- **Context-Heavy Tasks**: Migration planning, architectural analysis

**Avoid Using Gemini For:**
- **Critical Code Implementation**: Use Claude Code for precision
- **Security-Sensitive Logic**: Requires Claude's superior quality
- **Complex Business Logic**: Needs careful architectural consideration
- **Final Code Review**: Claude provides higher quality assurance

### Gemini Command Patterns

#### File and Directory Inclusion Syntax

Use the `@` syntax to include files and directories in your Gemini prompts. The paths should be relative to WHERE you run the gemini command:

**Single file analysis:**
```bash
gemini -p "@src/main.go Explain this file's purpose and structure"
```

**Multiple files:**
```bash
gemini -p "@go.mod @main.go Analyze the dependencies used in the code"
```

**Entire directory:**
```bash
gemini -p "@internal/ Summarize the architecture of this codebase"
```

**Multiple directories:**
```bash
gemini -p "@internal/ @api/ Analyze the API layer and internal logic"
```

**Current directory and subdirectories:**
```bash
gemini -p "@./ Give me an overview of this entire project"
```

#### Basic Analysis
```bash
# Full codebase analysis
gemini --all_files -p "Analyze this Go project for security vulnerabilities"

# Targeted file analysis with @ syntax
gemini -p "@internal/db/ Review database queries for optimization"

# Pattern detection
gemini --all_files -p "Find all functions that handle user authentication"
```

#### Implementation Verification Examples

**Check if a feature is implemented:**
```bash
gemini -p "@internal/ @api/ Has JWT authentication been implemented in this codebase? Show me the relevant files and functions"
```

**Verify specific patterns:**
```bash
gemini -p "@internal/ Are there any Go routines that handle concurrent harvesting? List them with file paths"
```

**Check for error handling:**
```bash
gemini -p "@internal/ @api/ Is proper error handling implemented for all database operations? Show examples of error handling patterns"
```

**Check for security measures:**
```bash
gemini -p "@internal/ @api/ Are SQL injection protections implemented? Show how user inputs are sanitized"
```

**Verify test coverage:**
```bash
gemini -p "@internal/harvest/ Are the harvest session mechanics fully tested? List all test cases"
```

#### Parallel Execution Strategy
```bash
# Parallel analysis of non-overlapping modules
gemini -p "@internal/chunk/ Analyze only chunk management layer" &
gemini -p "@internal/resource/ Analyze only resource node logic" &
gemini -p "@internal/harvest/ Analyze only harvest session handling" &
wait
```

#### File-Specific Parallel Tasks
```bash
# Each instance handles specific files to avoid conflicts
gemini -p "@internal/chunk/manager.go Suggest optimizations for this file only" &
gemini -p "@internal/db/queries.go Review SQL queries in this file only" &
gemini -p "@api/handlers.go Analyze validation in this file only" &
wait
```

### Integration Workflow

1. **Research Phase** (Gemini)
   - Use Gemini for bulk analysis and research
   - Gather suggestions and identify patterns
   - Generate implementation options

2. **Planning Phase** (Claude Code)
   - Review Gemini's findings critically
   - Make architectural decisions
   - Plan precise implementation strategy

3. **Implementation Phase** (Claude Code)
   - Write high-quality, precise code
   - Implement security-sensitive logic
   - Ensure proper error handling

4. **Review Phase** (Claude Code)
   - Final code review and quality assurance
   - Run tests and validation
   - Commit changes when appropriate

### Performance Benefits

- **3-5x Faster**: Multi-task completion through parallel execution
- **Token Efficiency**: Gemini handles bulk analysis (free/cheap)
- **Quality Maintenance**: Claude Code ensures implementation excellence
- **Safe Parallelization**: Non-overlapping scopes prevent conflicts

### Best Practices

- **Micro-Management**: Give Gemini specific, bounded tasks
- **Non-Overlapping Scope**: Ensure parallel tasks don't conflict
- **Quality Gates**: Always review Gemini output with Claude Code
- **Context Boundaries**: Use `--all_files` for full context when needed
- **Task Delegation**: Research to Gemini, implementation to Claude Code
- **Path Accuracy**: Ensure @ syntax paths are relative to current working directory
- **Specific Queries**: Be explicit about what you're looking for when checking implementations

### When to Use Gemini CLI

**Use gemini -p when:**
- Analyzing entire codebases or large directories
- Comparing multiple large files
- Need to understand project-wide patterns or architecture
- Current context window is insufficient for the task
- Working with files totaling more than 100KB
- Verifying if specific features, patterns, or security measures are implemented
- Checking for the presence of certain coding patterns across the entire codebase

**Important Notes:**
- Paths in @ syntax are relative to your current working directory when invoking gemini
- The CLI will include file contents directly in the context
- No need for --yolo flag for read-only analysis
- Gemini's context window can handle entire codebases that would overflow Claude's context
- When checking implementations, be specific about what you're looking for to get accurate results
