package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/VoidMesh/api/api/db"
	"github.com/VoidMesh/api/api/internal/logging"
	userV1 "github.com/VoidMesh/api/api/proto/user/v1"
	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type userServiceServer struct {
	userV1.UnimplementedUserServiceServer
	db     *pgxpool.Pool
	logger *log.Logger
}

func NewUserServer(db *pgxpool.Pool) userV1.UserServiceServer {
	logger := logging.WithComponent("user-handler")
	logger.Debug("Creating new UserService server instance")
	return &userServiceServer{db: db, logger: logger}
}

// Helper function to convert DB user to proto user
func (s *userServiceServer) dbUserToProto(user db.User) *userV1.User {
	protoUser := &userV1.User{
		Id:                  hex.EncodeToString(user.ID.Bytes[:]),
		Username:            user.Username,
		DisplayName:         user.DisplayName,
		Email:               user.Email,
		EmailVerified:       user.EmailVerified.Bool,
		FailedLoginAttempts: user.FailedLoginAttempts.Int32,
		AccountLocked:       user.AccountLocked.Bool,
	}

	if user.CreatedAt.Valid {
		protoUser.CreatedAt = timestamppb.New(user.CreatedAt.Time)
	}

	if user.LastLoginAt.Valid {
		protoUser.LastLoginAt = timestamppb.New(user.LastLoginAt.Time)
	}

	return protoUser
}

// Helper function to hash password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Helper function to check password
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Helper function to generate random token
// JWT secret key - loaded from environment variables with validation
var jwtSecret = []byte(getJWTSecret())

// getJWTSecret retrieves and validates the JWT secret from environment variables
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")

	// In production, JWT_SECRET must be set
	if secret == "" {
		// Check if we're in development mode
		if os.Getenv("ENVIRONMENT") == "development" || os.Getenv("GO_ENV") == "development" {
			fmt.Println("WARNING: Using default JWT secret for development. Set JWT_SECRET environment variable for production.")
			return "dev-jwt-secret-change-for-production-use-minimum-32-chars"
		}

		// In production, fail fast if JWT_SECRET is not set
		logger := logging.GetLogger()
		logger.Fatal("SECURITY ERROR: JWT_SECRET environment variable is required in production. " +
			"Please set a secure secret key with at least 32 characters.")
	}

	// Validate secret length for security
	if len(secret) < 32 {
		logger := logging.GetLogger()
		logger.Fatal("SECURITY ERROR: JWT_SECRET must be at least 32 characters long for adequate security", "current_length", len(secret), "required_length", 32)
	}

	// Validate secret complexity (basic check)
	if !isSecretComplex(secret) {
		fmt.Println("WARNING: JWT_SECRET should contain a mix of uppercase, lowercase, numbers, and special characters for better security.")
	}

	return secret
}

// isSecretComplex performs basic complexity validation on the JWT secret
func isSecretComplex(secret string) bool {
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range secret {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		default:
			// Consider any non-alphanumeric character as special
			hasSpecial = true
		}
	}

	// Require at least 3 out of 4 character types
	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasDigit {
		count++
	}
	if hasSpecial {
		count++
	}

	return count >= 3
}

// generateJWTToken creates a JWT token for the given user
func generateJWTToken(userID string, username string) (string, error) {
	// Create the claims
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // Token expires in 7 days
		"iat":      time.Now().Unix(),
		"iss":      "voidmesh-api",
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// generateToken generates a random token for password reset (keeping for backward compatibility)
func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Helper function to parse UUID string to pgtype.UUID
func parseUUID(uuidStr string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	err := uuid.Scan(uuidStr)
	return uuid, err
}

// CreateUser creates a new user
func (s *userServiceServer) CreateUser(ctx context.Context, req *userV1.CreateUserRequest) (*userV1.CreateUserResponse, error) {
	logger := logging.WithFields("operation", "CreateUser", "username", req.Username, "email", req.Email)
	logger.Debug("Starting user creation process")

	start := time.Now()
	// Hash the password
	logger.Debug("Hashing user password")
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		logger.Error("Failed to hash password", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}
	logger.Debug("Password hashed successfully")

	// Create user in database
	logger.Debug("Creating user record in database")
	user, err := db.New(s.db).CreateUser(ctx, db.CreateUserParams{
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "username") {
				logger.Warn("User creation failed: username already exists", "username", req.Username)
				return nil, status.Errorf(codes.AlreadyExists, "username already exists")
			}
			if strings.Contains(err.Error(), "email") {
				logger.Warn("User creation failed: email already exists", "email", req.Email)
				return nil, status.Errorf(codes.AlreadyExists, "email already exists")
			}
		}
		logger.Error("Failed to create user in database", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	duration := time.Since(start)
	userID := hex.EncodeToString(user.ID.Bytes[:])
	logger.Info("User created successfully", "user_id", userID, "duration", duration)

	return &userV1.CreateUserResponse{
		User: s.dbUserToProto(user),
	}, nil
}

// GetUser gets a user by ID
func (s *userServiceServer) GetUser(ctx context.Context, req *userV1.GetUserRequest) (*userV1.GetUserResponse, error) {
	logger := s.logger.With("operation", "GetUser", "user_id_request", req.Id)
	logger.Debug("Received GetUser request")

	uuid, err := parseUUID(req.Id)
	if err != nil {
		logger.Warn("Invalid user ID format", "user_id_request", req.Id, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	user, err := db.New(s.db).GetUserById(ctx, uuid)
	if err != nil {
		logger.Warn("User not found", "user_id", req.Id, "error", err)
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	logger.Info("User retrieved successfully", "user_id", req.Id, "username", user.Username)
	return &userV1.GetUserResponse{
		User: s.dbUserToProto(user),
	}, nil
}

// GetUserByEmail gets a user by email
func (s *userServiceServer) GetUserByEmail(ctx context.Context, req *userV1.GetUserByEmailRequest) (*userV1.GetUserByEmailResponse, error) {
	logger := s.logger.With("operation", "GetUserByEmail", "email", req.Email)
	logger.Debug("Received GetUserByEmail request")

	user, err := db.New(s.db).GetUserByEmail(ctx, req.Email)
	if err != nil {
		logger.Warn("User not found by email", "email", req.Email, "error", err)
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	logger.Info("User retrieved successfully by email", "user_id", hex.EncodeToString(user.ID.Bytes[:]), "username", user.Username)
	return &userV1.GetUserByEmailResponse{
		User: s.dbUserToProto(user),
	}, nil
}

// GetUserByUsername gets a user by username
func (s *userServiceServer) GetUserByUsername(ctx context.Context, req *userV1.GetUserByUsernameRequest) (*userV1.GetUserByUsernameResponse, error) {
	logger := s.logger.With("operation", "GetUserByUsername", "username", req.Username)
	logger.Debug("Received GetUserByUsername request")

	user, err := db.New(s.db).GetUserByUsername(ctx, req.Username)
	if err != nil {
		logger.Warn("User not found by username", "username", req.Username, "error", err)
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	logger.Info("User retrieved successfully by username", "user_id", hex.EncodeToString(user.ID.Bytes[:]), "username", user.Username)
	return &userV1.GetUserByUsernameResponse{
		User: s.dbUserToProto(user),
	}, nil
}

// UpdateUser updates a user
func (s *userServiceServer) UpdateUser(ctx context.Context, req *userV1.UpdateUserRequest) (*userV1.UpdateUserResponse, error) {
	logger := s.logger.With("operation", "UpdateUser", "user_id_request", req.Id)
	logger.Debug("Received UpdateUser request")

	uuid, err := parseUUID(req.Id)
	if err != nil {
		logger.Warn("Invalid user ID format", "user_id_request", req.Id, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	// Prepare update parameters
	updateParams := db.UpdateUserParams{
		ID: uuid,
	}

	// Handle optional fields
	if req.DisplayName != nil {
		updateParams.DisplayName = req.DisplayName.Value
		logger.Debug("Updating display name", "new_display_name", req.DisplayName.Value)
	}

	if req.Email != nil {
		updateParams.Email = req.Email.Value
		logger.Debug("Updating email", "new_email", req.Email.Value)
	}

	if req.EmailVerified != nil {
		updateParams.EmailVerified = pgtype.Bool{Bool: req.EmailVerified.Value, Valid: true}
		logger.Debug("Updating email verified status", "email_verified", req.EmailVerified.Value)
	}

	if req.Password != nil {
		logger.Debug("Updating password")
		hashedPassword, err := hashPassword(req.Password.Value)
		if err != nil {
			logger.Error("Failed to hash new password", "error", err)
			return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
		}
		updateParams.PasswordHash = hashedPassword
		logger.Debug("Password hashed successfully")
	}

	user, err := db.New(s.db).UpdateUser(ctx, updateParams)
	if err != nil {
		logger.Error("Failed to update user in database", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	logger.Info("User updated successfully", "user_id", req.Id)
	return &userV1.UpdateUserResponse{
		User: s.dbUserToProto(user),
	}, nil
}

// DeleteUser deletes a user
func (s *userServiceServer) DeleteUser(ctx context.Context, req *userV1.DeleteUserRequest) (*userV1.DeleteUserResponse, error) {
	logger := s.logger.With("operation", "DeleteUser", "user_id_request", req.Id)
	logger.Debug("Received DeleteUser request")

	uuid, err := parseUUID(req.Id)
	if err != nil {
		logger.Warn("Invalid user ID format", "user_id_request", req.Id, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	logger.Debug("Deleting user from database")
	err = db.New(s.db).DeleteUser(ctx, uuid)
	if err != nil {
		logger.Error("Failed to delete user from database", "user_id", req.Id, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	logger.Info("User deleted successfully", "user_id", req.Id)
	return &userV1.DeleteUserResponse{},
		nil
}

// ListUsers lists users with pagination
func (s *userServiceServer) ListUsers(ctx context.Context, req *userV1.ListUsersRequest) (*userV1.ListUsersResponse, error) {
	logger := s.logger.With("operation", "ListUsers", "limit", req.Limit, "offset", req.Offset)
	logger.Debug("Received ListUsers request")

	limit := req.Limit
	if limit <= 0 {
		limit = 50 // Default limit
		logger.Debug("Adjusted limit to default", "limit", limit)
	}
	if limit > 100 {
		limit = 100 // Max limit
		logger.Debug("Adjusted limit to max", "limit", limit)
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
		logger.Debug("Adjusted offset to minimum", "offset", offset)
	}

	logger.Debug("Fetching users from database", "actual_limit", limit, "actual_offset", offset)
	users, err := db.New(s.db).IndexUsers(ctx, db.IndexUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		logger.Error("Failed to list users from database", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	var protoUsers []*userV1.User
	for _, user := range users {
		protoUsers = append(protoUsers, s.dbUserToProto(user))
	}

	logger.Info("Successfully listed users", "count", len(protoUsers))
	return &userV1.ListUsersResponse{
		Users: protoUsers,
	}, nil
}

// Login authenticates a user
func (s *userServiceServer) Login(ctx context.Context, req *userV1.LoginRequest) (*userV1.LoginResponse, error) {
	logger := logging.WithFields("operation", "Login", "username_or_email", req.UsernameOrEmail)
	logger.Debug("Starting user authentication process")

	start := time.Now()
	var user db.User
	var err error

	// Try to find user by email or username
	if strings.Contains(req.UsernameOrEmail, "@") {
		logger.Debug("Looking up user by email")
		user, err = db.New(s.db).GetUserByEmail(ctx, req.UsernameOrEmail)
	} else {
		logger.Debug("Looking up user by username")
		user, err = db.New(s.db).GetUserByUsername(ctx, req.UsernameOrEmail)
	}

	if err != nil {
		logger.Warn("Authentication failed: user not found", "duration", time.Since(start))
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	userID := hex.EncodeToString(user.ID.Bytes[:])
	loggerWithUser := logger.With("user_id", userID, "username", user.Username)
	loggerWithUser.Debug("User found, validating credentials")

	// Check if account is locked
	if user.AccountLocked.Bool {
		loggerWithUser.Warn("Authentication failed: account is locked", "failed_attempts", user.FailedLoginAttempts.Int32)
		return nil, status.Errorf(codes.PermissionDenied, "account is locked")
	}

	// Check password
	loggerWithUser.Debug("Validating password hash")
	if !checkPasswordHash(req.Password, user.PasswordHash) {
		// Increment failed login attempts
		attempts := user.FailedLoginAttempts.Int32 + 1
		locked := attempts >= 5

		loggerWithUser.Warn("Authentication failed: invalid password",
			"attempts", attempts, "will_lock", locked, "duration", time.Since(start))

		_, err = db.New(s.db).UpdateLoginAttempts(ctx, db.UpdateLoginAttemptsParams{
			ID:                  user.ID,
			FailedLoginAttempts: pgtype.Int4{Int32: attempts, Valid: true},
			AccountLocked:       pgtype.Bool{Bool: locked, Valid: true},
		})
		if err != nil {
			loggerWithUser.Error("Failed to update login attempts", "error", err)
		}

		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	// Reset failed login attempts and update last login
	loggerWithUser.Debug("Resetting failed login attempts")
	_, err = db.New(s.db).UpdateLoginAttempts(ctx, db.UpdateLoginAttemptsParams{
		ID:                  user.ID,
		FailedLoginAttempts: pgtype.Int4{Int32: 0, Valid: true},
		AccountLocked:       pgtype.Bool{Bool: false, Valid: true},
	})
	if err != nil {
		loggerWithUser.Error("Failed to reset login attempts", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to update login attempts: %v", err)
	}

	// Update last login time
	loggerWithUser.Debug("Updating last login timestamp")
	_, err = db.New(s.db).UpdateLastLoginAt(ctx, db.UpdateLastLoginAtParams{
		ID:          user.ID,
		LastLoginAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		loggerWithUser.Error("Failed to update last login time", "error", err)
	}

	// Generate JWT token
	loggerWithUser.Debug("Generating JWT token")
	token, err := generateJWTToken(user.ID.String(), user.Username)
	if err != nil {
		loggerWithUser.Error("Failed to generate JWT token", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to generate JWT token: %v", err)
	}

	duration := time.Since(start)
	loggerWithUser.Info("User authentication successful", "duration", duration)

	return &userV1.LoginResponse{
		Token: token,
		User:  s.dbUserToProto(user),
	}, nil
}

// Logout logs out a user
func (s *userServiceServer) Logout(ctx context.Context, req *userV1.LogoutRequest) (*userV1.LogoutResponse, error) {
	logger := s.logger.With("operation", "Logout")
	logger.Debug("Received Logout request")

	// In a real implementation, you would invalidate the token
	// For now, just return success
	logger.Info("User logged out successfully (token invalidation not implemented)")
	return &userV1.LogoutResponse{
		Success: true,
	}, nil
}

// RequestPasswordReset initiates a password reset
func (s *userServiceServer) RequestPasswordReset(ctx context.Context, req *userV1.RequestPasswordResetRequest) (*userV1.RequestPasswordResetResponse, error) {
	logger := s.logger.With("operation", "RequestPasswordReset", "email", req.Email)
	logger.Debug("Received RequestPasswordReset request")

	user, err := db.New(s.db).GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Don't reveal if email exists or not for security reasons
		logger.Warn("Password reset requested for non-existent email or database error", "error", err)
		return &userV1.RequestPasswordResetResponse{
			Success: true,
		}, nil
	}

	userID := hex.EncodeToString(user.ID.Bytes[:])
	logger = logger.With("user_id", userID)

	// Generate reset token
	logger.Debug("Generating password reset token")
	token, err := generateToken(32)
	if err != nil {
		logger.Error("Failed to generate reset token", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to generate reset token: %v", err)
	}

	// Set token expiration to 1 hour
	expires := time.Now().Add(time.Hour)

	logger.Debug("Updating user with reset token and expiration")
	_, err = db.New(s.db).UpdatePasswordResetToken(ctx, db.UpdatePasswordResetTokenParams{
		ID:                   user.ID,
		ResetPasswordToken:   pgtype.Text{String: token, Valid: true},
		ResetPasswordExpires: pgtype.Timestamp{Time: expires, Valid: true},
	})
	if err != nil {
		logger.Error("Failed to save reset token to database", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to save reset token: %v", err)
	}

	// In a real implementation, send email with token
	logger.Info("Password reset token generated and saved", "token_length", len(token))
	logger.Debug("Password reset token (debug only)", "token", token) // Only show in debug mode

	return &userV1.RequestPasswordResetResponse{
		Success: true,
	}, nil
}

// ResetPassword resets a user's password using a token
func (s *userServiceServer) ResetPassword(ctx context.Context, req *userV1.ResetPasswordRequest) (*userV1.ResetPasswordResponse, error) {
	logger := s.logger.With("operation", "ResetPassword")
	logger.Debug("Received ResetPassword request")

	logger.Debug("Attempting to retrieve user by reset token")
	user, err := db.New(s.db).GetUserByResetToken(ctx, pgtype.Text{String: req.Token, Valid: true})
	if err != nil {
		logger.Warn("Invalid or expired reset token provided", "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid or expired reset token")
	}

	userID := hex.EncodeToString(user.ID.Bytes[:])
	logger = logger.With("user_id", userID)

	// Hash new password
	logger.Debug("Hashing new password")
	hashedPassword, err := hashPassword(req.NewPassword)
	if err != nil {
		logger.Error("Failed to hash new password", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}
	logger.Debug("New password hashed successfully")

	// Update password and clear reset token
	logger.Debug("Updating user password and clearing reset token in database")
	_, err = db.New(s.db).UpdateUser(ctx, db.UpdateUserParams{
		ID:           user.ID,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		logger.Error("Failed to update password in database", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to update password: %v", err)
	}

	// Clear reset token
	logger.Debug("Clearing password reset token")
	_, err = db.New(s.db).UpdatePasswordResetToken(ctx, db.UpdatePasswordResetTokenParams{
		ID:                   user.ID,
		ResetPasswordToken:   pgtype.Text{Valid: false},
		ResetPasswordExpires: pgtype.Timestamp{Valid: false},
	})
	if err != nil {
		logger.Error("Failed to clear reset token", "error", err)
	}

	logger.Info("Password reset successfully")
	return &userV1.ResetPasswordResponse{
		Success: true,
	}, nil
}

// VerifyEmail verifies a user's email
func (s *userServiceServer) VerifyEmail(ctx context.Context, req *userV1.VerifyEmailRequest) (*userV1.VerifyEmailResponse, error) {
	logger := s.logger.With("operation", "VerifyEmail", "user_id_request", req.Id)
	logger.Debug("Received VerifyEmail request")

	uuid, err := parseUUID(req.Id)
	if err != nil {
		logger.Warn("Invalid user ID format for email verification", "user_id_request", req.Id, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	logger.Debug("Attempting to verify email in database")
	_, err = db.New(s.db).VerifyEmail(ctx, uuid)
	if err != nil {
		logger.Error("Failed to verify email in database", "user_id", req.Id, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to verify email: %v", err)
	}

	logger.Info("Email verified successfully", "user_id", req.Id)
	return &userV1.VerifyEmailResponse{
		Success: true,
	}, nil
}
