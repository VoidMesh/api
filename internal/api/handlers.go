package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/VoidMesh/api/internal/chunk"
	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type Handler struct {
	chunkManager *chunk.Manager
}

func NewHandler(chunkManager *chunk.Manager) *Handler {
	return &Handler{
		chunkManager: chunkManager,
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

func (h *Handler) StartHarvest(w http.ResponseWriter, r *http.Request) {
	log.Debug("StartHarvest request received", "method", r.Method, "url", r.URL.Path, "remote_addr", r.RemoteAddr)
	start := time.Now()

	var req struct {
		NodeID   int64 `json:"node_id"`
		PlayerID int64 `json:"player_id"`
	}

	log.Debug("Parsing request body")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("Invalid request body", "error", err)
		h.renderError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	log.Debug("Request parsed", "node_id", req.NodeID, "player_id", req.PlayerID)

	if req.NodeID <= 0 {
		log.Debug("Invalid node_id", "node_id", req.NodeID)
		h.renderError(w, r, http.StatusBadRequest, "node_id must be positive", nil)
		return
	}

	if req.PlayerID <= 0 {
		log.Debug("Invalid player_id", "player_id", req.PlayerID)
		h.renderError(w, r, http.StatusBadRequest, "player_id must be positive", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	log.Debug("Starting harvest session", "node_id", req.NodeID, "player_id", req.PlayerID)
	session, err := h.chunkManager.StartHarvest(ctx, req.NodeID, req.PlayerID)
	if err != nil {
		log.Error("failed to start harvest", "error", err, "node_id", req.NodeID, "player_id", req.PlayerID, "duration", time.Since(start))
		h.renderError(w, r, http.StatusBadRequest, "failed to start harvest", err)
		return
	}

	log.Debug("Harvest session started successfully", "node_id", req.NodeID, "player_id", req.PlayerID, "session_id", session.SessionID, "duration", time.Since(start))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, session)
}

func (h *Handler) HarvestResource(w http.ResponseWriter, r *http.Request) {
	log.Debug("HarvestResource request received", "method", r.Method, "url", r.URL.Path, "remote_addr", r.RemoteAddr)
	start := time.Now()

	sessionIDStr := chi.URLParam(r, "sessionId")
	log.Debug("Parsing session ID", "session_id_str", sessionIDStr)

	sessionID, err := strconv.ParseInt(sessionIDStr, 10, 64)
	if err != nil {
		log.Debug("Invalid session ID", "session_id_str", sessionIDStr, "error", err)
		h.renderError(w, r, http.StatusBadRequest, "invalid session id", err)
		return
	}

	var req struct {
		HarvestAmount int64 `json:"harvest_amount"`
	}

	log.Debug("Parsing request body", "session_id", sessionID)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("Invalid request body", "error", err, "session_id", sessionID)
		h.renderError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	log.Debug("Request parsed", "session_id", sessionID, "harvest_amount", req.HarvestAmount)

	if req.HarvestAmount <= 0 {
		log.Debug("Invalid harvest amount", "harvest_amount", req.HarvestAmount, "session_id", sessionID)
		h.renderError(w, r, http.StatusBadRequest, "harvest_amount must be positive", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	log.Debug("Processing harvest", "session_id", sessionID, "harvest_amount", req.HarvestAmount)
	harvestResult, err := h.chunkManager.HarvestResource(ctx, sessionID, req.HarvestAmount)
	if err != nil {
		log.Error("failed to harvest resource", "error", err, "session_id", sessionID, "harvest_amount", req.HarvestAmount, "duration", time.Since(start))
		h.renderError(w, r, http.StatusBadRequest, "failed to harvest resource", err)
		return
	}

	log.Debug("Harvest completed successfully", "session_id", sessionID, "harvest_amount", req.HarvestAmount, "actual_harvested", harvestResult.AmountHarvested, "node_yield_after", harvestResult.NodeYieldAfter, "duration", time.Since(start))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, harvestResult)
}

func (h *Handler) GetPlayerSessions(w http.ResponseWriter, r *http.Request) {
	log.Debug("GetPlayerSessions request received", "method", r.Method, "url", r.URL.Path, "remote_addr", r.RemoteAddr)
	start := time.Now()

	playerIDStr := chi.URLParam(r, "playerId")
	log.Debug("Parsing player ID", "player_id_str", playerIDStr)

	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		log.Debug("Invalid player ID", "player_id_str", playerIDStr, "error", err)
		h.renderError(w, r, http.StatusBadRequest, "invalid player id", err)
		return
	}

	log.Debug("Player ID parsed successfully", "player_id", playerID)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	log.Debug("Getting player sessions", "player_id", playerID)
	sessions, err := h.chunkManager.GetPlayerSessions(ctx, playerID)
	if err != nil {
		log.Error("failed to get player sessions", "error", err, "player_id", playerID, "duration", time.Since(start))
		h.renderError(w, r, http.StatusInternalServerError, "failed to get player sessions", err)
		return
	}

	log.Debug("Player sessions retrieved successfully", "player_id", playerID, "session_count", len(sessions), "duration", time.Since(start))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"player_id": playerID,
		"sessions":  sessions,
	})
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
