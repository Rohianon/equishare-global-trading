CREATE TYPE mpesa_transaction_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');

CREATE TABLE mpesa_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    transaction_id UUID REFERENCES transactions(id),
    checkout_request_id VARCHAR(100) UNIQUE,
    merchant_request_id VARCHAR(100),
    amount DECIMAL(20, 2) NOT NULL CHECK (amount > 0),
    phone VARCHAR(20) NOT NULL,
    status mpesa_transaction_status DEFAULT 'pending',
    mpesa_receipt VARCHAR(50),
    result_code INT,
    result_desc TEXT,
    callback_payload JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_mpesa_transactions_user_id ON mpesa_transactions(user_id);
CREATE INDEX idx_mpesa_transactions_checkout_request_id ON mpesa_transactions(checkout_request_id);
CREATE INDEX idx_mpesa_transactions_status ON mpesa_transactions(status);
CREATE INDEX idx_mpesa_transactions_created_at ON mpesa_transactions(created_at DESC);
