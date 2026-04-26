-- Drop legacy authentication tables and pg_cron job. Auth is now handled
-- entirely by the BetterAuth-based auth-server, which manages its own
-- user/session/account tables in this same database.

-- Unschedule the refresh-token cleanup job if pg_cron is loaded; safe no-op
-- otherwise.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.unschedule('cleanup-expired-refresh-tokens');
    END IF;
EXCEPTION WHEN OTHERS THEN
    -- Job may not exist (e.g. test DB), ignore.
    NULL;
END $$;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

-- pg_cron is no longer needed by this application.
DROP EXTENSION IF EXISTS pg_cron;
