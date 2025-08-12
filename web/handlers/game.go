package handlers

import (
	characterV1 "github.com/VoidMesh/api/api/proto/character/v1"
	"github.com/VoidMesh/api/web/grpc"
	viewGameV1 "github.com/VoidMesh/api/web/views/game"
	"github.com/gofiber/fiber/v2"
)

type Game struct{ *App }

// ListCharacters displays the character list
func (h *Game) ListCharacters(c *fiber.Ctx) error {
	// Get JWT token from locals
	jwtToken := c.Locals("jwt_token").(string)
	ctx := grpc.WithAuth(c.Context(), jwtToken)

	// Get user ID from JWT token
	userID := c.Locals("user_id").(string)

	// Get characters for this user
	req := &characterV1.GetCharactersByUserRequest{UserId: userID}
	resp, err := h.API.CharacterService.GetCharactersByUser(ctx, req)
	if err != nil {
		return err
	}

	return renderTempl(c, viewGameV1.ListCharacters(c, resp))
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
	req := &characterV1.CreateCharacterRequest{
		UserId: userID,
		Name:   name,
		SpawnX: 0, // Default spawn at origin
		SpawnY: 0,
	}

	_, err := h.API.CharacterService.CreateCharacter(ctx, req)
	if err != nil {
		return c.Status(400).SendString("Failed to create character: " + err.Error())
	}

	// Redirect back to character select
	return c.Redirect("/game/characters")
}
