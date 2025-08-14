package world

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseInterface abstracts database operations for the world service.
// This enables dependency injection and makes the service easily testable.
type DatabaseInterface interface {
	GetDefaultWorld(ctx context.Context) (db.World, error)
	GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error)
	ListWorlds(ctx context.Context) ([]db.World, error)
	CreateWorld(ctx context.Context, arg db.CreateWorldParams) (db.World, error)
	UpdateWorld(ctx context.Context, arg db.UpdateWorldParams) (db.World, error)
	DeleteWorld(ctx context.Context, id pgtype.UUID) error
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

// GetDefaultWorld retrieves the default world.
func (d *DatabaseWrapper) GetDefaultWorld(ctx context.Context) (db.World, error) {
	return d.queries.GetDefaultWorld(ctx)
}

// GetWorldByID retrieves a world by ID.
func (d *DatabaseWrapper) GetWorldByID(ctx context.Context, id pgtype.UUID) (db.World, error) {
	return d.queries.GetWorldByID(ctx, id)
}

// ListWorlds retrieves all worlds.
func (d *DatabaseWrapper) ListWorlds(ctx context.Context) ([]db.World, error) {
	return d.queries.ListWorlds(ctx)
}

// CreateWorld creates a new world.
func (d *DatabaseWrapper) CreateWorld(ctx context.Context, arg db.CreateWorldParams) (db.World, error) {
	return d.queries.CreateWorld(ctx, arg)
}

// UpdateWorld updates a world's information.
func (d *DatabaseWrapper) UpdateWorld(ctx context.Context, arg db.UpdateWorldParams) (db.World, error) {
	return d.queries.UpdateWorld(ctx, arg)
}

// DeleteWorld deletes a world by ID.
func (d *DatabaseWrapper) DeleteWorld(ctx context.Context, id pgtype.UUID) error {
	return d.queries.DeleteWorld(ctx, id)
}