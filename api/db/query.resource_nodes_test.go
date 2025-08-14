package db

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateResourceNode(t *testing.T) {
	tests := []struct {
		name        string
		params      CreateResourceNodeParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, node ResourceNode)
	}{
		{
			name: "successful resource node creation",
			params: CreateResourceNodeParams{
				ResourceNodeTypeID: 1, // Wood
				WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:             10,
				ChunkY:             20,
				ClusterID:          "cluster_wood_001",
				PosX:               512,
				PosY:               768,
				Size:               3,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).AddRow(
					int32(1), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), "cluster_wood_001", int32(512), int32(768), int32(3), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO resource_nodes").
					WithArgs(int32(1), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20), "cluster_wood_001", int32(512), int32(768), int32(3)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, node ResourceNode) {
				assert.Equal(t, int32(1), node.ResourceNodeTypeID)
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), node.WorldID)
				assert.Equal(t, int32(10), node.ChunkX)
				assert.Equal(t, int32(20), node.ChunkY)
				assert.Equal(t, "cluster_wood_001", node.ClusterID)
				assert.Equal(t, int32(512), node.PosX)
				assert.Equal(t, int32(768), node.PosY)
				assert.Equal(t, int32(3), node.Size)
				assert.True(t, node.CreatedAt.Valid)
			},
		},
		{
			name: "resource node with minimum size",
			params: CreateResourceNodeParams{
				ResourceNodeTypeID: 2, // Stone
				WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:             0,
				ChunkY:             0,
				ClusterID:          "cluster_stone_001",
				PosX:               0,
				PosY:               0,
				Size:               1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).AddRow(
					int32(2), int32(2), "550e8400-e29b-41d4-a716-446655440000", int32(0), int32(0), "cluster_stone_001", int32(0), int32(0), int32(1), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO resource_nodes").
					WithArgs(int32(2), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(0), int32(0), "cluster_stone_001", int32(0), int32(0), int32(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, node ResourceNode) {
				assert.Equal(t, int32(2), node.ResourceNodeTypeID)
				assert.Equal(t, int32(1), node.Size)
				assert.Equal(t, int32(0), node.PosX)
				assert.Equal(t, int32(0), node.PosY)
			},
		},
		{
			name: "resource node with large size",
			params: CreateResourceNodeParams{
				ResourceNodeTypeID: 3, // Iron
				WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:             5,
				ChunkY:             5,
				ClusterID:          "cluster_iron_mega",
				PosX:               256,
				PosY:               384,
				Size:               50,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).AddRow(
					int32(3), int32(3), "550e8400-e29b-41d4-a716-446655440000", int32(5), int32(5), "cluster_iron_mega", int32(256), int32(384), int32(50), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO resource_nodes").
					WithArgs(int32(3), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(5), int32(5), "cluster_iron_mega", int32(256), int32(384), int32(50)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, node ResourceNode) {
				assert.Equal(t, int32(3), node.ResourceNodeTypeID)
				assert.Equal(t, int32(50), node.Size)
				assert.Equal(t, "cluster_iron_mega", node.ClusterID)
			},
		},
		{
			name: "resource node with negative coordinates",
			params: CreateResourceNodeParams{
				ResourceNodeTypeID: 1,
				WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:             -5,
				ChunkY:             -10,
				ClusterID:          "cluster_neg_coords",
				PosX:               -100,
				PosY:               -200,
				Size:               2,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).AddRow(
					int32(4), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(-5), int32(-10), "cluster_neg_coords", int32(-100), int32(-200), int32(2), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO resource_nodes").
					WithArgs(int32(1), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-5), int32(-10), "cluster_neg_coords", int32(-100), int32(-200), int32(2)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, node ResourceNode) {
				assert.Equal(t, int32(-5), node.ChunkX)
				assert.Equal(t, int32(-10), node.ChunkY)
				assert.Equal(t, int32(-100), node.PosX)
				assert.Equal(t, int32(-200), node.PosY)
			},
		},
		{
			name: "duplicate position constraint violation",
			params: CreateResourceNodeParams{
				ResourceNodeTypeID: 1,
				WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:             1,
				ChunkY:             1,
				ClusterID:          "cluster_duplicate",
				PosX:               100, // Same position as existing node
				PosY:               100,
				Size:               1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO resource_nodes").
					WithArgs(int32(1), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(1), int32(1), "cluster_duplicate", int32(100), int32(100), int32(1)).
					WillReturnError(sql.ErrConnDone) // Simulate unique constraint violation
			},
			wantErr: true,
		},
		{
			name: "invalid world foreign key",
			params: CreateResourceNodeParams{
				ResourceNodeTypeID: 1,
				WorldID:            mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), // Non-existent world
				ChunkX:             1,
				ChunkY:             1,
				ClusterID:          "cluster_invalid_world",
				PosX:               50,
				PosY:               50,
				Size:               1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO resource_nodes").
					WithArgs(int32(1), mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), int32(1), int32(1), "cluster_invalid_world", int32(50), int32(50), int32(1)).
					WillReturnError(sql.ErrConnDone) // Simulate foreign key constraint violation
			},
			wantErr: true,
		},
		{
			name: "invalid chunk foreign key",
			params: CreateResourceNodeParams{
				ResourceNodeTypeID: 1,
				WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:             999, // Non-existent chunk
				ChunkY:             999,
				ClusterID:          "cluster_invalid_chunk",
				PosX:               50,
				PosY:               50,
				Size:               1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO resource_nodes").
					WithArgs(int32(1), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(999), int32(999), "cluster_invalid_chunk", int32(50), int32(50), int32(1)).
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

			node, err := queries.CreateResourceNode(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, node)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetResourceNode(t *testing.T) {
	tests := []struct {
		name        string
		resourceID  int32
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, node ResourceNode)
	}{
		{
			name:       "successful resource node retrieval",
			resourceID: 1,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).AddRow(
					int32(1), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), "cluster_wood_001", int32(512), int32(768), int32(3), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.id = \\$1").
					WithArgs(int32(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, node ResourceNode) {
				assert.Equal(t, int32(1), node.ID)
				assert.Equal(t, int32(1), node.ResourceNodeTypeID)
				assert.Equal(t, mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), node.WorldID)
				assert.Equal(t, "cluster_wood_001", node.ClusterID)
				assert.Equal(t, int32(512), node.PosX)
				assert.Equal(t, int32(768), node.PosY)
				assert.Equal(t, int32(3), node.Size)
			},
		},
		{
			name:       "non-existent resource node",
			resourceID: 999,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.id = \\$1").
					WithArgs(int32(999)).
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

			node, err := queries.GetResourceNode(createTestContext(), tt.resourceID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, node)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetResourceNodesInChunk(t *testing.T) {
	tests := []struct {
		name        string
		params      GetResourceNodesInChunkParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, nodes []ResourceNode)
	}{
		{
			name: "multiple resource nodes in chunk",
			params: GetResourceNodesInChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).
					AddRow(
						int32(1), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), "cluster_wood_001", int32(100), int32(200), int32(2), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						int32(2), int32(2), "550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), "cluster_stone_001", int32(300), int32(400), int32(1), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						int32(3), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), "cluster_wood_001", int32(500), int32(600), int32(3), pgtype.Timestamp{Time: now, Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.world_id = \\$1 AND rn.chunk_x = \\$2 AND rn.chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, nodes []ResourceNode) {
				assert.Len(t, nodes, 3)
				// Verify all nodes are in the correct chunk
				for _, node := range nodes {
					assert.Equal(t, int32(10), node.ChunkX)
					assert.Equal(t, int32(20), node.ChunkY)
				}
				// Verify different resource types and clusters
				assert.Equal(t, int32(1), nodes[0].ResourceNodeTypeID) // Wood
				assert.Equal(t, int32(2), nodes[1].ResourceNodeTypeID) // Stone
				assert.Equal(t, int32(1), nodes[2].ResourceNodeTypeID) // Wood
				assert.Equal(t, "cluster_wood_001", nodes[0].ClusterID)
				assert.Equal(t, "cluster_stone_001", nodes[1].ClusterID)
				assert.Equal(t, "cluster_wood_001", nodes[2].ClusterID)
			},
		},
		{
			name: "empty chunk",
			params: GetResourceNodesInChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  999,
				ChunkY:  999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				})
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.world_id = \\$1 AND rn.chunk_x = \\$2 AND rn.chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(999), int32(999)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, nodes []ResourceNode) {
				assert.Empty(t, nodes)
			},
		},
		{
			name: "single resource node in chunk",
			params: GetResourceNodesInChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  5,
				ChunkY:  5,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).AddRow(
					int32(10), int32(3), "550e8400-e29b-41d4-a716-446655440000", int32(5), int32(5), "cluster_iron_001", int32(250), int32(250), int32(5), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.world_id = \\$1 AND rn.chunk_x = \\$2 AND rn.chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(5), int32(5)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, nodes []ResourceNode) {
				assert.Len(t, nodes, 1)
				assert.Equal(t, int32(3), nodes[0].ResourceNodeTypeID)
				assert.Equal(t, int32(5), nodes[0].Size)
				assert.Equal(t, "cluster_iron_001", nodes[0].ClusterID)
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

			nodes, err := queries.GetResourceNodesInChunk(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, nodes)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetResourceNodesInChunkRange(t *testing.T) {
	tests := []struct {
		name        string
		params      GetResourceNodesInChunkRangeParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, nodes []ResourceNode)
	}{
		{
			name: "resource nodes in 2x2 chunk range",
			params: GetResourceNodesInChunkRangeParams{
				WorldID:  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:   0,   // Min X
				ChunkX_2: 1,   // Max X
				ChunkY:   0,   // Min Y
				ChunkY_2: 1,   // Max Y
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).
					AddRow(
						int32(1), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(0), int32(0), "cluster_1", int32(50), int32(50), int32(1), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						int32(2), int32(2), "550e8400-e29b-41d4-a716-446655440000", int32(0), int32(1), "cluster_2", int32(100), int32(150), int32(2), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						int32(3), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(1), int32(0), "cluster_3", int32(200), int32(250), int32(1), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						int32(4), int32(3), "550e8400-e29b-41d4-a716-446655440000", int32(1), int32(1), "cluster_4", int32(300), int32(350), int32(3), pgtype.Timestamp{Time: now, Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.world_id = \\$1 AND rn.chunk_x >= \\$2 AND rn.chunk_x <= \\$3 AND rn.chunk_y >= \\$4 AND rn.chunk_y <= \\$5").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(0), int32(1), int32(0), int32(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, nodes []ResourceNode) {
				assert.Len(t, nodes, 4)
				// Verify all nodes are within the range
				for _, node := range nodes {
					assert.GreaterOrEqual(t, node.ChunkX, int32(0))
					assert.LessOrEqual(t, node.ChunkX, int32(1))
					assert.GreaterOrEqual(t, node.ChunkY, int32(0))
					assert.LessOrEqual(t, node.ChunkY, int32(1))
				}
				// Verify all four corners are represented
				chunks := make(map[string]bool)
				for _, node := range nodes {
					key := fmt.Sprintf("%d,%d", node.ChunkX, node.ChunkY)
					chunks[key] = true
				}
				assert.True(t, chunks["0,0"])
				assert.True(t, chunks["0,1"])
				assert.True(t, chunks["1,0"])
				assert.True(t, chunks["1,1"])
			},
		},
		{
			name: "empty range",
			params: GetResourceNodesInChunkRangeParams{
				WorldID:  mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:   100,
				ChunkX_2: 102,
				ChunkY:   100,
				ChunkY_2: 102,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				})
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.world_id = \\$1 AND rn.chunk_x >= \\$2 AND rn.chunk_x <= \\$3 AND rn.chunk_y >= \\$4 AND rn.chunk_y <= \\$5").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(100), int32(102), int32(100), int32(102)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, nodes []ResourceNode) {
				assert.Empty(t, nodes)
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

			nodes, err := queries.GetResourceNodesInChunkRange(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, nodes)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetResourceNodesInCluster(t *testing.T) {
	tests := []struct {
		name        string
		clusterID   string
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, nodes []ResourceNode)
	}{
		{
			name:      "multiple nodes in cluster",
			clusterID: "cluster_wood_001",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).
					AddRow(
						int32(1), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), "cluster_wood_001", int32(100), int32(100), int32(2), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						int32(2), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), "cluster_wood_001", int32(150), int32(150), int32(1), pgtype.Timestamp{Time: now, Valid: true},
					).
					AddRow(
						int32(3), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(10), int32(20), "cluster_wood_001", int32(200), int32(200), int32(3), pgtype.Timestamp{Time: now, Valid: true},
					)
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.cluster_id = \\$1").
					WithArgs("cluster_wood_001").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, nodes []ResourceNode) {
				assert.Len(t, nodes, 3)
				// Verify all nodes belong to the same cluster
				for _, node := range nodes {
					assert.Equal(t, "cluster_wood_001", node.ClusterID)
					assert.Equal(t, int32(1), node.ResourceNodeTypeID) // All wood
				}
				// Verify different positions and sizes
				positions := make(map[string]bool)
				for _, node := range nodes {
					key := fmt.Sprintf("%d,%d", node.PosX, node.PosY)
					positions[key] = true
				}
				assert.Len(t, positions, 3) // All different positions
			},
		},
		{
			name:      "empty cluster",
			clusterID: "non_existent_cluster",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				})
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.cluster_id = \\$1").
					WithArgs("non_existent_cluster").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, nodes []ResourceNode) {
				assert.Empty(t, nodes)
			},
		},
		{
			name:      "single node cluster",
			clusterID: "cluster_iron_solo",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).AddRow(
					int32(10), int32(3), "550e8400-e29b-41d4-a716-446655440000", int32(5), int32(5), "cluster_iron_solo", int32(256), int32(256), int32(10), pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("SELECT (.+) FROM resource_nodes rn WHERE rn.cluster_id = \\$1").
					WithArgs("cluster_iron_solo").
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, nodes []ResourceNode) {
				assert.Len(t, nodes, 1)
				assert.Equal(t, "cluster_iron_solo", nodes[0].ClusterID)
				assert.Equal(t, int32(3), nodes[0].ResourceNodeTypeID)
				assert.Equal(t, int32(10), nodes[0].Size)
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

			nodes, err := queries.GetResourceNodesInCluster(createTestContext(), tt.clusterID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, nodes)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestResourceNodeExistsAtPosition(t *testing.T) {
	tests := []struct {
		name      string
		params    ResourceNodeExistsAtPositionParams
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
		expected  bool
	}{
		{
			name: "resource node exists at position",
			params: ResourceNodeExistsAtPositionParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  10,
				ChunkY:  20,
				PosX:    512,
				PosY:    768,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20), int32(512), int32(768)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: true,
		},
		{
			name: "no resource node at position",
			params: ResourceNodeExistsAtPositionParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  10,
				ChunkY:  20,
				PosX:    999,
				PosY:    999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20), int32(999), int32(999)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: false,
		},
		{
			name: "negative coordinates check",
			params: ResourceNodeExistsAtPositionParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  -5,
				ChunkY:  -10,
				PosX:    -100,
				PosY:    -200,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-5), int32(-10), int32(-100), int32(-200)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			exists, err := queries.ResourceNodeExistsAtPosition(createTestContext(), tt.params)

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

func TestCountResourceNodesInChunk(t *testing.T) {
	tests := []struct {
		name      string
		params    CountResourceNodesInChunkParams
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
		expected  int64
	}{
		{
			name: "chunk with multiple resource nodes",
			params: CountResourceNodesInChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(15))
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM resource_nodes WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: 15,
		},
		{
			name: "empty chunk",
			params: CountResourceNodesInChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  999,
				ChunkY:  999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM resource_nodes WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(999), int32(999)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			count, err := queries.CountResourceNodesInChunk(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, count)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestCountResourceNodesByType(t *testing.T) {
	tests := []struct {
		name      string
		params    CountResourceNodesByTypeParams
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
		expected  int64
	}{
		{
			name: "count wood nodes in chunk",
			params: CountResourceNodesByTypeParams{
				WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:             10,
				ChunkY:             20,
				ResourceNodeTypeID: 1, // Wood
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(8))
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM resource_nodes WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3 AND resource_node_type_id = \\$4").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20), int32(1)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: 8,
		},
		{
			name: "no nodes of specific type",
			params: CountResourceNodesByTypeParams{
				WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:             10,
				ChunkY:             20,
				ResourceNodeTypeID: 999, // Non-existent type
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM resource_nodes WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3 AND resource_node_type_id = \\$4").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20), int32(999)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			count, err := queries.CountResourceNodesByType(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, count)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestDeleteResourceNodesInChunk(t *testing.T) {
	tests := []struct {
		name      string
		params    DeleteResourceNodesInChunkParams
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "successful deletion of resource nodes",
			params: DeleteResourceNodesInChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  10,
				ChunkY:  20,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM resource_nodes WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(10), int32(20)).
					WillReturnResult(pgxmock.NewResult("DELETE", 5)) // Deleted 5 nodes
			},
			wantErr: false,
		},
		{
			name: "delete from empty chunk",
			params: DeleteResourceNodesInChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  999,
				ChunkY:  999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM resource_nodes WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(999), int32(999)).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: false, // DELETE doesn't fail on empty chunks
		},
		{
			name: "delete with negative coordinates",
			params: DeleteResourceNodesInChunkParams{
				WorldID: mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
				ChunkX:  -5,
				ChunkY:  -10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM resource_nodes WHERE world_id = \\$1 AND chunk_x = \\$2 AND chunk_y = \\$3").
					WithArgs(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(-5), int32(-10)).
					WillReturnResult(pgxmock.NewResult("DELETE", 3))
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

			err = queries.DeleteResourceNodesInChunk(createTestContext(), tt.params)

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
func TestResourceNodeBusinessLogic(t *testing.T) {
	t.Run("resource node coordinate boundaries", func(t *testing.T) {
		// Test with extreme coordinate values
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := CreateResourceNodeParams{
			ResourceNodeTypeID: 1,
			WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			ChunkX:             2147483647,  // Max int32
			ChunkY:             -2147483648, // Min int32
			ClusterID:          "cluster_extreme",
			PosX:               2147483647,  // Max int32
			PosY:               -2147483648, // Min int32
			Size:               2147483647,  // Max int32
		}

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
		}).AddRow(
			int32(1), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(2147483647), int32(-2147483648), "cluster_extreme", int32(2147483647), int32(-2147483648), int32(2147483647), pgtype.Timestamp{Time: now, Valid: true},
		)

		mockPool.ExpectQuery("INSERT INTO resource_nodes").
			WithArgs(int32(1), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(2147483647), int32(-2147483648), "cluster_extreme", int32(2147483647), int32(-2147483648), int32(2147483647)).
			WillReturnRows(rows)

		node, err := queries.CreateResourceNode(createTestContext(), params)
		require.NoError(t, err)
		assert.Equal(t, int32(2147483647), node.ChunkX)
		assert.Equal(t, int32(-2147483648), node.ChunkY)
		assert.Equal(t, int32(2147483647), node.PosX)
		assert.Equal(t, int32(-2147483648), node.PosY)
		assert.Equal(t, int32(2147483647), node.Size)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("cluster ID variations", func(t *testing.T) {
		testCases := []struct {
			name      string
			clusterID string
			desc      string
		}{
			{"empty_cluster", "", "empty cluster ID"},
			{"long_cluster", strings.Repeat("cluster_", 100), "very long cluster ID"},
			{"special_chars", "cluster!@#$%^&*()_+-=[]{}|;':\",./<>?", "cluster with special characters"},
			{"unicode_cluster", "cluster_世界_🌍", "cluster with unicode characters"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockPool, err := pgxmock.NewPool()
				require.NoError(t, err)
				defer mockPool.Close()

				queries := New(mockPool)

				params := CreateResourceNodeParams{
					ResourceNodeTypeID: 1,
					WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
					ChunkX:             0,
					ChunkY:             0,
					ClusterID:          tc.clusterID,
					PosX:               0,
					PosY:               0,
					Size:               1,
				}

				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
				}).AddRow(
					int32(1), int32(1), "550e8400-e29b-41d4-a716-446655440000", int32(0), int32(0), tc.clusterID, int32(0), int32(0), int32(1), pgtype.Timestamp{Time: now, Valid: true},
				)

				mockPool.ExpectQuery("INSERT INTO resource_nodes").
					WithArgs(int32(1), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(0), int32(0), tc.clusterID, int32(0), int32(0), int32(1)).
					WillReturnRows(rows)

				node, err := queries.CreateResourceNode(createTestContext(), params)
				require.NoError(t, err)
				assert.Equal(t, tc.clusterID, node.ClusterID, tc.desc)

				assert.NoError(t, mockPool.ExpectationsWereMet())
			})
		}
	})

	t.Run("resource node type variations", func(t *testing.T) {
		// Test negative resource node type IDs
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := CreateResourceNodeParams{
			ResourceNodeTypeID: -1, // Negative resource type
			WorldID:            mustParseUUID("550e8400-e29b-41d4-a716-446655440000"),
			ChunkX:             0,
			ChunkY:             0,
			ClusterID:          "cluster_negative_type",
			PosX:               0,
			PosY:               0,
			Size:               1,
		}

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "resource_node_type_id", "world_id", "chunk_x", "chunk_y", "cluster_id", "pos_x", "pos_y", "size", "created_at",
		}).AddRow(
			int32(1), int32(-1), "550e8400-e29b-41d4-a716-446655440000", int32(0), int32(0), "cluster_negative_type", int32(0), int32(0), int32(1), pgtype.Timestamp{Time: now, Valid: true},
		)

		mockPool.ExpectQuery("INSERT INTO resource_nodes").
			WithArgs(int32(-1), mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), int32(0), int32(0), "cluster_negative_type", int32(0), int32(0), int32(1)).
			WillReturnRows(rows)

		node, err := queries.CreateResourceNode(createTestContext(), params)
		require.NoError(t, err)
		assert.Equal(t, int32(-1), node.ResourceNodeTypeID)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// Benchmark tests for performance validation
func BenchmarkCreateResourceNode(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkGetResourceNodesInChunk(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkGetResourceNodesInCluster(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}