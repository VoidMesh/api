package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserRepository defines the interface for user database operations.
// This abstraction allows for easy testing and different implementations.
type UserRepository interface {
	// CreateUser creates a new user in the database
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error)

	// GetUserById retrieves a user by their ID
	GetUserById(ctx context.Context, id pgtype.UUID) (db.User, error)

	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (db.User, error)

	// GetUserByUsername retrieves a user by their username
	GetUserByUsername(ctx context.Context, username string) (db.User, error)

	// GetUserByResetToken retrieves a user by their password reset token
	GetUserByResetToken(ctx context.Context, resetToken pgtype.Text) (db.User, error)

	// UpdateUser updates user information
	UpdateUser(ctx context.Context, params db.UpdateUserParams) (db.User, error)

	// UpdateLastLoginAt updates the user's last login timestamp
	UpdateLastLoginAt(ctx context.Context, params db.UpdateLastLoginAtParams) (db.User, error)

	// UpdateLoginAttempts updates failed login attempts and account lock status
	UpdateLoginAttempts(ctx context.Context, params db.UpdateLoginAttemptsParams) (db.User, error)

	// UpdatePasswordResetToken updates or clears password reset token and expiration
	UpdatePasswordResetToken(ctx context.Context, params db.UpdatePasswordResetTokenParams) (db.User, error)

	// VerifyEmail marks a user's email as verified
	VerifyEmail(ctx context.Context, id pgtype.UUID) (db.User, error)

	// DeleteUser removes a user from the database
	DeleteUser(ctx context.Context, id pgtype.UUID) error

	// IndexUsers lists users with pagination
	IndexUsers(ctx context.Context, params db.IndexUsersParams) ([]db.User, error)
}

// JWTService defines the interface for JWT token operations.
// This abstraction allows for easy testing and different JWT implementations.
type JWTService interface {
	// GenerateToken creates a new JWT token for the given user
	GenerateToken(userID string, username string) (string, error)

	// ValidateToken validates a JWT token and returns the claims
	// This method is not currently used in the user handler but is included
	// for completeness and future use
	ValidateToken(tokenString string) (map[string]interface{}, error)
}

// PasswordService defines the interface for password operations.
// This abstraction allows for easy testing and different hashing implementations.
type PasswordService interface {
	// HashPassword hashes a plain text password
	HashPassword(password string) (string, error)

	// CheckPassword verifies a password against its hash
	CheckPassword(password, hash string) bool
}

// TokenGenerator defines the interface for generating random tokens.
// This abstraction allows for easy testing and different token generation strategies.
type TokenGenerator interface {
	// GenerateToken generates a random token of the specified length
	GenerateToken(length int) (string, error)
}

// CharacterService defines the interface for character service operations.
// This abstraction allows for easy testing and dependency injection.
type CharacterService interface {
	// CreateCharacter creates a new character for a user
	CreateCharacter(ctx context.Context, userID string, req *characterV1.CreateCharacterRequest) (*characterV1.CreateCharacterResponse, error)

	// GetCharacter retrieves a character by ID
	GetCharacter(ctx context.Context, req *characterV1.GetCharacterRequest) (*characterV1.GetCharacterResponse, error)

	// GetUserCharacters retrieves all characters for a user
	GetUserCharacters(ctx context.Context, userID string) (*characterV1.GetMyCharactersResponse, error)

	// DeleteCharacter deletes a character
	DeleteCharacter(ctx context.Context, req *characterV1.DeleteCharacterRequest) (*characterV1.DeleteCharacterResponse, error)

	// MoveCharacter moves a character
	MoveCharacter(ctx context.Context, req *characterV1.MoveCharacterRequest) (*characterV1.MoveCharacterResponse, error)
}

// WorldService defines the interface for world service operations.
// This abstraction allows for easy testing and dependency injection.
type WorldService interface {
	// GetWorldByID retrieves a world by ID
	GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error)

	// GetDefaultWorld gets or creates the default world
	GetDefaultWorld(ctx context.Context) (db.World, error)

	// ListWorlds retrieves all worlds
	ListWorlds(ctx context.Context) ([]db.World, error)

	// UpdateWorld updates a world's name
	UpdateWorld(ctx context.Context, id pgtype.UUID, name string) (db.World, error)

	// DeleteWorld deletes a world
	DeleteWorld(ctx context.Context, id pgtype.UUID) error
}