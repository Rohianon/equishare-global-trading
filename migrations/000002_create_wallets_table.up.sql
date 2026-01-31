CREATE TYPE currency AS ENUM ('KES', 'USD');

CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency currency NOT NULL,
    balance DECIMAL(20, 4) DEFAULT 0 CHECK (balance >= 0),
    locked_balance DECIMAL(20, 4) DEFAULT 0 CHECK (locked_balance >= 0),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, currency)
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);
