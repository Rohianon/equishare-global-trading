package alpaca

import (
	"context"
	"fmt"
	"time"
)

// OrderSide represents buy or sell
type OrderSide string

const (
	Buy  OrderSide = "buy"
	Sell OrderSide = "sell"
)

// OrderType represents the order type
type OrderType string

const (
	Market        OrderType = "market"
	Limit         OrderType = "limit"
	Stop          OrderType = "stop"
	StopLimit     OrderType = "stop_limit"
	TrailingStop  OrderType = "trailing_stop"
)

// TimeInForce represents how long an order stays active
type TimeInForce string

const (
	Day TimeInForce = "day" // Valid for the day only
	GTC TimeInForce = "gtc" // Good til canceled
	IOC TimeInForce = "ioc" // Immediate or cancel
	FOK TimeInForce = "fok" // Fill or kill
)

// OrderStatus represents the current status of an order
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "new"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusDoneForDay      OrderStatus = "done_for_day"
	OrderStatusCanceled        OrderStatus = "canceled"
	OrderStatusExpired         OrderStatus = "expired"
	OrderStatusReplaced        OrderStatus = "replaced"
	OrderStatusPendingCancel   OrderStatus = "pending_cancel"
	OrderStatusPendingReplace  OrderStatus = "pending_replace"
	OrderStatusAccepted        OrderStatus = "accepted"
	OrderStatusPendingNew      OrderStatus = "pending_new"
	OrderStatusAcceptedForBidding OrderStatus = "accepted_for_bidding"
	OrderStatusStopped         OrderStatus = "stopped"
	OrderStatusRejected        OrderStatus = "rejected"
	OrderStatusSuspended       OrderStatus = "suspended"
	OrderStatusCalculated      OrderStatus = "calculated"
)

// CreateOrderRequest represents an order submission request
type CreateOrderRequest struct {
	Symbol        string      `json:"symbol"`
	Qty           string      `json:"qty,omitempty"`       // Number of shares (mutually exclusive with notional)
	Notional      string      `json:"notional,omitempty"`  // Dollar amount for fractional shares
	Side          OrderSide   `json:"side"`
	Type          OrderType   `json:"type"`
	TimeInForce   TimeInForce `json:"time_in_force"`
	LimitPrice    string      `json:"limit_price,omitempty"`
	StopPrice     string      `json:"stop_price,omitempty"`
	ClientOrderID string      `json:"client_order_id,omitempty"`
	ExtendedHours bool        `json:"extended_hours,omitempty"`
}

// Order represents an Alpaca order
type Order struct {
	ID             string      `json:"id"`
	ClientOrderID  string      `json:"client_order_id"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	SubmittedAt    time.Time   `json:"submitted_at"`
	FilledAt       *time.Time  `json:"filled_at"`
	ExpiredAt      *time.Time  `json:"expired_at"`
	CanceledAt     *time.Time  `json:"canceled_at"`
	FailedAt       *time.Time  `json:"failed_at"`
	ReplacedAt     *time.Time  `json:"replaced_at"`
	ReplacedBy     *string     `json:"replaced_by"`
	Replaces       *string     `json:"replaces"`
	AssetID        string      `json:"asset_id"`
	Symbol         string      `json:"symbol"`
	AssetClass     string      `json:"asset_class"`
	Qty            string      `json:"qty"`
	FilledQty      string      `json:"filled_qty"`
	FilledAvgPrice string      `json:"filled_avg_price"`
	OrderClass     string      `json:"order_class"`
	OrderType      OrderType   `json:"type"`
	Side           OrderSide   `json:"side"`
	TimeInForce    TimeInForce `json:"time_in_force"`
	LimitPrice     string      `json:"limit_price"`
	StopPrice      string      `json:"stop_price"`
	Status         OrderStatus `json:"status"`
	ExtendedHours  bool        `json:"extended_hours"`
	Notional       string      `json:"notional"`
}

// CreateOrder submits a new order to Alpaca
func (c *Client) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	url := fmt.Sprintf("%s/v2/orders", c.baseURL)
	resp, err := c.doRequest(ctx, "POST", url, req)
	if err != nil {
		return nil, err
	}

	var order Order
	if err := decodeResponse(resp, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

// GetOrder retrieves an order by ID
func (c *Client) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	url := fmt.Sprintf("%s/v2/orders/%s", c.baseURL, orderID)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var order Order
	if err := decodeResponse(resp, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

// GetOrderByClientID retrieves an order by client order ID
func (c *Client) GetOrderByClientID(ctx context.Context, clientOrderID string) (*Order, error) {
	url := fmt.Sprintf("%s/v2/orders:by_client_order_id?client_order_id=%s", c.baseURL, clientOrderID)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var order Order
	if err := decodeResponse(resp, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

// ListOrdersParams contains parameters for listing orders
type ListOrdersParams struct {
	Status    string // open, closed, all
	Limit     int
	After     *time.Time
	Until     *time.Time
	Direction string // asc, desc
	Nested    bool
	Symbols   string // comma-separated
}

// ListOrders retrieves a list of orders
func (c *Client) ListOrders(ctx context.Context, params *ListOrdersParams) ([]Order, error) {
	url := fmt.Sprintf("%s/v2/orders", c.baseURL)
	if params != nil {
		query := "?"
		if params.Status != "" {
			query += fmt.Sprintf("status=%s&", params.Status)
		}
		if params.Limit > 0 {
			query += fmt.Sprintf("limit=%d&", params.Limit)
		}
		if params.Direction != "" {
			query += fmt.Sprintf("direction=%s&", params.Direction)
		}
		if params.Symbols != "" {
			query += fmt.Sprintf("symbols=%s&", params.Symbols)
		}
		url += query
	}

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var orders []Order
	if err := decodeResponse(resp, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

// CancelOrder cancels an open order by ID
func (c *Client) CancelOrder(ctx context.Context, orderID string) error {
	url := fmt.Sprintf("%s/v2/orders/%s", c.baseURL, orderID)
	resp, err := c.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// CancelAllOrders cancels all open orders
func (c *Client) CancelAllOrders(ctx context.Context) error {
	url := fmt.Sprintf("%s/v2/orders", c.baseURL)
	resp, err := c.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// ReplaceOrderRequest contains fields for modifying an existing order
type ReplaceOrderRequest struct {
	Qty           string      `json:"qty,omitempty"`
	TimeInForce   TimeInForce `json:"time_in_force,omitempty"`
	LimitPrice    string      `json:"limit_price,omitempty"`
	StopPrice     string      `json:"stop_price,omitempty"`
	ClientOrderID string      `json:"client_order_id,omitempty"`
}

// ReplaceOrder modifies an existing order
func (c *Client) ReplaceOrder(ctx context.Context, orderID string, req *ReplaceOrderRequest) (*Order, error) {
	url := fmt.Sprintf("%s/v2/orders/%s", c.baseURL, orderID)
	resp, err := c.doRequest(ctx, "PATCH", url, req)
	if err != nil {
		return nil, err
	}

	var order Order
	if err := decodeResponse(resp, &order); err != nil {
		return nil, err
	}

	return &order, nil
}
