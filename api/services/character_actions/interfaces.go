package character_actions

import (
	"context"

	"github.com/VoidMesh/api/api/db"
	inventoryV1 "github.com/VoidMesh/api/api/proto/inventory/v1"
	resourceNodeV1 "github.com/VoidMesh/api/api/proto/resource_node/v1"
)

// DatabaseInterface defines the database operations needed by the character actions service.
type DatabaseInterface interface {
	GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error)
}

// InventoryServiceInterface defines the inventory operations needed.
type InventoryServiceInterface interface {
	AddInventoryItem(ctx context.Context, characterID string, resourceNodeTypeID resourceNodeV1.ResourceNodeTypeId, quantity int32) (*inventoryV1.InventoryItem, error)
}

// CharacterServiceInterface defines the character operations needed.
type CharacterServiceInterface interface {
	GetCharacterByID(ctx context.Context, characterID string) (*db.Character, error)
	// In the future: ValidateCharacterPosition, CheckCharacterPermissions, etc.
}

// ResourceNodeServiceInterface defines the resource node operations needed.
type ResourceNodeServiceInterface interface {
	GetResourceNodeTypes(ctx context.Context) ([]*resourceNodeV1.ResourceNodeType, error)
}

// LoggerInterface defines the logging operations.
type LoggerInterface interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	With(key string, value interface{}) LoggerInterface
}

// Concrete adapter implementations
type DatabaseWrapper struct {
	queries *db.Queries
}

func NewDatabaseWrapper(queries *db.Queries) *DatabaseWrapper {
	return &DatabaseWrapper{queries: queries}
}

func (d *DatabaseWrapper) GetResourceNode(ctx context.Context, id int32) (db.ResourceNode, error) {
	return d.queries.GetResourceNode(ctx, id)
}

// InventoryServiceAdapter adapts the inventory service to our interface
type InventoryServiceAdapter struct {
	service InventoryServiceInterface
}

func NewInventoryServiceAdapter(service InventoryServiceInterface) *InventoryServiceAdapter {
	return &InventoryServiceAdapter{service: service}
}

func (a *InventoryServiceAdapter) AddInventoryItem(ctx context.Context, characterID string, resourceNodeTypeID resourceNodeV1.ResourceNodeTypeId, quantity int32) (*inventoryV1.InventoryItem, error) {
	return a.service.AddInventoryItem(ctx, characterID, resourceNodeTypeID, quantity)
}

// CharacterServiceAdapter adapts the character service to our interface
type CharacterServiceAdapter struct {
	service CharacterServiceInterface
}

func NewCharacterServiceAdapter(service CharacterServiceInterface) *CharacterServiceAdapter {
	return &CharacterServiceAdapter{service: service}
}

func (a *CharacterServiceAdapter) GetCharacterByID(ctx context.Context, characterID string) (*db.Character, error) {
	return a.service.GetCharacterByID(ctx, characterID)
}

// ResourceNodeServiceAdapter adapts the resource node service to our interface
type ResourceNodeServiceAdapter struct {
	service ResourceNodeServiceInterface
}

func NewResourceNodeServiceAdapter(service ResourceNodeServiceInterface) *ResourceNodeServiceAdapter {
	return &ResourceNodeServiceAdapter{service: service}
}

func (a *ResourceNodeServiceAdapter) GetResourceNodeTypes(ctx context.Context) ([]*resourceNodeV1.ResourceNodeType, error) {
	return a.service.GetResourceNodeTypes(ctx)
}

// Default logger wrapper
type DefaultLoggerWrapper struct{}

func NewDefaultLoggerWrapper() *DefaultLoggerWrapper {
	return &DefaultLoggerWrapper{}
}

func (l *DefaultLoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	// In production, wire this to your actual logger
}

func (l *DefaultLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	// In production, wire this to your actual logger
}

func (l *DefaultLoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	// In production, wire this to your actual logger
}

func (l *DefaultLoggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	// In production, wire this to your actual logger
}

func (l *DefaultLoggerWrapper) With(key string, value interface{}) LoggerInterface {
	return l
}