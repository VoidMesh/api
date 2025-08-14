#!/bin/bash

# generate_mocks.sh - Generates Go mocks for VoidMesh API testing
# This script generates mocks for database interfaces, gRPC clients, and other dependencies

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TESTMOCKS_DIR="$API_DIR/internal/testmocks"

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}[$(date +'%Y-%m-%d %H:%M:%S')] ${message}${NC}"
}

# Function to check if mockgen is installed
check_mockgen() {
    if ! command -v mockgen >/dev/null 2>&1; then
        print_status "$RED" "mockgen not found. Installing..."
        go install go.uber.org/mock/mockgen@latest
        
        if ! command -v mockgen >/dev/null 2>&1; then
            print_status "$RED" "Failed to install mockgen. Please ensure GOPATH/bin is in your PATH."
            exit 1
        fi
        print_status "$GREEN" "mockgen installed successfully"
    else
        print_status "$BLUE" "mockgen already installed"
    fi
}

# Function to create directory structure
create_directories() {
    print_status "$YELLOW" "Creating mock directory structure..."
    
    mkdir -p "$TESTMOCKS_DIR/db"
    mkdir -p "$TESTMOCKS_DIR/grpc"
    mkdir -p "$TESTMOCKS_DIR/services"
    mkdir -p "$TESTMOCKS_DIR/external"
    mkdir -p "$TESTMOCKS_DIR/handlers"
    
    print_status "$GREEN" "Directory structure created"
}

# Function to generate database mocks
generate_db_mocks() {
    print_status "$YELLOW" "Generating database mocks..."
    
    # Mock the main Queries interface
    mockgen \
        -source="$API_DIR/db/db.go" \
        -destination="$TESTMOCKS_DIR/db/mock_queries.go" \
        -package=mockdb \
        -mock_names="DBTX=MockDBTX"
    
    # Create a comprehensive database mock interface file
    cat > "$TESTMOCKS_DIR/db/interfaces.go" << 'EOF'
// Package mockdb provides mock implementations of database interfaces for testing
package mockdb

import (
    "context"
    
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -source=interfaces.go -destination=mock_interfaces.go -package=mockdb

// Simple database interface for mocking without complex type dependencies
// These interfaces will be refined as needed for specific tests

// QuerierInterface represents a basic database querier interface
type QuerierInterface interface {
    // User operations - simplified signatures
    CreateUser(ctx context.Context, username, displayName, email, passwordHash string) (interface{}, error)
    GetUserByEmail(ctx context.Context, email string) (interface{}, error)
    GetUserByID(ctx context.Context, id string) (interface{}, error)
    GetUserByUsername(ctx context.Context, username string) (interface{}, error)
    
    // Character operations - simplified signatures
    CreateCharacter(ctx context.Context, userID, name string, x, y, chunkX, chunkY int32) (interface{}, error)
    GetCharacter(ctx context.Context, id string) (interface{}, error)
    GetCharactersByUserID(ctx context.Context, userID string) ([]interface{}, error)
    UpdateCharacterPosition(ctx context.Context, id string, x, y, chunkX, chunkY int32) (interface{}, error)
}

// PoolInterface represents the pgxpool.Pool interface for mocking
type PoolInterface interface {
    Acquire(ctx context.Context) (*pgxpool.Conn, error)
    Begin(ctx context.Context) (pgx.Tx, error)
    BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
    Close()
    Config() *pgxpool.Config
    Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
    Ping(ctx context.Context) error
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    Stat() *pgxpool.Stat
}
EOF
    
    print_status "$GREEN" "Database mocks generated"
}

# Function to generate gRPC service mocks
generate_grpc_mocks() {
    print_status "$YELLOW" "Generating gRPC service mocks..."
    
    # Generate mocks for all gRPC service interfaces
    local proto_dirs=(
        "user/v1"
        "character/v1" 
        "world/v1"
        "chunk/v1"
        "inventory/v1"
        "resource_node/v1"
        "terrain/v1"
    )
    
    for proto_dir in "${proto_dirs[@]}"; do
        service_name=$(echo "$proto_dir" | cut -d'/' -f1)
        
        if [[ -f "$API_DIR/proto/$proto_dir/${service_name}_grpc.pb.go" ]]; then
            print_status "$BLUE" "Generating mock for $service_name service..."
            
            mockgen \
                -source="$API_DIR/proto/$proto_dir/${service_name}_grpc.pb.go" \
                -destination="$TESTMOCKS_DIR/grpc/mock_${service_name}_service.go" \
                -package=mockgrpc
        fi
    done
    
    print_status "$GREEN" "gRPC service mocks generated"
}

# Function to generate service layer mocks
generate_service_mocks() {
    print_status "$YELLOW" "Generating service layer mocks..."
    
    # Create interface definitions for services that need mocking
    cat > "$TESTMOCKS_DIR/services/interfaces.go" << 'EOF'
// Package mockservices provides mock implementations of service interfaces for testing
package mockservices

import (
    "context"
)

//go:generate mockgen -source=interfaces.go -destination=mock_interfaces.go -package=mockservices

// Simple service interfaces for mocking without proto dependencies
// These will be refined as we develop the actual service interfaces

// CharacterServiceInterface represents a basic character service interface
type CharacterServiceInterface interface {
    CreateCharacter(ctx context.Context, userID, name string) (string, error)
    GetCharacter(ctx context.Context, id string) (interface{}, error)
    MoveCharacter(ctx context.Context, characterID string, x, y int32) error
    ListCharacters(ctx context.Context, userID string) ([]interface{}, error)
}

// ChunkServiceInterface represents a basic chunk service interface
type ChunkServiceInterface interface {
    GetChunk(ctx context.Context, worldID string, x, y int32) (interface{}, error)
    GenerateChunk(ctx context.Context, worldID string, x, y int32) (interface{}, error)
}

// WorldServiceInterface represents a basic world service interface  
type WorldServiceInterface interface {
    CreateWorld(ctx context.Context, name string, seed int64) (string, error)
    GetWorld(ctx context.Context, id string) (interface{}, error)
}
EOF
    
    # Generate the mocks
    mockgen \
        -source="$TESTMOCKS_DIR/services/interfaces.go" \
        -destination="$TESTMOCKS_DIR/services/mock_interfaces.go" \
        -package=mockservices
    
    print_status "$GREEN" "Service layer mocks generated"
}

# Function to generate handler mocks
generate_handler_mocks() {
    print_status "$YELLOW" "Generating handler dependency mocks..."
    
    # Create handler interface definitions if they don't exist
    if [[ ! -f "$TESTMOCKS_DIR/handlers/interfaces.go" ]]; then
        cat > "$TESTMOCKS_DIR/handlers/interfaces.go" << 'EOF'
// Package mockhandlers provides mock implementations of handler dependencies for testing
package mockhandlers

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	"github.com/jackc/pgx/v5/pgtype"
)

//go:generate mockgen -source=interfaces.go -destination=mock_interfaces.go -package=mockhandlers

// UserRepository defines the interface for user database operations.
// This mirrors the interface defined in server/handlers/interfaces.go
type UserRepository interface {
	// CreateUser creates a new user in the database
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error)

	// GetUserById retrieves a user by their ID
	GetUserById(ctx context.Context, id pgtype.UUID) (db.User, error)

	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (db.User, error)

	// GetUserByUsername retrieves a user by their username
	GetUserByUsername(ctx context.Context, username string) (db.User, error)

	// GetUserByResetToken retrieves a user by their password reset token
	GetUserByResetToken(ctx context.Context, resetToken pgtype.Text) (db.User, error)

	// UpdateUser updates user information
	UpdateUser(ctx context.Context, params db.UpdateUserParams) (db.User, error)

	// UpdateLastLoginAt updates the user's last login timestamp
	UpdateLastLoginAt(ctx context.Context, params db.UpdateLastLoginAtParams) (db.User, error)

	// UpdateLoginAttempts updates failed login attempts and account lock status
	UpdateLoginAttempts(ctx context.Context, params db.UpdateLoginAttemptsParams) (db.User, error)

	// UpdatePasswordResetToken updates or clears password reset token and expiration
	UpdatePasswordResetToken(ctx context.Context, params db.UpdatePasswordResetTokenParams) (db.User, error)

	// VerifyEmail marks a user's email as verified
	VerifyEmail(ctx context.Context, id pgtype.UUID) (db.User, error)

	// DeleteUser removes a user from the database
	DeleteUser(ctx context.Context, id pgtype.UUID) error

	// IndexUsers lists users with pagination
	IndexUsers(ctx context.Context, params db.IndexUsersParams) ([]db.User, error)
}

// JWTService defines the interface for JWT token operations.
// This mirrors the interface defined in server/handlers/interfaces.go
type JWTService interface {
	// GenerateToken creates a new JWT token for the given user
	GenerateToken(userID string, username string) (string, error)

	// ValidateToken validates a JWT token and returns the claims
	// This method is not currently used in the user handler but is included
	// for completeness and future use
	ValidateToken(tokenString string) (map[string]interface{}, error)
}

// PasswordService defines the interface for password operations.
// This mirrors the interface defined in server/handlers/interfaces.go
type PasswordService interface {
	// HashPassword hashes a plain text password
	HashPassword(password string) (string, error)

	// CheckPassword verifies a password against its hash
	CheckPassword(password, hash string) bool
}

// TokenGenerator defines the interface for generating random tokens.
// This mirrors the interface defined in server/handlers/interfaces.go
type TokenGenerator interface {
	// GenerateToken generates a random token of the specified length
	GenerateToken(length int) (string, error)
}

// CharacterService defines the interface for character service operations.
// This mirrors the interface defined in server/handlers/interfaces.go
type CharacterService interface {
	// CreateCharacter creates a new character for a user
	CreateCharacter(ctx context.Context, userID string, req *characterV1.CreateCharacterRequest) (*characterV1.CreateCharacterResponse, error)

	// GetCharacter retrieves a character by ID
	GetCharacter(ctx context.Context, req *characterV1.GetCharacterRequest) (*characterV1.GetCharacterResponse, error)

	// GetUserCharacters retrieves all characters for a user
	GetUserCharacters(ctx context.Context, userID string) (*characterV1.GetMyCharactersResponse, error)

	// DeleteCharacter deletes a character
	DeleteCharacter(ctx context.Context, req *characterV1.DeleteCharacterRequest) (*characterV1.DeleteCharacterResponse, error)

	// MoveCharacter moves a character
	MoveCharacter(ctx context.Context, req *characterV1.MoveCharacterRequest) (*characterV1.MoveCharacterResponse, error)
}

// WorldService defines the interface for world service operations.
// This mirrors the interface defined in server/handlers/interfaces.go
type WorldService interface {
	// GetWorldByID retrieves a world by ID
	GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error)

	// GetDefaultWorld gets or creates the default world
	GetDefaultWorld(ctx context.Context) (db.World, error)

	// ListWorlds retrieves all worlds
	ListWorlds(ctx context.Context) ([]db.World, error)

	// UpdateWorld updates a world's name
	UpdateWorld(ctx context.Context, id pgtype.UUID, name string) (db.World, error)

	// DeleteWorld deletes a world
	DeleteWorld(ctx context.Context, id pgtype.UUID) error
}
EOF
    fi
    
    # Generate handler mocks
    mockgen \
        -source="$TESTMOCKS_DIR/handlers/interfaces.go" \
        -destination="$TESTMOCKS_DIR/handlers/mock_interfaces.go" \
        -package=mockhandlers
    
    print_status "$GREEN" "Handler dependency mocks generated"
}

# Function to generate external dependency mocks
generate_external_mocks() {
    print_status "$YELLOW" "Generating external dependency mocks..."
    
    # Mock logger interface
    cat > "$TESTMOCKS_DIR/external/logger.go" << 'EOF'
// Package mockexternal provides mocks for external dependencies
package mockexternal

import (
    "context"
    "io"
    "time"
)

//go:generate mockgen -source=logger.go -destination=mock_logger.go -package=mockexternal

// LoggerInterface represents a basic logger interface for mocking
type LoggerInterface interface {
    Debug(msg interface{}, keyvals ...interface{})
    Info(msg interface{}, keyvals ...interface{})
    Warn(msg interface{}, keyvals ...interface{})
    Error(msg interface{}, keyvals ...interface{})
    Fatal(msg interface{}, keyvals ...interface{})
    With(keyvals ...interface{}) LoggerInterface
    SetOutput(w io.Writer)
}

// ContextInterface represents context operations for mocking
type ContextInterface interface {
    WithValue(parent context.Context, key, val interface{}) context.Context
    WithCancel(parent context.Context) (context.Context, context.CancelFunc)
    WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
}
EOF
    
    # Generate external mocks
    mockgen \
        -source="$TESTMOCKS_DIR/external/logger.go" \
        -destination="$TESTMOCKS_DIR/external/mock_logger.go" \
        -package=mockexternal
    
    print_status "$GREEN" "External dependency mocks generated"
}

# Function to create mock utilities
create_mock_utilities() {
    print_status "$YELLOW" "Creating mock utilities..."
    
    cat > "$TESTMOCKS_DIR/utils.go" << 'EOF'
// Package testmocks provides utilities for working with mocks in tests
package testmocks

import (
    "testing"
    
    "github.com/pashagolub/pgxmock/v4"
    "github.com/stretchr/testify/require"
    "go.uber.org/mock/gomock"
)

// MockController is a convenience wrapper around gomock.Controller
type MockController struct {
    *gomock.Controller
}

// NewMockController creates a new mock controller for the given test
func NewMockController(t *testing.T) *MockController {
    ctrl := gomock.NewController(t)
    
    // Ensure controller finishes properly
    t.Cleanup(ctrl.Finish)
    
    return &MockController{Controller: ctrl}
}

// NewPgxMock creates a new pgxmock for database testing
func NewPgxMock(t *testing.T) pgxmock.PgxPoolIface {
    mock, err := pgxmock.NewPool()
    require.NoError(t, err, "Failed to create pgx mock")
    
    t.Cleanup(func() {
        mock.Close()
    })
    
    return mock
}

// AssertMockExpectations verifies that all mock expectations were met
func AssertMockExpectations(t *testing.T, mocks ...interface{}) {
    t.Helper()
    
    for _, mock := range mocks {
        if pgxMock, ok := mock.(pgxmock.PgxPoolIface); ok {
            require.NoError(t, pgxMock.ExpectationsWereMet(), "pgx mock expectations were not met")
        }
        // Add other mock types as needed
    }
}
EOF
    
    print_status "$GREEN" "Mock utilities created"
}

# Function to generate all mocks using go generate
run_go_generate() {
    print_status "$YELLOW" "Running go generate to create remaining mocks..."
    
    # Change to the testmocks directory and run go generate
    cd "$TESTMOCKS_DIR"
    go generate ./...
    
    # Change back to API directory
    cd "$API_DIR"
    
    print_status "$GREEN" "go generate completed"
}

# Function to verify generated mocks compile
verify_mocks() {
    print_status "$YELLOW" "Verifying generated mocks compile..."
    
    # Try to build the testmocks package
    if go build ./internal/testmocks/...; then
        print_status "$GREEN" "All mocks compile successfully"
    else
        print_status "$RED" "Some mocks failed to compile"
        return 1
    fi
}

# Function to create makefile targets
create_makefile_targets() {
    print_status "$YELLOW" "Creating Makefile targets..."
    
    cat > "$API_DIR/scripts/test-targets.mk" << 'EOF'
# Test-related Makefile targets for VoidMesh API

# Check that all testing dependencies are installed and available
.PHONY: test-deps
test-deps:
	@echo "Checking testing dependencies..."
	@go list -m github.com/stretchr/testify >/dev/null 2>&1 || (echo "testify not found" && exit 1)
	@go list -m go.uber.org/mock >/dev/null 2>&1 || (echo "mock not found" && exit 1)
	@go list -m github.com/DATA-DOG/go-sqlmock >/dev/null 2>&1 || (echo "go-sqlmock not found" && exit 1)
	@go list -m github.com/pashagolub/pgxmock/v4 >/dev/null 2>&1 || (echo "pgxmock not found" && exit 1)
	@command -v mockgen >/dev/null 2>&1 || (echo "mockgen not found in PATH" && exit 1)
	@echo "All testing dependencies are available"

# Generate and verify all mocks
.PHONY: mocks
mocks:
	@echo "Generating all mocks..."
	@./scripts/generate_mocks.sh
	@echo "Mocks generated and verified successfully"

# Clean up generated mocks
.PHONY: clean-mocks
clean-mocks:
	@echo "Cleaning up generated mocks..."
	@rm -rf internal/testmocks/*/mock_*.go
	@echo "Generated mocks cleaned up"

# Run mock generation and verify they work
.PHONY: test-setup
test-setup: test-deps mocks
	@echo "Testing mock generation and compilation..."
	@go build ./internal/testmocks/...
	@go build ./internal/testutil/...
	@echo "Test setup verification completed successfully"

# Start test database
.PHONY: test-db-start
test-db-start:
	@echo "Starting test database..."
	@docker-compose -f docker-compose.test.yml up -d test-db
	@echo "Waiting for test database to be ready..."
	@until docker-compose -f docker-compose.test.yml exec test-db pg_isready -U meower -d meower_test; do sleep 1; done
	@echo "Test database is ready"

# Stop test database
.PHONY: test-db-stop
test-db-stop:
	@echo "Stopping test database..."
	@docker-compose -f docker-compose.test.yml down
	@echo "Test database stopped"

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests (requires test database)
.PHONY: test-integration
test-integration: test-db-start
	@echo "Running integration tests..."
	@TEST_DATABASE_URL="postgres://meower:meower@localhost:5433/meower_test?sslmode=disable" go test -v -tags=integration ./...
	@echo "Integration tests completed"

# Run all tests
.PHONY: test-all
test-all: test-setup test-coverage test-integration
	@echo "All tests completed successfully"
EOF
    
    print_status "$GREEN" "Makefile targets created"
}

# Main execution
main() {
    print_status "$BLUE" "Starting mock generation for VoidMesh API..."
    
    # Change to API directory
    cd "$API_DIR"
    
    # Execute all steps
    check_mockgen
    create_directories
    generate_db_mocks
    generate_grpc_mocks
    generate_service_mocks
    generate_handler_mocks
    generate_external_mocks
    create_mock_utilities
    run_go_generate
    verify_mocks
    create_makefile_targets
    
    print_status "$GREEN" "Mock generation completed successfully!"
    print_status "$BLUE" "To regenerate mocks, run: make mocks"
    print_status "$BLUE" "To test setup, run: make test-setup"
}

# Execute main function
main "$@"