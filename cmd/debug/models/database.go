package models

import (
	"database/sql"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/VoidMesh/api/cmd/debug/components"
	"github.com/VoidMesh/api/internal/db"
)

// DatabaseModel handles the database inspector view
type DatabaseModel struct {
	db      *sql.DB
	queries *db.Queries
	width   int
	height  int
}

// NewDatabaseModel creates a new database model
func NewDatabaseModel(database *sql.DB, queries *db.Queries) DatabaseModel {
	return DatabaseModel{
		db:      database,
		queries: queries,
	}
}

// Init initializes the database model
func (m DatabaseModel) Init() tea.Cmd {
	return nil
}

// Update handles database model messages
func (m DatabaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// TODO: Refresh data
			return m, nil
		}
	}

	return m, nil
}

// View renders the database inspector
func (m DatabaseModel) View() string {
	var s strings.Builder

	// Title
	title := components.TitleStyle.Render("Database Inspector")
	s.WriteString(title + "\n\n")

	// Placeholder content
	content := components.BorderStyle.Render("Database inspector coming soon...\n\nThis will show:\n• Query interface\n• Pre-built queries\n• Table browser\n• Data export")
	s.WriteString(content + "\n\n")

	// Status bar
	statusBar := components.StatusBarStyle.Width(m.width).Render("Press 'r' to refresh • 'q' to go back")
	s.WriteString(statusBar)

	return s.String()
}

// SetSize updates the database model size
func (m *DatabaseModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
