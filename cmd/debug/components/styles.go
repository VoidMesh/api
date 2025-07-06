package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Color definitions
var (
	// Primary colors
	PrimaryColor   = lipgloss.Color("#7D56F4")
	SecondaryColor = lipgloss.Color("#04B575")
	AccentColor    = lipgloss.Color("#FFD700")
	DangerColor    = lipgloss.Color("#F25D94")

	// Grayscale
	LightGray = lipgloss.Color("#D9D9D9")
	Gray      = lipgloss.Color("#8B8B8B")
	DarkGray  = lipgloss.Color("#383838")

	// Resource node colors
	IronOreColor = lipgloss.Color("#C0C0C0") // Silver
	GoldOreColor = lipgloss.Color("#FFD700") // Gold
	WoodColor    = lipgloss.Color("#8B4513") // SaddleBrown
	StoneColor   = lipgloss.Color("#696969") // DimGray

	// Quality colors
	PoorColor   = lipgloss.Color("#8B8B8B") // Gray
	NormalColor = lipgloss.Color("#FFFFFF") // White
	RichColor   = lipgloss.Color("#FFD700") // Gold

	// Status colors
	ActiveColor     = lipgloss.Color("#00FF00") // Lime
	InactiveColor   = lipgloss.Color("#FF0000") // Red
	DepletedColor   = lipgloss.Color("#8B0000") // DarkRed
	RespawningColor = lipgloss.Color("#FFA500") // Orange
)

// Base styles
var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#282828"))

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Align(lipgloss.Center).
			Padding(1, 2)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true).
			Padding(0, 1)

	// Border styles
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Gray).
			Padding(1)

	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PrimaryColor).
				Padding(1)

	// Menu styles
	MenuItemStyle = lipgloss.NewStyle().
			Foreground(LightGray).
			Padding(0, 2)

	SelectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(PrimaryColor).
				Bold(true).
				Padding(0, 2)

	// Info panel styles
	InfoPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(1).
			Width(30)

	// Status bar style
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(DarkGray).
			Padding(0, 1)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true).
				Align(lipgloss.Center).
				Padding(0, 1)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(LightGray).
			Padding(0, 1)

	TableSelectedCellStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(PrimaryColor).
				Padding(0, 1)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Italic(true).
			Padding(1)

	// Grid styles (for chunk visualization)
	GridCellStyle = lipgloss.NewStyle().
			Width(2).
			Height(1).
			Align(lipgloss.Center)

	GridSelectedCellStyle = lipgloss.NewStyle().
				Width(2).
				Height(1).
				Align(lipgloss.Center).
				Background(PrimaryColor).
				Foreground(lipgloss.Color("#FAFAFA"))

	// Form styles
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(Gray).
			Padding(0, 1).
			Width(20)

	FocusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(PrimaryColor).
				Padding(0, 1).
				Width(20)

	ButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(SecondaryColor).
			Bold(true).
			Padding(0, 2).
			MarginRight(1)

	FocusedButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(PrimaryColor).
				Bold(true).
				Padding(0, 2).
				MarginRight(1)
)

// Resource node symbols and styles
const (
	IronOreSymbol = "Fe"  // Iron chemical symbol
	GoldOreSymbol = "Au"  // Gold chemical symbol
	WoodSymbol    = "##"  // Tree/wood representation
	StoneSymbol   = "[]"  // Stone block
	EmptySymbol   = "  "  // Empty space
	CursorSymbol  = "><"  // Cursor indicator

	// Quality indicators
	PoorQualitySymbol   = "o"   // Poor quality
	NormalQualitySymbol = "O"   // Normal quality
	RichQualitySymbol   = "*"   // Rich quality (star)

	// Status indicators
	DepletedSymbol   = "xx"  // Depleted
	RespawningSymbol = ".."  // Respawning
	ActiveSymbol     = "OK"  // Active
)

// GetNodeSymbol returns the appropriate symbol for a resource node
func GetNodeSymbol(nodeType, nodeSubtype int64, isActive bool, currentYield int64) string {
	if !isActive {
		if currentYield <= 0 {
			return DepletedSymbol
		}
		return RespawningSymbol
	}

	var baseSymbol string
	switch nodeType {
	case 1: // IronOre
		baseSymbol = IronOreSymbol
	case 2: // GoldOre
		baseSymbol = GoldOreSymbol
	case 3: // Wood
		baseSymbol = WoodSymbol
	case 4: // Stone
		baseSymbol = StoneSymbol
	default:
		baseSymbol = EmptySymbol
	}

	return baseSymbol
}

// GetNodeColor returns the appropriate color for a resource node
func GetNodeColor(nodeType, nodeSubtype int64, isActive bool, currentYield int64) lipgloss.Color {
	if !isActive {
		if currentYield <= 0 {
			return DepletedColor
		}
		return RespawningColor
	}

	var baseColor lipgloss.Color
	switch nodeType {
	case 1: // IronOre
		baseColor = IronOreColor
	case 2: // GoldOre
		baseColor = GoldOreColor
	case 3: // Wood
		baseColor = WoodColor
	case 4: // Stone
		baseColor = StoneColor
	default:
		baseColor = Gray
	}

	// Modify based on quality
	switch nodeSubtype {
	case 0: // Poor
		return PoorColor
	case 2: // Rich
		return RichColor
	default: // Normal
		return baseColor
	}
}

// Layout helpers
func CenterText(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(text)
}

func LeftText(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Left).Render(text)
}

func RightText(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(text)
}
