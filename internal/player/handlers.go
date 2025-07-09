package player

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// PlayerHandlers contains all HTTP handlers for player operations
type PlayerHandlers struct {
	playerManager *Manager
}

// NewPlayerHandlers creates a new player handlers instance
func NewPlayerHandlers(playerManager *Manager) *PlayerHandlers {
	return &PlayerHandlers{
		playerManager: playerManager,
	}
}

// RegisterRoutes registers all player-related routes
func (h *PlayerHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/players", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.With(AuthMiddleware(h.playerManager)).Post("/logout", h.Logout)
		
		// Protected routes
		r.With(AuthMiddleware(h.playerManager)).Group(func(r chi.Router) {
			r.Get("/me", h.GetProfile)
			r.Put("/me/position", h.UpdatePosition)
			r.Get("/me/inventory", h.GetInventory)
			r.Get("/me/stats", h.GetStats)
		})
		
		// Public routes
		r.Get("/online", h.GetOnlinePlayers)
		r.Get("/{playerID}/profile", h.GetPlayerProfile)
	})
}

// Register handles player registration
func (h *PlayerHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req CreatePlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Failed to decode register request", "error", err)
		writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	player, err := h.playerManager.CreatePlayer(r.Context(), req)
	if err != nil {
		log.Error("Failed to create player", "error", err, "username", req.Username)
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Don't return sensitive information
	response := map[string]interface{}{
		"success":    true,
		"player_id":  player.PlayerID,
		"username":   player.Username,
		"created_at": player.CreatedAt,
	}

	render.JSON(w, r, response)
}

// Login handles player authentication
func (h *PlayerHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Failed to decode login request", "error", err)
		writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract IP address and user agent
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	loginResponse, err := h.playerManager.Login(r.Context(), req, ipAddress, userAgent)
	if err != nil {
		log.Error("Failed to login player", "error", err, "username", req.Username)
		writeErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	render.JSON(w, r, loginResponse)
}

// Logout handles player logout
func (h *PlayerHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Extract session token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeErrorResponse(w, "Authorization header required", http.StatusBadRequest)
		return
	}

	// Extract token from "Bearer <token>" format
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		writeErrorResponse(w, "Invalid authorization format", http.StatusBadRequest)
		return
	}

	sessionToken := tokenParts[1]
	err := h.playerManager.Logout(r.Context(), sessionToken)
	if err != nil {
		log.Error("Failed to logout player", "error", err)
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	}

	render.JSON(w, r, response)
}

// GetProfile returns the current player's profile
func (h *PlayerHandlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	player, err := RequirePlayer(r.Context())
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get player stats
	stats, err := h.playerManager.GetPlayerStats(r.Context(), player.PlayerID)
	if err != nil {
		log.Error("Failed to get player stats", "error", err, "player_id", player.PlayerID)
		// Continue without stats
	}

	// Build profile response
	profile := map[string]interface{}{
		"player_id":       player.PlayerID,
		"username":        player.Username,
		"email":           player.Email,
		"position": map[string]interface{}{
			"world_x":        player.WorldX,
			"world_y":        player.WorldY,
			"world_z":        player.WorldZ,
			"current_chunk_x": player.CurrentChunkX,
			"current_chunk_z": player.CurrentChunkZ,
		},
		"is_online":    player.IsOnline,
		"last_login":   player.LastLogin,
		"last_logout":  player.LastLogout,
		"created_at":   player.CreatedAt,
		"updated_at":   player.UpdatedAt,
	}

	if stats != nil {
		profile["stats"] = stats
	}

	render.JSON(w, r, profile)
}

// UpdatePosition updates the player's position
func (h *PlayerHandlers) UpdatePosition(w http.ResponseWriter, r *http.Request) {
	player, err := RequirePlayer(r.Context())
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req UpdatePositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Failed to decode position update request", "error", err)
		writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.playerManager.UpdatePlayerPosition(r.Context(), player.PlayerID, req)
	if err != nil {
		log.Error("Failed to update player position", "error", err, "player_id", player.PlayerID)
		writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Position updated successfully",
	}

	render.JSON(w, r, response)
}

// GetInventory returns the player's inventory
func (h *PlayerHandlers) GetInventory(w http.ResponseWriter, r *http.Request) {
	player, err := RequirePlayer(r.Context())
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	inventory, err := h.playerManager.GetPlayerInventory(r.Context(), player.PlayerID)
	if err != nil {
		log.Error("Failed to get player inventory", "error", err, "player_id", player.PlayerID)
		writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Format inventory response with resource names
	formattedInventory := make([]map[string]interface{}, len(inventory))
	for i, item := range inventory {
		formattedInventory[i] = map[string]interface{}{
			"inventory_id":     item.InventoryID,
			"resource_type":    item.ResourceType,
			"resource_name":    GetResourceName(item.ResourceType),
			"resource_subtype": item.ResourceSubtype,
			"quality_name":     GetQualityName(item.ResourceSubtype),
			"quantity":         item.Quantity,
			"first_obtained":   item.FirstObtained,
			"last_updated":     item.LastUpdated,
		}
	}

	response := map[string]interface{}{
		"inventory": formattedInventory,
		"total_items": len(formattedInventory),
	}

	render.JSON(w, r, response)
}

// GetStats returns the player's statistics
func (h *PlayerHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	player, err := RequirePlayer(r.Context())
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	stats, err := h.playerManager.GetPlayerStats(r.Context(), player.PlayerID)
	if err != nil {
		log.Error("Failed to get player stats", "error", err, "player_id", player.PlayerID)
		writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, stats)
}

// GetOnlinePlayers returns all currently online players
func (h *PlayerHandlers) GetOnlinePlayers(w http.ResponseWriter, r *http.Request) {
	players, err := h.playerManager.GetOnlinePlayers(r.Context())
	if err != nil {
		log.Error("Failed to get online players", "error", err)
		writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Format response to exclude sensitive information
	formattedPlayers := make([]map[string]interface{}, len(players))
	for i, player := range players {
		formattedPlayers[i] = map[string]interface{}{
			"player_id":       player.PlayerID,
			"username":        player.Username,
			"current_chunk_x": player.CurrentChunkX,
			"current_chunk_z": player.CurrentChunkZ,
			"last_login":      player.LastLogin,
		}
	}

	response := map[string]interface{}{
		"online_players": formattedPlayers,
		"total_count":    len(formattedPlayers),
	}

	render.JSON(w, r, response)
}

// GetPlayerProfile returns a specific player's public profile
func (h *PlayerHandlers) GetPlayerProfile(w http.ResponseWriter, r *http.Request) {
	playerIDStr := chi.URLParam(r, "playerID")
	_, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		writeErrorResponse(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	// Get player by ID (this would need to be implemented)
	// For now, return a placeholder response
	response := map[string]interface{}{
		"error": "Player profiles not yet implemented",
	}

	render.JSON(w, r, response)
}

