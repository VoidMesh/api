package models

import (
	"database/sql"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/log"

	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/db"
	"github.com/VoidMesh/api/internal/player"
)

// ViewType represents the different views in the debug tool
type ViewType int

const (
	MenuView ViewType = iota
	ChunkExplorerView
	SessionMonitorView
	DatabaseView
	NodeGeneratorView
	OverviewView
)

// App is the main application model
type App struct {
	// Database connections
	db            *sql.DB
	queries       *db.Queries
	chunkManager  *chunk.Manager
	playerManager *player.Manager

	// Current state
	currentView ViewType
	width       int
	height      int

	// View models
	menu           MenuModel
	chunkExplorer  ChunkExplorerModel
	sessionMonitor SessionMonitorModel
	database       DatabaseModel
	nodeGenerator  NodeGeneratorModel
	overview       OverviewModel

	// UI state
	showHelp bool
}

// NewApp creates a new application instance
func NewApp(database *sql.DB, queries *db.Queries, chunkManager *chunk.Manager, playerManager *player.Manager, startView string) *App {
	app := &App{
		db:            database,
		queries:       queries,
		chunkManager:  chunkManager,
		playerManager: playerManager,
		currentView:   MenuView,
	}

	// Initialize view models
	app.menu = NewMenuModel()
	app.chunkExplorer = NewChunkExplorerModel(database, queries, chunkManager, playerManager)
	app.sessionMonitor = NewSessionMonitorModel(database, queries)
	app.database = NewDatabaseModel(database, queries)
	app.nodeGenerator = NewNodeGeneratorModel(database, queries, chunkManager)
	app.overview = NewOverviewModel(database, queries)

	// Set starting view based on parameter
	switch startView {
	case "chunks":
		app.currentView = ChunkExplorerView
	case "sessions":
		app.currentView = SessionMonitorView
	case "database":
		app.currentView = DatabaseView
	case "generator":
		app.currentView = NodeGeneratorView
	case "overview":
		app.currentView = OverviewView
	default:
		app.currentView = MenuView
	}

	return app
}

// Init initializes the application
func (m *App) Init() tea.Cmd {
	log.Debug("Initializing debug tool")

	// Initialize current view
	switch m.currentView {
	case MenuView:
		return m.menu.Init()
	case ChunkExplorerView:
		return m.chunkExplorer.Init()
	case SessionMonitorView:
		return m.sessionMonitor.Init()
	case DatabaseView:
		return m.database.Init()
	case NodeGeneratorView:
		return m.nodeGenerator.Init()
	case OverviewView:
		return m.overview.Init()
	}

	return nil
}

// Update handles messages and updates the application state
func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update all view models with new size
		m.menu.SetSize(msg.Width, msg.Height)
		m.chunkExplorer.SetSize(msg.Width, msg.Height)
		m.sessionMonitor.SetSize(msg.Width, msg.Height)
		m.database.SetSize(msg.Width, msg.Height)
		m.nodeGenerator.SetSize(msg.Width, msg.Height)
		m.overview.SetSize(msg.Width, msg.Height)

		return m, nil

	case tea.KeyMsg:
		// Global key bindings
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentView == MenuView {
				return m, tea.Quit
			}
			// If not in menu, go back to menu instead of quitting
			// Reset chunk explorer state when leaving
			if m.currentView == ChunkExplorerView {
				m.chunkExplorer.Reset()
			}
			m.currentView = MenuView
			return m, m.menu.Init()

		case "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "tab":
			// Cycle through views
			// Reset chunk explorer state when leaving
			if m.currentView == ChunkExplorerView {
				m.chunkExplorer.Reset()
			}
			m.currentView = ViewType((int(m.currentView) + 1) % 6)
			return m, m.getCurrentViewModel().Init()

		case "1":
			if m.currentView == MenuView {
				m.currentView = ChunkExplorerView
				return m, m.chunkExplorer.Init()
			}
		case "2":
			if m.currentView == MenuView {
				m.currentView = SessionMonitorView
				return m, m.sessionMonitor.Init()
			}
		case "3":
			if m.currentView == MenuView {
				m.currentView = DatabaseView
				return m, m.database.Init()
			}
		case "4":
			if m.currentView == MenuView {
				m.currentView = NodeGeneratorView
				return m, m.nodeGenerator.Init()
			}
		case "5":
			if m.currentView == MenuView {
				m.currentView = OverviewView
				return m, m.overview.Init()
			}
		}

	case SwitchViewMsg:
		// Reset chunk explorer state when leaving
		if m.currentView == ChunkExplorerView {
			m.chunkExplorer.Reset()
		}
		m.currentView = msg.View
		return m, m.getCurrentViewModel().Init()
	}

	// Handle help view
	if m.showHelp {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "?" {
			m.showHelp = false
		}
		return m, nil
	}

	// Route message to current view
	switch m.currentView {
	case MenuView:
		newModel, cmd := m.menu.Update(msg)
		m.menu = newModel.(MenuModel)
		return m, cmd
	case ChunkExplorerView:
		newModel, cmd := m.chunkExplorer.Update(msg)
		m.chunkExplorer = newModel.(ChunkExplorerModel)
		return m, cmd
	case SessionMonitorView:
		newModel, cmd := m.sessionMonitor.Update(msg)
		m.sessionMonitor = newModel.(SessionMonitorModel)
		return m, cmd
	case DatabaseView:
		newModel, cmd := m.database.Update(msg)
		m.database = newModel.(DatabaseModel)
		return m, cmd
	case NodeGeneratorView:
		newModel, cmd := m.nodeGenerator.Update(msg)
		m.nodeGenerator = newModel.(NodeGeneratorModel)
		return m, cmd
	case OverviewView:
		newModel, cmd := m.overview.Update(msg)
		m.overview = newModel.(OverviewModel)
		return m, cmd
	}

	return m, cmd
}

// View renders the application
func (m *App) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	switch m.currentView {
	case MenuView:
		return m.menu.View()
	case ChunkExplorerView:
		return m.chunkExplorer.View()
	case SessionMonitorView:
		return m.sessionMonitor.View()
	case DatabaseView:
		return m.database.View()
	case NodeGeneratorView:
		return m.nodeGenerator.View()
	case OverviewView:
		return m.overview.View()
	}

	return "Unknown view"
}

// getCurrentViewModel returns the current view's model
func (m *App) getCurrentViewModel() tea.Model {
	switch m.currentView {
	case MenuView:
		return &m.menu
	case ChunkExplorerView:
		return &m.chunkExplorer
	case SessionMonitorView:
		return &m.sessionMonitor
	case DatabaseView:
		return &m.database
	case NodeGeneratorView:
		return &m.nodeGenerator
	case OverviewView:
		return &m.overview
	}
	return &m.menu
}

// renderHelp renders the help screen
func (m *App) renderHelp() string {
	help := `
┌─ VoidMesh Debug Tool - Help ─────────────────────────┐
│                                                      │
│ Global Keys:                                         │
│   q, Ctrl+C    Quit (from menu) / Back to menu      │
│   ?            Toggle this help                      │
│   Tab          Cycle through views                   │
│   1-5          Select view (from menu)               │
│                                                      │
│ Views:                                               │
│   1. 🗺️  Chunk Explorer  - Visualize world chunks   │
│   2. 👥 Session Monitor  - Monitor harvest sessions │
│   3. 🗄️  Database        - Query and analyze data   │
│   4. ⚙️  Node Generator  - Create and test nodes    │
│   5. 📊 Overview         - System dashboard         │
│                                                      │
│ Navigation:                                          │
│   Arrow keys   Navigate grids and menus             │
│   Enter        Select/Confirm                       │
│   Esc          Cancel/Back                           │
│   r            Refresh current view                  │
│                                                      │
│ Press ? again to close this help                    │
└──────────────────────────────────────────────────────┘
`
	return help
}

// SwitchViewMsg is a message to switch views
type SwitchViewMsg struct {
	View ViewType
}

// NewSwitchViewMsg creates a new switch view message
func NewSwitchViewMsg(view ViewType) SwitchViewMsg {
	return SwitchViewMsg{View: view}
}
