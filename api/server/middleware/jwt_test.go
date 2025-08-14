package middleware

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/VoidMesh/api/api/internal/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// mockUnaryHandler is a simple handler for testing interceptors
func mockUnaryHandler(ctx context.Context, req any) (any, error) {
	return &struct {
		Message string `json:"message"`
	}{
		Message: "success",
	}, nil
}

// mockUnaryInfo creates a grpc.UnaryServerInfo for testing
func mockUnaryInfo(fullMethod string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{
		FullMethod: fullMethod,
	}
}

func TestJWTAuthInterceptor_ValidToken(t *testing.T) {
	tests := []struct {
		name         string
		context      context.Context
		fullMethod   string
		wantErr      bool
		wantCode     codes.Code
		validateCtx  func(t *testing.T, ctx context.Context)
	}{
		{
			name:       "valid User1 token",
			context:    testutil.CreateTestContextForUser1(),
			fullMethod: "/character.v1.CharacterService/CreateCharacter",
			wantErr:    false,
			validateCtx: func(t *testing.T, ctx context.Context) {
				userID, ok := GetUserIDFromContext(ctx)
				require.True(t, ok, "User ID should be present in context")
				assert.Equal(t, testutil.UUIDTestData.User1, userID)

				username, ok := GetUsernameFromContext(ctx)
				require.True(t, ok, "Username should be present in context")
				assert.Equal(t, "testuser1", username)
			},
		},
		{
			name:       "valid User2 token",
			context:    testutil.CreateTestContextForUser2(),
			fullMethod: "/user.v1.UserService/GetProfile",
			wantErr:    false,
			validateCtx: func(t *testing.T, ctx context.Context) {
				userID, ok := GetUserIDFromContext(ctx)
				require.True(t, ok, "User ID should be present in context")
				assert.Equal(t, testutil.UUIDTestData.User2, userID)

				username, ok := GetUsernameFromContext(ctx)
				require.True(t, ok, "Username should be present in context")
				assert.Equal(t, "testuser2", username)
			},
		},
		{
			name:       "dynamically generated valid token",
			context:    testutil.CreateTestContextWithJWT(t, testutil.DefaultJWTTestConfig()),
			fullMethod: "/chunk.v1.ChunkService/GenerateChunk",
			wantErr:    false,
			validateCtx: func(t *testing.T, ctx context.Context) {
				userID, ok := GetUserIDFromContext(ctx)
				require.True(t, ok, "User ID should be present in context")
				assert.Equal(t, testutil.UUIDTestData.User1, userID)

				username, ok := GetUsernameFromContext(ctx)
				require.True(t, ok, "Username should be present in context")
				assert.Equal(t, "testuser", username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))

			// Create a handler that captures the context for validation
			var capturedCtx context.Context
			handler := func(ctx context.Context, req any) (any, error) {
				capturedCtx = ctx
				return mockUnaryHandler(ctx, req)
			}

			resp, err := interceptor(tt.context, nil, mockUnaryInfo(tt.fullMethod), handler)

			if tt.wantErr {
				testutil.AssertGRPCError(t, err, tt.wantCode)
			} else {
				testutil.AssertNoGRPCError(t, err)
				assert.NotNil(t, resp)

				if tt.validateCtx != nil {
					require.NotNil(t, capturedCtx, "Context should be captured")
					tt.validateCtx(t, capturedCtx)
				}
			}
		})
	}
}

func TestJWTAuthInterceptor_ExpiredToken(t *testing.T) {
	interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))
	ctx := testutil.CreateTestContextWithExpiredToken()

	resp, err := interceptor(ctx, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), mockUnaryHandler)

	require.Nil(t, resp)
	testutil.AssertGRPCError(t, err, codes.Unauthenticated, "invalid token")
	assert.Contains(t, err.Error(), "token is expired")
}

func TestJWTAuthInterceptor_InvalidToken(t *testing.T) {
	tests := []struct {
		name         string
		context      context.Context
		wantErrMsg   string
	}{
		{
			name:       "invalid signature",
			context:    testutil.CreateTestContextWithInvalidToken(),
			wantErrMsg: "signature is invalid",
		},
		{
			name: "malformed token",
			context: testutil.CreateTestContextWithPreGeneratedJWT(
				testutil.PreGeneratedJWTTokens.MalformedToken,
			),
			wantErrMsg: "token contains an invalid number of segments",
		},
		{
			name: "empty token after Bearer prefix",
			context: func() context.Context {
				md := metadata.Pairs("authorization", "Bearer ")
				return metadata.NewIncomingContext(context.Background(), md)
			}(),
			wantErrMsg: "token contains an invalid number of segments",
		},
		{
			name: "token without Bearer prefix",
			context: func() context.Context {
				md := metadata.Pairs("authorization", testutil.PreGeneratedJWTTokens.ValidUser1Token)
				return metadata.NewIncomingContext(context.Background(), md)
			}(),
			wantErrMsg: "invalid authorization header format",
		},
	}

	interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := interceptor(tt.context, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), mockUnaryHandler)

			require.Nil(t, resp)
			testutil.AssertGRPCError(t, err, codes.Unauthenticated)
			assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.wantErrMsg))
		})
	}
}

func TestJWTAuthInterceptor_MissingAuth(t *testing.T) {
	tests := []struct {
		name        string
		context     context.Context
		wantErrMsg  string
	}{
		{
			name:       "missing metadata",
			context:    context.Background(),
			wantErrMsg: "missing metadata",
		},
		{
			name: "missing authorization header",
			context: func() context.Context {
				md := metadata.Pairs("content-type", "application/grpc")
				return metadata.NewIncomingContext(context.Background(), md)
			}(),
			wantErrMsg: "missing authorization header",
		},
		{
			name: "empty authorization header",
			context: func() context.Context {
				md := metadata.Pairs("authorization", "")
				return metadata.NewIncomingContext(context.Background(), md)
			}(),
			wantErrMsg: "invalid authorization header format",
		},
	}

	interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := interceptor(tt.context, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), mockUnaryHandler)

			require.Nil(t, resp)
			testutil.AssertGRPCError(t, err, codes.Unauthenticated, tt.wantErrMsg)
		})
	}
}

func TestJWTAuthInterceptor_PublicMethods(t *testing.T) {
	publicMethods := []string{
		"/user.v1.UserService/Login",
		"/user.v1.UserService/CreateUser",
		"/user.v1.UserService/RequestPasswordReset",
		"/user.v1.UserService/ResetPassword",
		"/user.v1.UserService/VerifyEmail",
		"/grpc.health.v1.Health/Check",
	}

	interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))

	for _, method := range publicMethods {
		t.Run(fmt.Sprintf("public method %s", method), func(t *testing.T) {
			// Use context without any authentication
			ctx := context.Background()

			resp, err := interceptor(ctx, nil, mockUnaryInfo(method), mockUnaryHandler)

			testutil.AssertNoGRPCError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestJWTAuthInterceptor_WrongSecret(t *testing.T) {
	wrongSecret := []byte("wrong-secret-key-32-characters-long!")
	interceptor := JWTAuthInterceptor(wrongSecret)
	ctx := testutil.CreateTestContextForUser1()

	resp, err := interceptor(ctx, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), mockUnaryHandler)

	require.Nil(t, resp)
	testutil.AssertGRPCError(t, err, codes.Unauthenticated, "invalid token")
	assert.Contains(t, err.Error(), "signature is invalid")
}

func TestIsPublicMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		// Public methods
		{"/user.v1.UserService/Login", true},
		{"/user.v1.UserService/CreateUser", true},
		{"/user.v1.UserService/RequestPasswordReset", true},
		{"/user.v1.UserService/ResetPassword", true},
		{"/user.v1.UserService/VerifyEmail", true},
		{"/grpc.health.v1.Health/Check", true},

		// Private methods
		{"/user.v1.UserService/GetProfile", false},
		{"/character.v1.CharacterService/CreateCharacter", false},
		{"/chunk.v1.ChunkService/GenerateChunk", false},
		{"/world.v1.WorldService/CreateWorld", false},
		{"", false},
		{"/unknown/service/method", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := isPublicMethod(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateJWTToken(t *testing.T) {
	secretKey := []byte(testutil.TestJWTSecretKey)

	tests := []struct {
		name          string
		token         string
		secret        []byte
		wantErr       bool
		validateClaim func(t *testing.T, claims jwt.MapClaims)
	}{
		{
			name:   "valid User1 token",
			token:  testutil.PreGeneratedJWTTokens.ValidUser1Token,
			secret: secretKey,
			validateClaim: func(t *testing.T, claims jwt.MapClaims) {
				assert.Equal(t, testutil.UUIDTestData.User1, claims["user_id"])
				assert.Equal(t, "testuser1", claims["username"])
				assert.Equal(t, "user1@example.com", claims["email"])
			},
		},
		{
			name:   "valid User2 token",
			token:  testutil.PreGeneratedJWTTokens.ValidUser2Token,
			secret: secretKey,
			validateClaim: func(t *testing.T, claims jwt.MapClaims) {
				assert.Equal(t, testutil.UUIDTestData.User2, claims["user_id"])
				assert.Equal(t, "testuser2", claims["username"])
				assert.Equal(t, "user2@example.com", claims["email"])
			},
		},
		{
			name:    "expired token",
			token:   testutil.PreGeneratedJWTTokens.ExpiredToken,
			secret:  secretKey,
			wantErr: true,
		},
		{
			name:    "invalid signature",
			token:   testutil.PreGeneratedJWTTokens.InvalidSignature,
			secret:  secretKey,
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   testutil.PreGeneratedJWTTokens.MalformedToken,
			secret:  secretKey,
			wantErr: true,
		},
		{
			name:    "wrong secret",
			token:   testutil.PreGeneratedJWTTokens.ValidUser1Token,
			secret:  []byte("wrong-secret-key"),
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			secret:  secretKey,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validateJWTToken(tt.token, tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				require.NoError(t, err)
				require.NotNil(t, claims)
				
				if tt.validateClaim != nil {
					tt.validateClaim(t, claims)
				}
			}
		})
	}
}

func TestValidateJWTToken_SigningMethodValidation(t *testing.T) {
	secretKey := []byte(testutil.TestJWTSecretKey)

	// Use a malformed token that claims to use RS256 instead of HS256
	// This will fail because our validateJWTToken function only accepts HMAC signing methods
	tokenString := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdC11c2VyIiwidXNlcm5hbWUiOiJ0ZXN0dXNlciIsImV4cCI6MTYwOTQ1OTIwMCwiaWF0IjoxNjA5NDU5MjAwfQ.invalid"

	claims, err := validateJWTToken(tokenString, secretKey)

	require.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

func TestGetUserContextHelpers(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		username     string
		expectUserID bool
		expectUser   bool
	}{
		{
			name:         "both user ID and username present",
			userID:       testutil.UUIDTestData.User1,
			username:     "testuser1",
			expectUserID: true,
			expectUser:   true,
		},
		{
			name:         "only user ID present",
			userID:       testutil.UUIDTestData.User2,
			username:     "",
			expectUserID: true,
			expectUser:   true, // Type assertion succeeds, even for empty string
		},
		{
			name:         "only username present",
			userID:       "",
			username:     "testuser2",
			expectUserID: true, // Type assertion succeeds, even for empty string
			expectUser:   true,
		},
		{
			name:         "neither present",
			userID:       "",
			username:     "",
			expectUserID: true, // Type assertion succeeds, even for empty string
			expectUser:   true, // Type assertion succeeds, even for empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := CreateTestContextWithAuth(tt.userID, tt.username)

			// Test GetUserIDFromContext
			userID, ok := GetUserIDFromContext(ctx)
			assert.Equal(t, tt.expectUserID, ok)
			if tt.expectUserID {
				assert.Equal(t, tt.userID, userID)
			} else {
				assert.Empty(t, userID)
			}

			// Test GetUsernameFromContext
			username, ok := GetUsernameFromContext(ctx)
			assert.Equal(t, tt.expectUser, ok)
			if tt.expectUser {
				assert.Equal(t, tt.username, username)
			} else {
				assert.Empty(t, username)
			}
		})
	}
}

func TestCreateTestContextWithAuth(t *testing.T) {
	userID := testutil.UUIDTestData.User1
	username := "testuser1"

	ctx := CreateTestContextWithAuth(userID, username)

	// Verify user ID is correctly stored
	retrievedUserID, ok := GetUserIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, userID, retrievedUserID)

	// Verify username is correctly stored
	retrievedUsername, ok := GetUsernameFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, username, retrievedUsername)
}

// Integration test that simulates the full middleware flow
func TestJWTAuthInterceptor_Integration(t *testing.T) {
	// Create interceptor
	interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))

	// Test case 1: Successful authentication flow
	t.Run("successful authentication flow", func(t *testing.T) {
		ctx := testutil.CreateTestContextForUser1()

		// Create a handler that extracts user info
		handler := func(ctx context.Context, req any) (any, error) {
			userID, userIDOK := GetUserIDFromContext(ctx)
			username, usernameOK := GetUsernameFromContext(ctx)

			return map[string]interface{}{
				"user_id":      userID,
				"username":     username,
				"user_id_ok":   userIDOK,
				"username_ok":  usernameOK,
			}, nil
		}

		resp, err := interceptor(ctx, nil, mockUnaryInfo("/character.v1.CharacterService/CreateCharacter"), handler)

		require.NoError(t, err)
		require.NotNil(t, resp)

		result, ok := resp.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, testutil.UUIDTestData.User1, result["user_id"])
		assert.Equal(t, "testuser1", result["username"])
		assert.True(t, result["user_id_ok"].(bool))
		assert.True(t, result["username_ok"].(bool))
	})

	// Test case 2: Public method bypass
	t.Run("public method bypass", func(t *testing.T) {
		// Use context without any authentication
		ctx := context.Background()

		handler := func(ctx context.Context, req any) (any, error) {
			return "public method success", nil
		}

		resp, err := interceptor(ctx, nil, mockUnaryInfo("/user.v1.UserService/Login"), handler)

		require.NoError(t, err)
		assert.Equal(t, "public method success", resp)
	})
}

// Benchmark tests for middleware performance
func BenchmarkJWTAuthInterceptor_ValidToken(b *testing.B) {
	interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))
	ctx := testutil.CreateTestContextForUser1()
	info := mockUnaryInfo("/character.v1.CharacterService/CreateCharacter")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, nil, info, mockUnaryHandler)
	}
}

func BenchmarkJWTAuthInterceptor_PublicMethod(b *testing.B) {
	interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))
	ctx := context.Background()
	info := mockUnaryInfo("/user.v1.UserService/Login")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, nil, info, mockUnaryHandler)
	}
}

func BenchmarkValidateJWTToken(b *testing.B) {
	secretKey := []byte(testutil.TestJWTSecretKey)
	token := testutil.PreGeneratedJWTTokens.ValidUser1Token

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validateJWTToken(token, secretKey)
	}
}

func BenchmarkIsPublicMethod(b *testing.B) {
	methods := []string{
		"/user.v1.UserService/Login",
		"/character.v1.CharacterService/CreateCharacter",
		"/user.v1.UserService/CreateUser",
		"/chunk.v1.ChunkService/GenerateChunk",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		method := methods[i%len(methods)]
		_ = isPublicMethod(method)
	}
}

// Edge case and security tests
func TestJWTAuthInterceptor_EdgeCases(t *testing.T) {
	interceptor := JWTAuthInterceptor([]byte(testutil.TestJWTSecretKey))

	t.Run("nil request", func(t *testing.T) {
		ctx := testutil.CreateTestContextForUser1()
		resp, err := interceptor(ctx, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), mockUnaryHandler)
		
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("multiple authorization headers", func(t *testing.T) {
		md := metadata.Pairs(
			"authorization", "Bearer "+testutil.PreGeneratedJWTTokens.ValidUser1Token,
			"authorization", "Bearer invalid-token",
		)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		// Should use the first authorization header
		resp, err := interceptor(ctx, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), mockUnaryHandler)
		
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("case insensitive header handling", func(t *testing.T) {
		// gRPC metadata keys are case insensitive
		md := metadata.Pairs("Authorization", "Bearer "+testutil.PreGeneratedJWTTokens.ValidUser1Token)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		resp, err := interceptor(ctx, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), mockUnaryHandler)
		
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("token with extra whitespace", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer  "+testutil.PreGeneratedJWTTokens.ValidUser1Token+"  ")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		// This should fail because our implementation doesn't trim whitespace
		_, err := interceptor(ctx, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), mockUnaryHandler)
		
		testutil.AssertGRPCError(t, err, codes.Unauthenticated)
	})
}

func TestJWTAuthInterceptor_TokenWithMissingClaims(t *testing.T) {
	secretKey := []byte(testutil.TestJWTSecretKey)

	// Create token with missing claims
	incompleteToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
		// Missing user_id and username
	})

	tokenString, err := incompleteToken.SignedString(secretKey)
	require.NoError(t, err)

	ctx := testutil.CreateTestContextWithPreGeneratedJWT(tokenString)
	interceptor := JWTAuthInterceptor(secretKey)

	// The middleware should still succeed, but context values will be empty
	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "success", nil
	}

	resp, err := interceptor(ctx, nil, mockUnaryInfo("/user.v1.UserService/GetProfile"), handler)

	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify that missing claims result in empty context values
	userID, userIDOK := GetUserIDFromContext(capturedCtx)
	username, usernameOK := GetUsernameFromContext(capturedCtx)

	// The middleware always sets context values, even if claims are missing (empty strings)
	assert.True(t, userIDOK)
	assert.True(t, usernameOK)
	assert.Empty(t, userID)
	assert.Empty(t, username)
}