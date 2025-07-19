package routes

import (
	"github.com/gofiber/fiber/v2"
)

type route struct {
	Name string
	Path string
}

/*
Routes for a common resource using RESTful conventions.

| HTTP Verb | URL              | Route name    | Description                                  |
| --------- | ---------------- | ------------- | -------------------------------------------- |
| GET       | /photos          | index         | Display a list of all photos                 |
| GET       | /photos/new      | new           | Return an HTML form for creating a new photo |
| POST      | /photos          | create        | Create a new photo                           |
| GET       | /photos/:id      | show          | Display a specific photo                     |
| GET       | /photos/:id/edit | edit          | Return an HTML form for editing a photo      |
| PATCH/PUT | /photos/:id      | update        | Update a specific photo                      |
| DELETE    | /photos/:id      | destroy       | Delete a specific photo                      |

Feel free to use anything else that makes sense for your endpoints, resources and application.
*/

var (
	// Homepage
	Homepage = route{Name: "home.index", Path: "/"}

	// Authentication
	LoginShow  = route{Name: "auth.login.show", Path: "/login"}
	Login      = route{Name: "auth.login", Path: "/login"}
	SignupShow = route{Name: "auth.signup.show", Path: "/signup"}
	Signup     = route{Name: "auth.signup", Path: "/signup"}
	Logout     = route{Name: "auth.logout", Path: "/logout"}

	// Game
	GameCharacters      = route{Name: "game.characters", Path: "/game/characters"}
	GameCharacterCreate = route{Name: "game.characters.create", Path: "/game/characters/create"}
	GameWorldInfo       = route{Name: "game.world.info", Path: "/game/world/info"}
)

func (r *route) URL(c *fiber.Ctx, params fiber.Map) string {
	url, err := c.GetRouteURL(r.Name, params)
	if err != nil {
		c.Next()
		fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return url
}
