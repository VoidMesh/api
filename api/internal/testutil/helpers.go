package testutil

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// UUIDTestData contains commonly used test UUIDs for consistent testing
var UUIDTestData = struct {
	User1      string
	User2      string
	Character1 string
	Character2 string
	World1     string
	World2     string
	Chunk1     string
	Chunk2     string
}{
	User1:      "550e8400-e29b-41d4-a716-446655440000",
	User2:      "550e8400-e29b-41d4-a716-446655440001",
	Character1: "750e8400-e29b-41d4-a716-446655440000",
	Character2: "750e8400-e29b-41d4-a716-446655440001",
	World1:     "650e8400-e29b-41d4-a716-446655440000",
	World2:     "650e8400-e29b-41d4-a716-446655440001",
	Chunk1:     "850e8400-e29b-41d4-a716-446655440000",
	Chunk2:     "850e8400-e29b-41d4-a716-446655440001",
}

// TestJWTSecretKey is the standard secret key used for JWT signing in tests
const TestJWTSecretKey = "test-secret-key-32-characters-long!"

// PreGeneratedJWTTokens contains pre-generated JWT tokens for testing.
// These tokens are signed with TestJWTSecretKey and have long expiration times.
// Generated with: user_id, username, email, exp (year 2030), iat (fixed timestamp)
var PreGeneratedJWTTokens = struct {
	ValidUser1Token   string // User1 with valid claims
	ValidUser2Token   string // User2 with valid claims  
	ExpiredToken      string // Expired token
	InvalidSignature  string // Token with invalid signature
	MalformedToken    string // Malformed token string
}{
	// Valid token for User1 (expires in 2030)
	// Claims: {"user_id":"550e8400-e29b-41d4-a716-446655440000","username":"testuser1","email":"user1@example.com","exp":1893456000,"iat":1609459200}
	ValidUser1Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InVzZXIxQGV4YW1wbGUuY29tIiwiZXhwIjoxODkzNDU2MDAwLCJpYXQiOjE2MDk0NTkyMDAsInVzZXJfaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJ1c2VybmFtZSI6InRlc3R1c2VyMSJ9.1NX4AHR1Bc3XAfQDIAQgIczlTNVuXpJMFg58wsIi3BQ",
	
	// Valid token for User2 (expires in 2030)
	// Claims: {"user_id":"550e8400-e29b-41d4-a716-446655440001","username":"testuser2","email":"user2@example.com","exp":1893456000,"iat":1609459200}
	ValidUser2Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InVzZXIyQGV4YW1wbGUuY29tIiwiZXhwIjoxODkzNDU2MDAwLCJpYXQiOjE2MDk0NTkyMDAsInVzZXJfaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDEiLCJ1c2VybmFtZSI6InRlc3R1c2VyMiJ9.8_Xd8ToHYTBdeb6x9M6BJwNGY9Tq01y6D3i5hyhcuIg",
	
	// Expired token (expired in 2020)
	// Claims: {"user_id":"550e8400-e29b-41d4-a716-446655440000","username":"expireduser","email":"expired@example.com","exp":1577836800,"iat":1577750400}
	ExpiredToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImV4cGlyZWRAZXhhbXBsZS5jb20iLCJleHAiOjE1Nzc4MzY4MDAsImlhdCI6MTU3Nzc1MDQwMCwidXNlcl9pZCI6IjU1MGU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDAwMCIsInVzZXJuYW1lIjoiZXhwaXJlZHVzZXIifQ.PJHBVWrIlSZ7dqn5lT7J6VI5HDb8MXaXC024bvuolTA",
	
	// Token with invalid signature (same payload as ValidUser1Token but wrong signature)
	InvalidSignature: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InVzZXIxQGV4YW1wbGUuY29tIiwiZXhwIjoxODkzNDU2MDAwLCJpYXQiOjE2MDk0NTkyMDAsInVzZXJfaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJ1c2VybmFtZSI6InRlc3R1c2VyMSJ9.INVALID_SIGNATURE_HERE",
	
	// Malformed token
	MalformedToken: "not.a.valid.jwt.token",
}

// GenerateTestUUID generates a valid UUID string for testing.
// This ensures all test UUIDs are properly formatted and unique.
func GenerateTestUUID() string {
	return uuid.New().String()
}

// ParseTestUUID converts a UUID string to pgtype.UUID for database operations.
// It handles the conversion safely and provides clear error messages for debugging.
func ParseTestUUID(t *testing.T, uuidStr string) pgtype.UUID {
	t.Helper()

	parsedUUID, err := uuid.Parse(uuidStr)
	require.NoError(t, err, "Failed to parse UUID: %s", uuidStr)

	var pgUUID pgtype.UUID
	// Convert the UUID bytes to the pgtype.UUID format
	copy(pgUUID.Bytes[:], parsedUUID[:])
	pgUUID.Valid = true

	return pgUUID
}

// UUIDFromString is a convenience function to convert string UUID to pgtype.UUID
// without requiring a testing.T parameter. Use only when you're certain the UUID is valid.
func UUIDFromString(uuidStr string) pgtype.UUID {
	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(fmt.Sprintf("Invalid UUID string: %s", uuidStr))
	}

	var pgUUID pgtype.UUID
	copy(pgUUID.Bytes[:], parsedUUID[:])
	pgUUID.Valid = true

	return pgUUID
}

// TimestampFromTime converts a time.Time to pgtype.Timestamp for database operations
func TimestampFromTime(t time.Time) pgtype.Timestamp {
	var pgTimestamp pgtype.Timestamp
	pgTimestamp.Scan(t)
	return pgTimestamp
}

// NowTimestamp returns the current time as a pgtype.Timestamp
func NowTimestamp() pgtype.Timestamp {
	return TimestampFromTime(time.Now())
}

// JWTTestConfig holds configuration for JWT token generation in tests
type JWTTestConfig struct {
	UserID    string
	Username  string
	Email     string
	ExpiresIn time.Duration
	SecretKey string
}

// DefaultJWTTestConfig returns a default configuration for JWT testing
func DefaultJWTTestConfig() *JWTTestConfig {
	return &JWTTestConfig{
		UserID:    UUIDTestData.User1,
		Username:  "testuser",
		Email:     "test@example.com",
		ExpiresIn: time.Hour,
		SecretKey: TestJWTSecretKey,
	}
}

// GenerateTestJWT creates a JWT token for testing with the provided configuration.
// This token includes standard claims and can be used to test authentication.
func GenerateTestJWT(t *testing.T, config *JWTTestConfig) string {
	t.Helper()

	if config == nil {
		config = DefaultJWTTestConfig()
	}

	// Create the claims
	claims := jwt.MapClaims{
		"user_id":  config.UserID,
		"username": config.Username,
		"email":    config.Email,
		"exp":      time.Now().Add(config.ExpiresIn).Unix(),
		"iat":      time.Now().Unix(),
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	tokenString, err := token.SignedString([]byte(config.SecretKey))
	require.NoError(t, err, "Failed to sign JWT token")

	return tokenString
}

// CreateTestContext creates a context with metadata for gRPC testing.
// This includes authentication tokens and other metadata commonly used in tests.
func CreateTestContextWithAuth(userID, username string) context.Context {
	ctx := context.Background()

	// Add metadata for gRPC calls
	md := metadata.Pairs(
		"user-id", userID,
		"username", username,
	)

	return metadata.NewIncomingContext(ctx, md)
}

// CreateTestContextWithJWT creates a context with a JWT token in metadata
func CreateTestContextWithJWT(t *testing.T, config *JWTTestConfig) context.Context {
	t.Helper()

	token := GenerateTestJWT(t, config)

	md := metadata.Pairs(
		"authorization", "Bearer "+token,
	)

	return metadata.NewIncomingContext(context.Background(), md)
}

// CreateTestContextWithPreGeneratedJWT creates a context with a pre-generated JWT token
// This is faster than generating a new token and useful for consistent test data
func CreateTestContextWithPreGeneratedJWT(token string) context.Context {
	md := metadata.Pairs(
		"authorization", "Bearer "+token,
	)

	return metadata.NewIncomingContext(context.Background(), md)
}

// CreateTestContextForUser1 creates a context with User1's pre-generated valid JWT token
func CreateTestContextForUser1() context.Context {
	return CreateTestContextWithPreGeneratedJWT(PreGeneratedJWTTokens.ValidUser1Token)
}

// CreateTestContextForUser2 creates a context with User2's pre-generated valid JWT token
func CreateTestContextForUser2() context.Context {
	return CreateTestContextWithPreGeneratedJWT(PreGeneratedJWTTokens.ValidUser2Token)
}

// CreateTestContextWithExpiredToken creates a context with an expired JWT token
func CreateTestContextWithExpiredToken() context.Context {
	return CreateTestContextWithPreGeneratedJWT(PreGeneratedJWTTokens.ExpiredToken)
}

// CreateTestContextWithInvalidToken creates a context with an invalid JWT token
func CreateTestContextWithInvalidToken() context.Context {
	return CreateTestContextWithPreGeneratedJWT(PreGeneratedJWTTokens.InvalidSignature)
}

// AssertGRPCError verifies that an error is a gRPC error with the expected status code.
// This is essential for testing gRPC service implementations.
func AssertGRPCError(t *testing.T, err error, expectedCode codes.Code, msgAndArgs ...interface{}) {
	t.Helper()

	require.Error(t, err, "Expected gRPC error but got nil")

	st, ok := status.FromError(err)
	require.True(t, ok, "Error is not a gRPC status error")

	assert.Equal(t, expectedCode, st.Code(), "gRPC error code mismatch")

	if len(msgAndArgs) > 0 {
		expectedMessage, ok := msgAndArgs[0].(string)
		if ok {
			assert.Contains(t, st.Message(), expectedMessage, "gRPC error message should contain expected text")
		}
	}
}

// AssertNoGRPCError verifies that there is no gRPC error
func AssertNoGRPCError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()

	if err != nil {
		if st, ok := status.FromError(err); ok {
			require.NoError(t, err, "Unexpected gRPC error: code=%s, message=%s", st.Code(), st.Message())
		}
	}
	require.NoError(t, err, msgAndArgs...)
}

// CompareProtoMessages performs deep comparison of protocol buffer messages.
// This is more reliable than using == or reflect.DeepEqual with protobuf messages.
func CompareProtoMessages(t *testing.T, expected, actual proto.Message, msgAndArgs ...interface{}) {
	t.Helper()

	if expected == nil && actual == nil {
		return
	}

	require.NotNil(t, expected, "Expected proto message is nil")
	require.NotNil(t, actual, "Actual proto message is nil")

	// Use proto.Equal for proper protobuf comparison
	if !proto.Equal(expected, actual) {
		// Provide detailed diff information
		expectedStr := fmt.Sprintf("%v", expected)
		actualStr := fmt.Sprintf("%v", actual)

		require.Fail(t, "Proto messages are not equal",
			"Expected: %s\nActual: %s\n%v",
			expectedStr, actualStr, msgAndArgs)
	}
}

// AssertProtoFieldEqual compares a specific field in proto messages
func AssertProtoFieldEqual(t *testing.T, expected, actual proto.Message, fieldName string, msgAndArgs ...interface{}) {
	t.Helper()

	expectedReflect := expected.ProtoReflect()
	actualReflect := actual.ProtoReflect()

	expectedFields := expectedReflect.Descriptor().Fields()
	actualFields := actualReflect.Descriptor().Fields()

	expectedField := expectedFields.ByName(protoreflect.Name(fieldName))
	actualField := actualFields.ByName(protoreflect.Name(fieldName))

	require.NotNil(t, expectedField, "Field %s not found in expected message", fieldName)
	require.NotNil(t, actualField, "Field %s not found in actual message", fieldName)

	expectedValue := expectedReflect.Get(expectedField)
	actualValue := actualReflect.Get(actualField)

	assert.Equal(t, expectedValue.Interface(), actualValue.Interface(), msgAndArgs...)
}

// GenerateRandomBytes generates random bytes of the specified length
func GenerateRandomBytes(length int) []byte {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate random bytes: %v", err))
	}
	return bytes
}

// GenerateRandomHexString generates a random hex string of the specified length
func GenerateRandomHexString(length int) string {
	bytes := GenerateRandomBytes((length + 1) / 2)
	return hex.EncodeToString(bytes)[:length]
}

// GenerateTestEmail generates a unique test email address
func GenerateTestEmail() string {
	return fmt.Sprintf("test-%s@example.com", GenerateRandomHexString(8))
}

// GenerateTestUsername generates a unique test username
func GenerateTestUsername() string {
	return fmt.Sprintf("testuser-%s", GenerateRandomHexString(6))
}

// WaitForCondition waits for a condition to be true within a timeout.
// This is useful for testing asynchronous operations.
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			require.Fail(t, "Condition not met within timeout", message)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// RetryOperation retries an operation up to maxAttempts times with exponential backoff.
// This is useful for testing operations that might fail due to timing or external factors.
func RetryOperation(t *testing.T, operation func() error, maxAttempts int, baseDelay time.Duration) {
	t.Helper()

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return // Success
		}

		lastErr = err

		if attempt < maxAttempts {
			delay := time.Duration(attempt) * baseDelay
			t.Logf("Attempt %d failed, retrying in %v: %v", attempt, delay, err)
			time.Sleep(delay)
		}
	}

	require.NoError(t, lastErr, "Operation failed after %d attempts", maxAttempts)
}

// AssertEventuallyEqual waits for a value to eventually equal the expected value.
// This is useful for testing asynchronous operations where values change over time.
func AssertEventuallyEqual(t *testing.T, expected interface{}, getValue func() interface{}, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	WaitForCondition(t, func() bool {
		actual := getValue()
		return reflect.DeepEqual(expected, actual)
	}, timeout, fmt.Sprintf("Value never became equal to expected: %v", expected))
}

// TableTestCase represents a single test case in a table-driven test
type TableTestCase struct {
	Name       string
	Setup      func(t *testing.T)
	Run        func(t *testing.T)
	Cleanup    func(t *testing.T)
	Skip       bool
	SkipReason string
}

// RunTableTests executes a slice of table test cases with proper setup and cleanup
func RunTableTests(t *testing.T, testCases []TableTestCase) {
	t.Helper()

	for _, tc := range testCases {
		tc := tc // Capture range variable

		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip {
				if tc.SkipReason != "" {
					t.Skip(tc.SkipReason)
				} else {
					t.Skip("Test case skipped")
				}
			}

			// Run setup if provided
			if tc.Setup != nil {
				tc.Setup(t)
			}

			// Run cleanup at the end if provided
			if tc.Cleanup != nil {
				defer tc.Cleanup(t)
			}

			// Run the actual test
			if tc.Run != nil {
				tc.Run(t)
			}
		})
	}
}

// TestDataBuilder provides a fluent interface for building test data structures
type TestDataBuilder struct {
	data map[string]interface{}
}

// NewTestDataBuilder creates a new test data builder
func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{
		data: make(map[string]interface{}),
	}
}

// WithField adds a field to the test data
func (b *TestDataBuilder) WithField(key string, value interface{}) *TestDataBuilder {
	b.data[key] = value
	return b
}

// WithUserID adds a user ID to the test data
func (b *TestDataBuilder) WithUserID(userID string) *TestDataBuilder {
	return b.WithField("user_id", userID)
}

// WithCharacterID adds a character ID to the test data
func (b *TestDataBuilder) WithCharacterID(characterID string) *TestDataBuilder {
	return b.WithField("character_id", characterID)
}

// WithPosition adds position coordinates to the test data
func (b *TestDataBuilder) WithPosition(x, y int32) *TestDataBuilder {
	b.data["x"] = x
	b.data["y"] = y
	return b
}

// WithChunkPosition adds chunk coordinates to the test data
func (b *TestDataBuilder) WithChunkPosition(chunkX, chunkY int32) *TestDataBuilder {
	b.data["chunk_x"] = chunkX
	b.data["chunk_y"] = chunkY
	return b
}

// Build returns the constructed test data
func (b *TestDataBuilder) Build() map[string]interface{} {
	// Return a copy to prevent accidental modification
	result := make(map[string]interface{})
	for k, v := range b.data {
		result[k] = v
	}
	return result
}

// GetString safely gets a string value from the test data
func (b *TestDataBuilder) GetString(key string) string {
	if val, ok := b.data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt32 safely gets an int32 value from the test data
func (b *TestDataBuilder) GetInt32(key string) int32 {
	if val, ok := b.data[key]; ok {
		if i32, ok := val.(int32); ok {
			return i32
		}
	}
	return 0
}

// MustParseUUID parses a UUID string and panics on error.
// Use this for known valid UUIDs in test data.
func MustParseUUID(uuidStr string) pgtype.UUID {
	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(fmt.Sprintf("Invalid UUID string: %s", uuidStr))
	}

	var pgUUID pgtype.UUID
	copy(pgUUID.Bytes[:], parsedUUID[:])
	pgUUID.Valid = true

	return pgUUID
}

// GenerateTestString generates a random string of the specified length using alphanumeric characters
func GenerateTestString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)

	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
