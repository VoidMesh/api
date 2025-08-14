package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// jwtService implements the JWTService interface
type jwtService struct {
	secret []byte
}

// NewJWTService creates a new JWT service with the provided secret
func NewJWTService(secret string) (JWTService, error) {
	if err := validateJWTSecret(secret); err != nil {
		return nil, err
	}

	return &jwtService{
		secret: []byte(secret),
	}, nil
}

// GenerateToken creates a JWT token for the given user
func (j *jwtService) GenerateToken(userID string, username string) (string, error) {
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
	tokenString, err := token.SignedString(j.secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *jwtService) ValidateToken(tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// validateJWTSecret validates the JWT secret for security requirements
func validateJWTSecret(secret string) error {
	// Validate secret length for security
	if len(secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long for adequate security (current length: %d, required: 32)", len(secret))
	}

	// Validate secret complexity (basic check)
	if !isSecretComplex(secret) {
		// This is a warning, not an error, so we just print it
		fmt.Println("WARNING: JWT_SECRET should contain a mix of uppercase, lowercase, numbers, and special characters for better security.")
	}

	return nil
}

// isSecretComplex performs basic complexity validation on the JWT secret
func isSecretComplex(secret string) bool {
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range secret {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		default:
			// Consider any non-alphanumeric character as special
			hasSpecial = true
		}
	}

	// Require at least 3 out of 4 character types
	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasDigit {
		count++
	}
	if hasSpecial {
		count++
	}

	return count >= 3
}

// passwordService implements the PasswordService interface
type passwordService struct{}

// NewPasswordService creates a new password service
func NewPasswordService() PasswordService {
	return &passwordService{}
}

// HashPassword hashes a plain text password using bcrypt
func (p *passwordService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword verifies a password against its bcrypt hash
func (p *passwordService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// tokenGenerator implements the TokenGenerator interface
type tokenGenerator struct{}

// NewTokenGenerator creates a new token generator
func NewTokenGenerator() TokenGenerator {
	return &tokenGenerator{}
}

// GenerateToken generates a random token of the specified length
func (t *tokenGenerator) GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetJWTSecretFromEnv retrieves and validates the JWT secret from environment variables
// This function provides a centralized way to handle JWT secret configuration with
// appropriate security validation and development defaults.
func GetJWTSecretFromEnv(environment string) (string, error) {
	// This function should be used by the main application to get the JWT secret
	// and pass it to NewJWTService. It's separated from the service itself to
	// allow for better testability.
	
	// Try to get JWT secret from environment
	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		return secret, nil
	}
	
	// Check if we're in development mode
	if environment == "development" || strings.Contains(strings.ToLower(environment), "dev") {
		fmt.Println("WARNING: Using default JWT secret for development. Set JWT_SECRET environment variable for production.")
		return "dev-jwt-secret-change-for-production-use-minimum-32-chars", nil
	}

	return "", fmt.Errorf("JWT_SECRET environment variable is required in production")
}