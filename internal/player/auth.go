package player

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// PasswordManager handles password hashing and verification
type PasswordManager struct {
	// Salt length in bytes
	saltLength int
	// Number of iterations for key derivation
	iterations int
}

// NewPasswordManager creates a new password manager with default settings
func NewPasswordManager() *PasswordManager {
	return &PasswordManager{
		saltLength: 32,
		iterations: 10000,
	}
}

// GenerateSalt creates a random salt
func (pm *PasswordManager) GenerateSalt() (string, error) {
	salt := make([]byte, pm.saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// HashPassword creates a password hash with the given salt
func (pm *PasswordManager) HashPassword(password, salt string) (string, error) {
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return "", fmt.Errorf("failed to decode salt: %w", err)
	}

	// Use SHA-256 for password hashing (in production, consider using bcrypt or argon2)
	hash := sha256.Sum256(append([]byte(password), saltBytes...))
	
	// Apply iterations to make it more secure
	for i := 0; i < pm.iterations; i++ {
		hash = sha256.Sum256(hash[:])
	}
	
	return hex.EncodeToString(hash[:]), nil
}

// VerifyPassword verifies if a password matches the stored hash
func (pm *PasswordManager) VerifyPassword(password, storedHash, salt string) bool {
	computedHash, err := pm.HashPassword(password, salt)
	if err != nil {
		return false
	}
	
	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}

// TokenManager handles session token generation and validation
type TokenManager struct {
	tokenLength int
}

// NewTokenManager creates a new token manager
func NewTokenManager() *TokenManager {
	return &TokenManager{
		tokenLength: 32,
	}
}

// GenerateToken creates a random session token
func (tm *TokenManager) GenerateToken() (string, error) {
	token := make([]byte, tm.tokenLength)
	_, err := rand.Read(token)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(token), nil
}

// ValidateToken performs basic token validation
func (tm *TokenManager) ValidateToken(token string) error {
	if len(token) == 0 {
		return errors.New("token cannot be empty")
	}
	
	// Decode to verify it's valid base64
	_, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("invalid token format: %w", err)
	}
	
	return nil
}

// CreateSessionExpiry creates an expiry time for a session
func CreateSessionExpiry() time.Time {
	return time.Now().Add(time.Duration(SessionTimeout) * time.Hour)
}

// IsSessionExpired checks if a session has expired
func IsSessionExpired(expiresAt time.Time) bool {
	return time.Now().After(expiresAt)
}

// ValidateUsername checks if a username meets requirements
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	if len(username) > 32 {
		return errors.New("username must not exceed 32 characters")
	}
	
	// Check for valid characters (alphanumeric and underscores)
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_') {
			return errors.New("username can only contain letters, numbers, and underscores")
		}
	}
	
	return nil
}

// ValidatePassword checks if a password meets requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if len(password) > 256 {
		return errors.New("password must not exceed 256 characters")
	}
	
	// Check for at least one letter and one number
	hasLetter := false
	hasNumber := false
	
	for _, char := range password {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
		}
		if char >= '0' && char <= '9' {
			hasNumber = true
		}
	}
	
	if !hasLetter {
		return errors.New("password must contain at least one letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	
	return nil
}

// ValidateEmail performs basic email validation
func ValidateEmail(email string) error {
	if email == "" {
		return nil // Email is optional
	}
	
	if len(email) > 255 {
		return errors.New("email must not exceed 255 characters")
	}
	
	// Basic email format check
	atCount := 0
	for _, char := range email {
		if char == '@' {
			atCount++
		}
	}
	
	if atCount != 1 {
		return errors.New("email must contain exactly one @ symbol")
	}
	
	return nil
}