package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VoidMesh/api/internal/api"
	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/config"
	"github.com/charmbracelet/log"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logging
	setupLogging(cfg.Logging)

	// Initialize database
	db, err := initializeDatabase(cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize database", "error", err)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatal("Failed to run database migrations", "error", err)
	}

	// Initialize chunk manager
	chunkManager := chunk.NewManager(db)

	// Start background services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startBackgroundServices(ctx, chunkManager)

	// Initialize API handlers
	handler := api.NewHandler(chunkManager)
	router := api.SetupRoutes(handler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Starting VoidMesh API server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Create context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited")
}

func setupLogging(cfg config.LoggingConfig) {
	// Set log level
	switch cfg.Level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.Warn("Invalid log level, using info", "level", cfg.Level)
		log.SetLevel(log.InfoLevel)
	}

	// Configure output format
	if cfg.Format == "pretty" || !cfg.Structured {
		log.SetReportCaller(true)
		log.SetReportTimestamp(true)
	}

	// Add service info context
	log.SetPrefix("[voidmesh-api] ")
}

func initializeDatabase(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database initialized", "path", cfg.Path)
	return db, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./internal/db/migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info("Database migrations completed")
	return nil
}

func startBackgroundServices(ctx context.Context, chunkManager *chunk.Manager) {
	// Resource regeneration ticker (every hour)
	regenTicker := time.NewTicker(time.Hour)
	defer regenTicker.Stop()

	// Session cleanup ticker (every 5 minutes)
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Background services stopped")
			return

		case <-regenTicker.C:
			if err := chunkManager.RegenerateResources(ctx); err != nil {
				log.Error("Failed to regenerate resources", "error", err)
			} else {
				log.Debug("Resources regenerated")
			}

		case <-cleanupTicker.C:
			if err := chunkManager.CleanupExpiredSessions(ctx); err != nil {
				log.Error("Failed to cleanup expired sessions", "error", err)
			} else {
				log.Debug("Expired sessions cleaned up")
			}
		}
	}
}
