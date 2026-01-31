package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/types"
)

type WalletRepository struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) Create(ctx context.Context, userID, currency string) (*types.Wallet, error) {
	var wallet types.Wallet

	err := r.db.QueryRow(ctx, `
		INSERT INTO wallets (user_id, currency, balance, locked_balance)
		VALUES ($1, $2, 0, 0)
		RETURNING id, user_id, currency, balance, locked_balance, created_at, updated_at
	`, userID, currency).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Currency,
		&wallet.Balance, &wallet.LockedBalance,
		&wallet.CreatedAt, &wallet.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	return &wallet, nil
}

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
