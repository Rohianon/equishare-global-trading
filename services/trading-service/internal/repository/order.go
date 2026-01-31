package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/trading-service/internal/types"
)

// OrderRepository handles order database operations
type OrderRepository struct {
	db *pgxpool.Pool
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create creates a new order
func (r *OrderRepository) Create(ctx context.Context, order *types.Order) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO orders (user_id, alpaca_order_id, symbol, side, type, amount, qty, status, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, order.UserID, order.AlpacaOrderID, order.Symbol, order.Side, order.Type,
		order.Amount, order.Qty, order.Status, order.Source,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(ctx context.Context, orderID string) (*types.Order, error) {
	var order types.Order
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, alpaca_order_id, symbol, side, type, amount, qty,
		       filled_qty, filled_avg_price, status, source, failed_reason,
		       filled_at, canceled_at, created_at, updated_at
		FROM orders WHERE id = $1
	`, orderID).Scan(
		&order.ID, &order.UserID, &order.AlpacaOrderID, &order.Symbol, &order.Side,
		&order.Type, &order.Amount, &order.Qty, &order.FilledQty, &order.FilledAvgPrice,
		&order.Status, &order.Source, &order.FailedReason,
		&order.FilledAt, &order.CanceledAt, &order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

// GetByAlpacaOrderID retrieves an order by Alpaca order ID
func (r *OrderRepository) GetByAlpacaOrderID(ctx context.Context, alpacaOrderID string) (*types.Order, error) {
	var order types.Order
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, alpaca_order_id, symbol, side, type, amount, qty,
		       filled_qty, filled_avg_price, status, source, failed_reason,
		       filled_at, canceled_at, created_at, updated_at
		FROM orders WHERE alpaca_order_id = $1
	`, alpacaOrderID).Scan(
		&order.ID, &order.UserID, &order.AlpacaOrderID, &order.Symbol, &order.Side,
		&order.Type, &order.Amount, &order.Qty, &order.FilledQty, &order.FilledAvgPrice,
		&order.Status, &order.Source, &order.FailedReason,
		&order.FilledAt, &order.CanceledAt, &order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get order by alpaca ID: %w", err)
	}

	return &order, nil
}

// ListByUser retrieves orders for a user
func (r *OrderRepository) ListByUser(ctx context.Context, userID string, status string, limit int) ([]types.Order, error) {
	query := `
		SELECT id, user_id, alpaca_order_id, symbol, side, type, amount, qty,
		       filled_qty, filled_avg_price, status, source, failed_reason,
		       filled_at, canceled_at, created_at, updated_at
		FROM orders WHERE user_id = $1
	`
	args := []any{userID}

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []types.Order
	for rows.Next() {
		var order types.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.AlpacaOrderID, &order.Symbol, &order.Side,
			&order.Type, &order.Amount, &order.Qty, &order.FilledQty, &order.FilledAvgPrice,
			&order.Status, &order.Source, &order.FailedReason,
			&order.FilledAt, &order.CanceledAt, &order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// UpdateStatus updates the order status
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, orderID)

	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// UpdateFill updates the order with fill information
func (r *OrderRepository) UpdateFill(ctx context.Context, alpacaOrderID string, filledQty, filledAvgPrice float64, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE orders
		SET filled_qty = $1, filled_avg_price = $2, status = $3, filled_at = NOW(), updated_at = NOW()
		WHERE alpaca_order_id = $4
	`, filledQty, filledAvgPrice, status, alpacaOrderID)

	if err != nil {
		return fmt.Errorf("failed to update order fill: %w", err)
	}

	return nil
}

// UpdateCanceled marks the order as canceled
func (r *OrderRepository) UpdateCanceled(ctx context.Context, alpacaOrderID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE orders SET status = 'canceled', canceled_at = NOW(), updated_at = NOW()
		WHERE alpaca_order_id = $1
	`, alpacaOrderID)

	if err != nil {
		return fmt.Errorf("failed to update order canceled: %w", err)
	}

	return nil
}

// UpdateFailed marks the order as failed
func (r *OrderRepository) UpdateFailed(ctx context.Context, alpacaOrderID, reason string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE orders SET status = 'failed', failed_reason = $1, updated_at = NOW()
		WHERE alpaca_order_id = $2
	`, reason, alpacaOrderID)

	if err != nil {
		return fmt.Errorf("failed to update order failed: %w", err)
	}

	return nil
}
