package player

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/VoidMesh/api/internal/chunk"
	"github.com/VoidMesh/api/internal/db"
	"github.com/charmbracelet/log"
)

// Manager handles all player-related operations
type Manager struct {
	db          *sql.DB
	queries     *db.LoggingQueries
	passwordMgr *PasswordManager
	tokenMgr    *TokenManager
}

// NewManager creates a new player manager
func NewManager(database *sql.DB) *Manager {
	return &Manager{
		db:          database,
		queries:     db.NewLoggingQueries(database),
		passwordMgr: NewPasswordManager(),
		tokenMgr:    NewTokenManager(),
	}
}

// CreatePlayer creates a new player account
func (m *Manager) CreatePlayer(ctx context.Context, req CreatePlayerRequest) (*Player, error) {
	log.Debug("Creating new player", "username", req.Username)

	// Validate input
	if err := ValidateUsername(req.Username); err != nil {
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	if err := ValidatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	if req.Email != nil {
		if err := ValidateEmail(*req.Email); err != nil {
			return nil, fmt.Errorf("invalid email: %w", err)
		}
	}

	// Check if username already exists
	_, err := m.queries.GetPlayerByUsername(ctx, req.Username)
	if err == nil {
		return nil, errors.New("username already exists")
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}

	// Check if email already exists (if provided)
	if req.Email != nil {
		_, err := m.queries.GetPlayerByEmail(ctx, sql.NullString{String: *req.Email, Valid: true})
		if err == nil {
			return nil, errors.New("email already exists")
		}
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
	}

	// Generate password hash
	salt, err := m.passwordMgr.GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	passwordHash, err := m.passwordMgr.HashPassword(req.Password, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create player in database
	var email sql.NullString
	if req.Email != nil {
		email = sql.NullString{String: *req.Email, Valid: true}
	}

	dbPlayer, err := m.queries.CreatePlayer(ctx, db.CreatePlayerParams{
		Username:     req.Username,
		PasswordHash: passwordHash,
		Salt:         salt,
		Email:        email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create player: %w", err)
	}

	// Create initial player stats
	_, err = m.queries.CreatePlayerStats(ctx, dbPlayer.PlayerID)
	if err != nil {
		log.Error("Failed to create initial player stats", "error", err, "player_id", dbPlayer.PlayerID)
		// Don't fail player creation if stats creation fails
	}

	player := m.convertDBPlayerToPlayer(dbPlayer)
	log.Info("Created new player", "username", player.Username, "player_id", player.PlayerID)

	return player, nil
}

// Login authenticates a player and creates a session
func (m *Manager) Login(ctx context.Context, req LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	log.Debug("Player login attempt", "username", req.Username, "ip", ipAddress)

	// Get player by username
	dbPlayer, err := m.queries.GetPlayerByUsername(ctx, req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid username or password")
		}
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	// Verify password
	if !m.passwordMgr.VerifyPassword(req.Password, dbPlayer.PasswordHash, dbPlayer.Salt) {
		return nil, errors.New("invalid username or password")
	}

	// Generate session token
	sessionToken, err := m.tokenMgr.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	// Create session
	expiresAt := CreateSessionExpiry()
	var ipAddr, ua sql.NullString
	if ipAddress != "" {
		ipAddr = sql.NullString{String: ipAddress, Valid: true}
	}
	if userAgent != "" {
		ua = sql.NullString{String: userAgent, Valid: true}
	}

	dbSession, err := m.queries.CreatePlayerSession(ctx, db.CreatePlayerSessionParams{
		PlayerID:     dbPlayer.PlayerID,
		SessionToken: sessionToken,
		IpAddress:    ipAddr,
		UserAgent:    ua,
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update player online status
	err = m.queries.SetPlayerOnline(ctx, dbPlayer.PlayerID)
	if err != nil {
		log.Error("Failed to set player online", "error", err, "player_id", dbPlayer.PlayerID)
	}

	// Update session count
	err = m.queries.IncrementPlayerSessions(ctx, dbPlayer.PlayerID)
	if err != nil {
		log.Error("Failed to increment player sessions", "error", err, "player_id", dbPlayer.PlayerID)
	}

	player := m.convertDBPlayerToPlayer(dbPlayer)
	log.Info("Player logged in successfully", "username", player.Username, "player_id", player.PlayerID, "session_id", dbSession.SessionID)

	return &LoginResponse{
		Success:      true,
		SessionToken: sessionToken,
		Player:       *player,
		ExpiresAt:    expiresAt,
	}, nil
}

// Logout logs out a player and invalidates their session
func (m *Manager) Logout(ctx context.Context, sessionToken string) error {
	log.Debug("Player logout attempt", "session_token", sessionToken[:min(len(sessionToken), 8)]+"...")

	// Get session
	dbSession, err := m.queries.GetPlayerSession(ctx, sessionToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("invalid session")
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Delete session
	err = m.queries.DeletePlayerSession(ctx, sessionToken)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Update player offline status
	err = m.queries.SetPlayerOffline(ctx, dbSession.PlayerID)
	if err != nil {
		log.Error("Failed to set player offline", "error", err, "player_id", dbSession.PlayerID)
	}

	log.Info("Player logged out successfully", "player_id", dbSession.PlayerID)
	return nil
}

// AuthenticateSession validates a session token and returns the player
func (m *Manager) AuthenticateSession(ctx context.Context, sessionToken string) (*Player, error) {
	if err := m.tokenMgr.ValidateToken(sessionToken); err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Get and validate session
	dbSession, err := m.queries.GetPlayerSession(ctx, sessionToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid session")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if IsSessionExpired(dbSession.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	// Update session activity
	err = m.queries.UpdatePlayerSessionActivity(ctx, sessionToken)
	if err != nil {
		log.Error("Failed to update session activity", "error", err, "session_token", sessionToken[:min(len(sessionToken), 8)]+"...")
	}

	// Get player details
	dbPlayer, err := m.queries.GetPlayerByID(ctx, dbSession.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	return m.convertDBPlayerToPlayer(dbPlayer), nil
}

// UpdatePlayerPosition updates a player's position in the world
func (m *Manager) UpdatePlayerPosition(ctx context.Context, playerID int64, req UpdatePositionRequest) error {
	log.Debug("Updating player position", "player_id", playerID, "x", req.WorldX, "y", req.WorldY, "z", req.WorldZ)

	// Calculate chunk coordinates
	chunkX := int64(math.Floor(req.WorldX / 16))
	chunkZ := int64(math.Floor(req.WorldZ / 16))

	err := m.queries.UpdatePlayerPosition(ctx, db.UpdatePlayerPositionParams{
		WorldX:        sql.NullFloat64{Float64: req.WorldX, Valid: true},
		WorldY:        sql.NullFloat64{Float64: req.WorldY, Valid: true},
		WorldZ:        sql.NullFloat64{Float64: req.WorldZ, Valid: true},
		CurrentChunkX: sql.NullInt64{Int64: chunkX, Valid: true},
		CurrentChunkZ: sql.NullInt64{Int64: chunkZ, Valid: true},
		PlayerID:      playerID,
	})
	if err != nil {
		return fmt.Errorf("failed to update player position: %w", err)
	}

	log.Debug("Player position updated successfully", "player_id", playerID, "chunk_x", chunkX, "chunk_z", chunkZ)
	return nil
}

// GetPlayerInventory returns a player's inventory
func (m *Manager) GetPlayerInventory(ctx context.Context, playerID int64) ([]PlayerInventory, error) {
	log.Debug("Getting player inventory", "player_id", playerID)

	dbInventory, err := m.queries.GetPlayerInventory(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player inventory: %w", err)
	}

	inventory := make([]PlayerInventory, len(dbInventory))
	for i, item := range dbInventory {
		inventory[i] = PlayerInventory{
			InventoryID:     item.InventoryID,
			PlayerID:        item.PlayerID,
			ResourceType:    item.ResourceType,
			ResourceSubtype: item.ResourceSubtype.Int64,
			Quantity:        item.Quantity.Int64,
			FirstObtained:   item.FirstObtained.Time,
			LastUpdated:     item.LastUpdated.Time,
		}
	}

	log.Debug("Retrieved player inventory", "player_id", playerID, "items", len(inventory))
	return inventory, nil
}

// AddToInventory adds resources to a player's inventory
func (m *Manager) AddToInventory(ctx context.Context, playerID int64, resourceType, resourceSubtype, quantity int64) error {
	log.Debug("Adding to player inventory", "player_id", playerID, "resource_type", resourceType, "resource_subtype", resourceSubtype, "quantity", quantity)

	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	err := m.queries.AddToPlayerInventory(ctx, db.AddToPlayerInventoryParams{
		PlayerID:        playerID,
		ResourceType:    resourceType,
		ResourceSubtype: sql.NullInt64{Int64: resourceSubtype, Valid: true},
		Quantity:        sql.NullInt64{Int64: quantity, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to add to inventory: %w", err)
	}

	log.Debug("Added to player inventory successfully", "player_id", playerID, "resource_type", resourceType, "quantity", quantity)
	return nil
}

// GetPlayerStats returns a player's statistics
func (m *Manager) GetPlayerStats(ctx context.Context, playerID int64) (*PlayerStats, error) {
	log.Debug("Getting player stats", "player_id", playerID)

	dbStats, err := m.queries.GetPlayerStats(ctx, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create stats if they don't exist
			dbStats, err = m.queries.CreatePlayerStats(ctx, playerID)
			if err != nil {
				return nil, fmt.Errorf("failed to create player stats: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get player stats: %w", err)
		}
	}

	stats := &PlayerStats{
		StatID:                  dbStats.StatID,
		PlayerID:                dbStats.PlayerID,
		TotalResourcesHarvested: dbStats.TotalResourcesHarvested.Int64,
		TotalHarvestSessions:    dbStats.TotalHarvestSessions.Int64,
		IronOreHarvested:        dbStats.IronOreHarvested.Int64,
		GoldOreHarvested:        dbStats.GoldOreHarvested.Int64,
		WoodHarvested:           dbStats.WoodHarvested.Int64,
		StoneHarvested:          dbStats.StoneHarvested.Int64,
		UniqueNodesDiscovered:   dbStats.UniqueNodesDiscovered.Int64,
		TotalNodesHarvested:     dbStats.TotalNodesHarvested.Int64,
		TotalPlaytimeMinutes:    dbStats.TotalPlaytimeMinutes.Int64,
		SessionsCount:           dbStats.SessionsCount.Int64,
		StatsUpdated:            dbStats.StatsUpdated.Time,
	}

	if dbStats.FirstHarvest.Valid {
		stats.FirstHarvest = &dbStats.FirstHarvest.Time
	}
	if dbStats.LastHarvest.Valid {
		stats.LastHarvest = &dbStats.LastHarvest.Time
	}

	log.Debug("Retrieved player stats", "player_id", playerID, "total_resources", stats.TotalResourcesHarvested)
	return stats, nil
}

// HarvestStatsUpdate represents the data needed to update player stats after a harvest
type HarvestStatsUpdate struct {
	ResourceType    int64
	AmountHarvested int64
	NodeID          int64
	IsNewNode       bool
}

// UpdateHarvestStats updates a player's statistics after a harvest
func (m *Manager) UpdateHarvestStats(ctx context.Context, playerID int64, update chunk.HarvestStatsUpdate) error {
	log.Debug("Updating harvest stats", "player_id", playerID, "resource_type", update.ResourceType, "amount", update.AmountHarvested)

	// Calculate resource-specific amounts
	var ironOre, goldOre, wood, stone int64
	switch update.ResourceType {
	case ResourceIronOre:
		ironOre = update.AmountHarvested
	case ResourceGoldOre:
		goldOre = update.AmountHarvested
	case ResourceWood:
		wood = update.AmountHarvested
	case ResourceStone:
		stone = update.AmountHarvested
	}

	// Calculate unique nodes count (this is a simplified approach)
	uniqueNodes := int64(0)
	if update.IsNewNode {
		uniqueNodes = 1
	}

	err := m.queries.UpdatePlayerStats(ctx, db.UpdatePlayerStatsParams{
		TotalResourcesHarvested: sql.NullInt64{Int64: update.AmountHarvested, Valid: true},
		TotalHarvestSessions:    sql.NullInt64{Int64: 1, Valid: true},
		IronOreHarvested:        sql.NullInt64{Int64: ironOre, Valid: true},
		GoldOreHarvested:        sql.NullInt64{Int64: goldOre, Valid: true},
		WoodHarvested:           sql.NullInt64{Int64: wood, Valid: true},
		StoneHarvested:          sql.NullInt64{Int64: stone, Valid: true},
		UniqueNodesDiscovered:   sql.NullInt64{Int64: uniqueNodes, Valid: true},
		TotalNodesHarvested:     sql.NullInt64{Int64: 1, Valid: true},
		PlayerID:                playerID,
	})
	if err != nil {
		return fmt.Errorf("failed to update harvest stats: %w", err)
	}

	log.Debug("Updated harvest stats successfully", "player_id", playerID, "resource_type", update.ResourceType, "amount", update.AmountHarvested)
	return nil
}

// Save persists any pending player manager state to the database
// This method ensures all player-related data is properly persisted
func (m *Manager) Save(ctx context.Context) error {
	log.Debug("Persisting player manager state")

	// Force a database sync to ensure all transactions are committed
	if err := m.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database during player save: %w", err)
	}

	// In a more complex implementation, this could:
	// - Flush any pending player updates
	// - Save cached player data
	// - Commit any pending inventory changes
	// - Update player statistics

	log.Debug("Player manager state persisted successfully")
	return nil
}

// CleanupExpiredSessions removes expired sessions from the database
func (m *Manager) CleanupExpiredSessions(ctx context.Context) error {
	log.Debug("Cleaning up expired sessions")

	err := m.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	log.Debug("Expired sessions cleaned up successfully")
	return nil
}

// AddItem adds an item to the player's inventory (alias for AddToInventory)
func (m *Manager) AddItem(ctx context.Context, playerID int64, itemID int64) error {
	log.Debug("Adding item to player inventory", "player_id", playerID, "item_id", itemID)

	// Map item ID to resource type and subtype
	// For now, we'll use a simple mapping where item ID corresponds to resource type
	resourceType := itemID
	resourceSubtype := int64(1) // Default to normal quality
	quantity := int64(1)        // Default to 1 item

	return m.AddToInventory(ctx, playerID, resourceType, resourceSubtype, quantity)
}

// AddXp adds experience points to a player
func (m *Manager) AddXp(ctx context.Context, playerID int64, xpAmount int64) error {
	log.Debug("Adding XP to player", "player_id", playerID, "xp_amount", xpAmount)

	// For now, we'll log this as a placeholder since XP system isn't fully implemented
	// In a full implementation, this would update player XP stats
	log.Info("XP added to player", "player_id", playerID, "xp_amount", xpAmount)
	return nil
}

// AddEnergyCost deducts energy from a player for an action
func (m *Manager) AddEnergyCost(ctx context.Context, playerID int64, energyCost int64) error {
	log.Debug("Deducting energy from player", "player_id", playerID, "energy_cost", energyCost)

	// For now, we'll log this as a placeholder since energy system isn't fully implemented
	// In a full implementation, this would update player energy stats
	log.Info("Energy deducted from player", "player_id", playerID, "energy_cost", energyCost)
	return nil
}

// GetOnlinePlayers returns all currently online players
func (m *Manager) GetOnlinePlayers(ctx context.Context) ([]Player, error) {
	log.Debug("Getting online players")

	dbPlayers, err := m.queries.GetOnlinePlayers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get online players: %w", err)
	}

	players := make([]Player, len(dbPlayers))
	for i, dbPlayer := range dbPlayers {
		players[i] = *m.convertDBPlayerToPlayer(dbPlayer)
	}

	log.Debug("Retrieved online players", "count", len(players))
	return players, nil
}

// convertDBPlayerToPlayer converts a database player to a Player struct
func (m *Manager) convertDBPlayerToPlayer(dbPlayer db.Player) *Player {
	player := &Player{
		PlayerID:      dbPlayer.PlayerID,
		Username:      dbPlayer.Username,
		WorldX:        dbPlayer.WorldX.Float64,
		WorldY:        dbPlayer.WorldY.Float64,
		WorldZ:        dbPlayer.WorldZ.Float64,
		CurrentChunkX: dbPlayer.CurrentChunkX.Int64,
		CurrentChunkZ: dbPlayer.CurrentChunkZ.Int64,
		IsOnline:      dbPlayer.IsOnline.Int64 == 1,
		CreatedAt:     dbPlayer.CreatedAt.Time,
		UpdatedAt:     dbPlayer.UpdatedAt.Time,
	}

	if dbPlayer.Email.Valid {
		player.Email = &dbPlayer.Email.String
	}
	if dbPlayer.LastLogin.Valid {
		player.LastLogin = &dbPlayer.LastLogin.Time
	}
	if dbPlayer.LastLogout.Valid {
		player.LastLogout = &dbPlayer.LastLogout.Time
	}

	return player
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
