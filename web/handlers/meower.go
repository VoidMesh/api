package handlers

import (
	meowV1 "github.com/VoidMesh/platform/api/proto/meow/v1"

	"github.com/VoidMesh/platform/web/grpc"
	viewMeowV1 "github.com/VoidMesh/platform/web/views/services/meows/v1"
	"github.com/gofiber/fiber/v2"
)

type Meower struct{ *App }

func (h *Meower) New(c *fiber.Ctx) error {
	return renderTempl(c, viewMeowV1.New(c))
}

func (h *Meower) Create(c *fiber.Ctx) error {
	content := c.FormValue("content")
	req := &meowV1.CreateMeowRequest{Content: content}

	// Get JWT token from locals
	jwtToken := c.Locals("jwt_token").(string)
	ctx := grpc.WithAuth(c.Context(), jwtToken)

	resp, err := h.API.MeowService.CreateMeow(ctx, req)
	if err != nil {
		return err
	}

	return renderTempl(c, viewMeowV1.Create(c, resp))
}

func (h *Meower) Index(c *fiber.Ctx) error {
	req := &meowV1.IndexMeowRequest{}

	// Get JWT token from locals
	jwtToken := c.Locals("jwt_token").(string)
	ctx := grpc.WithAuth(c.Context(), jwtToken)

	resp, err := h.API.MeowService.IndexMeow(ctx, req)
	if err != nil {
		return err
	}

	return renderTempl(c, viewMeowV1.Index(c, resp))
}
