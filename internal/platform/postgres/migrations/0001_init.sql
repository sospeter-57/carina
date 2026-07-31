-- 0001_init.sql
-- Migration: Initial schema for Carina
--
-- Tables:
--   users        - authentication and identity
--   files        - file metadata
--   shares       - share links for unauthenticated access
--
-- Notes:
--   - All tables use UUID primary keys
--   - Foreign keys are CASCADE where appropriate
--   - Timestamps are UTC with timezone

-- ===================================================================
-- Users
-- ===================================================================

CREATE TABLE IF NOT EXISTS users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,          -- bcrypt hash
    full_name    TEXT,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL
);

-- Lookup by email is the most common auth query
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- ===================================================================
-- Refresh Tokens
-- ===================================================================

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   TEXT UNIQUE NOT NULL,    -- SHA-256 of the refresh token
    expires_at   TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at   TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
    user_agent   TEXT,
    ip_address   INET
);

-- Needed for lookups during refresh and revocation
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);

-- ===================================================================
-- Files
-- ===================================================================

CREATE TABLE IF NOT EXISTS files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL,
    content_type TEXT NOT NULL,
    storage_key  TEXT NOT NULL UNIQUE,    -- S3/MinIO object key
    hash_sha256  TEXT,                    -- for deduplication/integrity
    status       TEXT NOT NULL DEFAULT 'pending',  -- pending | available | rejected
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_files_owner_id ON files (owner_id);
CREATE INDEX IF NOT EXISTS idx_files_status ON files (status);

-- ===================================================================
-- Shares
-- ===================================================================

CREATE TABLE IF NOT EXISTS shares (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id      UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    token        TEXT UNIQUE NOT NULL,     -- URL-safe random token
    created_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at   TIMESTAMP WITH TIME ZONE,
    max_downloads INTEGER,
    download_count INTEGER DEFAULT 0 NOT NULL,
    revoked_at   TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_shares_token ON shares (token);
CREATE INDEX IF NOT EXISTS idx_shares_file_id ON shares (file_id);

-- ===================================================================
-- Update timestamp triggers
-- ===================================================================

-- Function to automatically update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_files_updated_at
    BEFORE UPDATE ON files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();