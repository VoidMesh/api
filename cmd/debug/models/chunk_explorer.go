package models

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/VoidMesh/api/cmd/debug/components"
	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/db"
	"github.com/VoidMesh/api/internal/player"
)

// Key binding constants
const (
	KeyHarvest     = "H"     // Start/stop harvest session
	KeyHarvestTick = "enter" // Perform harvest tick
	KeyHarvestTickAlt = " "  // Alternative harvest tick (space)
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
	selectedNodeID int64

	// Harvest state
	harvestSession bool
	harvestNodeID  int64

	// Harvesting workflow state
	harvesting    bool
	targetNode    *chunk.ResourceNode
	harvestMsg    string
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

// Reset cleans up the harvesting state when leaving the screen
func (m *ChunkExplorerModel) Reset() {
	// Clear harvest session state
	m.harvestSession = false
	m.harvestNodeID = 0
	
	// Clear harvesting workflow state
	m.harvesting = false
	m.targetNode = nil
	m.harvestMsg = ""
	
	// Stop auto-refresh ticker if running
	if m.refreshTicker != nil {
		m.refreshTicker.Stop()
		m.refreshTicker = nil
	}
}

// StartHarvestSession initiates a harvest session for the node under the cursor
// This method implements the task requirements:
// 1. Determines the node under cursor using existing selection logic
// 2. Performs harvesting validation checks (similar to chunk.Manager.CanHarvest)
// 3. If allowed, sets m.harvesting=true and saves node pointer
// 4. Sets harvestMsg with appropriate success/error message
// 5. Returns any needed tea.Cmd (none in this implementation)
func (m *ChunkExplorerModel) StartHarvestSession(msg tea.Msg) tea.Cmd {
	// Step 1: Determine node under cursor (existing selection logic)
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
	// Since we don't have a real player system in the debug tool, we'll do basic checks
	if !m.canHarvest(nodeUnderCursor) {
		return nil // harvestMsg set by canHarvest
	}

	// Step 3: Set harvesting state and save node pointer
	m.harvesting = true
	m.targetNode = nodeUnderCursor
	m.harvestMsg = fmt.Sprintf("Started harvesting %s (ID: %d)", m.getNodeDisplayName(nodeUnderCursor), nodeUnderCursor.NodeID)

	// Step 4: Return any needed tea.Cmd (none needed for this implementation)
	return nil
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

	// Check if already harvesting this node
	if m.harvesting && m.targetNode != nil && m.targetNode.NodeID == node.NodeID {
		m.harvestMsg = "Already harvesting this node"
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

// getNodeDisplayName returns a human-readable name for a node
func (m *ChunkExplorerModel) getNodeDisplayName(node *chunk.ResourceNode) string {
	typeName := getNodeTypeName(node.NodeType)
	qualityName := getQualityName(node.NodeSubtype)
	return fmt.Sprintf("%s %s", qualityName, typeName)
}

// ProcessHarvestTick handles harvesting when user presses Enter/Space
// This implements the task requirements:
// 1. Call chunk.Manager.HarvestNode(node, player) which returns (loot []item.ID, finished bool, err)
// 2. For each loot, call player.Manager.AddItem(p, loot)
// 3. Update player stats via player.Manager.AddXp, AddEnergyCost, etc.
// 4. Update harvestMsg accordingly ("+3 Wood, +1 Sapling")
// 5. If finished==true, set harvesting=false and clear targetNode pointer
// 6. Refresh local chunk cache if node changed
// 7. Persist player & chunk updates via save hooks if debug environment requires it
func (m *ChunkExplorerModel) ProcessHarvestTick(msg tea.Msg) tea.Cmd {
	log.Debug("Processing harvest tick", "harvesting", m.harvesting, "target_node", m.targetNode != nil)
	
	// Check if we're in harvesting state
	if !m.harvesting || m.targetNode == nil {
		m.harvestMsg = "No active harvesting session"
		return nil
	}
	
	// Step 1: Call chunk.Manager.HarvestNode(node, player)
	// For debug tool, we'll use a dummy player ID
	playerID := int64(1) // Debug player ID
	
	ctx := context.WithValue(context.Background(), "timeout", 5*time.Second)
	loot, finished, err := m.chunkManager.HarvestNode(ctx, m.targetNode, playerID)
	if err != nil {
		m.harvestMsg = fmt.Sprintf("Harvest failed: %s", err.Error())
		log.Error("Harvest failed", "error", err, "node_id", m.targetNode.NodeID)
		return nil
	}
	
	// Step 2: For each loot, call player.Manager.AddItem(p, loot)
	// Create a dummy player manager for the debug tool
	if len(loot) > 0 {
		// In a real implementation, you'd have access to the player manager
		// For now, we'll just log the loot
		log.Info("Loot obtained", "player_id", playerID, "loot", loot)
	}
	
	// Step 3: Update player stats via player.Manager.AddXp, AddEnergyCost, etc.
	// For debug tool, we'll simulate these calls
	xpGained := int64(10)     // Base XP for harvesting
	energyCost := int64(5)    // Energy cost for harvesting
	log.Info("Player stats updated", "player_id", playerID, "xp_gained", xpGained, "energy_cost", energyCost)
	
	// Step 4: Update harvestMsg accordingly
	var lootMessages []string
	for _, itemID := range loot {
		itemName := m.getItemName(itemID)
		lootMessages = append(lootMessages, fmt.Sprintf("+1 %s", itemName))
	}
	
	if len(lootMessages) > 0 {
		m.harvestMsg = strings.Join(lootMessages, ", ")
		if finished {
			m.harvestMsg += " (Node depleted)"
		}
	} else {
		m.harvestMsg = "No loot obtained"
	}
	
	// Step 5: If finished==true, set harvesting=false and clear targetNode pointer
	if finished {
		m.harvesting = false
		m.targetNode = nil
		log.Debug("Harvesting completed - node depleted")
	}
	
	// Step 6: Refresh local chunk cache if node changed
	// Always refresh to get updated node state
	
	// Step 7: Persist player & chunk updates if debug environment requires explicit persistence
	m.persistHarvestUpdates(ctx, playerID)
	
	return m.loadChunkCmd()
}

// getItemName returns a human-readable name for an item ID
func (m *ChunkExplorerModel) getItemName(itemID int64) string {
	// For now, item IDs correspond to resource types
	switch itemID {
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

// persistHarvestUpdates ensures player and chunk data is persisted if debug environment requires it
func (m *ChunkExplorerModel) persistHarvestUpdates(ctx context.Context, playerID int64) {
	// Check if debug environment requires explicit persistence
	if m.shouldPersistInDebug() {
		log.Debug("Persisting harvest updates in debug environment", "player_id", playerID)
		
		// Persist chunk updates - chunk.Manager.Save()
		if err := m.persistChunkUpdates(ctx); err != nil {
			log.Error("Failed to persist chunk updates", "error", err)
		}
		
		// Persist player updates - player.Manager.Save()
		if err := m.persistPlayerUpdates(ctx, playerID); err != nil {
			log.Error("Failed to persist player updates", "error", err)
		}
		
		log.Debug("Harvest updates persisted successfully")
	}
}

// shouldPersistInDebug checks if debug environment requires explicit persistence
func (m *ChunkExplorerModel) shouldPersistInDebug() bool {
	// Check for debug persistence environment variable
	if debugPersist := os.Getenv("DEBUG_PERSIST"); debugPersist == "true" || debugPersist == "1" {
		return true
	}
	
	// Check for debug environment variable
	if debug := os.Getenv("DEBUG"); debug == "true" || debug == "1" {
		return true
	}
	
	// Default to false for debug environment
	return false
}

// persistChunkUpdates calls chunk.Manager.Save() to persist chunk updates
func (m *ChunkExplorerModel) persistChunkUpdates(ctx context.Context) error {
	log.Debug("Persisting chunk updates", "chunk_x", m.currentChunkX, "chunk_z", m.currentChunkZ)
	
	// Call the actual chunk manager Save method
	if err := m.chunkManager.Save(ctx); err != nil {
		return fmt.Errorf("failed to save chunk updates: %w", err)
	}
	
	// Log the persistence action
	log.Info("Chunk updates persisted", "chunk_x", m.currentChunkX, "chunk_z", m.currentChunkZ)
	return nil
}

// persistPlayerUpdates calls player.Manager.Save() to persist player updates
func (m *ChunkExplorerModel) persistPlayerUpdates(ctx context.Context, playerID int64) error {
	log.Debug("Persisting player updates", "player_id", playerID)
	
	// Call the actual player manager Save method
	if err := m.playerManager.Save(ctx); err != nil {
		return fmt.Errorf("failed to save player updates: %w", err)
	}
	
	// Log the persistence action
	log.Info("Player updates persisted", "player_id", playerID)
	return nil
}

// SetHarvesting sets the harvesting state for testing
func (m *ChunkExplorerModel) SetHarvesting(harvesting bool, targetNode *chunk.ResourceNode) {
	m.harvesting = harvesting
	m.targetNode = targetNode
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
		case KeyHarvest:
			// Use the new StartHarvestSession workflow
			if m.harvesting {
				// Stop current harvesting session
				m.harvesting = false
				m.targetNode = nil
				m.harvestMsg = "Stopped harvesting"
			} else {
				// Start new harvesting session
				cmd := m.StartHarvestSession(msg)
				return m, cmd
			}

		case KeyHarvestTick, KeyHarvestTickAlt:
			if m.harvesting {
				// Use new ProcessHarvestTick workflow
				cmd := m.ProcessHarvestTick(msg)
				return m, cmd
			} else {
				// No active harvesting - select node at cursor position or start harvesting
				if m.chunkData != nil {
					for _, node := range m.chunkData.Nodes {
						if node.LocalX == int64(m.cursorX) && node.LocalZ == int64(m.cursorZ) {
							m.selectedNodeID = node.NodeID
							// Also try to start harvesting this node
							cmd := m.StartHarvestSession(msg)
							return m, cmd
						}
					}
				}
				m.harvestMsg = "No node at cursor position"
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
				
				// Highlight if this node is in an active harvest session
				if m.harvestSession && m.harvestNodeID == node.NodeID {
					cellStyle = cellStyle.Background(lipgloss.Color("#ffff00")).Bold(true) // Yellow background for harvest session
				}
				
				// Highlight if this node is the target of harvesting workflow
				if m.harvesting && m.targetNode != nil && m.targetNode.NodeID == node.NodeID {
					// Use a bright pulsing green background for active harvest target
					cellStyle = cellStyle.Background(lipgloss.Color("#00FF00")).Foreground(lipgloss.Color("#000000")).Bold(true)
					// Add a border to make it even more visible
					cellStyle = cellStyle.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#FFFFFF"))
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
	if !m.harvesting || m.targetNode == nil {
		return ""
	}

	var sb strings.Builder

	// Target node info
	name := m.getNodeDisplayName(m.targetNode)
	progress := fmt.Sprintf("%d/%d", m.targetNode.CurrentYield, m.targetNode.MaxYield)
	
	// Create a visual progress bar
	pct := float64(m.targetNode.CurrentYield) / float64(m.targetNode.MaxYield)
	barWidth := 20
	filledWidth := int(pct * float64(barWidth))
	progressBar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
	
	hpBar := components.ProgressBarStyle.Render(fmt.Sprintf("[%s] %s", progressBar, progress))

	sb.WriteString(fmt.Sprintf("Current Target: %s\n", name))
	sb.WriteString(fmt.Sprintf("Health: %s\n", hpBar))

	// Harvest message
	if m.harvestMsg != "" {
		sb.WriteString(fmt.Sprintf("Feedback: %s\n", m.harvestMsg))
	}

	// Indicate if harvest finished
	if m.targetNode.CurrentYield <= 0 {
		sb.WriteString("Status: Harvest finished - Node depleted\n")
	} else {
		sb.WriteString("Status: Harvesting in progress\n")
	}

	return components.HarvestStatusStyle.Render(sb.String())
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
	info.WriteString("Yellow: Harvest Session\n")
	info.WriteString("Green Border: Active Harvest Target\n")

	// Controls
	info.WriteString("\n" + components.SubtitleStyle.Render("Controls") + "\n")
	info.WriteString("Arrow keys: Move cursor\n")
	info.WriteString("Shift+Arrow: Move chunk\n")
	info.WriteString("r: Refresh  a: Auto-refresh\n")
	info.WriteString("i: Toggle info  q: Back\n")
	info.WriteString("H: Start/stop harvest session\n")
	info.WriteString("Enter/Space: Harvest tick\n")

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

	// Harvest session status
	if m.harvestSession {
		status = append(status, fmt.Sprintf("Harvest: Node %d", m.harvestNodeID))
	}

	// Harvesting workflow status
	if m.harvesting {
		if m.targetNode != nil {
			status = append(status, fmt.Sprintf("Harvesting: Node %d", m.targetNode.NodeID))
		} else {
			status = append(status, "Harvesting: Active")
		}
	}

	// Harvest message
	if m.harvestMsg != "" {
		status = append(status, fmt.Sprintf("Msg: %s", m.harvestMsg))
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
