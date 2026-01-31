CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE kyc_status AS ENUM ('pending', 'submitted', 'verified', 'rejected');
CREATE TYPE kyc_tier AS ENUM ('tier1', 'tier2', 'tier3');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone VARCHAR(20) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255),
    pin_hash VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    kyc_status kyc_status DEFAULT 'pending',
    kyc_tier kyc_tier DEFAULT 'tier1',
    kyc_submitted_at TIMESTAMPTZ,
    kyc_verified_at TIMESTAMPTZ,
    alpaca_account_id VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_kyc_status ON users(kyc_status);
CREATE INDEX idx_users_alpaca_account ON users(alpaca_account_id) WHERE alpaca_account_id IS NOT NULL;
