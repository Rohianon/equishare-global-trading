package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/portfolio-service/internal/types"
)

// Repository handles database operations for portfolio service
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ListHoldingsByUser retrieves all holdings for a user
func (r *Repository) ListHoldingsByUser(ctx context.Context, userID string) ([]types.Holding, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, symbol, quantity, avg_cost_basis, total_cost_basis, created_at, updated_at
		FROM holdings
		WHERE user_id = $1 AND quantity > 0
		ORDER BY symbol ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list holdings: %w", err)
	}
	defer rows.Close()

	var holdings []types.Holding
	for rows.Next() {
		var h types.Holding
		err := rows.Scan(
			&h.ID, &h.UserID, &h.Symbol, &h.Quantity,
			&h.AvgCostBasis, &h.TotalCostBasis, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan holding: %w", err)
		}
		holdings = append(holdings, h)
	}

	return holdings, nil
}

// GetHoldingBySymbol retrieves a specific holding for a user
func (r *Repository) GetHoldingBySymbol(ctx context.Context, userID, symbol string) (*types.Holding, error) {
	var h types.Holding
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, symbol, quantity, avg_cost_basis, total_cost_basis, created_at, updated_at
		FROM holdings
		WHERE user_id = $1 AND symbol = $2 AND quantity > 0
	`, userID, symbol).Scan(
		&h.ID, &h.UserID, &h.Symbol, &h.Quantity,
		&h.AvgCostBasis, &h.TotalCostBasis, &h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("holding not found: %w", err)
	}

	return &h, nil
}

// GetWalletByUserAndCurrency retrieves a wallet for a user
func (r *Repository) GetWalletByUserAndCurrency(ctx context.Context, userID, currency string) (*types.Wallet, error) {
	var w types.Wallet
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, currency, balance, created_at, updated_at
		FROM wallets
		WHERE user_id = $1 AND currency = $2
	`, userID, currency).Scan(
		&w.ID, &w.UserID, &w.Currency, &w.Balance, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}

	return &w, nil
}

// GetTotalCashBalance retrieves total cash balance across all wallets for a user
func (r *Repository) GetTotalCashBalance(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(balance), 0)
		FROM wallets
		WHERE user_id = $1
	`, userID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get cash balance: %w", err)
	}

	return total, nil
}
