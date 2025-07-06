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

	// Harvest session timeout (minutes)
	SessionTimeout = 5
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

type HarvestSession struct {
	SessionID         int64     `json:"session_id"`
	NodeID            int64     `json:"node_id"`
	PlayerID          int64     `json:"player_id"`
	StartedAt         time.Time `json:"started_at"`
	LastActivity      time.Time `json:"last_activity"`
	ResourcesGathered int64     `json:"resources_gathered"`
}

type HarvestRequest struct {
	NodeID        int64 `json:"node_id"`
	PlayerID      int64 `json:"player_id"`
	HarvestAmount int64 `json:"harvest_amount"`
}

type HarvestResponse struct {
	Success           bool  `json:"success"`
	AmountHarvested   int64 `json:"amount_harvested"`
	NodeYieldAfter    int64 `json:"node_yield_after"`
	ResourcesGathered int64 `json:"resources_gathered"`
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
