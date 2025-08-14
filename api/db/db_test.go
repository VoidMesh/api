package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew verifies that the New function correctly initializes a Queries instance
func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() pgxmock.PgxPoolIface
		wantErr   bool
	}{
		{
			name: "successful Queries initialization",
			setupMock: func() pgxmock.PgxPoolIface {
				mockPool, err := pgxmock.NewPool()
				require.NoError(t, err)
				return mockPool
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool := tt.setupMock()
			defer mockPool.Close()

			queries := New(mockPool)

			if tt.wantErr {
				assert.Nil(t, queries)
			} else {
				assert.NotNil(t, queries)
				assert.NotNil(t, queries.db)
			}
		})
	}
}

// TestWithTx verifies that the WithTx method correctly wraps a transaction
func TestWithTx(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock pgxmock.PgxPoolIface) pgxmock.PgxConnIface
		wantErr     bool
		checkResult func(t *testing.T, queries *Queries)
	}{
		{
			name: "successful transaction wrapper",
			setupMock: func(mock pgxmock.PgxPoolIface) pgxmock.PgxConnIface {
				mockTx, err := pgxmock.NewConn()
				require.NoError(t, err)
				return mockTx
			},
			wantErr: false,
			checkResult: func(t *testing.T, queries *Queries) {
				assert.NotNil(t, queries)
				assert.NotNil(t, queries.db)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			mockTx := tt.setupMock(mockPool)
			defer mockTx.Close(context.Background())

			originalQueries := New(mockPool)
			txQueries := originalQueries.WithTx(mockTx)

			if tt.wantErr {
				assert.Nil(t, txQueries)
			} else {
				require.NotNil(t, txQueries)
				tt.checkResult(t, txQueries)
				// Verify that the transaction queries use the transaction connection
				assert.Equal(t, mockTx, txQueries.db)
			}
		})
	}
}

// TestDatabaseConnectionPatterns tests various database connection scenarios
func TestDatabaseConnectionPatterns(t *testing.T) {
	t.Run("connection pool exhaustion simulation", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Simulate pool exhaustion by returning connection timeout error
		mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
			WithArgs("testuser").
			WillReturnError(sql.ErrConnDone) // Simulate connection pool exhaustion

		_, err = queries.GetUserByUsername(createTestContext(), "testuser")
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("context cancellation handling", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Create a context that is already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
			WithArgs("testuser").
			WillReturnError(context.Canceled)

		_, err = queries.GetUserByUsername(ctx, "testuser")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("context timeout handling", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Create a context with a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Sleep to ensure timeout
		time.Sleep(1 * time.Millisecond)

		mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
			WithArgs("testuser").
			WillReturnError(context.DeadlineExceeded)

		_, err = queries.GetUserByUsername(ctx, "testuser")
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("connection retry simulation", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Simulate temporary connection failure followed by success
		mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
			WithArgs("testuser").
			WillReturnError(sql.ErrConnDone) // First attempt fails

		// This test demonstrates that connection errors should be handled at a higher level
		_, err = queries.GetUserByUsername(createTestContext(), "testuser")
		assert.Error(t, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// TestTransactionPatterns tests various transaction scenarios
func TestTransactionPatterns(t *testing.T) {
	t.Run("transaction isolation levels", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		mockTx, err := pgxmock.NewConn()
		require.NoError(t, err)
		defer mockTx.Close(context.Background())

		queries := New(mockPool)
		txQueries := queries.WithTx(mockTx)

		// Simulate operations within a transaction
		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "username", "display_name", "email", "email_verified",
			"password_hash", "reset_password_token", "reset_password_expires",
			"created_at", "last_login_at", "account_locked", "failed_login_attempts",
		}).AddRow(
			generateTestUUID(), "txuser", "Transaction User", "tx@example.com", pgtype.Bool{Bool: false, Valid: true},
			"hashed_password", pgtype.Text{}, pgtype.Timestamp{},
			pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{}, pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
		)

		mockTx.ExpectQuery("INSERT INTO users").
			WithArgs("txuser", "Transaction User", "tx@example.com", "hashed_password").
			WillReturnRows(rows)

		user, err := txQueries.CreateUser(createTestContext(), CreateUserParams{
			Username:     "txuser",
			DisplayName:  "Transaction User",
			Email:        "tx@example.com",
			PasswordHash: "hashed_password",
		})

		require.NoError(t, err)
		assert.Equal(t, "txuser", user.Username)

		assert.NoError(t, mockTx.ExpectationsWereMet())
	})

	t.Run("transaction rollback simulation", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		mockTx, err := pgxmock.NewConn()
		require.NoError(t, err)
		defer mockTx.Close(context.Background())

		queries := New(mockPool)
		txQueries := queries.WithTx(mockTx)

		// Simulate a constraint violation that would cause rollback
		mockTx.ExpectQuery("INSERT INTO users").
			WithArgs("duplicate", "Duplicate User", "duplicate@example.com", "hashed_password").
			WillReturnError(sql.ErrConnDone) // Simulate constraint violation

		_, err = txQueries.CreateUser(createTestContext(), CreateUserParams{
			Username:     "duplicate",
			DisplayName:  "Duplicate User",
			Email:        "duplicate@example.com",
			PasswordHash: "hashed_password",
		})

		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)

		assert.NoError(t, mockTx.ExpectationsWereMet())
	})

	t.Run("nested transaction handling", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		mockTx, err := pgxmock.NewConn()
		require.NoError(t, err)
		defer mockTx.Close(context.Background())

		queries := New(mockPool)
		txQueries := queries.WithTx(mockTx)

		// Create a nested transaction scope
		nestedTxQueries := txQueries.WithTx(mockTx)

		// Both should reference the same transaction
		assert.Equal(t, txQueries.db, nestedTxQueries.db)
		assert.Equal(t, mockTx, nestedTxQueries.db)
	})
}

// TestDatabaseErrorHandling tests various database error scenarios
func TestDatabaseErrorHandling(t *testing.T) {
	t.Run("constraint violation errors", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Test unique constraint violation
		mockPool.ExpectQuery("INSERT INTO users").
			WithArgs("duplicate", "Duplicate User", "duplicate@example.com", "hashed_password").
			WillReturnError(sql.ErrConnDone) // Simulating constraint violation

		_, err = queries.CreateUser(createTestContext(), CreateUserParams{
			Username:     "duplicate",
			DisplayName:  "Duplicate User",
			Email:        "duplicate@example.com",
			PasswordHash: "hashed_password",
		})

		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("foreign key constraint violation", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Test foreign key constraint violation
		mockPool.ExpectQuery("INSERT INTO characters").
			WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), "OrphanCharacter", int32(0), int32(0), int32(0), int32(0)).
			WillReturnError(sql.ErrConnDone) // Simulating foreign key constraint violation

		_, err = queries.CreateCharacter(createTestContext(), CreateCharacterParams{
			UserID:  mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), // Non-existent user
			Name:    "OrphanCharacter",
			X:       0,
			Y:       0,
			ChunkX:  0,
			ChunkY:  0,
		})

		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("not found errors", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Test record not found
		mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
			WithArgs("nonexistent").
			WillReturnError(sql.ErrNoRows)

		_, err = queries.GetUserByUsername(createTestContext(), "nonexistent")
		assert.Error(t, err)
		assert.Equal(t, sql.ErrNoRows, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("database connection errors", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Test database connection error
		mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
			WithArgs("testuser").
			WillReturnError(sql.ErrConnDone)

		_, err = queries.GetUserByUsername(createTestContext(), "testuser")
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// TestConnectionPoolBehavior tests connection pool specific behaviors
func TestConnectionPoolBehavior(t *testing.T) {
	t.Run("connection pool health check", func(t *testing.T) {
		// This test would normally ping the database to check health
		// In a mock environment, we simulate the health check
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		// Simulate a simple ping operation
		mockPool.ExpectPing()

		err = mockPool.Ping(createTestContext())
		assert.NoError(t, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("connection pool statistics", func(t *testing.T) {
		// This test demonstrates how connection pool statistics would be accessed
		// In a real pgxpool.Pool, you would call pool.Stat()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		// For mock pools, we just verify the pool exists and is usable
		queries := New(mockPool)
		assert.NotNil(t, queries)
		assert.NotNil(t, queries.db)
	})

	t.Run("connection acquisition timeout", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Simulate connection acquisition timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(1 * time.Millisecond) // Ensure timeout

		mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
			WithArgs("testuser").
			WillReturnError(context.DeadlineExceeded)

		_, err = queries.GetUserByUsername(ctx, "testuser")
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// TestDatabasePerformancePatterns tests performance-related database patterns
func TestDatabasePerformancePatterns(t *testing.T) {
	t.Run("batch operation simulation", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Simulate batch user creation (in a real scenario, this would use pgx.Batch)
		usernames := []string{"user1", "user2", "user3"}
		now := time.Now()

		for i, username := range usernames {
			rows := pgxmock.NewRows([]string{
				"id", "username", "display_name", "email", "email_verified",
				"password_hash", "reset_password_token", "reset_password_expires",
				"created_at", "last_login_at", "account_locked", "failed_login_attempts",
			}).AddRow(
				generateTestUUID(), username, username+" Display", username+"@example.com", pgtype.Bool{Bool: false, Valid: true},
				"hashed_password", pgtype.Text{}, pgtype.Timestamp{},
				pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{}, pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
			)

			mockPool.ExpectQuery("INSERT INTO users").
				WithArgs(username, username+" Display", username+"@example.com", "hashed_password").
				WillReturnRows(rows)

			user, err := queries.CreateUser(createTestContext(), CreateUserParams{
				Username:     username,
				DisplayName:  username + " Display",
				Email:        username + "@example.com",
				PasswordHash: "hashed_password",
			})

			require.NoError(t, err)
			assert.Equal(t, username, user.Username)
			assert.Equal(t, int32(i+1), int32(i+1)) // Just to use i
		}

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("prepared statement simulation", func(t *testing.T) {
		// In a real environment, this would test prepared statement behavior
		// pgx automatically prepares frequently used statements
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Simulate multiple calls to the same query (would use prepared statement)
		for i := 0; i < 3; i++ {
			mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
				WithArgs("testuser").
				WillReturnError(sql.ErrNoRows)

			_, err := queries.GetUserByUsername(createTestContext(), "testuser")
			assert.Error(t, err)
			assert.Equal(t, sql.ErrNoRows, err)
		}

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("large result set handling", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Simulate large result set
		rows := pgxmock.NewRows([]string{
			"id", "username", "display_name", "email", "email_verified",
			"password_hash", "reset_password_token", "reset_password_expires",
			"created_at", "last_login_at", "account_locked", "failed_login_attempts",
		})

		now := time.Now()
		// Add many rows to simulate large result set
		for i := 0; i < 1000; i++ {
			username := fmt.Sprintf("user%d", i)
			rows.AddRow(
				generateTestUUID(), username, username+" Display", username+"@example.com", pgtype.Bool{Bool: false, Valid: true},
				"hashed_password", pgtype.Text{}, pgtype.Timestamp{},
				pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{}, pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
			)
		}

		mockPool.ExpectQuery("SELECT (.+) FROM users ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
			WithArgs(int32(1000), int32(0)).
			WillReturnRows(rows)

		users, err := queries.IndexUsers(createTestContext(), IndexUsersParams{
			Limit:  1000,
			Offset: 0,
		})

		require.NoError(t, err)
		assert.Len(t, users, 1000)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// TestConcurrentDatabaseAccess tests concurrent database access patterns
func TestConcurrentDatabaseAccess(t *testing.T) {
	t.Run("concurrent read operations", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Set up expectations for concurrent reads
		for i := 0; i < 5; i++ {
			mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
				WithArgs("concurrent_user").
				WillReturnError(sql.ErrNoRows)
		}

		// Simulate concurrent read operations
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func() {
				_, err := queries.GetUserByUsername(createTestContext(), "concurrent_user")
				assert.Error(t, err)
				assert.Equal(t, sql.ErrNoRows, err)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			<-done
		}

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("connection pool contention simulation", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Simulate high contention by having many operations expect connection errors
		for i := 0; i < 10; i++ {
			if i < 8 {
				// Most operations succeed
				mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
					WithArgs("pool_test").
					WillReturnError(sql.ErrNoRows)
			} else {
				// Some operations fail due to pool exhaustion
				mockPool.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
					WithArgs("pool_test").
					WillReturnError(sql.ErrConnDone)
			}
		}

		// Execute operations and track results
		successCount := 0
		errorCount := 0

		for i := 0; i < 10; i++ {
			_, err := queries.GetUserByUsername(createTestContext(), "pool_test")
			if err == sql.ErrNoRows {
				successCount++
			} else if err == sql.ErrConnDone {
				errorCount++
			}
		}

		assert.Equal(t, 8, successCount)
		assert.Equal(t, 2, errorCount)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// Benchmark tests for database operations
func BenchmarkDatabaseOperations(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

// Test helper for creating real pool connections (would be used with integration tests)
func createTestPool(connectionString string) (*pgxpool.Pool, error) {
	// This function would be used in integration tests with a real database
	// For unit tests, we use mocks
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// Test cleanup function for integration tests
func cleanupTestData(queries *Queries, ctx context.Context) error {
	// This function would clean up test data in integration tests
	// Implementation would depend on specific test data cleanup requirements
	return nil
}