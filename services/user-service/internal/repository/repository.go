package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/user-service/internal/types"
)

// Repository handles database operations for user service
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, userID string) (*types.User, error) {
	var u types.User
	err := r.db.QueryRow(ctx, `
		SELECT id, phone, email, first_name, last_name, kyc_status, kyc_tier,
		       kyc_submitted_at, kyc_verified_at, alpaca_account_id, is_active,
		       last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&u.ID, &u.Phone, &u.Email, &u.FirstName, &u.LastName, &u.KYCStatus, &u.KYCTier,
		&u.KYCSubmittedAt, &u.KYCVerifiedAt, &u.AlpacaAccountID, &u.IsActive,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &u, nil
}

// UpdateProfile updates a user's profile information
func (r *Repository) UpdateProfile(ctx context.Context, userID string, req *types.UpdateProfileRequest) error {
	query := "UPDATE users SET updated_at = NOW()"
	args := []any{}
	argNum := 1

	if req.FirstName != nil {
		query += fmt.Sprintf(", first_name = $%d", argNum)
		args = append(args, *req.FirstName)
		argNum++
	}
	if req.LastName != nil {
		query += fmt.Sprintf(", last_name = $%d", argNum)
		args = append(args, *req.LastName)
		argNum++
	}
	if req.Email != nil {
		query += fmt.Sprintf(", email = $%d", argNum)
		args = append(args, *req.Email)
		argNum++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, userID)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// DeactivateUser soft-deletes a user by setting is_active to false
func (r *Repository) DeactivateUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}

// GetUserSettings retrieves user settings (using defaults if not stored)
func (r *Repository) GetUserSettings(ctx context.Context, userID string) (*types.UserSettings, error) {
	// For now, return default settings since we don't have a settings table
	// In a real implementation, you'd have a user_settings table
	return &types.UserSettings{
		UserID:           userID,
		NotifySMS:        true,
		NotifyEmail:      false,
		NotifyPush:       true,
		DefaultCurrency:  "KES",
		Language:         "en",
		TwoFactorEnabled: false,
	}, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *Repository) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// CheckEmailExists checks if an email is already in use
func (r *Repository) CheckEmailExists(ctx context.Context, email, excludeUserID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE email = $1 AND id != $2
	`, email, excludeUserID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check email: %w", err)
	}

	return count > 0, nil
}
