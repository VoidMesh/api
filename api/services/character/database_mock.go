package character

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/VoidMesh/api/api/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

// MockDatabaseInterface implements DatabaseInterface for testing.
// It provides controlled behavior for database operations without requiring a real database.
type MockDatabaseInterface struct {
	characters       map[string]db.Character
	shouldReturnErr  bool
	nextCharacterID  string
	createCallCount  int
	getCallCount     int
	updateCallCount  int
	deleteCallCount  int
}

// NewMockDatabase creates a new mock database interface for testing.
func NewMockDatabase() *MockDatabaseInterface {
	return &MockDatabaseInterface{
		characters:      make(map[string]db.Character),
		nextCharacterID: "550e8400-e29b-41d4-a716-446655440000",
	}
}

// SetShouldReturnError configures the mock to return errors for all operations.
func (m *MockDatabaseInterface) SetShouldReturnError(shouldErr bool) {
	m.shouldReturnErr = shouldErr
}

// SetNextCharacterID sets the ID that will be used for the next created character.
func (m *MockDatabaseInterface) SetNextCharacterID(id string) {
	m.nextCharacterID = id
}

// AddCharacter manually adds a character to the mock database.
func (m *MockDatabaseInterface) AddCharacter(char db.Character) {
	key := fmt.Sprintf("%x", char.ID.Bytes)
	m.characters[key] = char
}

// GetCharacterById retrieves a character by ID.
func (m *MockDatabaseInterface) GetCharacterById(ctx context.Context, id pgtype.UUID) (db.Character, error) {
	m.getCallCount++
	
	if m.shouldReturnErr {
		return db.Character{}, assert.AnError
	}
	
	key := fmt.Sprintf("%x", id.Bytes)
	char, exists := m.characters[key]
	if !exists {
		return db.Character{}, sql.ErrNoRows
	}
	
	return char, nil
}

// CreateCharacter creates a new character.
func (m *MockDatabaseInterface) CreateCharacter(ctx context.Context, arg db.CreateCharacterParams) (db.Character, error) {
	m.createCallCount++
	
	if m.shouldReturnErr {
		return db.Character{}, assert.AnError
	}
	
	// Check for duplicate name (simplified - just check if name exists anywhere)
	for _, existing := range m.characters {
		if existing.Name == arg.Name {
			return db.Character{}, fmt.Errorf("duplicate key value violates unique constraint")
		}
	}
	
	// Parse the next character ID
	uuid, err := mockParseUUID(m.nextCharacterID)
	if err != nil {
		return db.Character{}, fmt.Errorf("invalid mock character ID: %v", err)
	}
	
	char := db.Character{
		ID:      uuid,
		UserID:  arg.UserID,
		Name:    arg.Name,
		X:       arg.X,
		Y:       arg.Y,
		ChunkX:  arg.ChunkX,
		ChunkY:  arg.ChunkY,
		CreatedAt: pgtype.Timestamp{Valid: true}, // Simplified timestamp
	}
	
	key := fmt.Sprintf("%x", char.ID.Bytes)
	m.characters[key] = char
	
	return char, nil
}

// UpdateCharacterPosition updates a character's position.
func (m *MockDatabaseInterface) UpdateCharacterPosition(ctx context.Context, arg db.UpdateCharacterPositionParams) (db.Character, error) {
	m.updateCallCount++
	
	if m.shouldReturnErr {
		return db.Character{}, assert.AnError
	}
	
	key := fmt.Sprintf("%x", arg.ID.Bytes)
	char, exists := m.characters[key]
	if !exists {
		return db.Character{}, sql.ErrNoRows
	}
	
	// Update position
	char.X = arg.X
	char.Y = arg.Y
	char.ChunkX = arg.ChunkX
	char.ChunkY = arg.ChunkY
	
	m.characters[key] = char
	
	return char, nil
}

// GetCharactersByUser retrieves all characters for a user.
func (m *MockDatabaseInterface) GetCharactersByUser(ctx context.Context, userID pgtype.UUID) ([]db.Character, error) {
	if m.shouldReturnErr {
		return nil, assert.AnError
	}
	
	var result []db.Character
	userKey := fmt.Sprintf("%x", userID.Bytes)
	
	for _, char := range m.characters {
		charUserKey := fmt.Sprintf("%x", char.UserID.Bytes)
		if charUserKey == userKey {
			result = append(result, char)
		}
	}
	
	return result, nil
}

// GetCharacterByUserAndName retrieves a character by user ID and name.
func (m *MockDatabaseInterface) GetCharacterByUserAndName(ctx context.Context, arg db.GetCharacterByUserAndNameParams) (db.Character, error) {
	if m.shouldReturnErr {
		return db.Character{}, assert.AnError
	}
	
	userKey := fmt.Sprintf("%x", arg.UserID.Bytes)
	
	for _, char := range m.characters {
		charUserKey := fmt.Sprintf("%x", char.UserID.Bytes)
		if charUserKey == userKey && char.Name == arg.Name {
			return char, nil
		}
	}
	
	return db.Character{}, sql.ErrNoRows
}

// DeleteCharacter deletes a character by ID.
func (m *MockDatabaseInterface) DeleteCharacter(ctx context.Context, id pgtype.UUID) error {
	m.deleteCallCount++
	
	if m.shouldReturnErr {
		return assert.AnError
	}
	
	key := fmt.Sprintf("%x", id.Bytes)
	if _, exists := m.characters[key]; !exists {
		return sql.ErrNoRows
	}
	
	delete(m.characters, key)
	return nil
}

// Test helper methods
func (m *MockDatabaseInterface) GetCreateCallCount() int {
	return m.createCallCount
}

func (m *MockDatabaseInterface) GetGetCallCount() int {
	return m.getCallCount
}

func (m *MockDatabaseInterface) GetUpdateCallCount() int {
	return m.updateCallCount
}

func (m *MockDatabaseInterface) GetDeleteCallCount() int {
	return m.deleteCallCount
}

// mockParseUUID helper function (renamed to avoid conflicts with character.go)
func mockParseUUID(uuidStr string) (pgtype.UUID, error) {
	var pgUUID pgtype.UUID
	err := pgUUID.Scan(uuidStr)
	return pgUUID, err
}