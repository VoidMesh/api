# JWT Implementation Guide

This document explains how to use the JWT (JSON Web Token) implementation in the VoidMesh API for production-ready authentication.

## Overview

The authentication system has been upgraded from simple random tokens to industry-standard JWT tokens with the following features:

- **Secure**: Uses HMAC-SHA256 signing algorithm
- **Stateless**: No need to store tokens in database
- **Configurable**: JWT secret can be set via environment variables
- **Production-ready**: Includes proper token validation and middleware
- **Expiration**: Tokens expire after 7 days by default

## Features

### JWT Token Generation
- Tokens include user ID, username, expiration time, and issuer
- 7-day expiration period (configurable)
- Secure HMAC-SHA256 signing

### JWT Validation Middleware
- Automatic token validation for protected endpoints
- Extracts user information and adds to request context
- Configurable public endpoints that skip authentication

## Environment Setup

### Development
For development, a default JWT secret is provided, but you should set your own:

```bash
export JWT_SECRET="your-development-secret-key"
```

### Production
**CRITICAL**: You MUST set a strong JWT secret in production:

```bash
export JWT_SECRET="your-super-secure-production-secret-key-at-least-32-characters-long"
```

### Docker Environment
Add to your `docker-compose.yml` or `.env` file:

```yaml
environment:
  - JWT_SECRET=your-super-secure-production-secret-key
```

## Usage Examples

### 1. User Login
When a user logs in successfully, they receive a JWT token:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "user-uuid",
    "username": "john_doe",
    "email": "john@example.com"
  }
}
```

### 2. Making Authenticated Requests
Include the JWT token in the Authorization header:

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     http://localhost:50051/user.v1.UserService/GetUser
```

### 3. Token Validation
The middleware automatically validates tokens and extracts user information:

```go
// In your handler, access user info from context
func (s *userServiceServer) GetProfile(ctx context.Context, req *userV1.GetProfileRequest) (*userV1.GetProfileResponse, error) {
    userID, ok := middleware.GetUserIDFromContext(ctx)
    if !ok {
        return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
    }
    
    username, _ := middleware.GetUsernameFromContext(ctx)
    // Use userID and username...
}
```

## Security Considerations

### JWT Secret
- **Length**: Use at least 32 characters
- **Complexity**: Include uppercase, lowercase, numbers, and special characters
- **Uniqueness**: Use a different secret for each environment
- **Rotation**: Regularly rotate secrets in production

### Token Expiration
- Default: 7 days
- Configurable in `generateJWTToken` function
- Consider shorter expiration for high-security applications

### Public Endpoints
The following endpoints skip JWT authentication:
- `/user.v1.UserService/Login`
- `/user.v1.UserService/CreateUser`
- `/user.v1.UserService/RequestPasswordReset`
- `/user.v1.UserService/ResetPassword`
- `/user.v1.UserService/VerifyEmail`

## Integration with gRPC Server

To enable JWT middleware in your gRPC server:

```go
// In your server setup
import "github.com/VoidMesh/platform/api/server/middleware"

func main() {
    // Get JWT secret
    jwtSecret := []byte(os.Getenv("JWT_SECRET"))
    if len(jwtSecret) == 0 {
        log.Fatal("JWT_SECRET environment variable is required")
    }
    
    // Create gRPC server with JWT middleware
    s := grpc.NewServer(
        grpc.UnaryInterceptor(middleware.JWTAuthInterceptor(jwtSecret)),
    )
    
    // Register your services...
}
```

## Token Structure

JWT tokens contain the following claims:

```json
{
  "user_id": "uuid-string",
  "username": "user_username",
  "exp": 1234567890,  // Expiration timestamp
  "iat": 1234567890,  // Issued at timestamp
  "iss": "voidmesh-api"  // Issuer
}
```

## Migration from Simple Tokens

### What Changed
1. **Login endpoint**: Now returns JWT tokens instead of random strings
2. **Token validation**: Moved from database lookup to cryptographic verification
3. **Token format**: Structured JWT instead of random hex strings
4. **Expiration**: Built-in token expiration

### Backward Compatibility
- Password reset still uses simple random tokens (stored in database)
- Existing password reset functionality unchanged
- Database schema remains the same

## Troubleshooting

### Common Issues

1. **"Invalid token" errors**
   - Check JWT secret matches between token generation and validation
   - Verify token hasn't expired
   - Ensure proper "Bearer " prefix in Authorization header

2. **"Missing authorization header"**
   - Include `Authorization: Bearer <token>` in requests
   - Check if endpoint requires authentication

3. **"Unexpected signing method"**
   - Token was signed with different algorithm
   - Possible token tampering

### Debug Mode
To debug JWT issues, you can decode tokens at [jwt.io](https://jwt.io) (never paste production tokens there!).

## Best Practices

1. **Environment Variables**: Always use environment variables for secrets
2. **HTTPS**: Use HTTPS in production to protect tokens in transit
3. **Token Storage**: Store tokens securely on client side (httpOnly cookies recommended for web)
4. **Refresh Tokens**: Consider implementing refresh tokens for longer sessions
5. **Logging**: Never log JWT tokens or secrets
6. **Validation**: Always validate tokens on server side

## Future Enhancements

- Refresh token implementation
- Token blacklisting for logout
- Role-based access control (RBAC)
- Token introspection endpoint
- Configurable token expiration