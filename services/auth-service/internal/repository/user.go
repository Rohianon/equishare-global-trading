package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/types"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// userColumns is the list of columns to select for a user.
const userColumns = `id, phone, email, username, display_name, avatar_url, password_hash, pin_hash,
	first_name, last_name, kyc_status, kyc_tier, primary_auth_provider, phone_verified,
	email_verified, alpaca_account_id, is_active, created_at, updated_at`

func scanUser(row pgx.Row) (*types.User, error) {
	var user types.User
	err := row.Scan(
		&user.ID, &user.Phone, &user.Email, &user.Username, &user.DisplayName, &user.AvatarURL,
		&user.PasswordHash, &user.PINHash, &user.FirstName, &user.LastName,
		&user.KYCStatus, &user.KYCTier, &user.PrimaryAuthProvider, &user.PhoneVerified,
		&user.EmailVerified, &user.AlpacaAccountID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
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
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO users (phone, pin_hash, password_hash, phone_verified, primary_auth_provider)
		VALUES ($1, $2, $3, true, 'phone')
		RETURNING %s
	`, userColumns), phone, pinHash, passwordHash)

	return scanUser(row)
}

func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (*types.User, error) {
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM users WHERE phone = $1
	`, userColumns), phone)

	return scanUser(row)
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*types.User, error) {
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM users WHERE id = $1
	`, userColumns), id)

	return scanUser(row)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*types.User, error) {
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM users WHERE email = $1
	`, userColumns), email)

	return scanUser(row)
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*types.User, error) {
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM users WHERE username = $1
	`, userColumns), username)

	return scanUser(row)
}

func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}
	return exists, nil
}

// CreateOAuthUser creates a new user from OAuth provider data.
func (r *UserRepository) CreateOAuthUser(ctx context.Context, params types.CreateOAuthUserParams) (*types.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create user
	var email *string
	if params.Email != "" {
		email = &params.Email
	}
	var displayName *string
	if params.DisplayName != "" {
		displayName = &params.DisplayName
	}
	var avatarURL *string
	if params.AvatarURL != "" {
		avatarURL = &params.AvatarURL
	}

	row := tx.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO users (email, display_name, avatar_url, email_verified, primary_auth_provider)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING %s
	`, userColumns), email, displayName, avatarURL, params.EmailVerified, params.Provider)

	user, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create OAuth identity
	_, err = tx.Exec(ctx, `
		INSERT INTO oauth_identities (user_id, provider, provider_user_id, provider_email, provider_name)
		VALUES ($1, $2, $3, $4, $5)
	`, user.ID, params.Provider, params.ProviderUserID, email, displayName)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth identity: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return user, nil
}

// CreateEmailUser creates a new user from magic link email.
func (r *UserRepository) CreateEmailUser(ctx context.Context, email string) (*types.User, error) {
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO users (email, email_verified, primary_auth_provider)
		VALUES ($1, true, 'email')
		RETURNING %s
	`, userColumns), email)

	return scanUser(row)
}

// GetOAuthIdentity retrieves an OAuth identity by provider and provider user ID.
func (r *UserRepository) GetOAuthIdentity(ctx context.Context, provider, providerUserID string) (*types.OAuthIdentity, error) {
	var identity types.OAuthIdentity
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_user_id, provider_email, provider_name, created_at, updated_at
		FROM oauth_identities
		WHERE provider = $1 AND provider_user_id = $2
	`, provider, providerUserID).Scan(
		&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUserID,
		&identity.ProviderEmail, &identity.ProviderName, &identity.CreatedAt, &identity.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth identity: %w", err)
	}
	return &identity, nil
}

// GetOAuthIdentitiesByUser retrieves all OAuth identities for a user.
func (r *UserRepository) GetOAuthIdentitiesByUser(ctx context.Context, userID string) ([]*types.OAuthIdentity, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, provider, provider_user_id, provider_email, provider_name, created_at, updated_at
		FROM oauth_identities
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth identities: %w", err)
	}
	defer rows.Close()

	var identities []*types.OAuthIdentity
	for rows.Next() {
		var identity types.OAuthIdentity
		if err := rows.Scan(
			&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUserID,
			&identity.ProviderEmail, &identity.ProviderName, &identity.CreatedAt, &identity.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan OAuth identity: %w", err)
		}
		identities = append(identities, &identity)
	}

	return identities, nil
}

// CreateOAuthIdentity links an OAuth provider to an existing user.
func (r *UserRepository) CreateOAuthIdentity(ctx context.Context, userID, provider, providerUserID string, email, name *string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO oauth_identities (user_id, provider, provider_user_id, provider_email, provider_name)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, provider, providerUserID, email, name)
	if err != nil {
		return fmt.Errorf("failed to create OAuth identity: %w", err)
	}
	return nil
}

// DeleteOAuthIdentity removes an OAuth identity.
func (r *UserRepository) DeleteOAuthIdentity(ctx context.Context, userID, provider string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM oauth_identities
		WHERE user_id = $1 AND provider = $2
	`, userID, provider)
	if err != nil {
		return fmt.Errorf("failed to delete OAuth identity: %w", err)
	}
	return nil
}

// UpdateUsername sets the username for a user.
func (r *UserRepository) UpdateUsername(ctx context.Context, userID, username string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET username = $2, updated_at = NOW()
		WHERE id = $1
	`, userID, username)
	if err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}
	return nil
}

// LinkPhone links a phone number to an existing user.
func (r *UserRepository) LinkPhone(ctx context.Context, userID, phone, pinHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET phone = $2, pin_hash = $3, phone_verified = true, updated_at = NOW()
		WHERE id = $1
	`, userID, phone, pinHash)
	if err != nil {
		return fmt.Errorf("failed to link phone: %w", err)
	}
	return nil
}
