package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/VoidMesh/api/cmd/debug/components"
	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/db"
)

// ChunkExplorerModel handles the chunk visualization view
type ChunkExplorerModel struct {
	// Database connections
	db           *sql.DB
	queries      *db.Queries
	chunkManager *chunk.Manager

	// Current state
	currentChunkX int64
	currentChunkZ int64
	cursorX       int
	cursorZ       int
	width         int
	height        int

	// Data
	chunkData   *chunk.ChunkResponse
	isLoading   bool
	lastUpdated time.Time
	errorMsg    string

	// UI state
	autoRefresh    bool
	refreshTicker  *time.Ticker
	showNodeInfo   bool
	selectedNodeID int64
}

// NewChunkExplorerModel creates a new chunk explorer model
func NewChunkExplorerModel(database *sql.DB, queries *db.Queries, chunkManager *chunk.Manager) ChunkExplorerModel {
	return ChunkExplorerModel{
		db:            database,
		queries:       queries,
		chunkManager:  chunkManager,
		currentChunkX: 0,
		currentChunkZ: 0,
		cursorX:       8, // Center of chunk
		cursorZ:       8,
		autoRefresh:   true,
		showNodeInfo:  true,
	}
}

// Init initializes the chunk explorer
func (m ChunkExplorerModel) Init() tea.Cmd {
	log.Debug("Initializing chunk explorer", "chunk_x", m.currentChunkX, "chunk_z", m.currentChunkZ)
	return tea.Batch(
		m.loadChunkCmd(),
		m.tickCmd(),
	)
}

// Update handles chunk explorer messages
func (m ChunkExplorerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Cursor movement within chunk
		case "up", "k":
			if m.cursorZ > 0 {
				m.cursorZ--
			}
		case "down", "j":
			if m.cursorZ < chunk.ChunkSize-1 {
				m.cursorZ++
			}
		case "left", "h":
			if m.cursorX > 0 {
				m.cursorX--
			}
		case "right", "l":
			if m.cursorX < chunk.ChunkSize-1 {
				m.cursorX++
			}

		// Chunk navigation
		case "shift+up", "K":
			m.currentChunkZ--
			m.cursorX = chunk.ChunkSize / 2
			m.cursorZ = chunk.ChunkSize / 2
			return m, m.loadChunkCmd()

		case "shift+down", "J":
			m.currentChunkZ++
			m.cursorX = chunk.ChunkSize / 2
			m.cursorZ = chunk.ChunkSize / 2
			return m, m.loadChunkCmd()

		case "shift+left", "H":
			m.currentChunkX--
			m.cursorX = chunk.ChunkSize / 2
			m.cursorZ = chunk.ChunkSize / 2
			return m, m.loadChunkCmd()

		case "shift+right", "L":
			m.currentChunkX++
			m.cursorX = chunk.ChunkSize / 2
			m.cursorZ = chunk.ChunkSize / 2
			return m, m.loadChunkCmd()

		// Actions
		case "r":
			return m, m.loadChunkCmd()

		case "g":
			// TODO: Implement "go to chunk" input
			return m, nil

		case "a":
			m.autoRefresh = !m.autoRefresh
			if m.autoRefresh {
				return m, m.tickCmd()
			} else if m.refreshTicker != nil {
				m.refreshTicker.Stop()
			}

		case "i":
			m.showNodeInfo = !m.showNodeInfo

		case "enter", " ":
			// Select node at cursor position
			if m.chunkData != nil {
				for _, node := range m.chunkData.Nodes {
					if node.LocalX == int64(m.cursorX) && node.LocalZ == int64(m.cursorZ) {
						m.selectedNodeID = node.NodeID
						break
					}
				}
			}
		}

	case chunkLoadedMsg:
		m.chunkData = msg.chunk
		m.isLoading = false
		m.lastUpdated = time.Now()
		m.errorMsg = ""

	case chunkErrorMsg:
		m.isLoading = false
		m.errorMsg = string(msg)
		log.Error("Failed to load chunk", "error", msg, "chunk_x", m.currentChunkX, "chunk_z", m.currentChunkZ)

	case tickMsg:
		if m.autoRefresh {
			return m, tea.Batch(
				m.loadChunkCmd(),
				m.tickCmd(),
			)
		}
	}

	return m, nil
}

// View renders the chunk explorer
func (m ChunkExplorerModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	var s strings.Builder

	// Title
	title := components.TitleStyle.Render(fmt.Sprintf("Chunk Explorer - (%d, %d)", m.currentChunkX, m.currentChunkZ))
	s.WriteString(title + "\n")

	// Main content area
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderGrid(),
		m.renderInfoPanel(),
	)
	s.WriteString(mainContent + "\n")

	// Status bar
	statusBar := m.renderStatusBar()
	s.WriteString(statusBar)

	return s.String()
}

// renderGrid renders the chunk grid
func (m ChunkExplorerModel) renderGrid() string {
	if m.chunkData == nil {
		if m.isLoading {
			return components.BorderStyle.Render("Loading chunk data...")
		}
		if m.errorMsg != "" {
			return components.BorderStyle.Render("Error: " + m.errorMsg)
		}
		return components.BorderStyle.Render("No data")
	}

	// Create a map of positions to nodes for quick lookup
	nodeMap := make(map[string]*chunk.ResourceNode)
	for i := range m.chunkData.Nodes {
		node := &m.chunkData.Nodes[i]
		key := fmt.Sprintf("%d,%d", node.LocalX, node.LocalZ)
		nodeMap[key] = node
	}

	var gridRows []string
	for z := range chunk.ChunkSize {
		var row []string
		for x := range chunk.ChunkSize {
			cellContent := components.EmptySymbol
			var cellStyle lipgloss.Style

			// Check if there's a node at this position
			key := fmt.Sprintf("%d,%d", x, z)
			if node, exists := nodeMap[key]; exists {
				cellContent = components.GetNodeSymbol(node.NodeType, node.NodeSubtype, node.IsActive, node.CurrentYield)
				cellColor := components.GetNodeColor(node.NodeType, node.NodeSubtype, node.IsActive, node.CurrentYield)
				cellStyle = components.GridCellStyle.Foreground(cellColor)
			} else {
				cellStyle = components.GridCellStyle.Foreground(components.Gray)
			}

			// Highlight cursor position
			if x == m.cursorX && z == m.cursorZ {
				cellStyle = components.GridSelectedCellStyle
				if cellContent == components.EmptySymbol {
					cellContent = components.CursorSymbol
				}
			}

			row = append(row, cellStyle.Render(cellContent))
		}
		gridRows = append(gridRows, strings.Join(row, ""))
	}

	grid := strings.Join(gridRows, "\n")

	// Add border and title
	gridWithBorder := components.BorderStyle.
		Width(chunk.ChunkSize*2 + 2).
		Height(chunk.ChunkSize + 2).
		Render(grid)

	return gridWithBorder
}

// renderInfoPanel renders the information panel
func (m ChunkExplorerModel) renderInfoPanel() string {
	if !m.showNodeInfo {
		return ""
	}

	var info strings.Builder

	// Current cursor position info
	info.WriteString(components.SubtitleStyle.Render("Position Info") + "\n")
	info.WriteString(fmt.Sprintf("Chunk: (%d, %d)\n", m.currentChunkX, m.currentChunkZ))
	info.WriteString(fmt.Sprintf("Local: (%d, %d)\n", m.cursorX, m.cursorZ))
	info.WriteString(fmt.Sprintf("World: (%d, %d)\n\n",
		m.currentChunkX*chunk.ChunkSize+int64(m.cursorX),
		m.currentChunkZ*chunk.ChunkSize+int64(m.cursorZ)))

	// Node at cursor position
	if m.chunkData != nil {
		var nodeAtCursor *chunk.ResourceNode
		for i := range m.chunkData.Nodes {
			node := &m.chunkData.Nodes[i]
			if node.LocalX == int64(m.cursorX) && node.LocalZ == int64(m.cursorZ) {
				nodeAtCursor = node
				break
			}
		}

		info.WriteString(components.SubtitleStyle.Render("Node Info") + "\n")
		if nodeAtCursor != nil {
			info.WriteString(fmt.Sprintf("ID: %d\n", nodeAtCursor.NodeID))
			info.WriteString(fmt.Sprintf("Type: %s\n", getNodeTypeName(nodeAtCursor.NodeType)))
			info.WriteString(fmt.Sprintf("Quality: %s\n", getQualityName(nodeAtCursor.NodeSubtype)))
			info.WriteString(fmt.Sprintf("Yield: %d/%d\n", nodeAtCursor.CurrentYield, nodeAtCursor.MaxYield))
			info.WriteString(fmt.Sprintf("Active: %v\n", nodeAtCursor.IsActive))
			info.WriteString(fmt.Sprintf("Spawned: %s\n", nodeAtCursor.SpawnedAt.Format("15:04")))
			if nodeAtCursor.LastHarvest != nil {
				info.WriteString(fmt.Sprintf("Last Harvest: %s\n", nodeAtCursor.LastHarvest.Format("15:04")))
			}
			if nodeAtCursor.RespawnTimer != nil {
				info.WriteString(fmt.Sprintf("Respawn: %s\n", nodeAtCursor.RespawnTimer.Format("15:04")))
			}
		} else {
			info.WriteString("No node at this position\n")
		}
	}

	// Legend
	info.WriteString("\n" + components.SubtitleStyle.Render("Legend") + "\n")
	info.WriteString("Fe Iron Ore  Au Gold Ore\n")
	info.WriteString("## Wood      [] Stone\n")
	info.WriteString("*  Rich      O  Normal\n")
	info.WriteString("o  Poor      xx Depleted\n")
	info.WriteString(".. Respawning >< Cursor\n")

	// Controls
	info.WriteString("\n" + components.SubtitleStyle.Render("Controls") + "\n")
	info.WriteString("Arrow keys: Move cursor\n")
	info.WriteString("Shift+Arrow: Move chunk\n")
	info.WriteString("r: Refresh  a: Auto-refresh\n")
	info.WriteString("i: Toggle info  q: Back\n")

	return components.InfoPanelStyle.Render(info.String())
}

// renderStatusBar renders the status bar
func (m ChunkExplorerModel) renderStatusBar() string {
	var status []string

	// Chunk info
	if m.chunkData != nil {
		status = append(status, fmt.Sprintf("Nodes: %d", len(m.chunkData.Nodes)))
	}

	// Auto-refresh status
	if m.autoRefresh {
		status = append(status, "Auto-refresh: ON")
	} else {
		status = append(status, "Auto-refresh: OFF")
	}

	// Last updated
	if !m.lastUpdated.IsZero() {
		status = append(status, fmt.Sprintf("Updated: %s", m.lastUpdated.Format("15:04:05")))
	}

	// Error status
	if m.errorMsg != "" {
		status = append(status, fmt.Sprintf("Error: %s", m.errorMsg))
	}

	statusText := strings.Join(status, " • ")
	return components.StatusBarStyle.Width(m.width).Render(statusText)
}

// SetSize updates the chunk explorer size
func (m *ChunkExplorerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// loadChunkCmd creates a command to load chunk data
func (m ChunkExplorerModel) loadChunkCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		chunk, err := m.chunkManager.LoadChunk(ctx, m.currentChunkX, m.currentChunkZ)
		if err != nil {
			return chunkErrorMsg(err.Error())
		}

		return chunkLoadedMsg{chunk: chunk}
	}
}

// tickCmd creates a command for auto-refresh
func (m ChunkExplorerModel) tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Helper functions
func getNodeTypeName(nodeType int64) string {
	switch nodeType {
	case 1:
		return "Iron Ore"
	case 2:
		return "Gold Ore"
	case 3:
		return "Wood"
	case 4:
		return "Stone"
	default:
		return "Unknown"
	}
}

func getQualityName(quality int64) string {
	switch quality {
	case 0:
		return "Poor"
	case 1:
		return "Normal"
	case 2:
		return "Rich"
	default:
		return "Unknown"
	}
}

// Messages
type chunkLoadedMsg struct {
	chunk *chunk.ChunkResponse
}

type chunkErrorMsg string

type tickMsg struct{}
