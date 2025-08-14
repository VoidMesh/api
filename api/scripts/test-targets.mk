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
