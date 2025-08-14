package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions for this test file only to avoid import cycles

func generateTestUUID() string {
	return uuid.New().String()
}

func mustParseUUID(uuidStr string) pgtype.UUID {
	// Validate the UUID format first
	_, err := uuid.Parse(uuidStr)
	if err != nil {
		panic("Invalid UUID: " + uuidStr)
	}

	var pgUUID pgtype.UUID
	// Use the string directly for pgtype.UUID
	err = pgUUID.Scan(uuidStr)
	if err != nil {
		panic("Failed to convert UUID: " + uuidStr + " error: " + err.Error())
	}

	return pgUUID
}

func createTestContext() context.Context {
	return context.Background()
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name        string
		params      CreateUserParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, user User)
	}{
		{
			name: "successful user creation",
			params: CreateUserParams{
				Username:     "newuser",
				DisplayName:  "New User",
				Email:        "newuser@example.com",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				testUUID := generateTestUUID()
				rows := pgxmock.NewRows([]string{
					"id", "username", "display_name", "email", "email_verified",
					"password_hash", "reset_password_token", "reset_password_expires",
					"created_at", "last_login_at", "account_locked", "failed_login_attempts",
				}).AddRow(
					testUUID, "newuser", "New User", "newuser@example.com", pgtype.Bool{Bool: false, Valid: true},
					"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
					pgtype.Text{}, pgtype.Timestamp{},
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{},
					pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("newuser", "New User", "newuser@example.com", "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, user User) {
				assert.Equal(t, "newuser", user.Username)
				assert.Equal(t, "New User", user.DisplayName)
				assert.Equal(t, "newuser@example.com", user.Email)
				assert.False(t, user.EmailVerified.Bool)
				assert.False(t, user.AccountLocked.Bool)
				assert.Equal(t, int32(0), user.FailedLoginAttempts.Int32)
			},
		},
		{
			name: "duplicate username error",
			params: CreateUserParams{
				Username:     "existinguser",
				DisplayName:  "Existing User",
				Email:        "existing@example.com",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("existinguser", "Existing User", "existing@example.com", "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy").
					WillReturnError(sql.ErrConnDone) // Simulate unique constraint violation
			},
			wantErr: true,
		},
		{
			name: "invalid input validation",
			params: CreateUserParams{
				Username:     "", // Empty username
				DisplayName:  "Test User",
				Email:        "invalid-email",
				PasswordHash: "weak",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("", "Test User", "invalid-email", "weak").
					WillReturnError(sql.ErrConnDone) // Simulate validation error
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			// Execute test
			user, err := queries.CreateUser(createTestContext(), tt.params)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, user)
			}

			// Verify all expectations were met
			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetUserByUsername(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, user User)
	}{
		{
			name:     "existing user retrieval",
			username: "testuser",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				testUUID := generateTestUUID()
				rows := pgxmock.NewRows([]string{
					"id", "username", "display_name", "email", "email_verified",
					"password_hash", "reset_password_token", "reset_password_expires",
					"created_at", "last_login_at", "account_locked", "failed_login_attempts",
				}).AddRow(
					testUUID, "testuser", "Test User", "test@example.com", pgtype.Bool{Bool: true, Valid: true},
					"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
					pgtype.Text{}, pgtype.Timestamp{},
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
					pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, user User) {
				assert.Equal(t, "testuser", user.Username)
				assert.Equal(t, "Test User", user.DisplayName)
				assert.Equal(t, "test@example.com", user.Email)
				assert.True(t, user.EmailVerified.Bool)
			},
		},
		{
			name:     "non-existent user handling",
			username: "nonexistent",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM users WHERE username = \\$1").
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			user, err := queries.GetUserByUsername(createTestContext(), tt.username)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, user)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetUserById(t *testing.T) {
	tests := []struct {
		name        string
		userID      pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, user User)
	}{
		{
			name:   "valid ID retrieval",
			userID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "username", "display_name", "email", "email_verified",
					"password_hash", "reset_password_token", "reset_password_expires",
					"created_at", "last_login_at", "account_locked", "failed_login_attempts",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", "testuser", "Test User", "test@example.com", pgtype.Bool{Bool: true, Valid: true},
					"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
					pgtype.Text{}, pgtype.Timestamp{},
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
					pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM users WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, user User) {
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), user.ID)
				assert.Equal(t, "testuser", user.Username)
			},
		},
		{
			name:   "invalid UUID format",
			userID: pgtype.UUID{}, // Invalid UUID
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM users WHERE id = \\$1").
					WithArgs(pgtype.UUID{}).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			user, err := queries.GetUserById(createTestContext(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, user)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestUpdateLastLoginAt(t *testing.T) {
	tests := []struct {
		name        string
		params      UpdateLastLoginAtParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, user User)
	}{
		{
			name: "timestamp update verification",
			params: UpdateLastLoginAtParams{
				ID:          mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				LastLoginAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "username", "display_name", "email", "email_verified",
					"password_hash", "reset_password_token", "reset_password_expires",
					"created_at", "last_login_at", "account_locked", "failed_login_attempts",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", "testuser", "Test User", "test@example.com", pgtype.Bool{Bool: true, Valid: true},
					"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
					pgtype.Text{}, pgtype.Timestamp{},
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
					pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
				)
				mock.ExpectQuery("UPDATE users SET last_login_at = \\$2 WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, user User) {
				assert.True(t, user.LastLoginAt.Valid)
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), user.ID)
			},
		},
		{
			name: "non-existent user handling",
			params: UpdateLastLoginAtParams{
				ID:          mustParseUUID("550e8400-e29b-41d4-a716-446655440999"),
				LastLoginAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE users SET last_login_at = \\$2 WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440999"), pgxmock.AnyArg()).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			user, err := queries.UpdateLastLoginAt(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, user)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestUpdateLoginAttempts(t *testing.T) {
	tests := []struct {
		name        string
		params      UpdateLoginAttemptsParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, user User)
	}{
		{
			name: "increment failed attempts",
			params: UpdateLoginAttemptsParams{
				ID:                  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				FailedLoginAttempts: pgtype.Int4{Int32: 3, Valid: true},
				AccountLocked:       pgtype.Bool{Bool: false, Valid: true},
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "username", "display_name", "email", "email_verified",
					"password_hash", "reset_password_token", "reset_password_expires",
					"created_at", "last_login_at", "account_locked", "failed_login_attempts",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", "testuser", "Test User", "test@example.com", pgtype.Bool{Bool: true, Valid: true},
					"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
					pgtype.Text{}, pgtype.Timestamp{},
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
					pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 3, Valid: true},
				)
				mock.ExpectQuery("UPDATE users SET failed_login_attempts = \\$2, account_locked = \\$3 WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), pgtype.Int4{Int32: 3, Valid: true}, pgtype.Bool{Bool: false, Valid: true}).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, user User) {
				assert.Equal(t, int32(3), user.FailedLoginAttempts.Int32)
				assert.False(t, user.AccountLocked.Bool)
			},
		},
		{
			name: "lock account after max attempts",
			params: UpdateLoginAttemptsParams{
				ID:                  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				FailedLoginAttempts: pgtype.Int4{Int32: 5, Valid: true},
				AccountLocked:       pgtype.Bool{Bool: true, Valid: true},
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "username", "display_name", "email", "email_verified",
					"password_hash", "reset_password_token", "reset_password_expires",
					"created_at", "last_login_at", "account_locked", "failed_login_attempts",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", "testuser", "Test User", "test@example.com", pgtype.Bool{Bool: true, Valid: true},
					"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
					pgtype.Text{}, pgtype.Timestamp{},
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
					pgtype.Bool{Bool: true, Valid: true}, pgtype.Int4{Int32: 5, Valid: true},
				)
				mock.ExpectQuery("UPDATE users SET failed_login_attempts = \\$2, account_locked = \\$3 WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), pgtype.Int4{Int32: 5, Valid: true}, pgtype.Bool{Bool: true, Valid: true}).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, user User) {
				assert.Equal(t, int32(5), user.FailedLoginAttempts.Int32)
				assert.True(t, user.AccountLocked.Bool)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			user, err := queries.UpdateLoginAttempts(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, user)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestIndexUsers(t *testing.T) {
	tests := []struct {
		name        string
		params      IndexUsersParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, users []User)
	}{
		{
			name: "successful pagination",
			params: IndexUsersParams{
				Limit:  10,
				Offset: 0,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "username", "display_name", "email", "email_verified",
					"password_hash", "reset_password_token", "reset_password_expires",
					"created_at", "last_login_at", "account_locked", "failed_login_attempts",
				}).
					AddRow(
						generateTestUUID(), "user1", "User 1", "user1@example.com", pgtype.Bool{Bool: true, Valid: true},
						"hash1", pgtype.Text{}, pgtype.Timestamp{},
						pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
						pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
					).
					AddRow(
						generateTestUUID(), "user2", "User 2", "user2@example.com", pgtype.Bool{Bool: false, Valid: true},
						"hash2", pgtype.Text{}, pgtype.Timestamp{},
						pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{},
						pgtype.Bool{Bool: false, Valid: true}, pgtype.Int4{Int32: 0, Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM users ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
					WithArgs(int32(10), int32(0)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, users []User) {
				assert.Len(t, users, 2)
				assert.Equal(t, "user1", users[0].Username)
				assert.Equal(t, "user2", users[1].Username)
			},
		},
		{
			name: "empty result",
			params: IndexUsersParams{
				Limit:  10,
				Offset: 100,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "username", "display_name", "email", "email_verified",
					"password_hash", "reset_password_token", "reset_password_expires",
					"created_at", "last_login_at", "account_locked", "failed_login_attempts",
				})
				mock.ExpectQuery("SELECT (.+) FROM users ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
					WithArgs(int32(10), int32(100)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, users []User) {
				assert.Empty(t, users)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			users, err := queries.IndexUsers(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, users)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name      string
		userID    pgtype.UUID
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name:   "successful deletion",
			userID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000")).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:   "non-existent user",
			userID: mustParseUUID("550e8400-e29b-41d4-a716-446655440999"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440999")).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: false, // DELETE doesn't fail on non-existent records
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			err = queries.DeleteUser(createTestContext(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkCreateUser(b *testing.B) {
	// This would require a real database connection for meaningful benchmarks
	// For now, we'll skip this test in the mock environment
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkGetUserByUsername(b *testing.B) {
	// This would require a real database connection for meaningful benchmarks
	// For now, we'll skip this test in the mock environment
	b.Skip("Benchmark tests require real database connection")
}
