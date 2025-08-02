package handlers

import (
	viewHomepage "github.com/VoidMesh/api/web/views/pages/homepage"

	"github.com/gofiber/fiber/v2"
)

type Homepage struct{ *App }

func (h *Homepage) Index(c *fiber.Ctx) error {
	return renderTempl(c, viewHomepage.Index(c))
}
