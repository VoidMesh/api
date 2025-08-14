package testutil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// TestPreGeneratedJWTTokens verifies that the pre-generated JWT tokens are valid
func TestPreGeneratedJWTTokens(t *testing.T) {
	t.Run("ValidUser1Token", func(t *testing.T) {
		token, err := jwt.Parse(PreGeneratedJWTTokens.ValidUser1Token, func(token *jwt.Token) (interface{}, error) {
			return []byte(TestJWTSecretKey), nil
		})
		require.NoError(t, err)
		require.True(t, token.Valid)

		claims, ok := token.Claims.(jwt.MapClaims)
		require.True(t, ok)

		assert.Equal(t, UUIDTestData.User1, claims["user_id"])
		assert.Equal(t, "testuser1", claims["username"])
		assert.Equal(t, "user1@example.com", claims["email"])
		
		// Check expiration is in the future (year 2030)
		exp, ok := claims["exp"].(float64)
		require.True(t, ok)
		expTime := time.Unix(int64(exp), 0)
		assert.True(t, expTime.After(time.Now()), "Token should not be expired")
		assert.Equal(t, 2030, expTime.Year(), "Token should expire in 2030")
	})

	t.Run("ValidUser2Token", func(t *testing.T) {
		token, err := jwt.Parse(PreGeneratedJWTTokens.ValidUser2Token, func(token *jwt.Token) (interface{}, error) {
			return []byte(TestJWTSecretKey), nil
		})
		require.NoError(t, err)
		require.True(t, token.Valid)

		claims, ok := token.Claims.(jwt.MapClaims)
		require.True(t, ok)

		assert.Equal(t, UUIDTestData.User2, claims["user_id"])
		assert.Equal(t, "testuser2", claims["username"])
		assert.Equal(t, "user2@example.com", claims["email"])
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		token, err := jwt.Parse(PreGeneratedJWTTokens.ExpiredToken, func(token *jwt.Token) (interface{}, error) {
			return []byte(TestJWTSecretKey), nil
		})
		
		// Token parsing succeeds but validation should fail due to expiration
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token is expired")
		assert.False(t, token.Valid)
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		_, err := jwt.Parse(PreGeneratedJWTTokens.InvalidSignature, func(token *jwt.Token) (interface{}, error) {
			return []byte(TestJWTSecretKey), nil
		})
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signature")
	})

	t.Run("MalformedToken", func(t *testing.T) {
		_, err := jwt.Parse(PreGeneratedJWTTokens.MalformedToken, func(token *jwt.Token) (interface{}, error) {
			return []byte(TestJWTSecretKey), nil
		})
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token contains an invalid number of segments")
	})
}

// TestContextHelpers verifies that the context helper functions work correctly
func TestContextHelpers(t *testing.T) {
	t.Run("CreateTestContextForUser1", func(t *testing.T) {
		ctx := CreateTestContextForUser1()
		require.NotNil(t, ctx)
		
		// Verify that the context contains the authorization metadata
		md, ok := metadata.FromIncomingContext(ctx)
		require.True(t, ok)
		
		auth := md.Get("authorization")
		require.Len(t, auth, 1)
		assert.Equal(t, "Bearer "+PreGeneratedJWTTokens.ValidUser1Token, auth[0])
	})

	t.Run("CreateTestContextForUser2", func(t *testing.T) {
		ctx := CreateTestContextForUser2()
		require.NotNil(t, ctx)
		
		md, ok := metadata.FromIncomingContext(ctx)
		require.True(t, ok)
		
		auth := md.Get("authorization")
		require.Len(t, auth, 1)
		assert.Equal(t, "Bearer "+PreGeneratedJWTTokens.ValidUser2Token, auth[0])
	})

	t.Run("CreateTestContextWithExpiredToken", func(t *testing.T) {
		ctx := CreateTestContextWithExpiredToken()
		require.NotNil(t, ctx)
		
		md, ok := metadata.FromIncomingContext(ctx)
		require.True(t, ok)
		
		auth := md.Get("authorization")
		require.Len(t, auth, 1)
		assert.Equal(t, "Bearer "+PreGeneratedJWTTokens.ExpiredToken, auth[0])
	})

	t.Run("CreateTestContextWithInvalidToken", func(t *testing.T) {
		ctx := CreateTestContextWithInvalidToken()
		require.NotNil(t, ctx)
		
		md, ok := metadata.FromIncomingContext(ctx)
		require.True(t, ok)
		
		auth := md.Get("authorization")
		require.Len(t, auth, 1)
		assert.Equal(t, "Bearer "+PreGeneratedJWTTokens.InvalidSignature, auth[0])
	})
}