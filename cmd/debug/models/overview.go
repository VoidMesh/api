package models

import (
	"database/sql"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/VoidMesh/api/cmd/debug/components"
	"github.com/VoidMesh/api/internal/db"
)

// OverviewModel handles the system overview view
type OverviewModel struct {
	db      *sql.DB
	queries *db.Queries
	width   int
	height  int
}

// NewOverviewModel creates a new overview model
func NewOverviewModel(database *sql.DB, queries *db.Queries) OverviewModel {
	return OverviewModel{
		db:      database,
		queries: queries,
	}
}

// Init initializes the overview
func (m OverviewModel) Init() tea.Cmd {
	return nil
}

// Update handles overview messages
func (m OverviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View renders the overview
func (m OverviewModel) View() string {
	var s strings.Builder

	// Title
	title := components.TitleStyle.Render("System Overview")
	s.WriteString(title + "\n\n")

	// Placeholder content
	content := components.BorderStyle.Render("System overview coming soon...\n\nThis will show:\n• Key metrics\n• System health\n• Performance charts\n• Activity summaries")
	s.WriteString(content + "\n\n")

	// Status bar
	statusBar := components.StatusBarStyle.Width(m.width).Render("Press 'r' to refresh • 'q' to go back")
	s.WriteString(statusBar)

	return s.String()
}

// SetSize updates the overview size
func (m *OverviewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
