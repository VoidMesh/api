package handlers

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/testmocks/handlers"
	"github.com/VoidMesh/api/api/internal/testutil"
	worldV1 "github.com/VoidMesh/api/api/proto/world/v1"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestWorldServiceServer_GetWorld demonstrates comprehensive testing patterns for world retrieval
func TestWorldServiceServer_GetWorld(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock
	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	// Create server instance
	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *worldV1.GetWorldRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *worldV1.GetWorldResponse)
	}{
		{
			name: "successful world retrieval",
			request: &worldV1.GetWorldRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				expectedWorld := db.World{
					ID:        worldUUID,
					Name:      "Test World",
					Seed:      12345,
					CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
				}

				mockWorldService.EXPECT().
					GetWorldByID(gomock.Any(), worldUUID).
					Return(expectedWorld, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *worldV1.GetWorldResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.World)
				
				expectedWorldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				assert.Equal(t, expectedWorldUUID.Bytes[:], resp.World.Id)
				assert.Equal(t, "Test World", resp.World.Name)
				assert.Equal(t, int64(12345), resp.World.Seed)
				assert.NotNil(t, resp.World.CreatedAt)
			},
		},
		{
			name: "invalid UUID format",
			request: &worldV1.GetWorldRequest{
				WorldId: []byte("invalid-uuid-format"),
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
			wantMsg:    "Invalid world ID",
		},
		{
			name: "empty world ID",
			request: &worldV1.GetWorldRequest{
				WorldId: []byte(""),
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
			wantMsg:    "Invalid world ID",
		},
		{
			name: "world not found",
			request: &worldV1.GetWorldRequest{
				WorldId: []byte(testutil.UUIDTestData.World2),
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World2)
				mockWorldService.EXPECT().
					GetWorldByID(gomock.Any(), worldUUID).
					Return(db.World{}, errors.New("no rows in result set"))
			},
			wantErr:  true,
			wantCode: codes.NotFound,
			wantMsg:  "World not found",
		},
		{
			name: "service layer error",
			request: &worldV1.GetWorldRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				mockWorldService.EXPECT().
					GetWorldByID(gomock.Any(), worldUUID).
					Return(db.World{}, errors.New("database connection failed"))
			},
			wantErr:  true,
			wantCode: codes.NotFound,
			wantMsg:  "World not found",
		},
		{
			name: "malformed bytes in world ID",
			request: &worldV1.GetWorldRequest{
				WorldId: []byte{0x00, 0x01, 0x02}, // Invalid UUID bytes
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
			wantMsg:    "Invalid world ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			tt.setupMocks()

			// Execute request
			resp, err := server.GetWorld(context.Background(), tt.request)

			// Validate error expectation
			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestWorldServiceServer_GetDefaultWorld demonstrates testing patterns for default world retrieval
func TestWorldServiceServer_GetDefaultWorld(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *worldV1.GetDefaultWorldRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *worldV1.GetDefaultWorldResponse)
	}{
		{
			name:    "successful default world retrieval",
			request: &worldV1.GetDefaultWorldRequest{},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				expectedWorld := db.World{
					ID:        worldUUID,
					Name:      "Default World",
					Seed:      54321,
					CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
				}

				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(expectedWorld, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *worldV1.GetDefaultWorldResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.World)
				
				expectedWorldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				assert.Equal(t, expectedWorldUUID.Bytes[:], resp.World.Id)
				assert.Equal(t, "Default World", resp.World.Name)
				assert.Equal(t, int64(54321), resp.World.Seed)
				assert.NotNil(t, resp.World.CreatedAt)
			},
		},
		{
			name:    "no default world configured",
			request: &worldV1.GetDefaultWorldRequest{},
			setupMocks: func() {
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(db.World{}, errors.New("no default world found"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to get default world",
		},
		{
			name:    "service layer database error",
			request: &worldV1.GetDefaultWorldRequest{},
			setupMocks: func() {
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(db.World{}, errors.New("database connection error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to get default world",
		},
		{
			name:    "service layer timeout error",
			request: &worldV1.GetDefaultWorldRequest{},
			setupMocks: func() {
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(db.World{}, context.DeadlineExceeded)
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to get default world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.GetDefaultWorld(context.Background(), tt.request)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestWorldServiceServer_ListWorlds demonstrates testing patterns for world listing
func TestWorldServiceServer_ListWorlds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *worldV1.ListWorldsRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *worldV1.ListWorldsResponse)
	}{
		{
			name:    "successful multiple worlds retrieval",
			request: &worldV1.ListWorldsRequest{},
			setupMocks: func() {
				world1UUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				world2UUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World2)
				
				expectedWorlds := []db.World{
					{
						ID:        world1UUID,
						Name:      "World One",
						Seed:      12345,
						CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
					},
					{
						ID:        world2UUID,
						Name:      "World Two",
						Seed:      67890,
						CreatedAt: pgtype.Timestamp{Time: time.Now().Add(-time.Hour), Valid: true},
					},
				}

				mockWorldService.EXPECT().
					ListWorlds(gomock.Any()).
					Return(expectedWorlds, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *worldV1.ListWorldsResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Worlds, 2)
				
				world1UUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				world2UUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World2)
				
				assert.Equal(t, world1UUID.Bytes[:], resp.Worlds[0].Id)
				assert.Equal(t, "World One", resp.Worlds[0].Name)
				assert.Equal(t, int64(12345), resp.Worlds[0].Seed)
				
				assert.Equal(t, world2UUID.Bytes[:], resp.Worlds[1].Id)
				assert.Equal(t, "World Two", resp.Worlds[1].Name)
				assert.Equal(t, int64(67890), resp.Worlds[1].Seed)
			},
		},
		{
			name:    "successful empty list retrieval",
			request: &worldV1.ListWorldsRequest{},
			setupMocks: func() {
				mockWorldService.EXPECT().
					ListWorlds(gomock.Any()).
					Return([]db.World{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *worldV1.ListWorldsResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Worlds, 0)
				assert.NotNil(t, resp.Worlds) // Should be empty slice, not nil
			},
		},
		{
			name:    "large result set handling",
			request: &worldV1.ListWorldsRequest{},
			setupMocks: func() {
				var worlds []db.World
				for i := 0; i < 100; i++ {
					worldUUID := testutil.UUIDFromString(testutil.GenerateTestUUID())
					worlds = append(worlds, db.World{
						ID:        worldUUID,
						Name:      testutil.GenerateTestString(10),
						Seed:      int64(i),
						CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
					})
				}

				mockWorldService.EXPECT().
					ListWorlds(gomock.Any()).
					Return(worlds, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *worldV1.ListWorldsResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Worlds, 100)
				
				// Verify all worlds have valid data
				for i, world := range resp.Worlds {
					assert.NotEmpty(t, world.Id)
					assert.NotEmpty(t, world.Name)
					assert.Equal(t, int64(i), world.Seed)
					assert.NotNil(t, world.CreatedAt)
				}
			},
		},
		{
			name:    "service layer database error",
			request: &worldV1.ListWorldsRequest{},
			setupMocks: func() {
				mockWorldService.EXPECT().
					ListWorlds(gomock.Any()).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to list worlds",
		},
		{
			name:    "service layer timeout error",
			request: &worldV1.ListWorldsRequest{},
			setupMocks: func() {
				mockWorldService.EXPECT().
					ListWorlds(gomock.Any()).
					Return(nil, context.DeadlineExceeded)
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to list worlds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.ListWorlds(context.Background(), tt.request)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestWorldServiceServer_UpdateWorldName demonstrates testing patterns for world name updates
func TestWorldServiceServer_UpdateWorldName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *worldV1.UpdateWorldNameRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *worldV1.UpdateWorldNameResponse)
	}{
		{
			name: "successful name update",
			request: &worldV1.UpdateWorldNameRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
				Name:    "Updated World Name",
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				updatedWorld := db.World{
					ID:        worldUUID,
					Name:      "Updated World Name",
					Seed:      12345,
					CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
				}

				mockWorldService.EXPECT().
					UpdateWorld(gomock.Any(), worldUUID, "Updated World Name").
					Return(updatedWorld, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *worldV1.UpdateWorldNameResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.World)
				
				expectedWorldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				assert.Equal(t, expectedWorldUUID.Bytes[:], resp.World.Id)
				assert.Equal(t, "Updated World Name", resp.World.Name)
				assert.Equal(t, int64(12345), resp.World.Seed)
			},
		},
		{
			name: "invalid UUID format",
			request: &worldV1.UpdateWorldNameRequest{
				WorldId: []byte("invalid-uuid"),
				Name:    "Updated Name",
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
			wantMsg:    "Invalid world ID",
		},
		{
			name: "empty world ID",
			request: &worldV1.UpdateWorldNameRequest{
				WorldId: []byte(""),
				Name:    "Updated Name",
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
			wantMsg:    "Invalid world ID",
		},
		{
			name: "empty name validation",
			request: &worldV1.UpdateWorldNameRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
				Name:    "",
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				mockWorldService.EXPECT().
					UpdateWorld(gomock.Any(), worldUUID, "").
					Return(db.World{}, errors.New("name cannot be empty"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to update world",
		},
		{
			name: "long name validation (>255 chars)",
			request: &worldV1.UpdateWorldNameRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
				Name:    strings.Repeat("a", 300), // Exceeds typical limit
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				longName := strings.Repeat("a", 300)
				mockWorldService.EXPECT().
					UpdateWorld(gomock.Any(), worldUUID, longName).
					Return(db.World{}, errors.New("name too long"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to update world",
		},
		{
			name: "unicode name support",
			request: &worldV1.UpdateWorldNameRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
				Name:    "世界名前🌍🎮テスト",
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				unicodeName := "世界名前🌍🎮テスト"
				updatedWorld := db.World{
					ID:        worldUUID,
					Name:      unicodeName,
					Seed:      12345,
					CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
				}

				mockWorldService.EXPECT().
					UpdateWorld(gomock.Any(), worldUUID, unicodeName).
					Return(updatedWorld, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *worldV1.UpdateWorldNameResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.World)
				assert.Equal(t, "世界名前🌍🎮テスト", resp.World.Name)
			},
		},
		{
			name: "world not found",
			request: &worldV1.UpdateWorldNameRequest{
				WorldId: []byte(testutil.UUIDTestData.World2),
				Name:    "Updated Name",
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World2)
				mockWorldService.EXPECT().
					UpdateWorld(gomock.Any(), worldUUID, "Updated Name").
					Return(db.World{}, errors.New("world not found"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to update world",
		},
		{
			name: "service layer database error",
			request: &worldV1.UpdateWorldNameRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
				Name:    "Updated Name",
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				mockWorldService.EXPECT().
					UpdateWorld(gomock.Any(), worldUUID, "Updated Name").
					Return(db.World{}, errors.New("database constraint violation"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to update world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.UpdateWorldName(context.Background(), tt.request)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestWorldServiceServer_DeleteWorld demonstrates testing patterns for world deletion
func TestWorldServiceServer_DeleteWorld(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *worldV1.DeleteWorldRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *worldV1.DeleteWorldResponse)
	}{
		{
			name: "successful world deletion",
			request: &worldV1.DeleteWorldRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				mockWorldService.EXPECT().
					DeleteWorld(gomock.Any(), worldUUID).
					Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *worldV1.DeleteWorldResponse) {
				require.NotNil(t, resp)
				// DeleteWorldResponse is empty, just verify it's not nil
			},
		},
		{
			name: "invalid UUID format",
			request: &worldV1.DeleteWorldRequest{
				WorldId: []byte("invalid-uuid"),
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
			wantMsg:    "Invalid world ID",
		},
		{
			name: "empty world ID",
			request: &worldV1.DeleteWorldRequest{
				WorldId: []byte(""),
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
			wantMsg:    "Invalid world ID",
		},
		{
			name: "world not found",
			request: &worldV1.DeleteWorldRequest{
				WorldId: []byte(testutil.UUIDTestData.World2),
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World2)
				mockWorldService.EXPECT().
					DeleteWorld(gomock.Any(), worldUUID).
					Return(errors.New("world not found"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to delete world",
		},
		{
			name: "foreign key constraint violation",
			request: &worldV1.DeleteWorldRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				mockWorldService.EXPECT().
					DeleteWorld(gomock.Any(), worldUUID).
					Return(errors.New("foreign key constraint violation"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to delete world",
		},
		{
			name: "service layer database error",
			request: &worldV1.DeleteWorldRequest{
				WorldId: []byte(testutil.UUIDTestData.World1),
			},
			setupMocks: func() {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				mockWorldService.EXPECT().
					DeleteWorld(gomock.Any(), worldUUID).
					Return(errors.New("database connection failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "Failed to delete world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.DeleteWorld(context.Background(), tt.request)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode, tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				testutil.AssertNoGRPCError(t, err)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestWorldServiceServer_EdgeCases demonstrates edge case testing
func TestWorldServiceServer_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "null responses handling",
			testFunc: func(t *testing.T) {
				// Test GetDefaultWorld with nil response handling
				mockWorldService.EXPECT().
					GetDefaultWorld(gomock.Any()).
					Return(db.World{}, nil) // Valid but empty world

				resp, err := server.GetDefaultWorld(context.Background(), &worldV1.GetDefaultWorldRequest{})
				
				testutil.AssertNoGRPCError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.World)
				// Empty world should still create valid proto message
				assert.NotNil(t, resp.World.Id)
				assert.Equal(t, "", resp.World.Name)
				assert.Equal(t, int64(0), resp.World.Seed)
			},
		},
		{
			name: "concurrent access patterns simulation",
			testFunc: func(t *testing.T) {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				expectedWorld := db.World{
					ID:        worldUUID,
					Name:      "Concurrent World",
					Seed:      99999,
					CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
				}

				// Simulate multiple concurrent requests
				mockWorldService.EXPECT().
					GetWorldByID(gomock.Any(), worldUUID).
					Return(expectedWorld, nil).
					Times(5)

				// Execute 5 concurrent requests
				done := make(chan bool, 5)
				for i := 0; i < 5; i++ {
					go func() {
						defer func() { done <- true }()
						
						resp, err := server.GetWorld(context.Background(), &worldV1.GetWorldRequest{
							WorldId: []byte(testutil.UUIDTestData.World1),
						})
						
						assert.NoError(t, err)
						assert.NotNil(t, resp)
						assert.Equal(t, "Concurrent World", resp.World.Name)
					}()
				}

				// Wait for all requests to complete
				for i := 0; i < 5; i++ {
					<-done
				}
			},
		},
		{
			name: "protocol buffer field validation",
			testFunc: func(t *testing.T) {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				now := time.Now()
				expectedWorld := db.World{
					ID:        worldUUID,
					Name:      "Proto Test World",
					Seed:      42,
					CreatedAt: pgtype.Timestamp{Time: now, Valid: true},
				}

				mockWorldService.EXPECT().
					GetWorldByID(gomock.Any(), worldUUID).
					Return(expectedWorld, nil)

				resp, err := server.GetWorld(context.Background(), &worldV1.GetWorldRequest{
					WorldId: []byte(testutil.UUIDTestData.World1),
				})

				testutil.AssertNoGRPCError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.World)
				
				// Validate proto timestamp conversion
				assert.Equal(t, timestamppb.New(now), resp.World.CreatedAt)
				
				// Validate all required fields are present
				assert.NotEmpty(t, resp.World.Id)
				assert.NotEmpty(t, resp.World.Name)
				assert.NotZero(t, resp.World.Seed)
				assert.NotNil(t, resp.World.CreatedAt)
				
				// Validate UUID bytes conversion
				assert.Equal(t, worldUUID.Bytes[:], resp.World.Id)
			},
		},
		{
			name: "maximum seed value handling",
			testFunc: func(t *testing.T) {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				expectedWorld := db.World{
					ID:        worldUUID,
					Name:      "Max Seed World",
					Seed:      9223372036854775807, // max int64
					CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
				}

				mockWorldService.EXPECT().
					GetWorldByID(gomock.Any(), worldUUID).
					Return(expectedWorld, nil)

				resp, err := server.GetWorld(context.Background(), &worldV1.GetWorldRequest{
					WorldId: []byte(testutil.UUIDTestData.World1),
				})

				testutil.AssertNoGRPCError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, int64(9223372036854775807), resp.World.Seed)
			},
		},
		{
			name: "minimum seed value handling",
			testFunc: func(t *testing.T) {
				worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
				expectedWorld := db.World{
					ID:        worldUUID,
					Name:      "Min Seed World",
					Seed:      -9223372036854775808, // min int64
					CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
				}

				mockWorldService.EXPECT().
					GetWorldByID(gomock.Any(), worldUUID).
					Return(expectedWorld, nil)

				resp, err := server.GetWorld(context.Background(), &worldV1.GetWorldRequest{
					WorldId: []byte(testutil.UUIDTestData.World1),
				})

				testutil.AssertNoGRPCError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, int64(-9223372036854775808), resp.World.Seed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

// TestWorldServiceServer_ProtocolBufferMessageValidation demonstrates proto message validation testing
func TestWorldServiceServer_ProtocolBufferMessageValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	t.Run("get world response proto validation", func(t *testing.T) {
		worldUUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
		now := time.Now()
		expectedWorld := db.World{
			ID:        worldUUID,
			Name:      "Proto Validation World",
			Seed:      123456789,
			CreatedAt: pgtype.Timestamp{Time: now, Valid: true},
		}

		expectedResponse := &worldV1.GetWorldResponse{
			World: &worldV1.World{
				Id:        worldUUID.Bytes[:],
				Name:      "Proto Validation World",
				Seed:      123456789,
				CreatedAt: timestamppb.New(now),
			},
		}

		mockWorldService.EXPECT().
			GetWorldByID(gomock.Any(), worldUUID).
			Return(expectedWorld, nil)

		resp, err := server.GetWorld(context.Background(), &worldV1.GetWorldRequest{
			WorldId: []byte(testutil.UUIDTestData.World1),
		})

		testutil.AssertNoGRPCError(t, err)
		testutil.CompareProtoMessages(t, expectedResponse, resp)

		// Validate specific proto fields
		testutil.AssertProtoFieldEqual(t, expectedResponse.World, resp.World, "name")
		testutil.AssertProtoFieldEqual(t, expectedResponse.World, resp.World, "seed")
	})

	t.Run("list worlds response proto validation", func(t *testing.T) {
		world1UUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World1)
		world2UUID := testutil.ParseTestUUID(t, testutil.UUIDTestData.World2)
		now := time.Now()
		
		expectedWorlds := []db.World{
			{
				ID:        world1UUID,
				Name:      "World Alpha",
				Seed:      111,
				CreatedAt: pgtype.Timestamp{Time: now, Valid: true},
			},
			{
				ID:        world2UUID,
				Name:      "World Beta",
				Seed:      222,
				CreatedAt: pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true},
			},
		}

		expectedResponse := &worldV1.ListWorldsResponse{
			Worlds: []*worldV1.World{
				{
					Id:        world1UUID.Bytes[:],
					Name:      "World Alpha",
					Seed:      111,
					CreatedAt: timestamppb.New(now),
				},
				{
					Id:        world2UUID.Bytes[:],
					Name:      "World Beta",
					Seed:      222,
					CreatedAt: timestamppb.New(now.Add(-time.Hour)),
				},
			},
		}

		mockWorldService.EXPECT().
			ListWorlds(gomock.Any()).
			Return(expectedWorlds, nil)

		resp, err := server.ListWorlds(context.Background(), &worldV1.ListWorldsRequest{})

		testutil.AssertNoGRPCError(t, err)
		testutil.CompareProtoMessages(t, expectedResponse, resp)

		// Validate array handling
		assert.Len(t, resp.Worlds, 2)
		testutil.AssertProtoFieldEqual(t, expectedResponse.Worlds[0], resp.Worlds[0], "name")
		testutil.AssertProtoFieldEqual(t, expectedResponse.Worlds[1], resp.Worlds[1], "name")
	})
}

// BenchmarkWorldServiceServer_GetWorld measures performance of world retrieval
func BenchmarkWorldServiceServer_GetWorld(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	// Setup mock for all benchmark iterations
	worldUUID := testutil.MustParseUUID(testutil.UUIDTestData.World1)
	expectedWorld := db.World{
		ID:        worldUUID,
		Name:      "Benchmark World",
		Seed:      12345,
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	mockWorldService.EXPECT().
		GetWorldByID(gomock.Any(), worldUUID).
		Return(expectedWorld, nil).
		AnyTimes()

	request := &worldV1.GetWorldRequest{
		WorldId: []byte(testutil.UUIDTestData.World1),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.GetWorld(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWorldServiceServer_GetDefaultWorld measures performance of default world retrieval
func BenchmarkWorldServiceServer_GetDefaultWorld(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	// Setup mock for all benchmark iterations
	worldUUID := testutil.MustParseUUID(testutil.UUIDTestData.World1)
	expectedWorld := db.World{
		ID:        worldUUID,
		Name:      "Default Benchmark World",
		Seed:      54321,
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	mockWorldService.EXPECT().
		GetDefaultWorld(gomock.Any()).
		Return(expectedWorld, nil).
		AnyTimes()

	request := &worldV1.GetDefaultWorldRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.GetDefaultWorld(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWorldServiceServer_ListWorlds measures performance of world listing
func BenchmarkWorldServiceServer_ListWorlds(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	// Setup mock with multiple worlds for realistic benchmarking
	var worlds []db.World
	for i := 0; i < 10; i++ {
		worldUUID := testutil.UUIDFromString(testutil.GenerateTestUUID())
		worlds = append(worlds, db.World{
			ID:        worldUUID,
			Name:      testutil.GenerateTestString(10),
			Seed:      int64(i),
			CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		})
	}

	mockWorldService.EXPECT().
		ListWorlds(gomock.Any()).
		Return(worlds, nil).
		AnyTimes()

	request := &worldV1.ListWorldsRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.ListWorlds(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWorldServiceServer_UpdateWorldName measures performance of world name updates
func BenchmarkWorldServiceServer_UpdateWorldName(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	// Setup mock for all benchmark iterations
	worldUUID := testutil.MustParseUUID(testutil.UUIDTestData.World1)
	updatedWorld := db.World{
		ID:        worldUUID,
		Name:      "Benchmark Updated World",
		Seed:      12345,
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	mockWorldService.EXPECT().
		UpdateWorld(gomock.Any(), worldUUID, "Benchmark Updated World").
		Return(updatedWorld, nil).
		AnyTimes()

	request := &worldV1.UpdateWorldNameRequest{
		WorldId: []byte(testutil.UUIDTestData.World1),
		Name:    "Benchmark Updated World",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.UpdateWorldName(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWorldServiceServer_DeleteWorld measures performance of world deletion
func BenchmarkWorldServiceServer_DeleteWorld(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockWorldService := mockhandlers.NewMockWorldService(ctrl)

	server := &worldServiceServer{
		worldService: mockWorldService,
		logger:       log.New(io.Discard),
	}

	// Setup mock for all benchmark iterations
	worldUUID := testutil.MustParseUUID(testutil.UUIDTestData.World1)
	mockWorldService.EXPECT().
		DeleteWorld(gomock.Any(), worldUUID).
		Return(nil).
		AnyTimes()

	request := &worldV1.DeleteWorldRequest{
		WorldId: []byte(testutil.UUIDTestData.World1),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.DeleteWorld(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}