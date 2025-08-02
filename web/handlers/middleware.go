package handlers

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware checks if the user is authenticated
func AuthMiddleware(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Printf("DEBUG: AuthMiddleware called for path: %s\n", c.Path())

		sess, err := store.Get(c)
		if err != nil {
			fmt.Printf("DEBUG: Session error in AuthMiddleware: %v\n", err)
			return c.Redirect("/login")
		}

		userID := sess.Get("user_id")
		if userID == nil {
			fmt.Printf("DEBUG: No user_id in session, redirecting to login\n")
			return c.Redirect("/login")
		}

		fmt.Printf("DEBUG: User authenticated: %v\n", userID)

		// Generate JWT token for gRPC calls
		jwtToken, err := generateJWTToken(userID.(string), sess.Get("username").(string))
		if err != nil {
			fmt.Printf("DEBUG: Failed to generate JWT token: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Authentication error")
		}

		// Add user info to locals for use in handlers
		c.Locals("user_id", userID)
		c.Locals("username", sess.Get("username"))
		c.Locals("display_name", sess.Get("display_name"))
		c.Locals("jwt_token", jwtToken)

		return c.Next()
	}
}

// GuestMiddleware redirects authenticated users away from guest-only pages
func GuestMiddleware(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.Next()
		}

		userID := sess.Get("user_id")
		if userID != nil {
			return c.Redirect("/")
		}

		return c.Next()
	}
}

// generateJWTToken creates a JWT token for the given user (matching the API implementation)
func generateJWTToken(userID string, username string) (string, error) {
	jwtSecret := []byte(getJWTSecret())

	// Create the claims
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // Token expires in 7 days
		"iat":      time.Now().Unix(),
		"iss":      "voidmesh-api",
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// getJWTSecret retrieves the JWT secret from environment variables
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Use same default as API for development
		return "dev-jwt-secret-change-for-production-use-minimum-32-chars"
	}
	return secret
}
