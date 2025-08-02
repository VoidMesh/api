package logging

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

var Logger *log.Logger

// LogLevel represents available log levels
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
)

// InitLogger initializes the global logger with configuration from environment variables
func InitLogger() {
	// Create new logger instance
	Logger = log.New(os.Stderr)

	// Set log level from environment variable (default: debug)
	logLevel := getLogLevelFromEnv()
	setLogLevel(Logger, logLevel)

	// Configure logger appearance
	Logger.SetReportTimestamp(true)
	Logger.SetReportCaller(true)

	Logger.Debug("Logger initialized successfully", "level", logLevel)
}

// getLogLevelFromEnv reads log level from LOG_LEVEL environment variable
func getLogLevelFromEnv() LogLevel {
	envLevel := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))

	switch envLevel {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		// Default to debug level for maximum visibility
		return DebugLevel
	}
}

// setLogLevel configures the logger with the specified level
func setLogLevel(logger *log.Logger, level LogLevel) {
	switch level {
	case DebugLevel:
		logger.SetLevel(log.DebugLevel)
	case InfoLevel:
		logger.SetLevel(log.InfoLevel)
	case WarnLevel:
		logger.SetLevel(log.WarnLevel)
	case ErrorLevel:
		logger.SetLevel(log.ErrorLevel)
	default:
		logger.SetLevel(log.DebugLevel)
	}
}

// GetLogger returns the global logger instance
func GetLogger() *log.Logger {
	if Logger == nil {
		InitLogger()
	}
	return Logger
}

// WithFields creates a logger with contextual fields
func WithFields(fields ...interface{}) *log.Logger {
	return GetLogger().With(fields...)
}

// WithUserID creates a logger with user_id context
func WithUserID(userID string) *log.Logger {
	return WithFields("user_id", userID)
}

// WithCharacterID creates a logger with character_id context
func WithCharacterID(characterID string) *log.Logger {
	return WithFields("character_id", characterID)
}

// WithCoords creates a logger with coordinate context
func WithCoords(x, y int32) *log.Logger {
	return WithFields("x", x, "y", y)
}

// WithChunkCoords creates a logger with chunk coordinate context
func WithChunkCoords(chunkX, chunkY int32) *log.Logger {
	return WithFields("chunk_x", chunkX, "chunk_y", chunkY)
}

// WithDuration creates a logger with duration context (for performance logging)
func WithDuration(operation string, duration interface{}) *log.Logger {
	return WithFields("operation", operation, "duration", duration)
}
