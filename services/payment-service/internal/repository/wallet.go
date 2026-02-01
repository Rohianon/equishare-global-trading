package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/payment-service/internal/types"
)

type WalletRepository struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{db: db}
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

func (r *WalletRepository) CreateTransaction(ctx context.Context, userID, walletID, txType, provider, providerRef string, amount float64, description string) (*types.Transaction, error) {
	var tx types.Transaction

	err := r.db.QueryRow(ctx, `
		INSERT INTO transactions (user_id, wallet_id, type, status, amount, currency, provider, provider_ref, description, completed_at)
		VALUES ($1, $2, $3, 'completed', $4, 'KES', $5, $6, $7, NOW())
		RETURNING id, user_id, wallet_id, type, status, amount, fee, currency, provider,
		          provider_ref, description, completed_at, created_at, updated_at
	`, userID, walletID, txType, amount, provider, providerRef, description).Scan(
		&tx.ID, &tx.UserID, &tx.WalletID, &tx.Type, &tx.Status, &tx.Amount, &tx.Fee,
		&tx.Currency, &tx.Provider, &tx.ProviderRef, &tx.Description, &tx.CompletedAt,
		&tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &tx, nil
}

func (r *WalletRepository) GetTransactions(ctx context.Context, userID string, page, perPage int) ([]types.Transaction, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}
	offset := (page - 1) * perPage

	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, wallet_id, type, status, amount, fee, currency, provider,
		       provider_ref, description, completed_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transactions: %w", err)
	}
	defer rows.Close()

	var transactions []types.Transaction
	for rows.Next() {
		var tx types.Transaction
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.WalletID, &tx.Type, &tx.Status, &tx.Amount, &tx.Fee,
			&tx.Currency, &tx.Provider, &tx.ProviderRef, &tx.Description, &tx.CompletedAt,
			&tx.CreatedAt, &tx.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, total, nil
}
