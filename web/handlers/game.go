package handlers

import (
	"github.com/VoidMesh/platform/web/grpc"
	chunkV1 "github.com/VoidMesh/platform/api/proto/chunk/v1"
	worldV1 "github.com/VoidMesh/platform/api/proto/world/v1"
	viewGameV1 "github.com/VoidMesh/platform/web/views/game"
	"github.com/gofiber/fiber/v2"
)

type Game struct{ *App }

// CharacterSelect displays the character selection page
func (h *Game) CharacterSelect(c *fiber.Ctx) error {
	// Get JWT token from locals
	jwtToken := c.Locals("jwt_token").(string)
	ctx := grpc.WithAuth(c.Context(), jwtToken)

	// Get user ID from JWT token
	userID := c.Locals("user_id").(string)

	// Get characters for this user
	req := &worldV1.GetCharactersByUserRequest{UserId: userID}
	resp, err := h.API.WorldService.GetCharactersByUser(ctx, req)
	if err != nil {
		return err
	}

	return renderTempl(c, viewGameV1.CharacterSelect(c, resp))
}

// CreateCharacter handles character creation
func (h *Game) CreateCharacter(c *fiber.Ctx) error {
	name := c.FormValue("name")
	if name == "" {
		return c.Status(400).SendString("Character name is required")
	}

	// Get JWT token from locals
	jwtToken := c.Locals("jwt_token").(string)
	ctx := grpc.WithAuth(c.Context(), jwtToken)

	// Get user ID from JWT token
	userID := c.Locals("user_id").(string)

	// Create character
	req := &worldV1.CreateCharacterRequest{
		UserId: userID,
		Name:   name,
		SpawnX: 0, // Default spawn at origin
		SpawnY: 0,
	}

	_, err := h.API.WorldService.CreateCharacter(ctx, req)
	if err != nil {
		return c.Status(400).SendString("Failed to create character: " + err.Error())
	}

	// Redirect back to character select
	return c.Redirect("/game/characters")
}

// GameWorld displays the game world view for a character
func (h *Game) GameWorld(c *fiber.Ctx) error {
	characterID := c.Params("characterId")
	if characterID == "" {
		return c.Status(400).SendString("Character ID is required")
	}

	// Get JWT token from locals
	jwtToken := c.Locals("jwt_token").(string)
	ctx := grpc.WithAuth(c.Context(), jwtToken)

	// Get character details
	charReq := &worldV1.GetCharacterRequest{CharacterId: characterID}
	charResp, err := h.API.WorldService.GetCharacter(ctx, charReq)
	if err != nil {
		return c.Status(404).SendString("Character not found")
	}

	character := charResp.Character

	// Calculate which chunks to load (3x3 grid around character)
	chunkX := character.ChunkX
	chunkY := character.ChunkY
	
	// Get chunks in a 3x3 grid around the character
	chunksReq := &chunkV1.GetChunksRequest{
		MinChunkX: chunkX - 1,
		MaxChunkX: chunkX + 1,
		MinChunkY: chunkY - 1,
		MaxChunkY: chunkY + 1,
	}

	chunksResp, err := h.API.ChunkService.GetChunks(ctx, chunksReq)
	if err != nil {
		return c.Status(500).SendString("Failed to load world data")
	}

	return renderTempl(c, viewGameV1.WorldView(c, character, chunksResp.Chunks))
}

// MoveCharacter handles character movement via AJAX
func (h *Game) MoveCharacter(c *fiber.Ctx) error {
	characterID := c.Params("characterId")
	if characterID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Character ID is required"})
	}

	// Parse movement direction
	direction := c.FormValue("direction")
	if direction == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Direction is required"})
	}

	// Get JWT token from locals
	jwtToken := c.Locals("jwt_token").(string)
	ctx := grpc.WithAuth(c.Context(), jwtToken)

	// Get current character position
	charReq := &worldV1.GetCharacterRequest{CharacterId: characterID}
	charResp, err := h.API.WorldService.GetCharacter(ctx, charReq)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Character not found"})
	}

	character := charResp.Character
	newX := character.X
	newY := character.Y

	// Calculate new position based on direction
	switch direction {
	case "up":
		newY--
	case "down":
		newY++
	case "left":
		newX--
	case "right":
		newX++
	default:
		return c.Status(400).JSON(fiber.Map{"error": "Invalid direction"})
	}

	// Move character
	moveReq := &worldV1.MoveCharacterRequest{
		CharacterId: characterID,
		NewX:        newX,
		NewY:        newY,
	}

	moveResp, err := h.API.WorldService.MoveCharacter(ctx, moveReq)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to move character"})
	}

	if !moveResp.Success {
		return c.Status(400).JSON(fiber.Map{"error": moveResp.ErrorMessage})
	}

	// Return updated character position
	return c.JSON(fiber.Map{
		"success": true,
		"character": fiber.Map{
			"x":       moveResp.Character.X,
			"y":       moveResp.Character.Y,
			"chunk_x": moveResp.Character.ChunkX,
			"chunk_y": moveResp.Character.ChunkY,
		},
	})
}

// GetWorldInfo returns world information as JSON
func (h *Game) GetWorldInfo(c *fiber.Ctx) error {
	// Get JWT token from locals
	jwtToken := c.Locals("jwt_token").(string)
	ctx := grpc.WithAuth(c.Context(), jwtToken)

	req := &worldV1.GetWorldInfoRequest{}
	resp, err := h.API.WorldService.GetWorldInfo(ctx, req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get world info"})
	}

	return c.JSON(resp.WorldInfo)
}