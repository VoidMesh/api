package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	"github.com/VoidMesh/api/cmd/debug/components"
)

// MenuModel handles the main menu view
type MenuModel struct {
	choices []MenuChoice
	cursor  int
	width   int
	height  int
}

// MenuChoice represents a menu option
type MenuChoice struct {
	Title       string
	Description string
	Icon        string
	View        ViewType
}

// NewMenuModel creates a new menu model
func NewMenuModel() MenuModel {
	choices := []MenuChoice{
		{
			Title:       "Chunk Explorer",
			Description: "Visualize world chunks and resource nodes",
			Icon:        "🗺️",
			View:        ChunkExplorerView,
		},
		{
			Title:       "Session Monitor",
			Description: "Monitor active harvest sessions",
			Icon:        "👥",
			View:        SessionMonitorView,
		},
		{
			Title:       "Database Inspector",
			Description: "Query and analyze database contents",
			Icon:        "🗄️",
			View:        DatabaseView,
		},
		{
			Title:       "Node Generator",
			Description: "Create and test resource nodes",
			Icon:        "⚙️",
			View:        NodeGeneratorView,
		},
		{
			Title:       "System Overview",
			Description: "View system metrics and dashboard",
			Icon:        "📊",
			View:        OverviewView,
		},
	}

	return MenuModel{
		choices: choices,
		cursor:  0,
	}
}

// Init initializes the menu
func (m MenuModel) Init() tea.Cmd {
	return nil
}

// Update handles menu messages
func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.choices) - 1
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}

		case "enter", " ":
			// Switch to selected view
			selected := m.choices[m.cursor]
			return m, func() tea.Msg {
				return NewSwitchViewMsg(selected.View)
			}

		case "1", "2", "3", "4", "5":
			// Direct number selection
			choice := int(msg.String()[0] - '1')
			if choice >= 0 && choice < len(m.choices) {
				m.cursor = choice
				selected := m.choices[m.cursor]
				return m, func() tea.Msg {
					return NewSwitchViewMsg(selected.View)
				}
			}
		}
	}

	return m, nil
}

// View renders the menu
func (m MenuModel) View() string {
	var s strings.Builder

	// Title
	title := components.TitleStyle.Render("VoidMesh Debug Tool")
	s.WriteString(title + "\n\n")

	// Menu items
	menuStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(components.PrimaryColor).
		Padding(1, 2).
		Width(60)

	var menuItems []string
	for i, choice := range m.choices {
		// Format menu item
		number := fmt.Sprintf("%d.", i+1)
		title := fmt.Sprintf("%s %s", choice.Icon, choice.Title)
		description := choice.Description

		var itemStyle lipgloss.Style
		if i == m.cursor {
			itemStyle = components.SelectedMenuItemStyle
		} else {
			itemStyle = components.MenuItemStyle
		}

		// Create the full menu item
		item := fmt.Sprintf("%-3s %-25s %s", number, title, description)
		menuItems = append(menuItems, itemStyle.Render(item))
	}

	menu := menuStyle.Render(strings.Join(menuItems, "\n"))
	s.WriteString(menu + "\n\n")

	// Instructions
	instructions := components.HelpStyle.Render(
		"Use ↑/↓ or j/k to navigate • Enter or number to select • ? for help • q to quit",
	)
	s.WriteString(instructions)

	// Footer with version info
	footer := "\n\n" + components.StatusBarStyle.Render(
		"VoidMesh API Debug Tool v1.0.0 • Built with Bubble Tea & Lip Gloss",
	)
	s.WriteString(footer)

	// Center the content
	content := s.String()
	if m.width > 0 {
		contentWidth := lipgloss.Width(content)
		if contentWidth < m.width {
			leftPadding := (m.width - contentWidth) / 2
			content = lipgloss.NewStyle().PaddingLeft(leftPadding).Render(content)
		}
	}

	return content
}

// SetSize updates the menu size
func (m *MenuModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
