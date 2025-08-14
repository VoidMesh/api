package character

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseInterface abstracts database operations for the character service.
// This enables dependency injection and makes the service easily testable.
type DatabaseInterface interface {
	GetCharacterById(ctx context.Context, id pgtype.UUID) (db.Character, error)
	CreateCharacter(ctx context.Context, arg db.CreateCharacterParams) (db.Character, error)
	UpdateCharacterPosition(ctx context.Context, arg db.UpdateCharacterPositionParams) (db.Character, error)
	GetCharactersByUser(ctx context.Context, userID pgtype.UUID) ([]db.Character, error)
	GetCharacterByUserAndName(ctx context.Context, arg db.GetCharacterByUserAndNameParams) (db.Character, error)
	DeleteCharacter(ctx context.Context, id pgtype.UUID) error
}

// DatabaseWrapper implements DatabaseInterface using the actual database connection.
// This is the production implementation that wraps the SQLC generated queries.
type DatabaseWrapper struct {
	queries *db.Queries
}

// NewDatabaseWrapper creates a new database wrapper with the given connection pool.
func NewDatabaseWrapper(pool *pgxpool.Pool) DatabaseInterface {
	return &DatabaseWrapper{
		queries: db.New(pool),
	}
}

// GetCharacterById retrieves a character by ID.
func (d *DatabaseWrapper) GetCharacterById(ctx context.Context, id pgtype.UUID) (db.Character, error) {
	return d.queries.GetCharacterById(ctx, id)
}

// CreateCharacter creates a new character.
func (d *DatabaseWrapper) CreateCharacter(ctx context.Context, arg db.CreateCharacterParams) (db.Character, error) {
	return d.queries.CreateCharacter(ctx, arg)
}

// UpdateCharacterPosition updates a character's position.
func (d *DatabaseWrapper) UpdateCharacterPosition(ctx context.Context, arg db.UpdateCharacterPositionParams) (db.Character, error) {
	return d.queries.UpdateCharacterPosition(ctx, arg)
}

// GetCharactersByUser retrieves all characters for a user.
func (d *DatabaseWrapper) GetCharactersByUser(ctx context.Context, userID pgtype.UUID) ([]db.Character, error) {
	return d.queries.GetCharactersByUser(ctx, userID)
}

// GetCharacterByUserAndName retrieves a character by user ID and name.
func (d *DatabaseWrapper) GetCharacterByUserAndName(ctx context.Context, arg db.GetCharacterByUserAndNameParams) (db.Character, error) {
	return d.queries.GetCharacterByUserAndName(ctx, arg)
}

// DeleteCharacter deletes a character by ID.
func (d *DatabaseWrapper) DeleteCharacter(ctx context.Context, id pgtype.UUID) error {
	return d.queries.DeleteCharacter(ctx, id)
}