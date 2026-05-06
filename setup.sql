-- ============================================================
-- SovraEquitara — Native PostgreSQL Schema
-- Migrasi dari Supabase ke PostgreSQL (Docker)
-- ============================================================

-- ============================================================
-- PROFILES TABLE (Self-managed auth, no Supabase dependency)
-- ============================================================
CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL DEFAULT '',
    phone TEXT DEFAULT '',
    points INTEGER NOT NULL DEFAULT 0,
    role TEXT NOT NULL DEFAULT 'USER' CHECK (role IN ('USER', 'admin')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for fast email lookups during login/register
CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles(email);
-- Index for leaderboard queries
CREATE INDEX IF NOT EXISTS idx_profiles_points ON profiles(points DESC);

-- ============================================================
-- REPORTS TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    phone_number TEXT,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    location_detail TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'VALID', 'RESOLVED')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for filtering by profile
CREATE INDEX IF NOT EXISTS idx_reports_profile_id ON reports(profile_id);
-- Index for filtering by status
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);

-- ============================================================
-- OTP TABLE (Registration handoff)
-- ============================================================
CREATE TABLE IF NOT EXISTS otps (
    email TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================
-- FORGOT PASSWORD OTP TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS forgot_password_otps (
    email TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================
-- AUTO-UPDATE updated_at TRIGGER
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to profiles
DROP TRIGGER IF EXISTS trigger_profiles_updated_at ON profiles;
CREATE TRIGGER trigger_profiles_updated_at
    BEFORE UPDATE ON profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to reports
DROP TRIGGER IF EXISTS trigger_reports_updated_at ON reports;
CREATE TRIGGER trigger_reports_updated_at
    BEFORE UPDATE ON reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- AUTO-CLEANUP EXPIRED OTPs (older than 10 minutes)
-- Can be called periodically or before each verification
-- ============================================================
CREATE OR REPLACE FUNCTION cleanup_expired_otps()
RETURNS void AS $$
BEGIN
    DELETE FROM otps WHERE created_at < NOW() - INTERVAL '10 minutes';
    DELETE FROM forgot_password_otps WHERE created_at < NOW() - INTERVAL '10 minutes';
END;
$$ LANGUAGE plpgsql;
