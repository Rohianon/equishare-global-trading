package alpaca

import "context"

// TradingClient defines the interface for Alpaca trading operations
type TradingClient interface {
	// Account
	GetAccount(ctx context.Context) (*Account, error)

	// Orders
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error)
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	GetOrderByClientID(ctx context.Context, clientOrderID string) (*Order, error)
	ListOrders(ctx context.Context, params *ListOrdersParams) ([]Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	CancelAllOrders(ctx context.Context) error
	ReplaceOrder(ctx context.Context, orderID string, req *ReplaceOrderRequest) (*Order, error)

	// Positions
	ListPositions(ctx context.Context) ([]Position, error)
	GetPosition(ctx context.Context, symbol string) (*Position, error)
	ClosePosition(ctx context.Context, symbol string, qty string) (*Order, error)
	CloseAllPositions(ctx context.Context, cancelOrders bool) ([]Order, error)

	// Assets
	GetAsset(ctx context.Context, symbol string) (*Asset, error)
	ListAssets(ctx context.Context, params *ListAssetsParams) ([]Asset, error)

	// Market Data
	GetQuote(ctx context.Context, symbol string) (*Quote, error)
}

// Ensure Client and MockClient implement TradingClient
var _ TradingClient = (*Client)(nil)
var _ TradingClient = (*MockClient)(nil)
