package models

import (
	"database/sql"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/VoidMesh/api/cmd/debug/components"
	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/db"
)

// NodeGeneratorModel handles the node generator view
type NodeGeneratorModel struct {
	db           *sql.DB
	queries      *db.Queries
	chunkManager *chunk.Manager
	width        int
	height       int
}

// NewNodeGeneratorModel creates a new node generator model
func NewNodeGeneratorModel(database *sql.DB, queries *db.Queries, chunkManager *chunk.Manager) NodeGeneratorModel {
	return NodeGeneratorModel{
		db:           database,
		queries:      queries,
		chunkManager: chunkManager,
	}
}

// Init initializes the node generator
func (m NodeGeneratorModel) Init() tea.Cmd {
	return nil
}

// Update handles node generator messages
func (m NodeGeneratorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View renders the node generator
func (m NodeGeneratorModel) View() string {
	var s strings.Builder

	// Title
	title := components.TitleStyle.Render("Node Generator")
	s.WriteString(title + "\n\n")

	// Placeholder content
	content := components.BorderStyle.Render("Node generator coming soon...\n\nThis will show:\n• Node creation form\n• Template testing\n• Bulk operations\n• Spawn simulation")
	s.WriteString(content + "\n\n")

	// Status bar
	statusBar := components.StatusBarStyle.Width(m.width).Render("Press 'r' to refresh • 'q' to go back")
	s.WriteString(statusBar)

	return s.String()
}

// SetSize updates the node generator size
func (m *NodeGeneratorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
