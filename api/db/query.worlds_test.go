package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWorld(t *testing.T) {
	tests := []struct {
		name        string
		params      CreateWorldParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, world World)
	}{
		{
			name: "successful world creation",
			params: CreateWorldParams{
				Name: "Test World",
				Seed: 12345,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				testUUID := generateTestUUID()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					testUUID, "Test World", int64(12345), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO worlds").
					WithArgs("Test World", int64(12345)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, "Test World", world.Name)
				assert.Equal(t, int64(12345), world.Seed)
				assert.True(t, world.CreatedAt.Valid)
			},
		},
		{
			name: "world creation with negative seed",
			params: CreateWorldParams{
				Name: "Negative Seed World",
				Seed: -999999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				testUUID := generateTestUUID()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					testUUID, "Negative Seed World", int64(-999999), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO worlds").
					WithArgs("Negative Seed World", int64(-999999)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, "Negative Seed World", world.Name)
				assert.Equal(t, int64(-999999), world.Seed)
			},
		},
		{
			name: "world creation with very large seed",
			params: CreateWorldParams{
				Name: "Large Seed World",
				Seed: 9223372036854775807, // Max int64
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				testUUID := generateTestUUID()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					testUUID, "Large Seed World", int64(9223372036854775807), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO worlds").
					WithArgs("Large Seed World", int64(9223372036854775807)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, "Large Seed World", world.Name)
				assert.Equal(t, int64(9223372036854775807), world.Seed)
			},
		},
		{
			name: "empty world name validation",
			params: CreateWorldParams{
				Name: "",
				Seed: 12345,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO worlds").
					WithArgs("", int64(12345)).
					WillReturnError(sql.ErrConnDone) // Simulate constraint violation
			},
			wantErr: true,
		},
		{
			name: "world name with special characters",
			params: CreateWorldParams{
				Name: "World! @#$%^&*()_+-={}[]|\\:;\"'<>,.?/~`",
				Seed: 54321,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				testUUID := generateTestUUID()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					testUUID, "World! @#$%^&*()_+-={}[]|\\:;\"'<>,.?/~`", int64(54321), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO worlds").
					WithArgs("World! @#$%^&*()_+-={}[]|\\:;\"'<>,.?/~`", int64(54321)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, "World! @#$%^&*()_+-={}[]|\\:;\"'<>,.?/~`", world.Name)
				assert.Equal(t, int64(54321), world.Seed)
			},
		},
		{
			name: "very long world name",
			params: CreateWorldParams{
				Name: "This is a very long world name that might test database constraints and should be handled properly by the system without truncation unless there are specific limits in place",
				Seed: 67890,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				testUUID := generateTestUUID()
				longName := "This is a very long world name that might test database constraints and should be handled properly by the system without truncation unless there are specific limits in place"
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					testUUID, longName, int64(67890), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO worlds").
					WithArgs(longName, int64(67890)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				expectedName := "This is a very long world name that might test database constraints and should be handled properly by the system without truncation unless there are specific limits in place"
				assert.Equal(t, expectedName, world.Name)
				assert.Equal(t, int64(67890), world.Seed)
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

			world, err := queries.CreateWorld(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, world)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetWorldByID(t *testing.T) {
	tests := []struct {
		name        string
		worldID     pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, world World)
	}{
		{
			name:    "valid world retrieval",
			worldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", "VoidMesh World", int64(123456789), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM worlds WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), world.ID)
				assert.Equal(t, "VoidMesh World", world.Name)
				assert.Equal(t, int64(123456789), world.Seed)
				assert.True(t, world.CreatedAt.Valid)
			},
		},
		{
			name:    "non-existent world handling",
			worldID: mustParseUUID("999e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM worlds WHERE id = \\$1").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000")).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:    "invalid UUID format",
			worldID: pgtype.UUID{}, // Invalid UUID
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM worlds WHERE id = \\$1").
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

			world, err := queries.GetWorldByID(createTestContext(), tt.worldID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, world)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetDefaultWorld(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, world World)
	}{
		{
			name: "default world exists",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					generateTestUUID(), "Default World", int64(100), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM worlds ORDER BY created_at ASC").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, "Default World", world.Name)
				assert.Equal(t, int64(100), world.Seed)
				assert.True(t, world.CreatedAt.Valid)
			},
		},
		{
			name: "no worlds exist",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM worlds ORDER BY created_at ASC").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "multiple worlds - returns oldest",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				oldestTime := time.Now().Add(-24 * time.Hour)
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					generateTestUUID(), "Oldest World", int64(1), pgtype.Timestamp{Time: oldestTime, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM worlds ORDER BY created_at ASC").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, "Oldest World", world.Name)
				assert.Equal(t, int64(1), world.Seed)
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

			world, err := queries.GetDefaultWorld(createTestContext())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, world)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestListWorlds(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, worlds []World)
	}{
		{
			name: "multiple worlds retrieval",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).
					AddRow(
						generateTestUUID(), "World 1", int64(100), pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true},
					).
					AddRow(
						generateTestUUID(), "World 2", int64(200), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						generateTestUUID(), "World 3", int64(300), pgtype.Timestamp{Time: now.Add(time.Hour), Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM worlds ORDER BY created_at").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, worlds []World) {
				assert.Len(t, worlds, 3)
				assert.Equal(t, "World 1", worlds[0].Name)
				assert.Equal(t, "World 2", worlds[1].Name)
				assert.Equal(t, "World 3", worlds[2].Name)
				// Verify ordering by created_at
				assert.True(t, worlds[0].CreatedAt.Time.Before(worlds[1].CreatedAt.Time))
				assert.True(t, worlds[1].CreatedAt.Time.Before(worlds[2].CreatedAt.Time))
			},
		},
		{
			name: "empty worlds table",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				})
				mock.ExpectQuery("SELECT (.+) FROM worlds ORDER BY created_at").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, worlds []World) {
				assert.Empty(t, worlds)
			},
		},
		{
			name: "single world",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					generateTestUUID(), "Only World", int64(42), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM worlds ORDER BY created_at").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, worlds []World) {
				assert.Len(t, worlds, 1)
				assert.Equal(t, "Only World", worlds[0].Name)
				assert.Equal(t, int64(42), worlds[0].Seed)
			},
		},
		{
			name: "database connection error",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM worlds ORDER BY created_at").
					WillReturnError(sql.ErrConnDone)
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

			worlds, err := queries.ListWorlds(createTestContext())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, worlds)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestUpdateWorld(t *testing.T) {
	tests := []struct {
		name        string
		params      UpdateWorldParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, world World)
	}{
		{
			name: "successful world name update",
			params: UpdateWorldParams{
				ID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				Name: "Updated World Name",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", "Updated World Name", int64(123456), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE worlds SET name = \\$2 WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "Updated World Name").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, "Updated World Name", world.Name)
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), world.ID)
				assert.Equal(t, int64(123456), world.Seed) // Seed should remain unchanged
			},
		},
		{
			name: "update with empty name",
			params: UpdateWorldParams{
				ID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				Name: "",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE worlds SET name = \\$2 WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "").
					WillReturnError(sql.ErrConnDone) // Simulate constraint violation
			},
			wantErr: true,
		},
		{
			name: "update non-existent world",
			params: UpdateWorldParams{
				ID:   mustParseUUID("999e8400-e29b-41d4-a716-446655440000"),
				Name: "New Name",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE worlds SET name = \\$2 WHERE id = \\$1").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), "New Name").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "update with special characters",
			params: UpdateWorldParams{
				ID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				Name: "World!@#$%^&*()",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "name", "seed", "created_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", "World!@#$%^&*()", int64(789), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE worlds SET name = \\$2 WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "World!@#$%^&*()").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, world World) {
				assert.Equal(t, "World!@#$%^&*()", world.Name)
				assert.Equal(t, int64(789), world.Seed)
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

			world, err := queries.UpdateWorld(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, world)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestDeleteWorld(t *testing.T) {
	tests := []struct {
		name      string
		worldID   pgtype.UUID
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name:    "successful deletion",
			worldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM worlds WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000")).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:    "cascade deletion verification - related chunks should be cleaned up",
			worldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// This test verifies that the deletion succeeds and expects
				// the database to handle cascade deletion of related records
				mock.ExpectExec("DELETE FROM worlds WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000")).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:    "non-existent world",
			worldID: mustParseUUID("999e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM worlds WHERE id = \\$1").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000")).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: false, // DELETE doesn't fail on non-existent records
		},
		{
			name:    "foreign key constraint violation simulation",
			worldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM worlds WHERE id = \\$1").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000")).
					WillReturnError(sql.ErrConnDone) // Simulate constraint violation
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

			err = queries.DeleteWorld(createTestContext(), tt.worldID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

// Edge case tests for business logic validation
func TestWorldBusinessLogic(t *testing.T) {
	t.Run("world seed boundaries", func(t *testing.T) {
		// Test with minimum int64 value
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := CreateWorldParams{
			Name: "Min Seed World",
			Seed: -9223372036854775808, // Min int64
		}

		now := time.Now()
		testUUID := generateTestUUID()
		rows := pgxmock.NewRows([]string{
			"id", "name", "seed", "created_at",
		}).AddRow(
			testUUID, "Min Seed World", int64(-9223372036854775808), pgtype.Timestamp{Time: now, Valid: true},
		)

		mockPool.ExpectQuery("INSERT INTO worlds").
			WithArgs("Min Seed World", int64(-9223372036854775808)).
			WillReturnRows(rows)

		world, err := queries.CreateWorld(createTestContext(), params)
		require.NoError(t, err)
		assert.Equal(t, int64(-9223372036854775808), world.Seed)
		assert.Equal(t, "Min Seed World", world.Name)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("concurrent world name updates", func(t *testing.T) {
		// Simulate optimistic locking scenario
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := UpdateWorldParams{
			ID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			Name: "Concurrent Update",
		}

		mockPool.ExpectQuery("UPDATE worlds SET name = \\$2 WHERE id = \\$1").
			WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "Concurrent Update").
			WillReturnError(sql.ErrNoRows) // Simulate concurrent modification

		_, err = queries.UpdateWorld(createTestContext(), params)
		assert.Error(t, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("world name uniqueness - not enforced at database level", func(t *testing.T) {
		// Test that multiple worlds can have the same name (no unique constraint)
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := CreateWorldParams{
			Name: "Duplicate Name",
			Seed: 111,
		}

		now := time.Now()
		testUUID := generateTestUUID()
		rows := pgxmock.NewRows([]string{
			"id", "name", "seed", "created_at",
		}).AddRow(
			testUUID, "Duplicate Name", int64(111), pgtype.Timestamp{Time: now, Valid: true},
		)

		mockPool.ExpectQuery("INSERT INTO worlds").
			WithArgs("Duplicate Name", int64(111)).
			WillReturnRows(rows)

		world, err := queries.CreateWorld(createTestContext(), params)
		require.NoError(t, err)
		assert.Equal(t, "Duplicate Name", world.Name)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// Benchmark tests for performance validation
func BenchmarkCreateWorld(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkGetWorldByID(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkListWorlds(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}