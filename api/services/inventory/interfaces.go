package inventory

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
	"github.com/VoidMesh/api/api/services/character"
	"github.com/VoidMesh/api/api/services/resource_node"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseInterface abstracts database operations for the inventory service.
type DatabaseInterface interface {
	GetCharacterInventory(ctx context.Context, characterID pgtype.UUID) ([]db.GetCharacterInventoryRow, error)
	InventoryItemExists(ctx context.Context, arg db.InventoryItemExistsParams) (bool, error)
	AddInventoryItemQuantity(ctx context.Context, arg db.AddInventoryItemQuantityParams) (db.CharacterInventory, error)
	CreateInventoryItem(ctx context.Context, arg db.CreateInventoryItemParams) (db.CharacterInventory, error)
	RemoveInventoryItemQuantity(ctx context.Context, arg db.RemoveInventoryItemQuantityParams) (db.CharacterInventory, error)
	DeleteInventoryItem(ctx context.Context, arg db.DeleteInventoryItemParams) error
	GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error)
}

// DatabaseWrapper implements DatabaseInterface using the actual database connection.
type DatabaseWrapper struct {
	queries *db.Queries
}

// NewDatabaseWrapper creates a new database wrapper with the given connection pool.
func NewDatabaseWrapper(pool *pgxpool.Pool) DatabaseInterface {
	return &DatabaseWrapper{
		queries: db.New(pool),
	}
}

func (d *DatabaseWrapper) GetCharacterInventory(ctx context.Context, characterID pgtype.UUID) ([]db.GetCharacterInventoryRow, error) {
	return d.queries.GetCharacterInventory(ctx, characterID)
}

func (d *DatabaseWrapper) InventoryItemExists(ctx context.Context, arg db.InventoryItemExistsParams) (bool, error) {
	return d.queries.InventoryItemExists(ctx, arg)
}

func (d *DatabaseWrapper) AddInventoryItemQuantity(ctx context.Context, arg db.AddInventoryItemQuantityParams) (db.CharacterInventory, error) {
	return d.queries.AddInventoryItemQuantity(ctx, arg)
}

func (d *DatabaseWrapper) CreateInventoryItem(ctx context.Context, arg db.CreateInventoryItemParams) (db.CharacterInventory, error) {
	return d.queries.CreateInventoryItem(ctx, arg)
}

func (d *DatabaseWrapper) RemoveInventoryItemQuantity(ctx context.Context, arg db.RemoveInventoryItemQuantityParams) (db.CharacterInventory, error) {
	return d.queries.RemoveInventoryItemQuantity(ctx, arg)
}

func (d *DatabaseWrapper) DeleteInventoryItem(ctx context.Context, arg db.DeleteInventoryItemParams) error {
	return d.queries.DeleteInventoryItem(ctx, arg)
}

func (d *DatabaseWrapper) GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error) {
	return d.queries.GetResourceNode(ctx, id)
}

// CharacterServiceInterface defines the interface for character service operations.
type CharacterServiceInterface interface {
	// Add any character service methods that inventory service needs
}

// ResourceNodeServiceInterface defines the interface for resource node service operations.
type ResourceNodeServiceInterface interface {
	GetResourceNodeTypes(ctx context.Context) ([]*resourceNodeV1.ResourceNodeType, error)
}

// LoggerInterface abstracts logging operations for dependency injection.
type LoggerInterface interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) LoggerInterface
}

// Adapter types to implement interfaces for existing services

// CharacterServiceAdapter adapts a character.Service to our interface.
type CharacterServiceAdapter struct {
	service *character.Service
}

func NewCharacterServiceAdapter(service *character.Service) CharacterServiceInterface {
	return &CharacterServiceAdapter{service: service}
}

// ResourceNodeServiceAdapter adapts a resource_node.NodeService to our interface.
type ResourceNodeServiceAdapter struct {
	service *resource_node.NodeService
}

func NewResourceNodeServiceAdapter(service *resource_node.NodeService) ResourceNodeServiceInterface {
	return &ResourceNodeServiceAdapter{service: service}
}

func (r *ResourceNodeServiceAdapter) GetResourceNodeTypes(ctx context.Context) ([]*resourceNodeV1.ResourceNodeType, error) {
	return r.service.GetResourceNodeTypes(ctx)
}

// DefaultLoggerWrapper wraps the internal logging package.
type DefaultLoggerWrapper struct{}

func NewDefaultLoggerWrapper() LoggerInterface {
	return &DefaultLoggerWrapper{}
}

func (l *DefaultLoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	// Implementation uses internal logging package - will be implemented when needed
}

func (l *DefaultLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	// Implementation uses internal logging package - will be implemented when needed
}

func (l *DefaultLoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	// Implementation uses internal logging package - will be implemented when needed
}

func (l *DefaultLoggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	// Implementation uses internal logging package - will be implemented when needed
}

func (l *DefaultLoggerWrapper) With(keysAndValues ...interface{}) LoggerInterface {
	return l
}