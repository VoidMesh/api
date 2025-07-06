package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"

	"github.com/VoidMesh/api/cmd/debug/models"
	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/db"
)

func main() {
	dbPath := flag.String("db", "./game.db", "Path to the SQLite database")
	startView := flag.String("view", "menu", "Starting view (menu, chunks, sessions, database, generator, overview)")
	logLevel := flag.String("log", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Setup logging
	switch *logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	// Setup file logging for debug
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	// Initialize database connection
	database, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatal("Failed to open database", "error", err, "path", *dbPath)
	}
	defer database.Close()

	// Test database connection
	if err := database.Ping(); err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}

	// Create queries and chunk manager
	queries := db.New(database)
	chunkManager := chunk.NewManager(database)

	// Initialize the main app model
	app := models.NewApp(database, queries, chunkManager, *startView)

	// Create and run the Bubble Tea program
	program := tea.NewProgram(app, tea.WithAltScreen())

	log.Info("Starting VoidMesh Debug Tool", "db_path", *dbPath, "start_view", *startView)

	if _, err := program.Run(); err != nil {
		log.Fatal("Error running debug tool", "error", err)
	}
}
