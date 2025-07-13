package main

import (
	"os"
	"time"

	"github.com/VoidMesh/platform/web/grpc"
	"github.com/VoidMesh/platform/web/handlers"
	"github.com/VoidMesh/platform/web/routing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/gofiber/storage/sqlite3/v2"
)

func main() {
	// Connect to the internal gRPC API
	GrpcClient := grpc.NewClient()

	// Create SQLite storage for sessions
	storage := sqlite3.New(sqlite3.Config{
		Database:   "./data/sessions.db",
		Table:      "sessions",
		Reset:      false,
		GCInterval: 10 * time.Second,
	})

	// Create session store with SQLite storage
	sessionStore := session.New(session.Config{
		Storage:        storage,
		KeyLookup:      "cookie:session_id",
		CookieDomain:   "",
		CookiePath:     "/",
		CookieSecure:   os.Getenv("ENV") == "production",
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		Expiration:     24 * time.Hour, // 24 hour session expiration
	})

	// Create the Fiber app
	fiberApp := fiber.New(fiber.Config{
		ErrorHandler:      handlers.ErrorHandler,
		EnablePrintRoutes: true,
	})

	// Add middlewares
	if os.Getenv("ENV") == "production" {
		fiberApp.Use(compress.New()) // Enable gzip compression in production only, templ proxy does not support brotli
		fiberApp.Use(csrf.New())
	} else {
		fiberApp.Use(logger.New()) // Enable request logging in development
	}
	fiberApp.Use(requestid.New(requestid.Config{Generator: utils.UUIDv4}))
	fiberApp.Use(encryptcookie.New(encryptcookie.Config{
		Key: os.Getenv("COOKIE_SECRET_KEY"),
	}))

	app := &handlers.App{
		Web:          fiberApp,
		API:          GrpcClient,
		SessionStore: sessionStore,
	}

	// Mount public routes
	routing.RegisterRoutes(app)
	if err := app.Web.Listen("0.0.0.0:3000"); err != nil {
		panic(err)
	}
}
