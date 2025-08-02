package middleware

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
)

// JWTAuthInterceptor creates a gRPC interceptor for JWT authentication
func JWTAuthInterceptor(jwtSecret []byte) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Skip authentication for certain methods (like login, register)
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		authorization := md.Get("authorization")
		if len(authorization) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		// Extract token from "Bearer <token>" format
		token := strings.TrimPrefix(authorization[0], "Bearer ")
		if token == authorization[0] {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization header format")
		}

		// Validate JWT token
		claims, err := validateJWTToken(token, jwtSecret)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Add user info to context
		ctx = context.WithValue(ctx, userIDKey, claims[fmt.Sprint(userIDKey)])
		ctx = context.WithValue(ctx, usernameKey, claims[fmt.Sprint(usernameKey)])

		return handler(ctx, req)
	}
}

// isPublicMethod checks if a method should skip authentication
func isPublicMethod(method string) bool {
	publicMethods := []string{
		"/user.v1.UserService/Login",
		"/user.v1.UserService/CreateUser",
		"/user.v1.UserService/RequestPasswordReset",
		"/user.v1.UserService/ResetPassword",
		"/user.v1.UserService/VerifyEmail",
		"/grpc.health.v1.Health/Check",
	}

	return slices.Contains(publicMethods, method)
}

// validateJWTToken validates a JWT token and returns the claims
func validateJWTToken(tokenString string, jwtSecret []byte) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

// GetUsernameFromContext extracts username from context
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value("username").(string)
	return username, ok
}
