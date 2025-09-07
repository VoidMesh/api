package uuid

import (
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Normalize removes dashes from UUID string for consistent comparison
func Normalize(uuidStr string) string {
	return strings.ReplaceAll(uuidStr, "-", "")
}

// ParseToHexString converts any UUID format to 32-char hex string
func ParseToHexString(uuidStr string) (string, error) {
	// Parse standard UUID format (handles both with/without dashes)
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(parsed[:]), nil
}

// StringToPgtype converts UUID string to pgtype.UUID
func StringToPgtype(uuidStr string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return pgtype.UUID{}, err
	}
	
	var pgUUID pgtype.UUID
	copy(pgUUID.Bytes[:], parsed[:])
	pgUUID.Valid = true
	return pgUUID, nil
}

// PgtypeToString converts pgtype.UUID to standard string format with dashes
func PgtypeToString(pgUUID pgtype.UUID) string {
	if !pgUUID.Valid {
		return ""
	}
	return uuid.UUID(pgUUID.Bytes).String()
}

// PgtypeToNormalizedString converts pgtype.UUID to normalized string format without dashes
func PgtypeToNormalizedString(pgUUID pgtype.UUID) string {
	if !pgUUID.Valid {
		return ""
	}
	return Normalize(uuid.UUID(pgUUID.Bytes).String())
}

// Compare compares two UUIDs regardless of format (with/without dashes)
func Compare(uuid1, uuid2 string) bool {
	return Normalize(uuid1) == Normalize(uuid2)
}

// ValidateAndNormalize validates UUID format and returns normalized version without dashes
func ValidateAndNormalize(uuidStr string) (string, error) {
	_, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", err
	}
	return Normalize(uuidStr), nil
}

// ValidateFormat checks if a string is a valid UUID format (with or without dashes)
func ValidateFormat(uuidStr string) bool {
	_, err := uuid.Parse(uuidStr)
	return err == nil
}

// HexBytesToPgtype converts 16-byte array to pgtype.UUID
func HexBytesToPgtype(hexBytes [16]byte) pgtype.UUID {
	var pgUUID pgtype.UUID
	pgUUID.Bytes = hexBytes
	pgUUID.Valid = true
	return pgUUID
}

// GenerateNew generates a new UUID and returns it in standard format with dashes
func GenerateNew() string {
	return uuid.New().String()
}

// GenerateNewNormalized generates a new UUID and returns it in normalized format without dashes
func GenerateNewNormalized() string {
	return Normalize(uuid.New().String())
}