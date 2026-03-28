package auth

import (
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/platform/database"
)

type Service struct {
	db    *database.Pool
	cache *redis.Client
}

func NewService(db *database.Pool, cache *redis.Client) *Service {
	return &Service{db: db, cache: cache}
}
