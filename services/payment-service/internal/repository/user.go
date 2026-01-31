package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID       string
	Phone    string
	IsActive bool
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	var user User

	err := r.db.QueryRow(ctx, `
		SELECT id, phone, is_active FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Phone, &user.IsActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
