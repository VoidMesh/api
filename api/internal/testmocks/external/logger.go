// Package mockexternal provides mocks for external dependencies
package mockexternal

import (
    "context"
    "io"
    "time"
)

//go:generate mockgen -source=logger.go -destination=mock_logger.go -package=mockexternal

// LoggerInterface represents a basic logger interface for mocking
type LoggerInterface interface {
    Debug(msg interface{}, keyvals ...interface{})
    Info(msg interface{}, keyvals ...interface{})
    Warn(msg interface{}, keyvals ...interface{})
    Error(msg interface{}, keyvals ...interface{})
    Fatal(msg interface{}, keyvals ...interface{})
    With(keyvals ...interface{}) LoggerInterface
    SetOutput(w io.Writer)
}

// ContextInterface represents context operations for mocking
type ContextInterface interface {
    WithValue(parent context.Context, key, val interface{}) context.Context
    WithCancel(parent context.Context) (context.Context, context.CancelFunc)
    WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
}
