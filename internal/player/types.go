package player

import (
	"time"
)

// Player represents a player in the game
type Player struct {
	PlayerID       int64     `json:"player_id"`
	Username       string    `json:"username"`
	Email          *string   `json:"email,omitempty"`
	WorldX         float64   `json:"world_x"`
	WorldY         float64   `json:"world_y"`
	WorldZ         float64   `json:"world_z"`
	CurrentChunkX  int64     `json:"current_chunk_x"`
	CurrentChunkZ  int64     `json:"current_chunk_z"`
	IsOnline       bool      `json:"is_online"`
	LastLogin      *time.Time `json:"last_login,omitempty"`
	LastLogout     *time.Time `json:"last_logout,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PlayerStats represents player statistics
type PlayerStats struct {
	StatID                   int64      `json:"stat_id"`
	PlayerID                 int64      `json:"player_id"`
	TotalResourcesHarvested  int64      `json:"total_resources_harvested"`
	TotalHarvestSessions     int64      `json:"total_harvest_sessions"`
	IronOreHarvested         int64      `json:"iron_ore_harvested"`
	GoldOreHarvested         int64      `json:"gold_ore_harvested"`
	WoodHarvested            int64      `json:"wood_harvested"`
	StoneHarvested           int64      `json:"stone_harvested"`
	UniqueNodesDiscovered    int64      `json:"unique_nodes_discovered"`
	TotalNodesHarvested      int64      `json:"total_nodes_harvested"`
	TotalPlaytimeMinutes     int64      `json:"total_playtime_minutes"`
	SessionsCount            int64      `json:"sessions_count"`
	FirstHarvest             *time.Time `json:"first_harvest,omitempty"`
	LastHarvest              *time.Time `json:"last_harvest,omitempty"`
	StatsUpdated             time.Time  `json:"stats_updated"`
}

// PlayerInventory represents a single inventory item
type PlayerInventory struct {
	InventoryID     int64     `json:"inventory_id"`
	PlayerID        int64     `json:"player_id"`
	ResourceType    int64     `json:"resource_type"`
	ResourceSubtype int64     `json:"resource_subtype"`
	Quantity        int64     `json:"quantity"`
	FirstObtained   time.Time `json:"first_obtained"`
	LastUpdated     time.Time `json:"last_updated"`
}

// PlayerSession represents an active player session
type PlayerSession struct {
	SessionID    int64     `json:"session_id"`
	PlayerID     int64     `json:"player_id"`
	SessionToken string    `json:"session_token"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	UserAgent    *string   `json:"user_agent,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastActivity time.Time `json:"last_activity"`
}

// CreatePlayerRequest represents the request to create a new player
type CreatePlayerRequest struct {
	Username string  `json:"username" validate:"required,min=3,max=32"`
	Password string  `json:"password" validate:"required,min=8"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
}

// LoginRequest represents the request to log in
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	Success      bool           `json:"success"`
	SessionToken string         `json:"session_token"`
	Player       Player         `json:"player"`
	ExpiresAt    time.Time      `json:"expires_at"`
}

// UpdatePositionRequest represents the request to update player position
type UpdatePositionRequest struct {
	WorldX float64 `json:"world_x"`
	WorldY float64 `json:"world_y"`
	WorldZ float64 `json:"world_z"`
}

// PlayerProfile represents a player's public profile
type PlayerProfile struct {
	Username              string     `json:"username"`
	IsOnline              bool       `json:"is_online"`
	TotalResourcesHarvested int64    `json:"total_resources_harvested"`
	TotalHarvestSessions    int64    `json:"total_harvest_sessions"`
	SessionsCount           int64    `json:"sessions_count"`
	TotalPlaytimeMinutes    int64    `json:"total_playtime_minutes"`
	FirstHarvest           *time.Time `json:"first_harvest,omitempty"`
	LastHarvest            *time.Time `json:"last_harvest,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
}

// Resource type constants
const (
	ResourceIronOre = 1
	ResourceGoldOre = 2
	ResourceWood    = 3
	ResourceStone   = 4
)

// Resource subtype constants  
const (
	QualityPoor   = 0
	QualityNormal = 1
	QualityRich   = 2
)

// GetResourceName returns the human-readable name for a resource type
func GetResourceName(resourceType int64) string {
	switch resourceType {
	case ResourceIronOre:
		return "Iron Ore"
	case ResourceGoldOre:
		return "Gold Ore"
	case ResourceWood:
		return "Wood"
	case ResourceStone:
		return "Stone"
	default:
		return "Unknown"
	}
}

// GetQualityName returns the human-readable name for a quality level
func GetQualityName(quality int64) string {
	switch quality {
	case QualityPoor:
		return "Poor"
	case QualityNormal:
		return "Normal"
	case QualityRich:
		return "Rich"
	default:
		return "Unknown"
	}
}


// SessionTimeout defines how long a session remains valid (in hours)
const SessionTimeout = 24

// MaxSessionsPerPlayer defines the maximum number of active sessions per player
const MaxSessionsPerPlayer = 5