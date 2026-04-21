package bootstrap

import (
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/auth"
	"github.com/redis/go-redis/v9"
)

// InitializeTokenService sets up the token service for JWT authentication using a provided Redis client
func InitializeTokenService(redisClient *redis.Client) (*auth.TokenService, error) {
	tokenService := auth.NewTokenService(redisClient)
	auth.SetTokenService(tokenService)

	return tokenService, nil
}
