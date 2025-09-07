package uuid

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UUID with dashes",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: "550e8400e29b41d4a716446655440000",
		},
		{
			name:     "UUID without dashes",
			input:    "550e8400e29b41d4a716446655440000",
			expected: "550e8400e29b41d4a716446655440000",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Mixed case with dashes",
			input:    "550E8400-E29B-41D4-A716-446655440000",
			expected: "550E8400E29B41D4A716446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseToHexString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "Valid UUID with dashes",
			input:       "550e8400-e29b-41d4-a716-446655440000",
			expected:    "550e8400e29b41d4a716446655440000",
			expectError: false,
		},
		{
			name:        "Valid UUID without dashes",
			input:       "550e8400e29b41d4a716446655440000",
			expected:    "550e8400e29b41d4a716446655440000",
			expectError: false,
		},
		{
			name:        "Invalid UUID - too short",
			input:       "550e8400-e29b-41d4",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid UUID - invalid characters",
			input:       "550g8400-e29b-41d4-a716-446655440000",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseToHexString(tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestStringToPgtype(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectValid bool
	}{
		{
			name:        "Valid UUID with dashes",
			input:       "550e8400-e29b-41d4-a716-446655440000",
			expectError: false,
			expectValid: true,
		},
		{
			name:        "Valid UUID without dashes",
			input:       "550e8400e29b41d4a716446655440000",
			expectError: false,
			expectValid: true,
		},
		{
			name:        "Invalid UUID",
			input:       "invalid-uuid",
			expectError: true,
			expectValid: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToPgtype(tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectValid, result.Valid)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectValid, result.Valid)
				
				// Verify roundtrip conversion
				if result.Valid {
					backToString := PgtypeToString(result)
					assert.True(t, ValidateFormat(backToString))
					assert.True(t, Compare(tt.input, backToString))
				}
			}
		})
	}
}

func TestPgtypeToString(t *testing.T) {
	// Create a valid pgtype.UUID
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	parsedUUID, err := uuid.Parse(validUUID)
	require.NoError(t, err)
	
	pgUUID := pgtype.UUID{}
	copy(pgUUID.Bytes[:], parsedUUID[:])
	pgUUID.Valid = true
	
	result := PgtypeToString(pgUUID)
	assert.Equal(t, validUUID, result)
	
	// Test invalid UUID
	invalidPgUUID := pgtype.UUID{Valid: false}
	result = PgtypeToString(invalidPgUUID)
	assert.Equal(t, "", result)
}

func TestPgtypeToNormalizedString(t *testing.T) {
	// Create a valid pgtype.UUID
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	parsedUUID, err := uuid.Parse(validUUID)
	require.NoError(t, err)
	
	pgUUID := pgtype.UUID{}
	copy(pgUUID.Bytes[:], parsedUUID[:])
	pgUUID.Valid = true
	
	result := PgtypeToNormalizedString(pgUUID)
	expected := "550e8400e29b41d4a716446655440000"
	assert.Equal(t, expected, result)
	
	// Test invalid UUID
	invalidPgUUID := pgtype.UUID{Valid: false}
	result = PgtypeToNormalizedString(invalidPgUUID)
	assert.Equal(t, "", result)
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		uuid1    string
		uuid2    string
		expected bool
	}{
		{
			name:     "Same UUID with and without dashes",
			uuid1:    "550e8400-e29b-41d4-a716-446655440000",
			uuid2:    "550e8400e29b41d4a716446655440000",
			expected: true,
		},
		{
			name:     "Same UUID both with dashes",
			uuid1:    "550e8400-e29b-41d4-a716-446655440000",
			uuid2:    "550e8400-e29b-41d4-a716-446655440000",
			expected: true,
		},
		{
			name:     "Same UUID both without dashes",
			uuid1:    "550e8400e29b41d4a716446655440000",
			uuid2:    "550e8400e29b41d4a716446655440000",
			expected: true,
		},
		{
			name:     "Different UUIDs",
			uuid1:    "550e8400-e29b-41d4-a716-446655440000",
			uuid2:    "550e8400-e29b-41d4-a716-446655440001",
			expected: false,
		},
		{
			name:     "Case insensitive comparison",
			uuid1:    "550e8400-e29b-41d4-a716-446655440000",
			uuid2:    "550E8400E29B41D4A716446655440000",
			expected: false, // Note: Our normalize doesn't lowercase, so this should be false
		},
		{
			name:     "Empty strings",
			uuid1:    "",
			uuid2:    "",
			expected: true,
		},
		{
			name:     "One empty, one valid",
			uuid1:    "",
			uuid2:    "550e8400-e29b-41d4-a716-446655440000",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Compare(tt.uuid1, tt.uuid2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAndNormalize(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "Valid UUID with dashes",
			input:       "550e8400-e29b-41d4-a716-446655440000",
			expected:    "550e8400e29b41d4a716446655440000",
			expectError: false,
		},
		{
			name:        "Valid UUID without dashes", 
			input:       "550e8400e29b41d4a716446655440000",
			expected:    "550e8400e29b41d4a716446655440000",
			expectError: false,
		},
		{
			name:        "Invalid UUID",
			input:       "invalid-uuid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndNormalize(tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid UUID with dashes",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: true,
		},
		{
			name:     "Valid UUID without dashes",
			input:    "550e8400e29b41d4a716446655440000",
			expected: true,
		},
		{
			name:     "Invalid UUID - too short",
			input:    "550e8400-e29b-41d4",
			expected: false,
		},
		{
			name:     "Invalid UUID - invalid characters",
			input:    "550g8400-e29b-41d4-a716-446655440000",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Random string",
			input:    "not-a-uuid",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHexBytesToPgtype(t *testing.T) {
	// Create a test byte array
	hexBytes := [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}
	
	result := HexBytesToPgtype(hexBytes)
	
	assert.True(t, result.Valid)
	assert.Equal(t, hexBytes, result.Bytes)
	
	// Verify it can be converted back to string
	stringResult := PgtypeToString(result)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", stringResult)
}

func TestGenerateNew(t *testing.T) {
	result := GenerateNew()
	
	// Should be a valid UUID format
	assert.True(t, ValidateFormat(result))
	
	// Should contain dashes (standard format)
	assert.Contains(t, result, "-")
	
	// Should be 36 characters long (standard UUID format)
	assert.Len(t, result, 36)
}

func TestGenerateNewNormalized(t *testing.T) {
	result := GenerateNewNormalized()
	
	// Should be a valid UUID when dashes are added back
	withDashes := result[:8] + "-" + result[8:12] + "-" + result[12:16] + "-" + result[16:20] + "-" + result[20:]
	assert.True(t, ValidateFormat(withDashes))
	
	// Should not contain dashes
	assert.NotContains(t, result, "-")
	
	// Should be 32 characters long (normalized format)
	assert.Len(t, result, 32)
}

func TestRoundTripConversions(t *testing.T) {
	originalUUID := "550e8400-e29b-41d4-a716-446655440000"
	
	// Test string -> pgtype -> string
	pgUUID, err := StringToPgtype(originalUUID)
	require.NoError(t, err)
	backToString := PgtypeToString(pgUUID)
	assert.Equal(t, originalUUID, backToString)
	
	// Test string -> hex -> string
	hexString, err := ParseToHexString(originalUUID)
	require.NoError(t, err)
	withDashes := hexString[:8] + "-" + hexString[8:12] + "-" + hexString[12:16] + "-" + hexString[16:20] + "-" + hexString[20:]
	assert.Equal(t, originalUUID, withDashes)
	
	// Test normalize -> compare
	normalized := Normalize(originalUUID)
	assert.True(t, Compare(originalUUID, normalized))
}

func BenchmarkNormalize(b *testing.B) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	
	for i := 0; i < b.N; i++ {
		Normalize(uuid)
	}
}

func BenchmarkCompare(b *testing.B) {
	uuid1 := "550e8400-e29b-41d4-a716-446655440000"
	uuid2 := "550e8400e29b41d4a716446655440000"
	
	for i := 0; i < b.N; i++ {
		Compare(uuid1, uuid2)
	}
}

func BenchmarkParseToHexString(b *testing.B) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	
	for i := 0; i < b.N; i++ {
		ParseToHexString(uuid)
	}
}