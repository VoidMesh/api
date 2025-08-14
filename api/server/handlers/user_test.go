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
	userV1 "github.com/VoidMesh/api/api/proto/user/v1"
	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestUserServiceServer_CreateUser demonstrates comprehensive testing patterns for user creation
func TestUserServiceServer_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockRepo := mockhandlers.NewMockUserRepository(ctrl)
	mockJWT := mockhandlers.NewMockJWTService(ctrl)
	mockPassword := mockhandlers.NewMockPasswordService(ctrl)
	mockToken := mockhandlers.NewMockTokenGenerator(ctrl)

	// Create server instance
	server := &userServiceServer{
		userRepo:        mockRepo,
		jwtService:      mockJWT,
		passwordService: mockPassword,
		tokenGenerator:  mockToken,
		logger:          log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *userV1.CreateUserRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *userV1.CreateUserResponse)
	}{
		{
			name: "successful user creation",
			request: &userV1.CreateUserRequest{
				Username:    "testuser",
				DisplayName: "Test User",
				Email:       "test@example.com",
				Password:    "password123",
			},
			setupMocks: func() {
				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword("password123").
					Return("hashed_password", nil)

				// Mock successful user creation
				mockRepo.EXPECT().
					CreateUser(gomock.Any(), db.CreateUserParams{
						Username:     "testuser",
						DisplayName:  "Test User",
						Email:        "test@example.com",
						PasswordHash: "hashed_password",
					}).
					Return(db.User{
						ID:          testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
						Username:    "testuser",
						DisplayName: "Test User",
						Email:       "test@example.com",
						CreatedAt:   pgtype.Timestamp{Time: time.Now(), Valid: true},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *userV1.CreateUserResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.User)
				// UUID is converted to hex format without dashes
				expectedID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				assert.Equal(t, expectedID, resp.User.Id)
				assert.Equal(t, "testuser", resp.User.Username)
				assert.Equal(t, "Test User", resp.User.DisplayName)
				assert.Equal(t, "test@example.com", resp.User.Email)
				assert.NotNil(t, resp.User.CreatedAt)
			},
		},
		{
			name: "password hashing fails",
			request: &userV1.CreateUserRequest{
				Username:    "testuser",
				DisplayName: "Test User",
				Email:       "test@example.com",
				Password:    "password123",
			},
			setupMocks: func() {
				mockPassword.EXPECT().
					HashPassword("password123").
					Return("", errors.New("bcrypt error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "failed to hash password",
		},
		{
			name: "duplicate username",
			request: &userV1.CreateUserRequest{
				Username:    "duplicate",
				DisplayName: "Duplicate User",
				Email:       "duplicate@example.com",
				Password:    "password123",
			},
			setupMocks: func() {
				mockPassword.EXPECT().
					HashPassword("password123").
					Return("hashed_password", nil)

				mockRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, errors.New("duplicate key value violates unique constraint \"users_username_key\""))
			},
			wantErr:  true,
			wantCode: codes.AlreadyExists,
			wantMsg:  "username already exists",
		},
		{
			name: "duplicate email",
			request: &userV1.CreateUserRequest{
				Username:    "uniqueuser",
				DisplayName: "Unique User",
				Email:       "duplicate@example.com",
				Password:    "password123",
			},
			setupMocks: func() {
				mockPassword.EXPECT().
					HashPassword("password123").
					Return("hashed_password", nil)

				mockRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, errors.New("duplicate key value violates unique constraint \"users_email_key\""))
			},
			wantErr:  true,
			wantCode: codes.AlreadyExists,
			wantMsg:  "email already exists",
		},
		{
			name: "database error",
			request: &userV1.CreateUserRequest{
				Username:    "testuser",
				DisplayName: "Test User",
				Email:       "test@example.com",
				Password:    "password123",
			},
			setupMocks: func() {
				mockPassword.EXPECT().
					HashPassword("password123").
					Return("hashed_password", nil)

				mockRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, errors.New("connection refused"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "failed to create user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			tt.setupMocks()

			// Execute request
			resp, err := server.CreateUser(context.Background(), tt.request)

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

// TestUserServiceServer_GetUser demonstrates testing patterns for user retrieval
func TestUserServiceServer_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockhandlers.NewMockUserRepository(ctrl)
	mockJWT := mockhandlers.NewMockJWTService(ctrl)
	mockPassword := mockhandlers.NewMockPasswordService(ctrl)
	mockToken := mockhandlers.NewMockTokenGenerator(ctrl)

	server := &userServiceServer{
		userRepo:        mockRepo,
		jwtService:      mockJWT,
		passwordService: mockPassword,
		tokenGenerator:  mockToken,
		logger:          log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *userV1.GetUserRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		validate   func(t *testing.T, resp *userV1.GetUserResponse)
	}{
		{
			name: "successful user retrieval",
			request: &userV1.GetUserRequest{
				Id: testutil.UUIDTestData.User1,
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					GetUserById(gomock.Any(), testutil.ParseTestUUID(t, testutil.UUIDTestData.User1)).
					Return(db.User{
						ID:            testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
						Username:      "testuser",
						DisplayName:   "Test User",
						Email:         "test@example.com",
						EmailVerified: pgtype.Bool{Bool: true, Valid: true},
						CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *userV1.GetUserResponse) {
				require.NotNil(t, resp)
				require.NotNil(t, resp.User)
				expectedID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				assert.Equal(t, expectedID, resp.User.Id)
				assert.Equal(t, "testuser", resp.User.Username)
				assert.True(t, resp.User.EmailVerified)
			},
		},
		{
			name: "invalid UUID format",
			request: &userV1.GetUserRequest{
				Id: "invalid-uuid",
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
		},
		{
			name: "user not found",
			request: &userV1.GetUserRequest{
				Id: testutil.UUIDTestData.User2,
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					GetUserById(gomock.Any(), testutil.ParseTestUUID(t, testutil.UUIDTestData.User2)).
					Return(db.User{}, errors.New("no rows in result set"))
			},
			wantErr:  true,
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.GetUser(context.Background(), tt.request)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode)
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

// TestUserServiceServer_Login demonstrates comprehensive authentication testing
func TestUserServiceServer_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockhandlers.NewMockUserRepository(ctrl)
	mockJWT := mockhandlers.NewMockJWTService(ctrl)
	mockPassword := mockhandlers.NewMockPasswordService(ctrl)
	mockToken := mockhandlers.NewMockTokenGenerator(ctrl)

	server := &userServiceServer{
		userRepo:        mockRepo,
		jwtService:      mockJWT,
		passwordService: mockPassword,
		tokenGenerator:  mockToken,
		logger:          log.New(io.Discard),
	}

	testUser := db.User{
		ID:                  testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
		Username:            "testuser",
		Email:               "test@example.com",
		PasswordHash:        "hashed_password",
		FailedLoginAttempts: pgtype.Int4{Int32: 0, Valid: true},
		AccountLocked:       pgtype.Bool{Bool: false, Valid: true},
	}

	tests := []struct {
		name       string
		request    *userV1.LoginRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		wantMsg    string
		validate   func(t *testing.T, resp *userV1.LoginResponse)
	}{
		{
			name: "successful login with email",
			request: &userV1.LoginRequest{
				UsernameOrEmail: "test@example.com",
				Password:        "password123",
			},
			setupMocks: func() {
				// Mock user lookup by email
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), "test@example.com").
					Return(testUser, nil)

				// Mock password verification
				mockPassword.EXPECT().
					CheckPassword("password123", "hashed_password").
					Return(true)

				// Mock reset login attempts
				mockRepo.EXPECT().
					UpdateLoginAttempts(gomock.Any(), db.UpdateLoginAttemptsParams{
						ID:                  testUser.ID,
						FailedLoginAttempts: pgtype.Int4{Int32: 0, Valid: true},
						AccountLocked:       pgtype.Bool{Bool: false, Valid: true},
					}).
					Return(testUser, nil)

				// Mock update last login
				mockRepo.EXPECT().
					UpdateLastLoginAt(gomock.Any(), gomock.Any()).
					Return(testUser, nil)

				// Mock JWT generation - handler passes hex format without dashes
				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				mockJWT.EXPECT().
					GenerateToken(expectedUserID, "testuser").
					Return("jwt_token", nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *userV1.LoginResponse) {
				require.NotNil(t, resp)
				assert.Equal(t, "jwt_token", resp.Token)
				assert.NotNil(t, resp.User)
				expectedID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				assert.Equal(t, expectedID, resp.User.Id)
			},
		},
		{
			name: "successful login with username",
			request: &userV1.LoginRequest{
				UsernameOrEmail: "testuser",
				Password:        "password123",
			},
			setupMocks: func() {
				// Mock user lookup by username
				mockRepo.EXPECT().
					GetUserByUsername(gomock.Any(), "testuser").
					Return(testUser, nil)

				mockPassword.EXPECT().
					CheckPassword("password123", "hashed_password").
					Return(true)

				mockRepo.EXPECT().
					UpdateLoginAttempts(gomock.Any(), gomock.Any()).
					Return(testUser, nil)

				mockRepo.EXPECT().
					UpdateLastLoginAt(gomock.Any(), gomock.Any()).
					Return(testUser, nil)

				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				mockJWT.EXPECT().
					GenerateToken(expectedUserID, "testuser").
					Return("jwt_token", nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			request: &userV1.LoginRequest{
				UsernameOrEmail: "nonexistent@example.com",
				Password:        "password123",
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), "nonexistent@example.com").
					Return(db.User{}, errors.New("no rows in result set"))
			},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
			wantMsg:  "invalid credentials",
		},
		{
			name: "account locked",
			request: &userV1.LoginRequest{
				UsernameOrEmail: "locked@example.com",
				Password:        "password123",
			},
			setupMocks: func() {
				lockedUser := testUser
				lockedUser.AccountLocked = pgtype.Bool{Bool: true, Valid: true}
				lockedUser.FailedLoginAttempts = pgtype.Int4{Int32: 5, Valid: true}

				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), "locked@example.com").
					Return(lockedUser, nil)
			},
			wantErr:  true,
			wantCode: codes.PermissionDenied,
			wantMsg:  "account is locked",
		},
		{
			name: "invalid password increments attempts",
			request: &userV1.LoginRequest{
				UsernameOrEmail: "test@example.com",
				Password:        "wrongpassword",
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), "test@example.com").
					Return(testUser, nil)

				mockPassword.EXPECT().
					CheckPassword("wrongpassword", "hashed_password").
					Return(false)

				// Mock incrementing failed attempts
				mockRepo.EXPECT().
					UpdateLoginAttempts(gomock.Any(), db.UpdateLoginAttemptsParams{
						ID:                  testUser.ID,
						FailedLoginAttempts: pgtype.Int4{Int32: 1, Valid: true},
						AccountLocked:       pgtype.Bool{Bool: false, Valid: true},
					}).
					Return(testUser, nil)
			},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
			wantMsg:  "invalid credentials",
		},
		{
			name: "JWT generation fails",
			request: &userV1.LoginRequest{
				UsernameOrEmail: "test@example.com",
				Password:        "password123",
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), "test@example.com").
					Return(testUser, nil)

				mockPassword.EXPECT().
					CheckPassword("password123", "hashed_password").
					Return(true)

				mockRepo.EXPECT().
					UpdateLoginAttempts(gomock.Any(), gomock.Any()).
					Return(testUser, nil)

				mockRepo.EXPECT().
					UpdateLastLoginAt(gomock.Any(), gomock.Any()).
					Return(testUser, nil)

				expectedUserID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
				mockJWT.EXPECT().
					GenerateToken(expectedUserID, "testuser").
					Return("", errors.New("JWT signing error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
			wantMsg:  "failed to generate JWT token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.Login(context.Background(), tt.request)

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

// TestUserServiceServer_UpdateUser demonstrates testing patterns for user updates
func TestUserServiceServer_UpdateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockhandlers.NewMockUserRepository(ctrl)
	mockJWT := mockhandlers.NewMockJWTService(ctrl)
	mockPassword := mockhandlers.NewMockPasswordService(ctrl)
	mockToken := mockhandlers.NewMockTokenGenerator(ctrl)

	server := &userServiceServer{
		userRepo:        mockRepo,
		jwtService:      mockJWT,
		passwordService: mockPassword,
		tokenGenerator:  mockToken,
		logger:          log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *userV1.UpdateUserRequest
		setupMocks func()
		wantErr    bool
		wantCode   codes.Code
		validate   func(t *testing.T, resp *userV1.UpdateUserResponse)
	}{
		{
			name: "update display name only",
			request: &userV1.UpdateUserRequest{
				Id:          testutil.UUIDTestData.User1,
				DisplayName: wrapperspb.String("Updated Display Name"),
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					UpdateUser(gomock.Any(), db.UpdateUserParams{
						ID:          testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
						DisplayName: "Updated Display Name",
					}).
					Return(db.User{
						ID:          testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
						DisplayName: "Updated Display Name",
						Username:    "testuser",
						Email:       "test@example.com",
					}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *userV1.UpdateUserResponse) {
				require.NotNil(t, resp)
				assert.Equal(t, "Updated Display Name", resp.User.DisplayName)
			},
		},
		{
			name: "update password",
			request: &userV1.UpdateUserRequest{
				Id:       testutil.UUIDTestData.User1,
				Password: wrapperspb.String("newpassword123"),
			},
			setupMocks: func() {
				mockPassword.EXPECT().
					HashPassword("newpassword123").
					Return("new_hashed_password", nil)

				mockRepo.EXPECT().
					UpdateUser(gomock.Any(), db.UpdateUserParams{
						ID:           testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
						PasswordHash: "new_hashed_password",
					}).
					Return(db.User{
						ID:       testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
						Username: "testuser",
						Email:    "test@example.com",
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "invalid UUID format",
			request: &userV1.UpdateUserRequest{
				Id:          "invalid-uuid",
				DisplayName: wrapperspb.String("New Name"),
			},
			setupMocks: func() {},
			wantErr:    true,
			wantCode:   codes.InvalidArgument,
		},
		{
			name: "password hashing fails",
			request: &userV1.UpdateUserRequest{
				Id:       testutil.UUIDTestData.User1,
				Password: wrapperspb.String("newpassword"),
			},
			setupMocks: func() {
				mockPassword.EXPECT().
					HashPassword("newpassword").
					Return("", errors.New("bcrypt error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.UpdateUser(context.Background(), tt.request)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode)
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

// TestUserServiceServer_RequestPasswordReset demonstrates password reset flow testing
func TestUserServiceServer_RequestPasswordReset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockhandlers.NewMockUserRepository(ctrl)
	mockJWT := mockhandlers.NewMockJWTService(ctrl)
	mockPassword := mockhandlers.NewMockPasswordService(ctrl)
	mockToken := mockhandlers.NewMockTokenGenerator(ctrl)

	server := &userServiceServer{
		userRepo:        mockRepo,
		jwtService:      mockJWT,
		passwordService: mockPassword,
		tokenGenerator:  mockToken,
		logger:          log.New(io.Discard),
	}

	tests := []struct {
		name       string
		request    *userV1.RequestPasswordResetRequest
		setupMocks func()
		wantErr    bool
		validate   func(t *testing.T, resp *userV1.RequestPasswordResetResponse)
	}{
		{
			name: "successful password reset request",
			request: &userV1.RequestPasswordResetRequest{
				Email: "test@example.com",
			},
			setupMocks: func() {
				// Mock user lookup
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), "test@example.com").
					Return(db.User{
						ID:       testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
						Username: "testuser",
						Email:    "test@example.com",
					}, nil)

				// Mock token generation
				mockToken.EXPECT().
					GenerateToken(32).
					Return("reset_token_123", nil)

				// Mock saving reset token
				mockRepo.EXPECT().
					UpdatePasswordResetToken(gomock.Any(), gomock.Any()).
					Return(db.User{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *userV1.RequestPasswordResetResponse) {
				require.NotNil(t, resp)
				assert.True(t, resp.Success)
			},
		},
		{
			name: "user not found returns success for security",
			request: &userV1.RequestPasswordResetRequest{
				Email: "nonexistent@example.com",
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), "nonexistent@example.com").
					Return(db.User{}, errors.New("no rows in result set"))
			},
			wantErr: false,
			validate: func(t *testing.T, resp *userV1.RequestPasswordResetResponse) {
				require.NotNil(t, resp)
				assert.True(t, resp.Success) // Returns success even if user not found
			},
		},
		{
			name: "token generation fails",
			request: &userV1.RequestPasswordResetRequest{
				Email: "test@example.com",
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), "test@example.com").
					Return(db.User{
						ID:       testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
						Username: "testuser",
						Email:    "test@example.com",
					}, nil)

				mockToken.EXPECT().
					GenerateToken(32).
					Return("", errors.New("random generation error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.RequestPasswordReset(context.Background(), tt.request)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, codes.Internal)
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

// TestUserServiceServer_ListUsers demonstrates pagination testing
func TestUserServiceServer_ListUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockhandlers.NewMockUserRepository(ctrl)
	mockJWT := mockhandlers.NewMockJWTService(ctrl)
	mockPassword := mockhandlers.NewMockPasswordService(ctrl)
	mockToken := mockhandlers.NewMockTokenGenerator(ctrl)

	server := &userServiceServer{
		userRepo:        mockRepo,
		jwtService:      mockJWT,
		passwordService: mockPassword,
		tokenGenerator:  mockToken,
		logger:          log.New(io.Discard),
	}

	testUsers := []db.User{
		{
			ID:          testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
			Username:    "user1",
			DisplayName: "User One",
			Email:       "user1@example.com",
		},
		{
			ID:          testutil.ParseTestUUID(t, testutil.UUIDTestData.User2),
			Username:    "user2",
			DisplayName: "User Two",
			Email:       "user2@example.com",
		},
	}

	tests := []struct {
		name       string
		request    *userV1.ListUsersRequest
		setupMocks func()
		wantErr    bool
		validate   func(t *testing.T, resp *userV1.ListUsersResponse)
	}{
		{
			name: "default pagination",
			request: &userV1.ListUsersRequest{
				Limit:  0, // Should default to 50
				Offset: 0,
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					IndexUsers(gomock.Any(), db.IndexUsersParams{
						Limit:  50, // Default limit
						Offset: 0,
					}).
					Return(testUsers, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *userV1.ListUsersResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Users, 2)
				assert.Equal(t, "user1", resp.Users[0].Username)
				assert.Equal(t, "user2", resp.Users[1].Username)
			},
		},
		{
			name: "custom pagination",
			request: &userV1.ListUsersRequest{
				Limit:  10,
				Offset: 5,
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					IndexUsers(gomock.Any(), db.IndexUsersParams{
						Limit:  10,
						Offset: 5,
					}).
					Return(testUsers, nil)
			},
			wantErr: false,
		},
		{
			name: "limit exceeds maximum",
			request: &userV1.ListUsersRequest{
				Limit:  200, // Should be capped at 100
				Offset: 0,
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					IndexUsers(gomock.Any(), db.IndexUsersParams{
						Limit:  100, // Capped at max
						Offset: 0,
					}).
					Return(testUsers, nil)
			},
			wantErr: false,
		},
		{
			name: "database error",
			request: &userV1.ListUsersRequest{
				Limit:  10,
				Offset: 0,
			},
			setupMocks: func() {
				mockRepo.EXPECT().
					IndexUsers(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database connection error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			resp, err := server.ListUsers(context.Background(), tt.request)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, codes.Internal)
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

// BenchmarkUserServiceServer_CreateUser measures performance of user creation
func BenchmarkUserServiceServer_CreateUser(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockRepo := mockhandlers.NewMockUserRepository(ctrl)
	mockJWT := mockhandlers.NewMockJWTService(ctrl)
	mockPassword := mockhandlers.NewMockPasswordService(ctrl)
	mockToken := mockhandlers.NewMockTokenGenerator(ctrl)

	server := &userServiceServer{
		userRepo:        mockRepo,
		jwtService:      mockJWT,
		passwordService: mockPassword,
		tokenGenerator:  mockToken,
		logger:          log.New(io.Discard),
	}

	// Setup mocks for all benchmark iterations
	mockPassword.EXPECT().
		HashPassword(gomock.Any()).
		Return("hashed_password", nil).
		AnyTimes()

	mockRepo.EXPECT().
		CreateUser(gomock.Any(), gomock.Any()).
		Return(db.User{
			ID:          testutil.MustParseUUID(testutil.UUIDTestData.User1),
			Username:    "benchuser",
			DisplayName: "Bench User",
			Email:       "bench@example.com",
		}, nil).
		AnyTimes()

	request := &userV1.CreateUserRequest{
		Username:    "benchuser",
		DisplayName: "Bench User",
		Email:       "bench@example.com",
		Password:    "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.CreateUser(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUserServiceServer_Login measures authentication performance
func BenchmarkUserServiceServer_Login(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockRepo := mockhandlers.NewMockUserRepository(ctrl)
	mockJWT := mockhandlers.NewMockJWTService(ctrl)
	mockPassword := mockhandlers.NewMockPasswordService(ctrl)
	mockToken := mockhandlers.NewMockTokenGenerator(ctrl)

	server := &userServiceServer{
		userRepo:        mockRepo,
		jwtService:      mockJWT,
		passwordService: mockPassword,
		tokenGenerator:  mockToken,
		logger:          log.New(io.Discard),
	}

	testUser := db.User{
		ID:                  testutil.MustParseUUID(testutil.UUIDTestData.User1),
		Username:            "benchuser",
		Email:               "bench@example.com",
		PasswordHash:        "hashed_password",
		FailedLoginAttempts: pgtype.Int4{Int32: 0, Valid: true},
		AccountLocked:       pgtype.Bool{Bool: false, Valid: true},
	}

	// Setup mocks for all benchmark iterations
	mockRepo.EXPECT().
		GetUserByEmail(gomock.Any(), "bench@example.com").
		Return(testUser, nil).
		AnyTimes()

	mockPassword.EXPECT().
		CheckPassword("password123", "hashed_password").
		Return(true).
		AnyTimes()

	mockRepo.EXPECT().
		UpdateLoginAttempts(gomock.Any(), gomock.Any()).
		Return(testUser, nil).
		AnyTimes()

	mockRepo.EXPECT().
		UpdateLastLoginAt(gomock.Any(), gomock.Any()).
		Return(testUser, nil).
		AnyTimes()

	mockJWT.EXPECT().
		GenerateToken(gomock.Any(), gomock.Any()).
		Return("jwt_token", nil).
		AnyTimes()

	request := &userV1.LoginRequest{
		UsernameOrEmail: "bench@example.com",
		Password:        "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.Login(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Example test showing how to test the dbUserToProto conversion helper
func TestUserServiceServer_dbUserToProto(t *testing.T) {
	server := &userServiceServer{
		logger: log.New(io.Discard),
	}

	now := time.Now()
	dbUser := db.User{
		ID:            testutil.ParseTestUUID(t, testutil.UUIDTestData.User1),
		Username:      "testuser",
		DisplayName:   "Test User",
		Email:         "test@example.com",
		EmailVerified: pgtype.Bool{Bool: true, Valid: true},
		CreatedAt:     pgtype.Timestamp{Time: now, Valid: true},
		LastLoginAt:   pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true},
	}

	protoUser := server.dbUserToProto(dbUser)

	// Validate conversion - UUID is converted to hex format without dashes
	expectedID := strings.ReplaceAll(testutil.UUIDTestData.User1, "-", "")
	assert.Equal(t, expectedID, protoUser.Id)
	assert.Equal(t, "testuser", protoUser.Username)
	assert.Equal(t, "Test User", protoUser.DisplayName)
	assert.Equal(t, "test@example.com", protoUser.Email)
	assert.True(t, protoUser.EmailVerified)
	assert.NotNil(t, protoUser.CreatedAt)
	assert.NotNil(t, protoUser.LastLoginAt)

	// Validate timestamp conversion
	assert.Equal(t, timestamppb.New(now), protoUser.CreatedAt)
	assert.Equal(t, timestamppb.New(now.Add(-time.Hour)), protoUser.LastLoginAt)
}