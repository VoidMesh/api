package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/player"
	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type Handler struct {
	chunkManager  *chunk.Manager
	playerManager *player.Manager
}

func NewHandler(chunkManager *chunk.Manager, playerManager *player.Manager) *Handler {
	return &Handler{
		chunkManager:  chunkManager,
		playerManager: playerManager,
	}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "voidmesh-api",
		"version":   "1.0.0",
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

func (h *Handler) GetChunk(w http.ResponseWriter, r *http.Request) {
	log.Debug("GetChunk request received", "method", r.Method, "url", r.URL.Path, "remote_addr", r.RemoteAddr)
	start := time.Now()

	chunkXStr := chi.URLParam(r, "x")
	chunkZStr := chi.URLParam(r, "z")
	log.Debug("Parsing chunk coordinates", "chunk_x_str", chunkXStr, "chunk_z_str", chunkZStr)

	chunkX, err := strconv.ParseInt(chunkXStr, 10, 64)
	if err != nil {
		log.Debug("Invalid chunk X coordinate", "chunk_x_str", chunkXStr, "error", err)
		h.renderError(w, r, http.StatusBadRequest, "invalid chunk x coordinate", err)
		return
	}

	chunkZ, err := strconv.ParseInt(chunkZStr, 10, 64)
	if err != nil {
		log.Debug("Invalid chunk Z coordinate", "chunk_z_str", chunkZStr, "error", err)
		h.renderError(w, r, http.StatusBadRequest, "invalid chunk z coordinate", err)
		return
	}

	log.Debug("Chunk coordinates parsed successfully", "chunk_x", chunkX, "chunk_z", chunkZ)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	log.Debug("Loading chunk data", "chunk_x", chunkX, "chunk_z", chunkZ)
	chunkData, err := h.chunkManager.LoadChunk(ctx, chunkX, chunkZ)
	if err != nil {
		log.Error("failed to load chunk", "error", err, "chunk_x", chunkX, "chunk_z", chunkZ, "duration", time.Since(start))
		h.renderError(w, r, http.StatusInternalServerError, "failed to load chunk", err)
		return
	}

	log.Debug("Chunk loaded successfully", "chunk_x", chunkX, "chunk_z", chunkZ, "node_count", len(chunkData.Nodes), "duration", time.Since(start))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, chunkData)
}

func (h *Handler) HarvestNode(w http.ResponseWriter, r *http.Request) {
	log.Debug("HarvestNode request received", "method", r.Method, "url", r.URL.Path, "remote_addr", r.RemoteAddr)
	start := time.Now()

	// Get authenticated player
	authenticatedPlayer, ok := player.GetPlayerFromContext(r.Context())
	if !ok {
		h.renderError(w, r, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	// Parse node ID from URL
	nodeIDStr := chi.URLParam(r, "nodeId")
	log.Debug("Parsing node ID", "node_id_str", nodeIDStr)

	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		log.Debug("Invalid node ID", "node_id_str", nodeIDStr, "error", err)
		h.renderError(w, r, http.StatusBadRequest, "invalid node id", err)
		return
	}

	// Parse request body (future: tool_id, consumables, etc.)
	var req struct {
		// Future fields for tools and consumables
		ToolID         *int64   `json:"tool_id,omitempty"`
		UseConsumables []string `json:"use_consumables,omitempty"`
	}

	log.Debug("Parsing request body", "node_id", nodeID)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Request body is optional for now, just log and continue
		log.Debug("No request body or invalid JSON (continuing)", "error", err, "node_id", nodeID)
	}

	log.Debug("Request parsed", "node_id", nodeID, "player_id", authenticatedPlayer.PlayerID, "username", authenticatedPlayer.Username)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Create harvest context
	harvestCtx := chunk.HarvestContext{
		PlayerID: authenticatedPlayer.PlayerID,
		NodeID:   nodeID,
		// Future: populate tool and character stats
		ToolID: req.ToolID,
	}

	log.Debug("Processing direct harvest", "node_id", nodeID, "player_id", authenticatedPlayer.PlayerID, "username", authenticatedPlayer.Username)
	harvestResult, err := h.chunkManager.HarvestNode(ctx, harvestCtx)
	if err != nil {
		log.Error("failed to harvest node", "error", err, "node_id", nodeID, "player_id", authenticatedPlayer.PlayerID, "duration", time.Since(start))
		h.renderError(w, r, http.StatusBadRequest, "failed to harvest node", err)
		return
	}

	log.Debug("Harvest completed successfully", "node_id", nodeID, "player_id", authenticatedPlayer.PlayerID, "success", harvestResult.Success, "primary_loot_count", len(harvestResult.PrimaryLoot), "duration", time.Since(start))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, harvestResult)
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	errorResponse := chunk.ErrorResponse{
		Error:   message,
		Code:    status,
		Message: message,
	}

	if err != nil {
		log.Error("API error", "error", err, "message", message, "status", status)
		// Don't expose internal errors to the client
		if status >= 500 {
			errorResponse.Error = "Internal server error"
		}
	}

	render.Status(r, status)
	render.JSON(w, r, errorResponse)
}
