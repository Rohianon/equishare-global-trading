package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/types"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)", phone).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return exists, nil
}

func (r *UserRepository) Create(ctx context.Context, phone, pinHash string, passwordHash *string) (*types.User, error) {
	var user types.User

	err := r.db.QueryRow(ctx, `
		INSERT INTO users (phone, pin_hash, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, phone, email, password_hash, pin_hash, first_name, last_name,
		          kyc_status, kyc_tier, alpaca_account_id, is_active, created_at, updated_at
	`, phone, pinHash, passwordHash).Scan(
		&user.ID, &user.Phone, &user.Email, &user.PasswordHash, &user.PINHash,
		&user.FirstName, &user.LastName, &user.KYCStatus, &user.KYCTier,
		&user.AlpacaAccountID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (*types.User, error) {
	var user types.User

	err := r.db.QueryRow(ctx, `
		SELECT id, phone, email, password_hash, pin_hash, first_name, last_name,
		       kyc_status, kyc_tier, alpaca_account_id, is_active, created_at, updated_at
		FROM users WHERE phone = $1
	`, phone).Scan(
		&user.ID, &user.Phone, &user.Email, &user.PasswordHash, &user.PINHash,
		&user.FirstName, &user.LastName, &user.KYCStatus, &user.KYCTier,
		&user.AlpacaAccountID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*types.User, error) {
	var user types.User

	err := r.db.QueryRow(ctx, `
		SELECT id, phone, email, password_hash, pin_hash, first_name, last_name,
		       kyc_status, kyc_tier, alpaca_account_id, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Phone, &user.Email, &user.PasswordHash, &user.PINHash,
		&user.FirstName, &user.LastName, &user.KYCStatus, &user.KYCTier,
		&user.AlpacaAccountID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
