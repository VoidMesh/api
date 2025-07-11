package chunk

import (
	"time"
)

const (
	ChunkSize = 16

	// Node types
	IronOre = 1
	GoldOre = 2
	Wood    = 3
	Stone   = 4

	// Node subtypes
	PoorQuality   = 0
	NormalQuality = 1
	RichQuality   = 2

	// Spawn types
	RandomSpawn     = 0
	StaticDaily     = 1
	StaticPermanent = 2
)

type ResourceNode struct {
	NodeID           int64      `json:"node_id"`
	ChunkX           int64      `json:"chunk_x"`
	ChunkZ           int64      `json:"chunk_z"`
	LocalX           int64      `json:"local_x"`
	LocalZ           int64      `json:"local_z"`
	NodeType         int64      `json:"node_type"`
	NodeSubtype      int64      `json:"node_subtype"`
	MaxYield         int64      `json:"max_yield"`
	CurrentYield     int64      `json:"current_yield"`
	RegenerationRate int64      `json:"regeneration_rate"`
	SpawnedAt        time.Time  `json:"spawned_at"`
	LastHarvest      *time.Time `json:"last_harvest,omitempty"`
	RespawnTimer     *time.Time `json:"respawn_timer,omitempty"`
	SpawnType        int64      `json:"spawn_type"`
	IsActive         bool       `json:"is_active"`
}

// Future-ready harvest models

type HarvestContext struct {
	PlayerID int64 `json:"player_id"`
	NodeID   int64 `json:"node_id"`
	// Future: Character stats, tools, bonuses
	CharacterStats *CharacterStats `json:"character_stats,omitempty"`
	ToolID         *int64          `json:"tool_id,omitempty"`
	ToolStats      *ToolStats      `json:"tool_stats,omitempty"`
	Bonuses        []HarvestBonus  `json:"bonuses,omitempty"`
}

type HarvestResult struct {
	Success        bool           `json:"success"`
	PrimaryLoot    []LootItem     `json:"primary_loot"`
	BonusLoot      []LootItem     `json:"bonus_loot"`
	NodeState      NodeState      `json:"node_state"`
	HarvestDetails HarvestDetails `json:"harvest_details"`
	// Future: Experience, tool wear, etc.
	ExperienceGained *int64   `json:"experience_gained,omitempty"`
	ToolWear         *float64 `json:"tool_wear,omitempty"`
}

type LootItem struct {
	ItemType    int64   `json:"item_type"`
	ItemSubtype int64   `json:"item_subtype"`
	Quantity    int64   `json:"quantity"`
	Quality     float64 `json:"quality"`
	Source      string  `json:"source"` // "primary", "bonus", "tool_bonus"
}

type NodeState struct {
	CurrentYield int64      `json:"current_yield"`
	IsActive     bool       `json:"is_active"`
	RespawnTimer *time.Time `json:"respawn_timer"`
	LastHarvest  *time.Time `json:"last_harvest"`
}

type HarvestDetails struct {
	BaseYield  int64   `json:"base_yield"`
	StatBonus  int64   `json:"stat_bonus"`
	ToolBonus  int64   `json:"tool_bonus"`
	TotalYield int64   `json:"total_yield"`
	BonusRolls int64   `json:"bonus_rolls"`
	LuckFactor float64 `json:"luck_factor"`
}

// Future placeholder models

type CharacterStats struct {
	PlayerID    int64   `json:"player_id"`
	MiningLevel int64   `json:"mining_level"`
	MiningBonus float64 `json:"mining_bonus"`
	LuckBonus   float64 `json:"luck_bonus"`
	// More stats as needed
}

type ToolStats struct {
	ToolID          int64   `json:"tool_id"`
	ToolType        string  `json:"tool_type"`
	YieldMultiplier float64 `json:"yield_multiplier"`
	BonusChance     float64 `json:"bonus_chance"`
	Durability      int64   `json:"durability"`
	CurrentWear     float64 `json:"current_wear"`
}

type HarvestBonus struct {
	BonusType  string  `json:"bonus_type"`
	BonusValue float64 `json:"bonus_value"`
	Source     string  `json:"source"` // "consumable", "buff", "equipment"
	Duration   *int64  `json:"duration,omitempty"`
}

type BonusMaterial struct {
	NodeType     int64   `json:"node_type"`
	BonusType    int64   `json:"bonus_type"`
	BaseChance   float64 `json:"base_chance"`
	StatModifier string  `json:"stat_modifier"` // which stat affects this
}

type HarvestStatsUpdate struct {
	ResourceType    int64 `json:"resource_type"`
	AmountHarvested int64 `json:"amount_harvested"`
	NodeID          int64 `json:"node_id"`
	IsNewNode       bool  `json:"is_new_node"`
}

type ChunkResponse struct {
	ChunkX int64          `json:"chunk_x"`
	ChunkZ int64          `json:"chunk_z"`
	Nodes  []ResourceNode `json:"nodes"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}
