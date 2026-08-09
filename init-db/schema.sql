-- Metrix Multi-Tenant Analytics Platform Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Core Users table (one row per registered creator)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Password reset tokens (single-use, time-limited)
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Platform accounts — one per connected social channel
CREATE TABLE IF NOT EXISTS platform_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'connected',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. OAuth tokens stored per platform account (encrypted at rest in production)
CREATE TABLE IF NOT EXISTS platform_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    platform_account_id UUID NOT NULL REFERENCES platform_accounts(id) ON DELETE CASCADE,
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Aggregated metrics summaries (written by pipeline)
CREATE TABLE IF NOT EXISTS metrics_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_reach INT NOT NULL DEFAULT 0,
    avg_engagement NUMERIC(5, 2) NOT NULL DEFAULT 0.0,
    follower_growth INT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Daily timeseries metrics per platform
CREATE TABLE IF NOT EXISTS metrics_timeseries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    metric_date DATE NOT NULL,
    value INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 7. Top content items ingested from each platform
CREATE TABLE IF NOT EXISTS content_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    engagement NUMERIC(6,2) NOT NULL DEFAULT 0.0,
    reach INT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 8. Audience insights — demographic + geography breakdown per user
CREATE TABLE IF NOT EXISTS audience_insights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(50) NOT NULL,
    label VARCHAR(100) NOT NULL,
    value NUMERIC(5,2) NOT NULL DEFAULT 0.0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, category, label, recorded_at)
);
