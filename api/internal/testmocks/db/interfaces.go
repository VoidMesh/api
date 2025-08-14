// Package mockdb provides mock implementations of database interfaces for testing
package mockdb

import (
    "context"
    
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -source=interfaces.go -destination=mock_interfaces.go -package=mockdb

// Simple database interface for mocking without complex type dependencies
// These interfaces will be refined as needed for specific tests

// QuerierInterface represents a basic database querier interface
type QuerierInterface interface {
    // User operations - simplified signatures
    CreateUser(ctx context.Context, username, displayName, email, passwordHash string) (interface{}, error)
    GetUserByEmail(ctx context.Context, email string) (interface{}, error)
    GetUserByID(ctx context.Context, id string) (interface{}, error)
    GetUserByUsername(ctx context.Context, username string) (interface{}, error)
    
    // Character operations - simplified signatures
    CreateCharacter(ctx context.Context, userID, name string, x, y, chunkX, chunkY int32) (interface{}, error)
    GetCharacter(ctx context.Context, id string) (interface{}, error)
    GetCharactersByUserID(ctx context.Context, userID string) ([]interface{}, error)
    UpdateCharacterPosition(ctx context.Context, id string, x, y, chunkX, chunkY int32) (interface{}, error)
}

// PoolInterface represents the pgxpool.Pool interface for mocking
type PoolInterface interface {
    Acquire(ctx context.Context) (*pgxpool.Conn, error)
    Begin(ctx context.Context) (pgx.Tx, error)
    BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
    Close()
    Config() *pgxpool.Config
    Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
    Ping(ctx context.Context) error
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    Stat() *pgxpool.Stat
}
