package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/trading-service/internal/types"
)

// WalletRepository handles wallet database operations for trading
type WalletRepository struct {
	db *pgxpool.Pool
}

// NewWalletRepository creates a new wallet repository
func NewWalletRepository(db *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{db: db}
}

// GetByUserAndCurrency retrieves a wallet by user and currency
func (r *WalletRepository) GetByUserAndCurrency(ctx context.Context, userID, currency string) (*types.Wallet, error) {
	var wallet types.Wallet
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, currency, balance, locked_balance, created_at, updated_at
		FROM wallets WHERE user_id = $1 AND currency = $2
	`, userID, currency).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Currency,
		&wallet.Balance, &wallet.LockedBalance,
		&wallet.CreatedAt, &wallet.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	return &wallet, nil
}

// Lock locks funds for an order
func (r *WalletRepository) Lock(ctx context.Context, walletID string, amount float64) error {
	result, err := r.db.Exec(ctx, `
		UPDATE wallets
		SET locked_balance = locked_balance + $1, updated_at = NOW()
		WHERE id = $2 AND balance - locked_balance >= $1
	`, amount, walletID)

	if err != nil {
		return fmt.Errorf("failed to lock funds: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("insufficient balance to lock")
	}

	return nil
}

// Unlock unlocks previously locked funds (for canceled orders)
func (r *WalletRepository) Unlock(ctx context.Context, walletID string, amount float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE wallets
		SET locked_balance = locked_balance - $1, updated_at = NOW()
		WHERE id = $2
	`, amount, walletID)

	if err != nil {
		return fmt.Errorf("failed to unlock funds: %w", err)
	}

	return nil
}

// DebitLocked debits from locked balance (after order fills)
func (r *WalletRepository) DebitLocked(ctx context.Context, walletID string, amount float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE wallets
		SET balance = balance - $1, locked_balance = locked_balance - $1, updated_at = NOW()
		WHERE id = $2
	`, amount, walletID)

	if err != nil {
		return fmt.Errorf("failed to debit locked funds: %w", err)
	}

	return nil
}

// Credit credits the wallet (for sell proceeds)
func (r *WalletRepository) Credit(ctx context.Context, walletID string, amount float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE wallets
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2
	`, amount, walletID)

	if err != nil {
		return fmt.Errorf("failed to credit wallet: %w", err)
	}

	return nil
}
