# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VoidMesh is built using the Meower Framework, an opinionated, production-ready Go web framework for building modern applications. The project follows a microservice architecture with separate API and web servers communicating via gRPC.

### Architecture

- **API Server**: gRPC-based service handling business logic, database operations
  - Located in `/api` directory
  - Provides services for Users, World, Character, and Chunks
  - Uses PostgreSQL with SQLC for type-safe database queries
  - Structured logging system with debug level support

- **Web Server**: HTTP server with server-side rendering
  - Located in `/web` directory
  - Uses Fiber for HTTP handling
  - Uses Templ for type-safe HTML templating
  - Communicates with the API server via gRPC
  - Uses SQLite for session storage

## Key Directories

```
/api
├── proto/                # Protocol Buffer definitions
│   ├── user/v1/         # User service definitions
│   ├── world/v1/        # World service definitions
│   ├── character/v1/    # Character service definitions
│   └── chunk/v1/        # Chunk service definitions
├── server/              # gRPC server implementation
│   ├── handlers/        # gRPC service implementations
│   └── middleware/      # JWT authentication middleware
├── services/            # Core business logic
│   ├── character/       # Character management & movement
│   ├── chunk/           # World chunk generation
│   ├── noise/           # Procedural generation utilities
│   └── world/           # World settings and configuration
├── db/                  # Database layer (SQLC generated)
│   ├── migrations/      # SQL migrations
│   │   └── schema.sql   # Database schema
│   ├── queries/         # SQL queries by domain
│   │   ├── query.users.sql
│   │   ├── query.characters.sql
│   │   ├── query.chunks.sql
│   │   └── query.worlds.sql
│   └── sqlc.yaml        # SQLC configuration
├── internal/            # Internal packages
│   └── logging/         # Structured logging system
└── main.go              # API server entry point

/web
├── handlers/            # HTTP request handlers
├── views/               # Templ templates
│   ├── components/      # Reusable UI components
│   ├── layouts/         # Page layouts
│   ├── pages/           # Page templates
│   └── game/            # Game-specific templates
├── routes/              # Route definitions
├── grpc/                # gRPC client code
├── static/              # Static assets (CSS, JS)
└── main.go              # Web server entry point
```

## Development Commands

### Starting the Development Environment

```bash
# Start all services (API, Web, DB, etc.)
docker-compose up
```

This starts all services with hot reload enabled:
- API server (gRPC) at localhost:50051
- Web server at http://localhost:3000
- PostgreSQL database
- Tailwind CSS watcher
- gRPC UI at http://localhost:50050
- Database UI (pgweb) at http://localhost:5430
- Mailpit (email testing) at http://localhost:8025

### Running in Development Mode

The development environment uses the following hot-reload components:
- `templ generate --watch` for live template reloading
- `wgo` for Go code hot reloading 
- TailwindCSS watcher for CSS changes

### Code Generation

```bash
# Generate code from Protocol Buffer definitions
./scripts/generate_protobuf.sh

# Generate Go code from SQL queries
# This happens automatically in Docker when SQL files change
# Manual command in the API docker container:
sqlc generate -f /src/api/db/sqlc.yaml
```

### Environment Variables

```
# API Configuration
LOG_LEVEL=debug       # or info, warn, error
DATABASE_URL=postgres://meower:meower@db:5432/meower?sslmode=disable
JWT_SECRET=your-secret-key

# Web Configuration
API_ENDPOINT=api:50051
COOKIE_SECRET_KEY=your-secret-key
ENV=development  # or production
```

## Building for Production

```bash
# Build API server
cd api && go build -o api ./main.go

# Build web server
cd web && go build -o web ./main.go

# Docker build
docker build -f api/Dockerfile -t voidmesh-api .
docker build -f web/Dockerfile -t voidmesh-web .
```

## Core Services and Components

### Authentication Flow
- JWT-based authentication system
- Tokens passed between services via gRPC metadata
- Middleware for securing API endpoints

### Character System
- Character creation, retrieval, and movement
- Position tracking with chunk-based coordinates
- Validation of movement within world constraints

### World Generation
- Procedural chunk generation with noise algorithms
- Terrain types including water, grass, stone
- Persistent chunk storage in database

### Logging System
- Structured logging with context fields
- Support for different log levels via environment variables
- Special helper functions for common context types (user_id, character_id, etc.)

## Project-Specific Notes

1. The project recently switched from PostgreSQL to SQLite for session storage (commit 5923fa9)
2. The web interface to play online was removed in commit 50f1362
3. The world component was split into character and world in commit 50f1362
4. Move cooldown was reduced from 200ms to 50ms for smoother gameplay (commit aca8865)

## Data Model

The project uses the following main database tables:
- `users`: User accounts with authentication information
- `characters`: Player characters with position data
- `chunks`: World map chunks with terrain data
- `worlds`: World configuration with seed, name, and creation time