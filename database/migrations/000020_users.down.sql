BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_cron') THEN
        PERFORM cron.unschedule('cleanup-expired-refresh-tokens');
        DROP EXTENSION IF EXISTS pg_cron;
    END IF;
EXCEPTION WHEN OTHERS THEN
    NULL;
END
$$;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

COMMIT;
