-- Rollback: Remove OAuth providers support

-- Drop tables
DROP TABLE IF EXISTS magic_link_tokens;
DROP TABLE IF EXISTS pending_phone_verifications;
DROP TABLE IF EXISTS oauth_identities;

-- Remove columns from users
ALTER TABLE users
    DROP COLUMN IF EXISTS username,
    DROP COLUMN IF EXISTS display_name,
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS primary_auth_provider,
    DROP COLUMN IF EXISTS phone_verified,
    DROP COLUMN IF EXISTS email_verified;

-- Restore phone as NOT NULL (requires all users to have phone)
-- WARNING: This will fail if any users have NULL phone
DROP INDEX IF EXISTS idx_users_phone_unique;
DROP INDEX IF EXISTS idx_users_username;

-- Restore original phone constraint
ALTER TABLE users ALTER COLUMN phone SET NOT NULL;
CREATE INDEX idx_users_phone ON users(phone);

-- Drop enum type
DROP TYPE IF EXISTS auth_provider;
