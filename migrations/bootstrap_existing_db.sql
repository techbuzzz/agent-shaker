-- Bootstrap script for existing databases
-- This marks existing migrations as applied without re-running them
-- Run this ONCE if you have an existing database before the new migration system

-- Create schema_migrations table if it doesn't exist
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64)
);

-- Mark existing migrations as applied (only if tables exist)
DO $$
BEGIN
    -- Check if projects table exists (from 001_init.sql)
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'projects') THEN
        INSERT INTO schema_migrations (version, applied_at) 
        VALUES ('001_init.sql', CURRENT_TIMESTAMP)
        ON CONFLICT (version) DO NOTHING;
        RAISE NOTICE 'Marked 001_init.sql as applied';
    END IF;

    -- Check if sample data was loaded (check for any project)
    IF EXISTS (SELECT 1 FROM projects LIMIT 1) THEN
        INSERT INTO schema_migrations (version, applied_at)
        VALUES ('002_sample_data.sql', CURRENT_TIMESTAMP)
        ON CONFLICT (version) DO NOTHING;
        RAISE NOTICE 'Marked 002_sample_data.sql as applied';
    END IF;

    -- Check if daily_standups table exists (from 003_daily_standups.sql)
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'daily_standups') THEN
        INSERT INTO schema_migrations (version, applied_at)
        VALUES ('003_daily_standups.sql', CURRENT_TIMESTAMP)
        ON CONFLICT (version) DO NOTHING;
        RAISE NOTICE 'Marked 003_daily_standups.sql as applied';
    END IF;
END $$;

-- Display current migration status
SELECT version, applied_at 
FROM schema_migrations 
ORDER BY version;
