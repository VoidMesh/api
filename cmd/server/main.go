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
	"github.com/VoidMesh/api/internal/player"
	"github.com/charmbracelet/log"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load configuration
	cfg := config.Load()
	log.Debug("Configuration loaded", "server_port", cfg.Server.Port, "db_path", cfg.Database.Path, "log_level", cfg.Logging.Level)

	// Setup logging
	setupLogging(cfg.Logging)
	log.Debug("Logging configured", "level", cfg.Logging.Level, "format", cfg.Logging.Format)

	// Initialize database
	log.Debug("Initializing database connection", "path", cfg.Database.Path)
	db, err := initializeDatabase(cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize database", "error", err)
	}
	defer db.Close()
	log.Debug("Database connection established")

	// Run migrations
	log.Debug("Running database migrations")
	if err := runMigrations(db); err != nil {
		log.Fatal("Failed to run database migrations", "error", err)
	}
	log.Debug("Database migrations completed successfully")

	// Initialize player manager
	log.Debug("Initializing player manager")
	playerManager := player.NewManager(db)
	log.Debug("Player manager initialized")

	// Initialize chunk manager
	log.Debug("Initializing chunk manager")
	chunkManager := chunk.NewManager(db, playerManager)
	log.Debug("Chunk manager initialized")

	// Start background services
	log.Debug("Starting background services")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startBackgroundServices(ctx, chunkManager, playerManager)
	log.Debug("Background services started")

	// Initialize API handlers
	log.Debug("Initializing API handlers")
	handler := api.NewHandler(chunkManager, playerManager)
	playerHandlers := player.NewPlayerHandlers(playerManager)
	router := api.SetupRoutes(handler, playerHandlers)
	log.Debug("API routes configured")

	// Create HTTP server
	log.Debug("Creating HTTP server", "port", cfg.Server.Port, "read_timeout", cfg.Server.ReadTimeout, "write_timeout", cfg.Server.WriteTimeout, "idle_timeout", cfg.Server.IdleTimeout)
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
		log.Debug("Server listening on all interfaces", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
		log.Debug("Server stopped listening")
	}()

	// Wait for interrupt signal
	log.Debug("Server startup complete, waiting for shutdown signal")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	signal := <-quit

	log.Info("Shutting down server...", "signal", signal.String())
	log.Debug("Received shutdown signal, beginning graceful shutdown")

	// Create context for graceful shutdown
	log.Debug("Creating shutdown context", "timeout", cfg.Server.ShutdownTimeout)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown server
	log.Debug("Initiating server shutdown")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	} else {
		log.Debug("Server shutdown completed gracefully")
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
	log.Debug("Opening database connection", "path", cfg.Path)
	db, err := sql.Open("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	log.Debug("Configuring database connection pool", "max_open_conns", cfg.MaxOpenConns, "max_idle_conns", cfg.MaxIdleConns, "conn_max_lifetime", cfg.ConnMaxLifetime)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	log.Debug("Testing database connection")
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Debug("Database connection test successful")

	log.Info("Database initialized", "path", cfg.Path)
	return db, nil
}

func runMigrations(db *sql.DB) error {
	log.Debug("Creating migration driver")
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	log.Debug("Creating migration instance", "source", "./internal/db/migrations")
	m, err := migrate.NewWithDatabaseInstance(
		"file://./internal/db/migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	log.Debug("Running database migrations")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Debug("No new migrations to apply")
	} else {
		log.Debug("Successfully applied migrations")
	}

	log.Info("Database migrations completed")
	return nil
}

func startBackgroundServices(ctx context.Context, chunkManager *chunk.Manager, playerManager *player.Manager) {
	// Resource regeneration ticker (every hour)
	log.Debug("Starting resource regeneration ticker", "interval", "1h")
	regenTicker := time.NewTicker(time.Hour)
	defer regenTicker.Stop()

	// Player session cleanup ticker (every 5 minutes)
	log.Debug("Starting player session cleanup ticker", "interval", "5m")
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	log.Debug("Background services running")
	for {
		select {
		case <-ctx.Done():
			log.Info("Background services stopped")
			return

		case <-regenTicker.C:
			log.Debug("Starting resource regeneration cycle")
			start := time.Now()
			if err := chunkManager.RegenerateResources(ctx); err != nil {
				log.Error("Failed to regenerate resources", "error", err, "duration", time.Since(start))
			} else {
				log.Debug("Resources regenerated successfully", "duration", time.Since(start))
			}

		case <-cleanupTicker.C:
			log.Debug("Starting player session cleanup cycle")
			start := time.Now()

			// Cleanup player sessions only (harvest sessions removed)
			if err := playerManager.CleanupExpiredSessions(ctx); err != nil {
				log.Error("Failed to cleanup expired player sessions", "error", err, "duration", time.Since(start))
			} else {
				log.Debug("Expired player sessions cleaned up successfully", "duration", time.Since(start))
			}
		}
	}
}
