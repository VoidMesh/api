package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"

	"github.com/VoidMesh/api/api/db"
)

// TestDB represents a test database connection and associated utilities
type TestDB struct {
	Pool     *pgxpool.Pool
	MockPool pgxmock.PgxPoolIface
	Queries  *db.Queries
	Config   *pgxpool.Config
	IsMock   bool
	cleanup  func()
}

// DatabaseConfig holds configuration for test database setup
type DatabaseConfig struct {
	// DatabaseURL is the connection string for the test database
	DatabaseURL string
	// UseMockDB determines whether to use a mock database instead of a real one
	UseMockDB bool
	// SeedData determines whether to insert test seed data
	SeedData bool
	// IsolateTransactions determines whether each test should run in a transaction that's rolled back
	IsolateTransactions bool
	// MaxConnections is the maximum number of connections in the pool
	MaxConnections int32
}

// DefaultDatabaseConfig returns a default database configuration for testing
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		DatabaseURL:         getTestDatabaseURL(),
		UseMockDB:           false,
		SeedData:            true,
		IsolateTransactions: true,
		MaxConnections:      5, // Keep low for testing
	}
}

// MockDatabaseConfig returns a configuration for mock database testing
func MockDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		UseMockDB:           true,
		SeedData:            false,
		IsolateTransactions: false,
	}
}

// SetupTestDB creates and initializes a test database connection.
// It supports both real database connections and mock databases based on configuration.
//
// Usage with real database:
//
//	testDB := testutil.SetupTestDB(t, testutil.DefaultDatabaseConfig())
//	defer testDB.Close()
//
// Usage with mock database:
//
//	testDB := testutil.SetupTestDB(t, testutil.MockDatabaseConfig())
//	defer testDB.Close()
func SetupTestDB(t *testing.T, config *DatabaseConfig) *TestDB {
	t.Helper()

	if config.UseMockDB {
		return setupMockDB(t, config)
	}

	return setupRealDB(t, config)
}

// setupRealDB creates a connection to a real test database
func setupRealDB(t *testing.T, config *DatabaseConfig) *TestDB {
	t.Helper()

	ctx := CreateTestContext()

	// Parse the database URL
	pgConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	require.NoError(t, err, "Failed to parse database URL")

	// Configure connection pool
	pgConfig.MaxConns = config.MaxConnections
	pgConfig.MinConns = 1
	pgConfig.MaxConnLifetime = time.Hour
	pgConfig.MaxConnIdleTime = time.Minute * 30

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	require.NoError(t, err, "Failed to create database pool")

	// Test the connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping database")

	// Create test database instance
	testDB := &TestDB{
		Pool:    pool,
		Queries: db.New(pool),
		Config:  pgConfig,
		cleanup: func() { pool.Close() },
	}

	// Setup schema and seed data if requested
	if config.SeedData {
		testDB.SeedTestData(t)
	}

	t.Cleanup(testDB.cleanup)

	return testDB
}

// setupMockDB creates a mock database for testing without requiring a real database connection
func setupMockDB(t *testing.T, config *DatabaseConfig) *TestDB {
	t.Helper()

	// Create pgxmock instance
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err, "Failed to create mock database pool")

	testDB := &TestDB{
		MockPool: mockPool,
		Queries:  db.New(mockPool),
		IsMock:   true,
		cleanup:  func() { mockPool.Close() },
	}

	t.Cleanup(testDB.cleanup)

	return testDB
}

// Close closes the database connection and performs cleanup
func (tdb *TestDB) Close() {
	if tdb.cleanup != nil {
		tdb.cleanup()
	}
}

// SeedTestData inserts common test data into the database.
// This includes test users, worlds, characters, and other entities needed for testing.
func (tdb *TestDB) SeedTestData(t *testing.T) {
	t.Helper()

	// Skip seeding for mock databases
	if tdb.IsMock {
		return
	}

	ctx := CreateTestContext()

	// Read and execute schema migrations
	schemaPath := filepath.Join(GetProjectRoot(), "db", "migrations", "schema.sql")
	schemaSQL, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read schema file")

	// Split SQL statements and execute them
	statements := strings.Split(string(schemaSQL), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err := tdb.Pool.Exec(ctx, stmt)
		if err != nil {
			// Skip errors for statements that might already exist (like tables)
			t.Logf("Warning: Failed to execute statement (continuing): %v", err)
		}
	}

	// Insert additional test data
	tdb.insertTestUsers(t, ctx)
	tdb.insertTestWorlds(t, ctx)
	tdb.insertTestCharacters(t, ctx)
}

// insertTestUsers creates common test users
func (tdb *TestDB) insertTestUsers(t *testing.T, ctx context.Context) {
	t.Helper()

	testUsers := []struct {
		id           string
		username     string
		displayName  string
		email        string
		passwordHash string
	}{
		{
			id:           "550e8400-e29b-41d4-a716-446655440000",
			username:     "testuser1",
			displayName:  "Test User 1",
			email:        "test1@example.com",
			passwordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // "secret"
		},
		{
			id:           "550e8400-e29b-41d4-a716-446655440001",
			username:     "testuser2",
			displayName:  "Test User 2",
			email:        "test2@example.com",
			passwordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // "secret"
		},
	}

	for _, user := range testUsers {
		_, err := tdb.Pool.Exec(ctx, `
			INSERT INTO users (id, username, display_name, email, password_hash, email_verified) 
			VALUES ($1, $2, $3, $4, $5, true)
			ON CONFLICT (id) DO NOTHING`,
			user.id, user.username, user.displayName, user.email, user.passwordHash)
		require.NoError(t, err, "Failed to insert test user %s", user.username)
	}
}

// insertTestWorlds creates common test worlds
func (tdb *TestDB) insertTestWorlds(t *testing.T, ctx context.Context) {
	t.Helper()

	testWorlds := []struct {
		id   string
		name string
		seed int64
	}{
		{
			id:   "650e8400-e29b-41d4-a716-446655440000",
			name: "Test World 1",
			seed: 123456789,
		},
		{
			id:   "650e8400-e29b-41d4-a716-446655440001",
			name: "Test World 2",
			seed: 987654321,
		},
	}

	for _, world := range testWorlds {
		_, err := tdb.Pool.Exec(ctx, `
			INSERT INTO worlds (id, name, seed) 
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO NOTHING`,
			world.id, world.name, world.seed)
		require.NoError(t, err, "Failed to insert test world %s", world.name)
	}
}

// insertTestCharacters creates common test characters
func (tdb *TestDB) insertTestCharacters(t *testing.T, ctx context.Context) {
	t.Helper()

	testCharacters := []struct {
		id     string
		userID string
		name   string
		x      int32
		y      int32
		chunkX int32
		chunkY int32
	}{
		{
			id:     "750e8400-e29b-41d4-a716-446655440000",
			userID: "550e8400-e29b-41d4-a716-446655440000", // testuser1
			name:   "Character1",
			x:      10,
			y:      20,
			chunkX: 0,
			chunkY: 0,
		},
		{
			id:     "750e8400-e29b-41d4-a716-446655440001",
			userID: "550e8400-e29b-41d4-a716-446655440001", // testuser2
			name:   "Character2",
			x:      50,
			y:      60,
			chunkX: 1,
			chunkY: 1,
		},
	}

	for _, char := range testCharacters {
		_, err := tdb.Pool.Exec(ctx, `
			INSERT INTO characters (id, user_id, name, x, y, chunk_x, chunk_y) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING`,
			char.id, char.userID, char.name, char.x, char.y, char.chunkX, char.chunkY)
		require.NoError(t, err, "Failed to insert test character %s", char.name)
	}
}

// TeardownTestDB cleans up test database by truncating all tables.
// This should be called after tests that modify the database.
func (tdb *TestDB) TeardownTestDB(t *testing.T) {
	t.Helper()

	// Skip teardown for mock databases
	if tdb.IsMock {
		return
	}

	ctx := CreateTestContext()

	// List of tables to truncate in dependency order (children first)
	tables := []string{
		"character_inventories",
		"resource_nodes",
		"chunks",
		"characters",
		"worlds",
		"users",
	}

	for _, table := range tables {
		_, err := tdb.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
		if err != nil {
			t.Logf("Warning: Failed to truncate table %s: %v", table, err)
		}
	}
}

// WithTransaction creates a test that runs within a database transaction.
// The transaction is automatically rolled back at the end of the test.
func (tdb *TestDB) WithTransaction(t *testing.T, testFunc func(t *testing.T, tx pgx.Tx)) {
	t.Helper()

	// Mock databases don't support transactions in the traditional sense
	if tdb.IsMock {
		t.Skip("WithTransaction not supported for mock databases")
		return
	}

	ctx := CreateTestContext()

	tx, err := tdb.Pool.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && err != pgx.ErrTxClosed {
			t.Errorf("Failed to rollback transaction: %v", err)
		}
	}()

	testFunc(t, tx)
}

// CreateTestPool creates a pgx connection pool for testing with the specified configuration
func CreateTestPool(ctx context.Context, config *DatabaseConfig) (*pgxpool.Pool, error) {
	pgConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	pgConfig.MaxConns = config.MaxConnections
	pgConfig.MinConns = 1
	pgConfig.MaxConnLifetime = time.Hour
	pgConfig.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return pool, nil
}

// getTestDatabaseURL returns the database URL for testing.
// It checks environment variables and falls back to a default test database URL.
func getTestDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}

	// Default to a test database URL
	// This assumes a test database is available - for CI/CD this would be configured
	return "postgres://meower:meower@localhost:5432/meower_test?sslmode=disable"
}

// GetMockPool returns a pgxmock.Pool for use in tests that need to mock database interactions.
// This is useful for unit tests that don't require a real database connection.
func GetMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err, "Failed to create mock pool")

	t.Cleanup(func() {
		mockPool.Close()
	})

	return mockPool
}
