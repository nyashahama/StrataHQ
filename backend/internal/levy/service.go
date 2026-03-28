package levy

import "github.com/stratahq/backend/internal/platform/database"

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}
