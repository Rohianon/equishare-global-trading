CREATE TYPE transaction_type AS ENUM ('deposit', 'withdrawal', 'buy', 'sell', 'fee', 'dividend', 'transfer');
CREATE TYPE transaction_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');
CREATE TYPE payment_provider AS ENUM ('mpesa', 'bank', 'alpaca', 'internal');

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    type transaction_type NOT NULL,
    status transaction_status DEFAULT 'pending',
    amount DECIMAL(20, 4) NOT NULL,
    fee DECIMAL(20, 4) DEFAULT 0,
    currency currency NOT NULL,
    provider payment_provider,
    provider_ref VARCHAR(255),
    description TEXT,
    metadata JSONB DEFAULT '{}',
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_provider_ref ON transactions(provider_ref) WHERE provider_ref IS NOT NULL;
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
