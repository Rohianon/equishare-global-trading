package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/trading-service/internal/types"
)

// HoldingRepository handles holding database operations
type HoldingRepository struct {
	db *pgxpool.Pool
}

// NewHoldingRepository creates a new holding repository
func NewHoldingRepository(db *pgxpool.Pool) *HoldingRepository {
	return &HoldingRepository{db: db}
}

// Upsert creates or updates a holding (after order fill)
func (r *HoldingRepository) Upsert(ctx context.Context, userID, symbol string, qty, avgPrice float64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO holdings (user_id, symbol, quantity, average_cost)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, symbol) DO UPDATE SET
			quantity = holdings.quantity + EXCLUDED.quantity,
			average_cost = CASE
				WHEN holdings.quantity + EXCLUDED.quantity = 0 THEN 0
				ELSE (holdings.quantity * holdings.average_cost + EXCLUDED.quantity * EXCLUDED.average_cost)
				     / (holdings.quantity + EXCLUDED.quantity)
			END,
			updated_at = NOW()
	`, userID, symbol, qty, avgPrice)

	if err != nil {
		return fmt.Errorf("failed to upsert holding: %w", err)
	}

	return nil
}

// GetByUserAndSymbol retrieves a specific holding
func (r *HoldingRepository) GetByUserAndSymbol(ctx context.Context, userID, symbol string) (*types.Holding, error) {
	var holding types.Holding
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, symbol, quantity, average_cost, created_at, updated_at
		FROM holdings WHERE user_id = $1 AND symbol = $2
	`, userID, symbol).Scan(
		&holding.ID, &holding.UserID, &holding.Symbol, &holding.Qty,
		&holding.AvgEntryPrice, &holding.CreatedAt, &holding.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get holding: %w", err)
	}

	return &holding, nil
}

// ListByUser retrieves all holdings for a user
func (r *HoldingRepository) ListByUser(ctx context.Context, userID string) ([]types.Holding, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, symbol, quantity, average_cost, created_at, updated_at
		FROM holdings WHERE user_id = $1 AND quantity > 0
		ORDER BY symbol ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list holdings: %w", err)
	}
	defer rows.Close()

	var holdings []types.Holding
	for rows.Next() {
		var holding types.Holding
		err := rows.Scan(
			&holding.ID, &holding.UserID, &holding.Symbol, &holding.Qty,
			&holding.AvgEntryPrice, &holding.CreatedAt, &holding.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan holding: %w", err)
		}
		holdings = append(holdings, holding)
	}

	return holdings, nil
}

// ReduceQty reduces the quantity of a holding (for sell orders)
func (r *HoldingRepository) ReduceQty(ctx context.Context, userID, symbol string, qty float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE holdings
		SET quantity = quantity - $1, updated_at = NOW()
		WHERE user_id = $2 AND symbol = $3
	`, qty, userID, symbol)

	if err != nil {
		return fmt.Errorf("failed to reduce holding qty: %w", err)
	}

	return nil
}

// Delete removes a holding (when qty reaches zero)
func (r *HoldingRepository) Delete(ctx context.Context, userID, symbol string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM holdings WHERE user_id = $1 AND symbol = $2
	`, userID, symbol)

	if err != nil {
		return fmt.Errorf("failed to delete holding: %w", err)
	}

	return nil
}

// HasSufficientQty checks if user has enough shares to sell
func (r *HoldingRepository) HasSufficientQty(ctx context.Context, userID, symbol string, qty float64) (bool, error) {
	var holdingQty float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(quantity, 0) FROM holdings WHERE user_id = $1 AND symbol = $2
	`, userID, symbol).Scan(&holdingQty)

	if err != nil {
		// No holding means 0 qty
		return false, nil
	}

	return holdingQty >= qty, nil
}
