# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Core Development Workflow
```bash
# Start full development environment with hot reload
docker-compose up

# Generate protobuf files after .proto changes
./scripts/generate_protobuf.sh

# Generate type-safe database code after SQL changes
sqlc generate -f api/db/sqlc.yaml

# Build production CSS
npm run build-css-prod

# Build API server
cd api && go build -o bin/api

# Build web server  
cd web && go build -o bin/web
```

### Service Access
- **Web Interface**: http://localhost:3000
- **gRPC API**: localhost:50051
- **gRPC UI**: http://localhost:50050 (grpcui)
- **Database UI**: http://localhost:5430 (pgweb)
- **Mail Testing**: http://localhost:8025 (mailpit)

## Architecture Overview

This is a **Meower Framework** application with microservice architecture:

### API Server (`/api`)
- **Language**: Go with gRPC
- **Database**: PostgreSQL with SQLC for type-safe queries
- **Services**: User, Meow, World, Chunk services
- **Authentication**: JWT middleware with configurable authentication
- **Entry Point**: `api/main.go` → `server/server.go`

### Web Server (`/web`) 
- **Framework**: Fiber (Go HTTP framework)
- **Templates**: Templ for type-safe HTML templating  
- **Styling**: TailwindCSS with hot reload
- **Communication**: gRPC client calls to API server
- **Sessions**: SQLite-backed sessions with encrypted cookies
- **Entry Point**: `web/main.go`

### Database Layer
- **ORM**: SQLC generates type-safe Go code from SQL
- **Schema**: `api/db/schema.sql`
- **Queries**: `api/db/query.*.sql` files per service
- **Config**: `api/db/sqlc.yaml`

### Protocol Buffers
- **Services**: user/v1, meow/v1, world/v1, chunk/v1
- **Location**: `api/proto/*/v1/*.proto`
- **Generation**: `scripts/generate_protobuf.sh`

## Development Patterns

### Adding New Features
1. Define protobuf service in `api/proto/[service]/v1/[service].proto`
2. Add database schema to `api/db/schema.sql`
3. Create queries in `api/db/query.[service].sql`
4. Generate code: `sqlc generate` and `./scripts/generate_protobuf.sh`
5. Implement handler in `api/server/handlers/[service].go`
6. Register service in `api/server/server.go`
7. Add web handlers in `web/handlers/[service].go`
8. Create templates in `web/views/services/[service]/v1/`
9. Register routes in `web/routing/routing.go`

### Code Generation Requirements
- Run `sqlc generate` after any SQL schema or query changes
- Run `./scripts/generate_protobuf.sh` after any .proto file changes
- Both are automatically watched in development via docker-compose

### Environment Variables
- `DATABASE_URL`: PostgreSQL connection string (API server only)
- `JWT_SECRET`: Required for API authentication
- `API_ENDPOINT`: gRPC server address (api:50051 in docker)
- `COOKIE_SECRET_KEY`: Web session encryption key
- `ENV`: "development" or "production"

## File Structure Conventions

### API Structure
- `api/proto/[service]/v1/` - Service definitions
- `api/server/handlers/` - Business logic implementations  
- `api/db/` - Database schema, queries, and generated code
- `api/server/middleware/` - gRPC interceptors (JWT auth)

### Web Structure  
- `web/handlers/` - HTTP request handlers
- `web/views/layouts/` - Base page templates
- `web/views/pages/` - Individual page templates
- `web/views/services/[service]/v1/` - Service-specific UI
- `web/static/src/css/` - Source CSS files
- `web/static/public/css/` - Generated CSS (auto-built)
- `web/routes/` - Route constants
- `web/routing/` - Route registration

## Testing & Debugging

### gRPC Testing
- Use grpcui at http://localhost:50050 for interactive API testing
- Health checks available at grpc-health-probe on :50051

### Database Access  
- pgweb UI at http://localhost:5430
- Direct connection: `postgres://meower:meower@localhost:5432/meower`

### Development Logs
```bash
# View specific service logs
docker-compose logs api
docker-compose logs web
docker-compose logs -f  # Follow all logs
```