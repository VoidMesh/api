package models

import (
	"database/sql"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/VoidMesh/api/cmd/debug/components"
	"github.com/VoidMesh/api/internal/db"
)

// SessionMonitorModel handles the session monitoring view
type SessionMonitorModel struct {
	db      *sql.DB
	queries *db.Queries
	width   int
	height  int
}

// NewSessionMonitorModel creates a new session monitor model
func NewSessionMonitorModel(database *sql.DB, queries *db.Queries) SessionMonitorModel {
	return SessionMonitorModel{
		db:      database,
		queries: queries,
	}
}

// Init initializes the session monitor
func (m SessionMonitorModel) Init() tea.Cmd {
	return nil
}

// Update handles session monitor messages
func (m SessionMonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// TODO: Refresh sessions
			return m, nil
		}
	}

	return m, nil
}

// View renders the session monitor
func (m SessionMonitorModel) View() string {
	var s strings.Builder

	// Title
	title := components.TitleStyle.Render("Session Monitor")
	s.WriteString(title + "\n\n")

	// Placeholder content
	content := components.BorderStyle.Render("Session monitoring coming soon...\n\nThis will show:\n• Active harvest sessions\n• Player activity\n• Session timeouts\n• Real-time updates")
	s.WriteString(content + "\n\n")

	// Status bar
	statusBar := components.StatusBarStyle.Width(m.width).Render("Press 'r' to refresh • 'q' to go back")
	s.WriteString(statusBar)

	return s.String()
}

// SetSize updates the session monitor size
func (m *SessionMonitorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
