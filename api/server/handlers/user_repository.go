package handlers

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// userRepository implements the UserRepository interface using the real database
type userRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository implementation
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{
		db: db,
	}
}

// CreateUser creates a new user in the database
func (r *userRepository) CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	return db.New(r.db).CreateUser(ctx, params)
}

// GetUserById retrieves a user by their ID
func (r *userRepository) GetUserById(ctx context.Context, id pgtype.UUID) (db.User, error) {
	return db.New(r.db).GetUserById(ctx, id)
}

// GetUserByEmail retrieves a user by their email address
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return db.New(r.db).GetUserByEmail(ctx, email)
}

// GetUserByUsername retrieves a user by their username
func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (db.User, error) {
	return db.New(r.db).GetUserByUsername(ctx, username)
}

// GetUserByResetToken retrieves a user by their password reset token
func (r *userRepository) GetUserByResetToken(ctx context.Context, resetToken pgtype.Text) (db.User, error) {
	return db.New(r.db).GetUserByResetToken(ctx, resetToken)
}

// UpdateUser updates user information
func (r *userRepository) UpdateUser(ctx context.Context, params db.UpdateUserParams) (db.User, error) {
	return db.New(r.db).UpdateUser(ctx, params)
}

// UpdateLastLoginAt updates the user's last login timestamp
func (r *userRepository) UpdateLastLoginAt(ctx context.Context, params db.UpdateLastLoginAtParams) (db.User, error) {
	return db.New(r.db).UpdateLastLoginAt(ctx, params)
}

// UpdateLoginAttempts updates failed login attempts and account lock status
func (r *userRepository) UpdateLoginAttempts(ctx context.Context, params db.UpdateLoginAttemptsParams) (db.User, error) {
	return db.New(r.db).UpdateLoginAttempts(ctx, params)
}

// UpdatePasswordResetToken updates or clears password reset token and expiration
func (r *userRepository) UpdatePasswordResetToken(ctx context.Context, params db.UpdatePasswordResetTokenParams) (db.User, error) {
	return db.New(r.db).UpdatePasswordResetToken(ctx, params)
}

// VerifyEmail marks a user's email as verified
func (r *userRepository) VerifyEmail(ctx context.Context, id pgtype.UUID) (db.User, error) {
	return db.New(r.db).VerifyEmail(ctx, id)
}

// DeleteUser removes a user from the database
func (r *userRepository) DeleteUser(ctx context.Context, id pgtype.UUID) error {
	return db.New(r.db).DeleteUser(ctx, id)
}

// IndexUsers lists users with pagination
func (r *userRepository) IndexUsers(ctx context.Context, params db.IndexUsersParams) ([]db.User, error) {
	return db.New(r.db).IndexUsers(ctx, params)
}