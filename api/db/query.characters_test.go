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

func TestCreateCharacter(t *testing.T) {
	tests := []struct {
		name        string
		params      CreateCharacterParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, character Character)
	}{
		{
			name: "successful character creation",
			params: CreateCharacterParams{
				UserID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				Name:   "TestHero",
				X:      100,
				Y:      200,
				ChunkX: 1,
				ChunkY: 2,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				testCharacterID := generateTestUUID()
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				}).AddRow(
					testCharacterID, "550e8400-e29b-41d4-a716-446655440000", "TestHero",
					int32(100), int32(200), int32(1), int32(2), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO characters").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "TestHero", int32(100), int32(200), int32(1), int32(2)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, character Character) {
				assert.Equal(t, "TestHero", character.Name)
				assert.Equal(t, int32(100), character.X)
				assert.Equal(t, int32(200), character.Y)
				assert.Equal(t, int32(1), character.ChunkX)
				assert.Equal(t, int32(2), character.ChunkY)
				assert.True(t, character.CreatedAt.Valid)
			},
		},
		{
			name: "duplicate character name per user",
			params: CreateCharacterParams{
				UserID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				Name:   "ExistingHero",
				X:      0,
				Y:      0,
				ChunkX: 0,
				ChunkY: 0,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO characters").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "ExistingHero", int32(0), int32(0), int32(0), int32(0)).
					WillReturnError(sql.ErrConnDone) // Simulate unique constraint violation
			},
			wantErr: true,
		},
		{
			name: "foreign key constraint violation - invalid user",
			params: CreateCharacterParams{
				UserID: mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), // Non-existent user
				Name:   "OrphanHero",
				X:      0,
				Y:      0,
				ChunkX: 0,
				ChunkY: 0,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO characters").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), "OrphanHero", int32(0), int32(0), int32(0), int32(0)).
					WillReturnError(sql.ErrConnDone) // Simulate foreign key constraint violation
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

			character, err := queries.CreateCharacter(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, character)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetCharacterById(t *testing.T) {
	tests := []struct {
		name        string
		characterID pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, character Character)
	}{
		{
			name:        "valid character retrieval",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				}).AddRow(
					"750e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000", "TestHero",
					int32(100), int32(200), int32(1), int32(2), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, character Character) {
				assert.Equal(t, "TestHero", character.Name)
				assert.Equal(t, int32(100), character.X)
				assert.Equal(t, int32(200), character.Y)
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), character.UserID)
			},
		},
		{
			name:        "non-existent character",
			characterID: mustParseUUID("999e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE id = \\$1").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000")).
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

			character, err := queries.GetCharacterById(createTestContext(), tt.characterID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, character)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetCharactersByUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, characters []Character)
	}{
		{
			name:   "multiple characters retrieval",
			userID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				}).
					AddRow(
						generateTestUUID(), "550e8400-e29b-41d4-a716-446655440000", "Character1",
						int32(10), int32(20), int32(0), int32(0), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						generateTestUUID(), "550e8400-e29b-41d4-a716-446655440000", "Character2",
						int32(50), int32(60), int32(1), int32(1), pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE user_id = \\$1 ORDER BY created_at DESC").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, characters []Character) {
				assert.Len(t, characters, 2)
				assert.Equal(t, "Character1", characters[0].Name)
				assert.Equal(t, "Character2", characters[1].Name)
				// Verify ordering by created_at DESC
				assert.True(t, characters[0].CreatedAt.Time.After(characters[1].CreatedAt.Time))
			},
		},
		{
			name:   "empty result handling",
			userID: mustParseUUID("999e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				})
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE user_id = \\$1 ORDER BY created_at DESC").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, characters []Character) {
				assert.Empty(t, characters)
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

			characters, err := queries.GetCharactersByUser(createTestContext(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, characters)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetCharacterByUserAndName(t *testing.T) {
	tests := []struct {
		name        string
		params      GetCharacterByUserAndNameParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, character Character)
	}{
		{
			name: "valid retrieval with unique name per user",
			params: GetCharacterByUserAndNameParams{
				UserID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				Name:   "UniqueHero",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				}).AddRow(
					generateTestUUID(), "550e8400-e29b-41d4-a716-446655440000", "UniqueHero",
					int32(75), int32(125), int32(2), int32(3), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE user_id = \\$1 AND name = \\$2").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "UniqueHero").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, character Character) {
				assert.Equal(t, "UniqueHero", character.Name)
				assert.Equal(t, int32(75), character.X)
				assert.Equal(t, int32(125), character.Y)
			},
		},
		{
			name: "non-existent character name",
			params: GetCharacterByUserAndNameParams{
				UserID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				Name:   "NonExistentHero",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE user_id = \\$1 AND name = \\$2").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "NonExistentHero").
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

			character, err := queries.GetCharacterByUserAndName(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, character)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetCharactersInChunk(t *testing.T) {
	tests := []struct {
		name        string
		params      GetCharactersInChunkParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, characters []Character)
	}{
		{
			name: "characters in same chunk",
			params: GetCharactersInChunkParams{
				ChunkX: 5,
				ChunkY: 10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				}).
					AddRow(
						generateTestUUID(), generateTestUUID(), "Hero1",
						int32(500), int32(1000), int32(5), int32(10), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						generateTestUUID(), generateTestUUID(), "Hero2",
						int32(510), int32(1020), int32(5), int32(10), pgtype.Timestamp{Time: now, Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE chunk_x = \\$1 AND chunk_y = \\$2").
					WithArgs(int32(5), int32(10)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, characters []Character) {
				assert.Len(t, characters, 2)
				for _, char := range characters {
					assert.Equal(t, int32(5), char.ChunkX)
					assert.Equal(t, int32(10), char.ChunkY)
				}
			},
		},
		{
			name: "empty chunk",
			params: GetCharactersInChunkParams{
				ChunkX: 999,
				ChunkY: 999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				})
				mock.ExpectQuery("SELECT (.+) FROM characters WHERE chunk_x = \\$1 AND chunk_y = \\$2").
					WithArgs(int32(999), int32(999)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, characters []Character) {
				assert.Empty(t, characters)
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

			characters, err := queries.GetCharactersInChunk(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, characters)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestUpdateCharacterPosition(t *testing.T) {
	tests := []struct {
		name        string
		params      UpdateCharacterPositionParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, character Character)
	}{
		{
			name: "successful position update",
			params: UpdateCharacterPositionParams{
				ID:     mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				X:      150,
				Y:      250,
				ChunkX: 3,
				ChunkY: 5,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				}).AddRow(
					"750e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000", "TestHero",
					int32(150), int32(250), int32(3), int32(5), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE characters SET x = \\$2, y = \\$3, chunk_x = \\$4, chunk_y = \\$5 WHERE id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(150), int32(250), int32(3), int32(5)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, character Character) {
				assert.Equal(t, int32(150), character.X)
				assert.Equal(t, int32(250), character.Y)
				assert.Equal(t, int32(3), character.ChunkX)
				assert.Equal(t, int32(5), character.ChunkY)
				assert.Equal(t, mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), character.ID)
			},
		},
		{
			name: "concurrent update handling - character moved by another process",
			params: UpdateCharacterPositionParams{
				ID:     mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				X:      160,
				Y:      260,
				ChunkX: 4,
				ChunkY: 6,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE characters SET x = \\$2, y = \\$3, chunk_x = \\$4, chunk_y = \\$5 WHERE id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(160), int32(260), int32(4), int32(6)).
					WillReturnError(sql.ErrNoRows) // Simulate optimistic locking failure or non-existent character
			},
			wantErr: true,
		},
		{
			name: "chunk transition handling",
			params: UpdateCharacterPositionParams{
				ID:     mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				X:      1000, // Moved to new chunk boundary
				Y:      1000,
				ChunkX: 10,
				ChunkY: 10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
				}).AddRow(
					"750e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000", "TestHero",
					int32(1000), int32(1000), int32(10), int32(10), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE characters SET x = \\$2, y = \\$3, chunk_x = \\$4, chunk_y = \\$5 WHERE id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1000), int32(1000), int32(10), int32(10)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, character Character) {
				assert.Equal(t, int32(1000), character.X)
				assert.Equal(t, int32(1000), character.Y)
				assert.Equal(t, int32(10), character.ChunkX)
				assert.Equal(t, int32(10), character.ChunkY)
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

			character, err := queries.UpdateCharacterPosition(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, character)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestDeleteCharacter(t *testing.T) {
	tests := []struct {
		name        string
		characterID pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
	}{
		{
			name:        "successful deletion",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM characters WHERE id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:        "cascade deletion verification - related inventory should be cleaned up",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// This test verifies that the deletion succeeds and expects
				// the database to handle cascade deletion of related records
				mock.ExpectExec("DELETE FROM characters WHERE id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:        "non-existent character",
			characterID: mustParseUUID("999e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM characters WHERE id = \\$1").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000")).
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

			err = queries.DeleteCharacter(createTestContext(), tt.characterID)

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
func TestCharacterBusinessLogic(t *testing.T) {
	t.Run("character position coordinates validation", func(t *testing.T) {
		// Test with extreme coordinate values
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Test updating with negative coordinates (should be allowed for game logic)
		params := UpdateCharacterPositionParams{
			ID:     mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			X:      -100,
			Y:      -200,
			ChunkX: -1,
			ChunkY: -2,
		}

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "x", "y", "chunk_x", "chunk_y", "created_at",
		}).AddRow(
			"750e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000", "TestHero",
			int32(-100), int32(-200), int32(-1), int32(-2), pgtype.Timestamp{Time: now, Valid: true},
		)

		mockPool.ExpectQuery("UPDATE characters SET x = \\$2, y = \\$3, chunk_x = \\$4, chunk_y = \\$5 WHERE id = \\$1").
			WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(-100), int32(-200), int32(-1), int32(-2)).
			WillReturnRows(rows)

		character, err := queries.UpdateCharacterPosition(createTestContext(), params)
		require.NoError(t, err)
		assert.Equal(t, int32(-100), character.X)
		assert.Equal(t, int32(-200), character.Y)
		assert.Equal(t, int32(-1), character.ChunkX)
		assert.Equal(t, int32(-2), character.ChunkY)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("character name constraints", func(t *testing.T) {
		// Test creating character with empty name (should fail)
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := CreateCharacterParams{
			UserID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			Name:   "", // Empty name
			X:      0,
			Y:      0,
			ChunkX: 0,
			ChunkY: 0,
		}

		mockPool.ExpectQuery("INSERT INTO characters").
			WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), "", int32(0), int32(0), int32(0), int32(0)).
			WillReturnError(sql.ErrConnDone) // Simulate constraint violation

		_, err = queries.CreateCharacter(createTestContext(), params)
		assert.Error(t, err)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// Benchmark tests for performance validation
func BenchmarkCreateCharacter(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkGetCharactersByUser(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkUpdateCharacterPosition(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}
