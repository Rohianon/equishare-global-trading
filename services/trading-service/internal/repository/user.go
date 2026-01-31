package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/trading-service/internal/types"
)

// UserRepository handles user database operations for trading
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, userID string) (*types.User, error) {
	var user types.User
	err := r.db.QueryRow(ctx, `
		SELECT id, phone, is_active, is_kyc_verified
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Phone, &user.IsActive, &user.IsKYCVerified)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
