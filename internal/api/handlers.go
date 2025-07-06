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
	chunkXStr := chi.URLParam(r, "x")
	chunkZStr := chi.URLParam(r, "z")

	chunkX, err := strconv.ParseInt(chunkXStr, 10, 64)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "invalid chunk x coordinate", err)
		return
	}

	chunkZ, err := strconv.ParseInt(chunkZStr, 10, 64)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "invalid chunk z coordinate", err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	chunkData, err := h.chunkManager.LoadChunk(ctx, chunkX, chunkZ)
	if err != nil {
		log.Error("failed to load chunk", "error", err, "chunk_x", chunkX, "chunk_z", chunkZ)
		h.renderError(w, r, http.StatusInternalServerError, "failed to load chunk", err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, chunkData)
}

func (h *Handler) StartHarvest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID   int64 `json:"node_id"`
		PlayerID int64 `json:"player_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.NodeID <= 0 {
		h.renderError(w, r, http.StatusBadRequest, "node_id must be positive", nil)
		return
	}

	if req.PlayerID <= 0 {
		h.renderError(w, r, http.StatusBadRequest, "player_id must be positive", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	session, err := h.chunkManager.StartHarvest(ctx, req.NodeID, req.PlayerID)
	if err != nil {
		log.Error("failed to start harvest", "error", err, "node_id", req.NodeID, "player_id", req.PlayerID)
		h.renderError(w, r, http.StatusBadRequest, "failed to start harvest", err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, session)
}

func (h *Handler) HarvestResource(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chi.URLParam(r, "sessionId")
	sessionID, err := strconv.ParseInt(sessionIDStr, 10, 64)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "invalid session id", err)
		return
	}

	var req struct {
		HarvestAmount int64 `json:"harvest_amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.HarvestAmount <= 0 {
		h.renderError(w, r, http.StatusBadRequest, "harvest_amount must be positive", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	harvestResult, err := h.chunkManager.HarvestResource(ctx, sessionID, req.HarvestAmount)
	if err != nil {
		log.Error("failed to harvest resource", "error", err, "session_id", sessionID, "harvest_amount", req.HarvestAmount)
		h.renderError(w, r, http.StatusBadRequest, "failed to harvest resource", err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, harvestResult)
}

func (h *Handler) GetPlayerSessions(w http.ResponseWriter, r *http.Request) {
	playerIDStr := chi.URLParam(r, "playerId")
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		h.renderError(w, r, http.StatusBadRequest, "invalid player id", err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sessions, err := h.chunkManager.GetPlayerSessions(ctx, playerID)
	if err != nil {
		log.Error("failed to get player sessions", "error", err, "player_id", playerID)
		h.renderError(w, r, http.StatusInternalServerError, "failed to get player sessions", err)
		return
	}

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
