package bootstrap

import (
	"github.com/luong-vh/Digimart_Backend/internal/auth"
	"github.com/redis/go-redis/v9"
)

// InitializeTokenService sets up the token service for JWT authentication using a provided Redis client
func InitializeTokenService(redisClient *redis.Client) (*auth.TokenService, error) {
	tokenService := auth.NewTokenService(redisClient)
	auth.SetTokenService(tokenService)

	return tokenService, nil
}
