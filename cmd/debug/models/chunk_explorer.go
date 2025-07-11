package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	"github.com/VoidMesh/api/cmd/debug/components"
	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/db"
	"github.com/VoidMesh/api/internal/player"
)

// Key binding constants
const (
	KeyHarvest     = "H"     // Harvest node directly
	KeyHarvestTick = "enter" // Harvest node directly
	KeyHarvestTickAlt = " "  // Alternative harvest (space)
)

// ChunkExplorerModel handles the chunk visualization view
type ChunkExplorerModel struct {
	// Database connections
	db            *sql.DB
	queries       *db.Queries
	chunkManager  *chunk.Manager
	playerManager *player.Manager

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

	// Harvest state
	harvestMsg    string
	lastHarvested map[int64]bool // Track which nodes were harvested today
}

// NewChunkExplorerModel creates a new chunk explorer model
func NewChunkExplorerModel(database *sql.DB, queries *db.Queries, chunkManager *chunk.Manager, playerManager *player.Manager) ChunkExplorerModel {
	return ChunkExplorerModel{
		db:            database,
		queries:       queries,
		chunkManager:  chunkManager,
		playerManager: playerManager,
		currentChunkX: 0,
		currentChunkZ: 0,
		cursorX:       8, // Center of chunk
		cursorZ:       8,
		autoRefresh:   true,
		showNodeInfo:  true,
		lastHarvested: make(map[int64]bool),
	}
}

// Init initializes the chunk explorer
func (m ChunkExplorerModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadChunkCmd(),
		m.tickCmd(),
	)
}

// Reset cleans up the harvesting state when leaving the screen
func (m *ChunkExplorerModel) Reset() {
	// Clear harvest state
	m.harvestMsg = ""
	m.lastHarvested = make(map[int64]bool)
	
	// Stop auto-refresh ticker if running
	if m.refreshTicker != nil {
		m.refreshTicker.Stop()
		m.refreshTicker = nil
	}
}

// HarvestNode directly harvests the node under the cursor using the new API
func (m *ChunkExplorerModel) HarvestNode(msg tea.Msg) tea.Cmd {
	// Step 1: Determine node under cursor
	var nodeUnderCursor *chunk.ResourceNode
	if m.chunkData != nil {
		for i := range m.chunkData.Nodes {
			node := &m.chunkData.Nodes[i]
			if node.LocalX == int64(m.cursorX) && node.LocalZ == int64(m.cursorZ) {
				nodeUnderCursor = node
				break
			}
		}
	}

	// If no node found at cursor position
	if nodeUnderCursor == nil {
		m.harvestMsg = "No node at cursor position"
		return nil
	}

	// Step 2: Check if harvesting is allowed
	if !m.canHarvest(nodeUnderCursor) {
		return nil // harvestMsg set by canHarvest
	}

	// Step 3: Perform direct harvest
	return m.performHarvest(nodeUnderCursor)
}

// canHarvest performs basic validation to determine if a node can be harvested
func (m *ChunkExplorerModel) canHarvest(node *chunk.ResourceNode) bool {
	// Check if node is active
	if !node.IsActive {
		m.harvestMsg = "Node is not active"
		return false
	}

	// Check if node has yield
	if node.CurrentYield <= 0 {
		m.harvestMsg = "Node is depleted"
		return false
	}

	// Check if already harvested today (simulate daily limit)
	if m.lastHarvested[node.NodeID] {
		m.harvestMsg = "Already harvested this node today"
		return false
	}

	// In a real implementation, we would check:
	// - Distance from player to node
	// - Player has appropriate tool
	// - Player has sufficient stamina
	// - Player level/skill requirements
	// For now, we'll just do a basic distance check (cursor is at node, so distance is 0)

	return true
}



// performHarvest executes the direct harvest using the new API
func (m *ChunkExplorerModel) performHarvest(node *chunk.ResourceNode) tea.Cmd {
	return func() tea.Msg {
		// For debug tool, we'll use a dummy player ID
		playerID := int64(1) // Debug player ID
		
		// Create harvest context
		harvestCtx := chunk.HarvestContext{
			PlayerID: playerID,
			NodeID:   node.NodeID,
		}
		
		// Perform direct harvest
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result, err := m.chunkManager.HarvestNode(ctx, harvestCtx)
		if err != nil {
			// Error already logged in chunk manager
			return harvestResultMsg{
				success: false,
				message: fmt.Sprintf("Harvest failed: %s", err.Error()),
			}
		}
		
		// Mark node as harvested for the day
		m.lastHarvested[node.NodeID] = true
		
		// Create success message
		var lootMessages []string
		for _, loot := range result.PrimaryLoot {
			itemName := m.getItemName(loot.ItemType)
			lootMessages = append(lootMessages, fmt.Sprintf("+%d %s", loot.Quantity, itemName))
		}
		
		for _, loot := range result.BonusLoot {
			itemName := m.getItemName(loot.ItemType)
			lootMessages = append(lootMessages, fmt.Sprintf("+%d %s (bonus)", loot.Quantity, itemName))
		}
		
		message := "Harvest successful"
		if len(lootMessages) > 0 {
			message = strings.Join(lootMessages, ", ")
		}
		
		if !result.NodeState.IsActive {
			message += " (Node depleted)"
		}
		
		// Harvest success logged in chunk manager
		
		return harvestResultMsg{
			success: true,
			message: message,
		}
	}
}

// getItemName returns a human-readable name for an item type
func (m *ChunkExplorerModel) getItemName(itemType int64) string {
	// Item types correspond to resource types
	switch itemType {
	case 1:
		return "Iron Ore"
	case 2:
		return "Gold Ore"
	case 3:
		return "Wood"
	case 4:
		return "Stone"
	default:
		return "Unknown Item"
	}
}

// GetHarvestMsg returns the current harvest message
func (m *ChunkExplorerModel) GetHarvestMsg() string {
	return m.harvestMsg
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

		case "shift+left":
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

		// Harvest functionality
		case KeyHarvest, KeyHarvestTick, KeyHarvestTickAlt:
			// Direct harvest using new API
			cmd := m.HarvestNode(msg)
			return m, cmd
		}

	case chunkLoadedMsg:
		m.chunkData = msg.chunk
		m.isLoading = false
		m.lastUpdated = time.Now()
		m.errorMsg = ""

	case chunkErrorMsg:
		m.isLoading = false
		m.errorMsg = string(msg)
		// Error already logged in loadChunkCmd

	case harvestResultMsg:
		m.harvestMsg = msg.message
		if msg.success {
			// Refresh chunk data to show updated node state
			return m, m.loadChunkCmd()
		}

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

	// Harvest status section
	harvestStatus := m.renderHarvestStatus()
	if harvestStatus != "" {
		s.WriteString(harvestStatus + "\n")
	}

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
				
				// Highlight if this node was harvested today
				if m.lastHarvested[node.NodeID] {
					cellStyle = cellStyle.Background(lipgloss.Color("#444444")).Bold(true) // Gray background for harvested nodes
				}
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

// renderHarvestStatus renders the harvest status line
func (m ChunkExplorerModel) renderHarvestStatus() string {
	if m.harvestMsg == "" {
		return ""
	}

	return components.HarvestStatusStyle.Render(fmt.Sprintf("Last Harvest: %s\n", m.harvestMsg))
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
	info.WriteString(".. Respawning ><  Cursor\n")
	info.WriteString("Gray: Harvested Today\n")

	// Controls
	info.WriteString("\n" + components.SubtitleStyle.Render("Controls") + "\n")
	info.WriteString("Arrow keys: Move cursor\n")
	info.WriteString("Shift+Arrow: Move chunk\n")
	info.WriteString("r: Refresh  a: Auto-refresh\n")
	info.WriteString("i: Toggle info  q: Back\n")
	info.WriteString("H/Enter/Space: Harvest node\n")

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

	// Harvest message
	if m.harvestMsg != "" {
		status = append(status, fmt.Sprintf("Last: %s", m.harvestMsg))
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

type harvestResultMsg struct {
	success bool
	message string
}

type tickMsg struct{}
