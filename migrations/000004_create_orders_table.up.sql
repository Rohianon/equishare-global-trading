CREATE TYPE order_side AS ENUM ('buy', 'sell');
CREATE TYPE order_type AS ENUM ('market', 'limit', 'stop', 'stop_limit');
CREATE TYPE order_status AS ENUM ('pending', 'submitted', 'partial', 'filled', 'cancelled', 'rejected', 'expired');
CREATE TYPE time_in_force AS ENUM ('day', 'gtc', 'ioc', 'fok');

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alpaca_order_id VARCHAR(100),
    symbol VARCHAR(20) NOT NULL,
    side order_side NOT NULL,
    type order_type NOT NULL,
    status order_status DEFAULT 'pending',
    time_in_force time_in_force DEFAULT 'day',
    quantity DECIMAL(20, 8) NOT NULL CHECK (quantity > 0),
    filled_quantity DECIMAL(20, 8) DEFAULT 0,
    limit_price DECIMAL(20, 4),
    stop_price DECIMAL(20, 4),
    filled_avg_price DECIMAL(20, 4),
    total_cost DECIMAL(20, 4),
    commission DECIMAL(20, 4) DEFAULT 0,
    submitted_at TIMESTAMPTZ,
    filled_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    reject_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_alpaca_order_id ON orders(alpaca_order_id) WHERE alpaca_order_id IS NOT NULL;
CREATE INDEX idx_orders_symbol ON orders(symbol);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
