-- Migration: Add OAuth providers support
-- Enables Google, Apple, and Email magic link authentication

-- Auth provider enum
CREATE TYPE auth_provider AS ENUM ('phone', 'google', 'apple', 'email');

-- OAuth identities table (supports multiple providers per user)
CREATE TABLE oauth_identities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider auth_provider NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    provider_name VARCHAR(255),
    raw_user_info JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_oauth_identities_user ON oauth_identities(user_id);
CREATE INDEX idx_oauth_identities_provider_email ON oauth_identities(provider, provider_email);

-- Pending phone verifications (for social users adding phone later)
CREATE TABLE pending_phone_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone VARCHAR(20) NOT NULL,
    otp_hash VARCHAR(255) NOT NULL,
    attempts INT DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_pending_phone_user ON pending_phone_verifications(user_id);
CREATE INDEX idx_pending_phone_phone ON pending_phone_verifications(phone);

-- Magic link tokens table
CREATE TABLE magic_link_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_magic_link_email ON magic_link_tokens(email);
CREATE INDEX idx_magic_link_expires ON magic_link_tokens(expires_at) WHERE used_at IS NULL;
CREATE INDEX idx_magic_link_token ON magic_link_tokens(token_hash) WHERE used_at IS NULL;

-- Add new columns to users table
ALTER TABLE users
    ADD COLUMN username VARCHAR(50),
    ADD COLUMN display_name VARCHAR(100),
    ADD COLUMN avatar_url TEXT,
    ADD COLUMN primary_auth_provider auth_provider DEFAULT 'phone',
    ADD COLUMN phone_verified BOOLEAN DEFAULT false,
    ADD COLUMN email_verified BOOLEAN DEFAULT false;

-- Update existing users: generate username from phone, mark as phone verified
UPDATE users SET
    phone_verified = true,
    primary_auth_provider = 'phone',
    username = 'user_' || SUBSTRING(REPLACE(phone, '+', '') FROM '.{6}$')
WHERE phone IS NOT NULL;

-- Add unique constraint on username (after populating existing)
CREATE UNIQUE INDEX idx_users_username ON users(username) WHERE username IS NOT NULL;

-- Make phone nullable for social-first registrations
-- First drop the existing unique constraint/index
DROP INDEX IF EXISTS idx_users_phone;
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;

-- Add partial unique index: phone must be unique when not null
CREATE UNIQUE INDEX idx_users_phone_unique ON users(phone) WHERE phone IS NOT NULL;
