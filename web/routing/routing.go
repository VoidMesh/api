package routing

import (
	"github.com/VoidMesh/platform/web/handlers"
	"github.com/VoidMesh/platform/web/routes"
)

func RegisterRoutes(app *handlers.App) {
	// Static
	app.Web.Static("/static", "/src/web/static/public/").Name("static")

	// Homepage (authenticated users only) - Register first to avoid conflicts
	homepage := handlers.Homepage{App: app}
	app.Web.Get(routes.Homepage.Path, handlers.AuthMiddleware(app.SessionStore), homepage.Index).Name(routes.Homepage.Name)

	// Debug route (temporary)
	auth := handlers.Auth{App: app}
	app.Web.Get("/debug/session", auth.Debug)

	// Authentication routes (guest only)
	app.Web.Get(routes.LoginShow.Path, handlers.GuestMiddleware(app.SessionStore), auth.ShowLogin).Name(routes.LoginShow.Name)
	app.Web.Post(routes.Login.Path, handlers.GuestMiddleware(app.SessionStore), auth.Login).Name(routes.Login.Name)
	app.Web.Get(routes.SignupShow.Path, handlers.GuestMiddleware(app.SessionStore), auth.ShowSignup).Name(routes.SignupShow.Name)
	app.Web.Post(routes.Signup.Path, handlers.GuestMiddleware(app.SessionStore), auth.Signup).Name(routes.Signup.Name)

	// Logout route (authenticated users only)
	app.Web.Post(routes.Logout.Path, handlers.AuthMiddleware(app.SessionStore), auth.Logout).Name(routes.Logout.Name)

	// Meower routes (authenticated users only)
	meower := handlers.Meower{App: app}
	app.Web.Get(routes.MeowIndex.Path, handlers.AuthMiddleware(app.SessionStore), meower.Index).Name(routes.MeowIndex.Name)
	app.Web.Get(routes.MeowNew.Path, handlers.AuthMiddleware(app.SessionStore), meower.New).Name(routes.MeowNew.Name)
	app.Web.Post(routes.MeowCreate.Path, handlers.AuthMiddleware(app.SessionStore), meower.Create).Name(routes.MeowCreate.Name)

	// Game routes (authenticated users only)
	game := handlers.Game{App: app}
	app.Web.Get(routes.GameCharacters.Path, handlers.AuthMiddleware(app.SessionStore), game.CharacterSelect).Name(routes.GameCharacters.Name)
	app.Web.Post(routes.GameCharacterCreate.Path, handlers.AuthMiddleware(app.SessionStore), game.CreateCharacter).Name(routes.GameCharacterCreate.Name)
	app.Web.Get(routes.GameWorldInfo.Path, handlers.AuthMiddleware(app.SessionStore), game.GetWorldInfo).Name(routes.GameWorldInfo.Name)
	app.Web.Get(routes.GameWorld.Path, handlers.AuthMiddleware(app.SessionStore), game.GameWorld).Name(routes.GameWorld.Name)
	app.Web.Post(routes.GameMove.Path, handlers.AuthMiddleware(app.SessionStore), game.MoveCharacter).Name(routes.GameMove.Name)
}
