package handlers

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/VoidMesh/api/api/internal/testmocks/handlers"
	"github.com/VoidMesh/api/api/internal/testutil"
	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	"github.com/VoidMesh/api/api/server/middleware"
	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)


// TestCharacterServiceServer_CreateCharacter demonstrates comprehensive testing patterns for character creation
func TestCharacterServiceServer_CreateCharacter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock
	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	// Create server instance
	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *characterV1.CreateCharacterRequest
		context    context.Context
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *characterV1.CreateCharacterResponse)
	}{
		{
			name: "successful character creation",
			request: &characterV1.CreateCharacterRequest{
				Name:    "TestHero",
				SpawnX:  100,
				SpawnY:  200,
			},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), expectedUserID, &characterV1.CreateCharacterRequest{
						Name:   "TestHero",
						SpawnX: 100,
						SpawnY: 200,
					}).
					Return(&characterV1.CreateCharacterResponse{
						Character: &characterV1.Character{
							Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
							UserId:   expectedUserID,
							Name:     "TestHero",
							X:        100,
							Y:        200,
							ChunkX:   3,
							ChunkY:   6,
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.CreateCharacterResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.Character)
				
				expectedCharacterID := strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", "")
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				assert.Equal(t, expectedCharacterID, resp.Character.Id)
				assert.Equal(t, expectedUserID, resp.Character.UserId)
				assert.Equal(t, "TestHero", resp.Character.Name)
				assert.Equal(t, int32(100), resp.Character.X)
				assert.Equal(t, int32(200), resp.Character.Y)
				assert.Equal(t, int32(3), resp.Character.ChunkX)
				assert.Equal(t, int32(6), resp.Character.ChunkY)
				assert.NotNil(t, resp.Character.CreatedAt)
			},
		},
		{
			name: "successful character creation with default spawn",
			request: &characterV1.CreateCharacterRequest{
				Name: "DefaultSpawnHero",
			},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), expectedUserID, &characterV1.CreateCharacterRequest{
						Name:   "DefaultSpawnHero",
						SpawnX: 0,
						SpawnY: 0,
					}).
					Return(&characterV1.CreateCharacterResponse{
						Character: &characterV1.Character{
							Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
							UserId:   expectedUserID,
							Name:     "DefaultSpawnHero",
							X:        0,
							Y:        0,
							ChunkX:   0,
							ChunkY:   0,
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "missing authentication",
			request: &characterV1.CreateCharacterRequest{
				Name:   "UnauthenticatedHero",
				SpawnX: 100,
				SpawnY: 200,
			},
			context:    context.Background(),
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.Unauthenticated,
			wantMsg:    "user not authenticated",
		},
		{
			name: "invalid authentication token",
			request: &characterV1.CreateCharacterRequest{
				Name:   "InvalidTokenHero",
				SpawnX: 100,
				SpawnY: 200,
			},
			context:    context.Background(),
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.Unauthenticated,
			wantMsg:    "user not authenticated",
		},
		{
			name: "expired authentication token",
			request: &characterV1.CreateCharacterRequest{
				Name:   "ExpiredTokenHero",
				SpawnX: 100,
				SpawnY: 200,
			},
			context:    context.Background(),
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.Unauthenticated,
			wantMsg:    "user not authenticated",
		},
		{
			name: "empty character name",
			request: &characterV1.CreateCharacterRequest{
				Name:   "",
				SpawnX: 100,
				SpawnY: 200,
			},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), expectedUserID, gomock.Any()).
					Return(nil, status.Errorf(codes.InvalidArgument, "character name cannot be empty"))
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "duplicate character name",
			request: &characterV1.CreateCharacterRequest{
				Name:   "DuplicateHero",
				SpawnX: 100,
				SpawnY: 200,
			},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), expectedUserID, gomock.Any()).
					Return(nil, status.Errorf(codes.AlreadyExists, "character name already exists"))
			},
			wantErr:  true,
			wantCode: codes.AlreadyExists,
		},
		{
			name: "invalid spawn position",
			request: &characterV1.CreateCharacterRequest{
				Name:   "InvalidPositionHero",
				SpawnX: -1000,
				SpawnY: -1000,
			},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), expectedUserID, gomock.Any()).
					Return(nil, status.Errorf(codes.InvalidArgument, "invalid spawn position"))
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "service layer error",
			request: &characterV1.CreateCharacterRequest{
				Name:   "ServiceErrorHero",
				SpawnX: 100,
				SpawnY: 200,
			},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), expectedUserID, gomock.Any()).
					Return(nil, status.Errorf(codes.Internal, "database connection failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			tt.setupMocks()

			// Execute request
			resp, err := server.CreateCharacter(tt.context, tt.request)

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

// TestCharacterServiceServer_GetCharacter demonstrates testing patterns for character retrieval
func TestCharacterServiceServer_GetCharacter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *characterV1.GetCharacterRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *characterV1.GetCharacterResponse)
	}{
		{
			name: "successful character retrieval",
			request: &characterV1.GetCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					GetCharacter(gomock.Any(), &characterV1.GetCharacterRequest{
						CharacterId: testutil.UUIDTestData.Character1,
					}).
					Return(&characterV1.GetCharacterResponse{
						Character: &characterV1.Character{
							Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
							UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
							Name:     "TestHero",
							X:        100,
							Y:        200,
							ChunkX:   3,
							ChunkY:   6,
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.GetCharacterResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.Character)
				
				expectedCharacterID := strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", "")
				assert.Equal(t, expectedCharacterID, resp.Character.Id)
				assert.Equal(t, "TestHero", resp.Character.Name)
				assert.Equal(t, int32(100), resp.Character.X)
				assert.Equal(t, int32(200), resp.Character.Y)
			},
		},
		{
			name: "invalid character ID format",
			request: &characterV1.GetCharacterRequest{
				CharacterId: "invalid-uuid-format",
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					GetCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.InvalidArgument, "invalid character ID format"))
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "character not found",
			request: &characterV1.GetCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character2,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					GetCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.NotFound, "character not found"))
			},
			wantErr:  true,
			wantCode: codes.NotFound,
		},
		{
			name: "empty character ID",
			request: &characterV1.GetCharacterRequest{
				CharacterId: "",
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					GetCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.InvalidArgument, "character ID is required"))
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "service layer error",
			request: &characterV1.GetCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					GetCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.Internal, "database connection error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.GetCharacter(context.Background(), tt.request)

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

// TestCharacterServiceServer_GetMyCharacters demonstrates testing patterns for user character listing
func TestCharacterServiceServer_GetMyCharacters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *characterV1.GetMyCharactersRequest
		context    context.Context
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *characterV1.GetMyCharactersResponse)
	}{
		{
			name:    "successful character list retrieval with multiple characters",
			request: &characterV1.GetMyCharactersRequest{},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				mockCharacterService.EXPECT().
					GetUserCharacters(gomock.Any(), expectedUserID).
					Return(&characterV1.GetMyCharactersResponse{
						Characters: []*characterV1.Character{
							{
								Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
								UserId:   expectedUserID,
								Name:     "Hero1",
								X:        100,
								Y:        200,
								ChunkX:   3,
								ChunkY:   6,
								CreatedAt: timestamppb.New(time.Now()),
							},
							{
								Id:       strings.ReplaceAll(testutil.UUIDTestData.Character2, "-", ""),
								UserId:   expectedUserID,
								Name:     "Hero2",
								X:        150,
								Y:        250,
								ChunkX:   4,
								ChunkY:   7,
								CreatedAt: timestamppb.New(time.Now()),
							},
						},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.GetMyCharactersResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Characters, 2)
				
				assert.Equal(t, "Hero1", resp.Characters[0].Name)
				assert.Equal(t, "Hero2", resp.Characters[1].Name)
				assert.Equal(t, int32(100), resp.Characters[0].X)
				assert.Equal(t, int32(150), resp.Characters[1].X)
			},
		},
		{
			name:    "successful character list retrieval with empty list",
			request: &characterV1.GetMyCharactersRequest{},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User2, "-", ""), "testuser2"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User2, "-", "")
				
				mockCharacterService.EXPECT().
					GetUserCharacters(gomock.Any(), expectedUserID).
					Return(&characterV1.GetMyCharactersResponse{
						Characters: []*characterV1.Character{},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.GetMyCharactersResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Characters, 0)
			},
		},
		{
			name:       "missing authentication",
			request:    &characterV1.GetMyCharactersRequest{},
			context:    context.Background(),
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.Unauthenticated,
			wantMsg:    "user not authenticated",
		},
		{
			name:       "invalid authentication token",
			request:    &characterV1.GetMyCharactersRequest{},
			context:    context.Background(),
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.Unauthenticated,
			wantMsg:    "user not authenticated",
		},
		{
			name:       "expired authentication token",
			request:    &characterV1.GetMyCharactersRequest{},
			context:    context.Background(),
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.Unauthenticated,
			wantMsg:    "user not authenticated",
		},
		{
			name:    "service layer error",
			request: &characterV1.GetMyCharactersRequest{},
			context: middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1"),
			setupMocks: func() {
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				
				mockCharacterService.EXPECT().
					GetUserCharacters(gomock.Any(), expectedUserID).
					Return(nil, status.Errorf(codes.Internal, "database connection error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.GetMyCharacters(tt.context, tt.request)

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

// TestCharacterServiceServer_DeleteCharacter demonstrates testing patterns for character deletion
func TestCharacterServiceServer_DeleteCharacter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *characterV1.DeleteCharacterRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *characterV1.DeleteCharacterResponse)
	}{
		{
			name: "successful character deletion",
			request: &characterV1.DeleteCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					DeleteCharacter(gomock.Any(), &characterV1.DeleteCharacterRequest{
						CharacterId: testutil.UUIDTestData.Character1,
					}).
					Return(&characterV1.DeleteCharacterResponse{
						Success: true,
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.DeleteCharacterResponse) {
				require.NotNil(t, resp)
				assert.True(t, resp.Success)
			},
		},
		{
			name: "invalid character ID format",
			request: &characterV1.DeleteCharacterRequest{
				CharacterId: "invalid-uuid-format",
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					DeleteCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.InvalidArgument, "invalid character ID format"))
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "character not found",
			request: &characterV1.DeleteCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character2,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					DeleteCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.NotFound, "character not found"))
			},
			wantErr:  true,
			wantCode: codes.NotFound,
		},
		{
			name: "empty character ID",
			request: &characterV1.DeleteCharacterRequest{
				CharacterId: "",
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					DeleteCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.InvalidArgument, "character ID is required"))
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "permission denied - not character owner",
			request: &characterV1.DeleteCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					DeleteCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.PermissionDenied, "permission denied: character belongs to another user"))
			},
			wantErr:  true,
			wantCode: codes.PermissionDenied,
		},
		{
			name: "service layer error",
			request: &characterV1.DeleteCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					DeleteCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.Internal, "database connection error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.DeleteCharacter(context.Background(), tt.request)

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

// TestCharacterServiceServer_MoveCharacter demonstrates testing patterns for character movement
func TestCharacterServiceServer_MoveCharacter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *characterV1.MoveCharacterRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *characterV1.MoveCharacterResponse)
	}{
		{
			name: "successful character movement",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
				NewX:        150,
				NewY:        250,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					MoveCharacter(gomock.Any(), &characterV1.MoveCharacterRequest{
						CharacterId: testutil.UUIDTestData.Character1,
						NewX:        150,
						NewY:        250,
					}).
					Return(&characterV1.MoveCharacterResponse{
						Character: &characterV1.Character{
							Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
							UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
							Name:     "TestHero",
							X:        150,
							Y:        250,
							ChunkX:   4,
							ChunkY:   7,
							CreatedAt: timestamppb.New(time.Now()),
						},
						Success:      true,
						ErrorMessage: "",
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.MoveCharacterResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.Character)
				assert.True(t, resp.Success)
				assert.Empty(t, resp.ErrorMessage)
				assert.Equal(t, int32(150), resp.Character.X)
				assert.Equal(t, int32(250), resp.Character.Y)
				assert.Equal(t, int32(4), resp.Character.ChunkX)
				assert.Equal(t, int32(7), resp.Character.ChunkY)
			},
		},
		{
			name: "movement validation failure",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
				NewX:        -1000,
				NewY:        -1000,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					MoveCharacter(gomock.Any(), gomock.Any()).
					Return(&characterV1.MoveCharacterResponse{
						Character:    nil,
						Success:      false,
						ErrorMessage: "movement out of bounds",
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.MoveCharacterResponse) {
				require.NotNil(t, resp)
				assert.False(t, resp.Success)
				assert.Equal(t, "movement out of bounds", resp.ErrorMessage)
			},
		},
		{
			name: "movement blocked by terrain",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
				NewX:        100,
				NewY:        200,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					MoveCharacter(gomock.Any(), gomock.Any()).
					Return(&characterV1.MoveCharacterResponse{
						Character:    nil,
						Success:      false,
						ErrorMessage: "movement blocked by terrain",
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.MoveCharacterResponse) {
				require.NotNil(t, resp)
				assert.False(t, resp.Success)
				assert.Equal(t, "movement blocked by terrain", resp.ErrorMessage)
			},
		},
		{
			name: "invalid character ID format",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "invalid-uuid-format",
				NewX:        150,
				NewY:        250,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					MoveCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.InvalidArgument, "invalid character ID format"))
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "character not found",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character2,
				NewX:        150,
				NewY:        250,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					MoveCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.NotFound, "character not found"))
			},
			wantErr:  true,
			wantCode: codes.NotFound,
		},
		{
			name: "empty character ID",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: "",
				NewX:        150,
				NewY:        250,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					MoveCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.InvalidArgument, "character ID is required"))
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "movement cooldown active",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
				NewX:        150,
				NewY:        250,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					MoveCharacter(gomock.Any(), gomock.Any()).
					Return(&characterV1.MoveCharacterResponse{
						Character:    nil,
						Success:      false,
						ErrorMessage: "movement cooldown active",
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *characterV1.MoveCharacterResponse) {
				require.NotNil(t, resp)
				assert.False(t, resp.Success)
				assert.Equal(t, "movement cooldown active", resp.ErrorMessage)
			},
		},
		{
			name: "service layer error",
			request: &characterV1.MoveCharacterRequest{
				CharacterId: testutil.UUIDTestData.Character1,
				NewX:        150,
				NewY:        250,
			},
			setupMocks: func() {
				mockCharacterService.EXPECT().
					MoveCharacter(gomock.Any(), gomock.Any()).
					Return(nil, status.Errorf(codes.Internal, "database connection error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.MoveCharacter(context.Background(), tt.request)

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

// BenchmarkCharacterServiceServer_CreateCharacter measures performance of character creation
func BenchmarkCharacterServiceServer_CreateCharacter(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	// Setup mocks for all benchmark iterations
	mockCharacterService.EXPECT().
		CreateCharacter(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&characterV1.CreateCharacterResponse{
			Character: &characterV1.Character{
				Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
				UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
				Name:     "BenchHero",
				X:        100,
				Y:        200,
				ChunkX:   3,
				ChunkY:   6,
				CreatedAt: timestamppb.New(time.Now()),
			},
		}, nil).
		AnyTimes()

	request := &characterV1.CreateCharacterRequest{
		Name:   "BenchHero",
		SpawnX: 100,
		SpawnY: 200,
	}

	ctx := middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.CreateCharacter(ctx, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCharacterServiceServer_GetCharacter measures performance of character retrieval
func BenchmarkCharacterServiceServer_GetCharacter(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	// Setup mocks for all benchmark iterations
	mockCharacterService.EXPECT().
		GetCharacter(gomock.Any(), gomock.Any()).
		Return(&characterV1.GetCharacterResponse{
			Character: &characterV1.Character{
				Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
				UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
				Name:     "BenchHero",
				X:        100,
				Y:        200,
				ChunkX:   3,
				ChunkY:   6,
				CreatedAt: timestamppb.New(time.Now()),
			},
		}, nil).
		AnyTimes()

	request := &characterV1.GetCharacterRequest{
		CharacterId: testutil.UUIDTestData.Character1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.GetCharacter(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCharacterServiceServer_GetMyCharacters measures performance of user character listing
func BenchmarkCharacterServiceServer_GetMyCharacters(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	// Setup mocks for all benchmark iterations
	mockCharacterService.EXPECT().
		GetUserCharacters(gomock.Any(), gomock.Any()).
		Return(&characterV1.GetMyCharactersResponse{
			Characters: []*characterV1.Character{
				{
					Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
					UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
					Name:     "BenchHero1",
					X:        100,
					Y:        200,
					ChunkX:   3,
					ChunkY:   6,
					CreatedAt: timestamppb.New(time.Now()),
				},
				{
					Id:       strings.ReplaceAll(testutil.UUIDTestData.Character2, "-", ""),
					UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
					Name:     "BenchHero2",
					X:        150,
					Y:        250,
					ChunkX:   4,
					ChunkY:   7,
					CreatedAt: timestamppb.New(time.Now()),
				},
			},
		}, nil).
		AnyTimes()

	request := &characterV1.GetMyCharactersRequest{}
	ctx := middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.GetMyCharacters(ctx, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCharacterServiceServer_MoveCharacter measures performance of character movement
func BenchmarkCharacterServiceServer_MoveCharacter(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	// Setup mocks for all benchmark iterations
	mockCharacterService.EXPECT().
		MoveCharacter(gomock.Any(), gomock.Any()).
		Return(&characterV1.MoveCharacterResponse{
			Character: &characterV1.Character{
				Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
				UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
				Name:     "BenchHero",
				X:        150,
				Y:        250,
				ChunkX:   4,
				ChunkY:   7,
				CreatedAt: timestamppb.New(time.Now()),
			},
			Success:      true,
			ErrorMessage: "",
		}, nil).
		AnyTimes()

	request := &characterV1.MoveCharacterRequest{
		CharacterId: testutil.UUIDTestData.Character1,
		NewX:        150,
		NewY:        250,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.MoveCharacter(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestCharacterServiceServer_ProtocolBufferMessageValidation demonstrates proto message validation testing
func TestCharacterServiceServer_ProtocolBufferMessageValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	t.Run("create character response proto validation", func(t *testing.T) {
		now := time.Now()
		expectedResponse := &characterV1.CreateCharacterResponse{
			Character: &characterV1.Character{
				Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
				UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
				Name:     "ProtoTestHero",
				X:        100,
				Y:        200,
				ChunkX:   3,
				ChunkY:   6,
				CreatedAt: timestamppb.New(now),
			},
		}

		mockCharacterService.EXPECT().
			CreateCharacter(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(expectedResponse, nil)

		request := &characterV1.CreateCharacterRequest{
			Name:   "ProtoTestHero",
			SpawnX: 100,
			SpawnY: 200,
		}

		ctx := middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1")
		resp, err := server.CreateCharacter(ctx, request)

		testutil.AssertNoGRPCError(t, err)
		testutil.CompareProtoMessages(t, expectedResponse, resp)

		// Validate specific proto fields
		testutil.AssertProtoFieldEqual(t, expectedResponse.Character, resp.Character, "name")
		testutil.AssertProtoFieldEqual(t, expectedResponse.Character, resp.Character, "x")
		testutil.AssertProtoFieldEqual(t, expectedResponse.Character, resp.Character, "y")
	})

	t.Run("move character response proto validation", func(t *testing.T) {
		expectedResponse := &characterV1.MoveCharacterResponse{
			Character: &characterV1.Character{
				Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
				UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
				Name:     "MoveTestHero",
				X:        150,
				Y:        250,
				ChunkX:   4,
				ChunkY:   7,
				CreatedAt: timestamppb.New(time.Now()),
			},
			Success:      true,
			ErrorMessage: "",
		}

		mockCharacterService.EXPECT().
			MoveCharacter(gomock.Any(), gomock.Any()).
			Return(expectedResponse, nil)

		request := &characterV1.MoveCharacterRequest{
			CharacterId: testutil.UUIDTestData.Character1,
			NewX:        150,
			NewY:        250,
		}

		resp, err := server.MoveCharacter(context.Background(), request)

		testutil.AssertNoGRPCError(t, err)
		testutil.CompareProtoMessages(t, expectedResponse, resp)

		// Validate specific fields
		assert.True(t, resp.Success)
		assert.Empty(t, resp.ErrorMessage)
		testutil.AssertProtoFieldEqual(t, expectedResponse.Character, resp.Character, "x")
		testutil.AssertProtoFieldEqual(t, expectedResponse.Character, resp.Character, "y")
	})
}

// TestCharacterServiceServer_EdgeCases demonstrates edge case testing
func TestCharacterServiceServer_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCharacterService := mockhandlers.NewMockCharacterService(ctrl)

	server := &characterServiceServer{
		characterService: mockCharacterService,
		logger:           log.New(io.Discard),
	}

	tests := []struct {
		name       string
		testFunc   func(t *testing.T)
	}{
		{
			name: "maximum coordinate values",
			testFunc: func(t *testing.T) {
				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&characterV1.CreateCharacterResponse{
						Character: &characterV1.Character{
							Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
							UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
							Name:     "MaxCoordHero",
							X:        2147483647,  // max int32
							Y:        2147483647,  // max int32
							ChunkX:   67108863,    // max chunk coordinate
							ChunkY:   67108863,    // max chunk coordinate
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)

				request := &characterV1.CreateCharacterRequest{
					Name:   "MaxCoordHero",
					SpawnX: 2147483647,
					SpawnY: 2147483647,
				}

				ctx := middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1")
				resp, err := server.CreateCharacter(ctx, request)

				testutil.AssertNoGRPCError(t, err)
				assert.Equal(t, int32(2147483647), resp.Character.X)
				assert.Equal(t, int32(2147483647), resp.Character.Y)
			},
		},
		{
			name: "minimum coordinate values",
			testFunc: func(t *testing.T) {
				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&characterV1.CreateCharacterResponse{
						Character: &characterV1.Character{
							Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
							UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
							Name:     "MinCoordHero",
							X:        -2147483648, // min int32
							Y:        -2147483648, // min int32
							ChunkX:   -67108864,   // min chunk coordinate
							ChunkY:   -67108864,   // min chunk coordinate
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)

				request := &characterV1.CreateCharacterRequest{
					Name:   "MinCoordHero",
					SpawnX: -2147483648,
					SpawnY: -2147483648,
				}

				ctx := middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1")
				resp, err := server.CreateCharacter(ctx, request)

				testutil.AssertNoGRPCError(t, err)
				assert.Equal(t, int32(-2147483648), resp.Character.X)
				assert.Equal(t, int32(-2147483648), resp.Character.Y)
			},
		},
		{
			name: "very long character name",
			testFunc: func(t *testing.T) {
				longName := testutil.GenerateTestString(255) // Maximum typical name length

				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&characterV1.CreateCharacterResponse{
						Character: &characterV1.Character{
							Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
							UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
							Name:     longName,
							X:        0,
							Y:        0,
							ChunkX:   0,
							ChunkY:   0,
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)

				request := &characterV1.CreateCharacterRequest{
					Name:   longName,
					SpawnX: 0,
					SpawnY: 0,
				}

				ctx := middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1")
				resp, err := server.CreateCharacter(ctx, request)

				testutil.AssertNoGRPCError(t, err)
				assert.Equal(t, longName, resp.Character.Name)
			},
		},
		{
			name: "unicode character names",
			testFunc: func(t *testing.T) {
				unicodeName := "英雄🦸‍♂️테스트🎮"

				mockCharacterService.EXPECT().
					CreateCharacter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&characterV1.CreateCharacterResponse{
						Character: &characterV1.Character{
							Id:       strings.ReplaceAll(testutil.UUIDTestData.Character1, "-", ""),
							UserId:   strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""),
							Name:     unicodeName,
							X:        0,
							Y:        0,
							ChunkX:   0,
							ChunkY:   0,
							CreatedAt: timestamppb.New(time.Now()),
						},
					}, nil)

				request := &characterV1.CreateCharacterRequest{
					Name:   unicodeName,
					SpawnX: 0,
					SpawnY: 0,
				}

				ctx := middleware.CreateTestContextWithAuth(strings.ReplaceAll(testutil.UUIDTestData.User1, "-", ""), "testuser1")
				resp, err := server.CreateCharacter(ctx, request)

				testutil.AssertNoGRPCError(t, err)
				assert.Equal(t, unicodeName, resp.Character.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}