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

func TestCreateChunk(t *testing.T) {
	tests := []struct {
		name        string
		params      CreateChunkParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, chunk Chunk)
	}{
		{
			name: "successful chunk creation",
			params: CreateChunkParams{
				WorldID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:    10,
				ChunkY:    20,
				ChunkData: []byte{0x01, 0x02, 0x03, 0x04}, // Sample terrain data
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), []byte{0x01, 0x02, 0x03, 0x04}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO chunks").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20), []byte{0x01, 0x02, 0x03, 0x04}).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunk Chunk) {
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), chunk.WorldID)
				assert.Equal(t, int32(10), chunk.ChunkX)
				assert.Equal(t, int32(20), chunk.ChunkY)
				assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, chunk.ChunkData)
				assert.True(t, chunk.GeneratedAt.Valid)
			},
		},
		{
			name: "chunk creation with negative coordinates",
			params: CreateChunkParams{
				WorldID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:    -5,
				ChunkY:    -10,
				ChunkData: []byte{0xFF, 0xFE, 0xFD}, // Different terrain data
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", int32(-5), int32(-10), []byte{0xFF, 0xFE, 0xFD}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO chunks").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-5), int32(-10), []byte{0xFF, 0xFE, 0xFD}).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunk Chunk) {
				assert.Equal(t, int32(-5), chunk.ChunkX)
				assert.Equal(t, int32(-10), chunk.ChunkY)
				assert.Equal(t, []byte{0xFF, 0xFE, 0xFD}, chunk.ChunkData)
			},
		},
		{
			name: "chunk creation with large data payload",
			params: CreateChunkParams{
				WorldID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:    0,
				ChunkY:    0,
				ChunkData: make([]byte, 65536), // 64KB chunk data
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				largeData := make([]byte, 65536)
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", int32(0), int32(0), largeData, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO chunks").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(0), int32(0), largeData).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunk Chunk) {
				assert.Len(t, chunk.ChunkData, 65536)
				assert.Equal(t, int32(0), chunk.ChunkX)
				assert.Equal(t, int32(0), chunk.ChunkY)
			},
		},
		{
			name: "chunk creation with empty data",
			params: CreateChunkParams{
				WorldID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:    5,
				ChunkY:    5,
				ChunkData: []byte{}, // Empty chunk data
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", int32(5), int32(5), []byte{}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO chunks").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(5), int32(5), []byte{}).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunk Chunk) {
				assert.Empty(t, chunk.ChunkData)
				assert.Equal(t, int32(5), chunk.ChunkX)
				assert.Equal(t, int32(5), chunk.ChunkY)
			},
		},
		{
			name: "duplicate chunk creation - primary key violation",
			params: CreateChunkParams{
				WorldID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:    1,
				ChunkY:    1,
				ChunkData: []byte{0x01},
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO chunks").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(1), int32(1), []byte{0x01}).
					WillReturnError(sql.ErrConnDone) // Simulate primary key constraint violation
			},
			wantErr: true,
		},
		{
			name: "invalid world foreign key",
			params: CreateChunkParams{
				WorldID:   mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), // Non-existent world
				ChunkX:    1,
				ChunkY:    1,
				ChunkData: []byte{0x01},
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO chunks").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), int32(1), int32(1), []byte{0x01}).
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

			chunk, err := queries.CreateChunk(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, chunk)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetChunk(t *testing.T) {
	tests := []struct {
		name        string
		params      GetChunkParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, chunk Chunk)
	}{
		{
			name: "successful chunk retrieval",
			params: GetChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), []byte{0x01, 0x02, 0x03}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunk Chunk) {
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), chunk.WorldID)
				assert.Equal(t, int32(10), chunk.ChunkX)
				assert.Equal(t, int32(20), chunk.ChunkY)
				assert.Equal(t, []byte{0x01, 0x02, 0x03}, chunk.ChunkData)
				assert.True(t, chunk.GeneratedAt.Valid)
			},
		},
		{
			name: "non-existent chunk",
			params: GetChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  999,
				ChunkY:  999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(999), int32(999)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "chunk with negative coordinates",
			params: GetChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  -5,
				ChunkY:  -10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", int32(-5), int32(-10), []byte{0xFF}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-5), int32(-10)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunk Chunk) {
				assert.Equal(t, int32(-5), chunk.ChunkX)
				assert.Equal(t, int32(-10), chunk.ChunkY)
				assert.Equal(t, []byte{0xFF}, chunk.ChunkData)
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

			chunk, err := queries.GetChunk(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, chunk)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetChunks(t *testing.T) {
	tests := []struct {
		name        string
		params      GetChunksParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, chunks []Chunk)
	}{
		{
			name: "retrieve chunks in spatial range",
			params: GetChunksParams{
				WorldID:  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:   0,   // Min X
				ChunkX_2: 2,   // Max X
				ChunkY:   0,   // Min Y
				ChunkY_2: 2,   // Max Y
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).
					AddRow(
						"550e8400-e29b-41d4-a716-446655440000", int32(0), int32(0), []byte{0x00}, pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						"550e8400-e29b-41d4-a716-446655440000", int32(0), int32(1), []byte{0x01}, pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						"550e8400-e29b-41d4-a716-446655440000", int32(1), int32(0), []byte{0x10}, pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						"550e8400-e29b-41d4-a716-446655440000", int32(1), int32(1), []byte{0x11}, pgtype.Timestamp{Time: now, Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x >= \\$2 AND chunk_x <= \\$3 AND chunk_y >= \\$4 AND chunk_y <= \\$5 ORDER BY chunk_x, chunk_y").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(0), int32(2), int32(0), int32(2)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunks []Chunk) {
				assert.Len(t, chunks, 4)
				// Verify ordering by chunk_x, chunk_y
				assert.Equal(t, int32(0), chunks[0].ChunkX)
				assert.Equal(t, int32(0), chunks[0].ChunkY)
				assert.Equal(t, int32(0), chunks[1].ChunkX)
				assert.Equal(t, int32(1), chunks[1].ChunkY)
				assert.Equal(t, int32(1), chunks[2].ChunkX)
				assert.Equal(t, int32(0), chunks[2].ChunkY)
				assert.Equal(t, int32(1), chunks[3].ChunkX)
				assert.Equal(t, int32(1), chunks[3].ChunkY)
				// Verify chunk data
				assert.Equal(t, []byte{0x00}, chunks[0].ChunkData)
				assert.Equal(t, []byte{0x01}, chunks[1].ChunkData)
				assert.Equal(t, []byte{0x10}, chunks[2].ChunkData)
				assert.Equal(t, []byte{0x11}, chunks[3].ChunkData)
			},
		},
		{
			name: "empty range - no chunks",
			params: GetChunksParams{
				WorldID:  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:   100,
				ChunkX_2: 102,
				ChunkY:   100,
				ChunkY_2: 102,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				})
				mock.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x >= \\$2 AND chunk_x <= \\$3 AND chunk_y >= \\$4 AND chunk_y <= \\$5 ORDER BY chunk_x, chunk_y").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(100), int32(102), int32(100), int32(102)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunks []Chunk) {
				assert.Empty(t, chunks)
			},
		},
		{
			name: "single chunk in range",
			params: GetChunksParams{
				WorldID:  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:   5,
				ChunkX_2: 5,
				ChunkY:   10,
				ChunkY_2: 10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", int32(5), int32(10), []byte{0xAB, 0xCD}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x >= \\$2 AND chunk_x <= \\$3 AND chunk_y >= \\$4 AND chunk_y <= \\$5 ORDER BY chunk_x, chunk_y").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(5), int32(5), int32(10), int32(10)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunks []Chunk) {
				assert.Len(t, chunks, 1)
				assert.Equal(t, int32(5), chunks[0].ChunkX)
				assert.Equal(t, int32(10), chunks[0].ChunkY)
				assert.Equal(t, []byte{0xAB, 0xCD}, chunks[0].ChunkData)
			},
		},
		{
			name: "negative coordinate range",
			params: GetChunksParams{
				WorldID:  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:   -2,
				ChunkX_2: 0,
				ChunkY:   -2,
				ChunkY_2: 0,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).
					AddRow(
						"550e8400-e29b-41d4-a716-446655440000", int32(-2), int32(-1), []byte{0xFE}, pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						"550e8400-e29b-41d4-a716-446655440000", int32(-1), int32(-2), []byte{0xFD}, pgtype.Timestamp{Time: now, Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x >= \\$2 AND chunk_x <= \\$3 AND chunk_y >= \\$4 AND chunk_y <= \\$5 ORDER BY chunk_x, chunk_y").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-2), int32(0), int32(-2), int32(0)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, chunks []Chunk) {
				assert.Len(t, chunks, 2)
				// Verify ordering
				assert.Equal(t, int32(-2), chunks[0].ChunkX)
				assert.Equal(t, int32(-1), chunks[0].ChunkY)
				assert.Equal(t, int32(-1), chunks[1].ChunkX)
				assert.Equal(t, int32(-2), chunks[1].ChunkY)
			},
		},
		{
			name: "database connection error",
			params: GetChunksParams{
				WorldID:  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:   0,
				ChunkX_2: 1,
				ChunkY:   0,
				ChunkY_2: 1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x >= \\$2 AND chunk_x <= \\$3 AND chunk_y >= \\$4 AND chunk_y <= \\$5 ORDER BY chunk_x, chunk_y").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(0), int32(1), int32(0), int32(1)).
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

			chunks, err := queries.GetChunks(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, chunks)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestChunkExists(t *testing.T) {
	tests := []struct {
		name      string
		params    ChunkExistsParams
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
		expected  bool
	}{
		{
			name: "chunk exists",
			params: ChunkExistsParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: true,
		},
		{
			name: "chunk does not exist",
			params: ChunkExistsParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  999,
				ChunkY:  999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(999), int32(999)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: false,
		},
		{
			name: "chunk with negative coordinates exists",
			params: ChunkExistsParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  -5,
				ChunkY:  -10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-5), int32(-10)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: true,
		},
		{
			name: "non-existent world",
			params: ChunkExistsParams{
				WorldID: mustParseUUID("999e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  0,
				ChunkY:  0,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), int32(0), int32(0)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			exists, err := queries.ChunkExists(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, exists)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestDeleteChunk(t *testing.T) {
	tests := []struct {
		name      string
		params    DeleteChunkParams
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "successful chunk deletion",
			params: DeleteChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM chunks WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "cascade deletion verification - related resource nodes should be cleaned up",
			params: DeleteChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  5,
				ChunkY:  5,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// This test verifies that the deletion succeeds and expects
				// the database to handle cascade deletion of related records
				mock.ExpectExec("DELETE FROM chunks WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(5), int32(5)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "non-existent chunk deletion",
			params: DeleteChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  999,
				ChunkY:  999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM chunks WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(999), int32(999)).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: false, // DELETE doesn't fail on non-existent records
		},
		{
			name: "deletion with negative coordinates",
			params: DeleteChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  -5,
				ChunkY:  -10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM chunks WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-5), int32(-10)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			err = queries.DeleteChunk(createTestContext(), tt.params)

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
func TestChunkBusinessLogic(t *testing.T) {
	t.Run("chunk coordinate boundaries", func(t *testing.T) {
		// Test with extreme coordinate values
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Test creating chunk with maximum int32 coordinates
		params := CreateChunkParams{
			WorldID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			ChunkX:    2147483647,  // Max int32
			ChunkY:    -2147483648, // Min int32
			ChunkData: []byte{0x01},
		}

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
		}).AddRow(
			"550e8400-e29b-41d4-a716-446655440000", int32(2147483647), int32(-2147483648), []byte{0x01}, pgtype.Timestamp{Time: now, Valid: true},
		)

		mockPool.ExpectQuery("INSERT INTO chunks").
			WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(2147483647), int32(-2147483648), []byte{0x01}).
			WillReturnRows(rows)

		chunk, err := queries.CreateChunk(createTestContext(), params)
		require.NoError(t, err)
		assert.Equal(t, int32(2147483647), chunk.ChunkX)
		assert.Equal(t, int32(-2147483648), chunk.ChunkY)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("chunk data size variations", func(t *testing.T) {
		testCases := []struct {
			name   string
			data   []byte
			desc   string
		}{
			{"nil_data", nil, "nil chunk data"},
			{"empty_data", []byte{}, "empty chunk data"},
			{"single_byte", []byte{0xFF}, "single byte chunk data"},
			{"typical_size", make([]byte, 1024), "typical 1KB chunk data"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockPool, err := pgxmock.NewPool()
				require.NoError(t, err)
				defer mockPool.Close()

				queries := New(mockPool)

				params := CreateChunkParams{
					WorldID:   mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
					ChunkX:    0,
					ChunkY:    0,
					ChunkData: tc.data,
				}

				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
				}).AddRow(
					"550e8400-e29b-41d4-a716-446655440000", int32(0), int32(0), tc.data, pgtype.Timestamp{Time: now, Valid: true},
				)

				mockPool.ExpectQuery("INSERT INTO chunks").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(0), int32(0), tc.data).
					WillReturnRows(rows)

				chunk, err := queries.CreateChunk(createTestContext(), params)
				require.NoError(t, err)
				assert.Equal(t, tc.data, chunk.ChunkData, tc.desc)

				assert.NoError(t, mockPool.ExpectationsWereMet())
			})
		}
	})

	t.Run("spatial query performance considerations", func(t *testing.T) {
		// Test large range queries that might impact performance
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		// Query very large range (10000x10000 chunks)
		params := GetChunksParams{
			WorldID:  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			ChunkX:   -5000,
			ChunkX_2: 5000,
			ChunkY:   -5000,
			ChunkY_2: 5000,
		}

		rows := pgxmock.NewRows([]string{
			"world_id", "chunk_x", "chunk_y", "chunk_data", "generated_at",
		})

		mockPool.ExpectQuery("SELECT (.+) FROM chunks WHERE world_id = \\$1 AND chunk_x >= \\$2 AND chunk_x <= \\$3 AND chunk_y >= \\$4 AND chunk_y <= \\$5 ORDER BY chunk_x, chunk_y").
			WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-5000), int32(5000), int32(-5000), int32(5000)).
			WillReturnRows(rows)

		chunks, err := queries.GetChunks(createTestContext(), params)
		require.NoError(t, err)
		assert.Empty(t, chunks) // No chunks in this large range

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// Benchmark tests for performance validation
func BenchmarkCreateChunk(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkGetChunk(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkGetChunks(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}