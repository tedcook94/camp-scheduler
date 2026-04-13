BEGIN;

DO $$
BEGIN
    IF current_database() = 'camp_scheduler' THEN
        PERFORM cron.unschedule('cleanup-expired-refresh-tokens');
        DROP EXTENSION IF EXISTS pg_cron;
    END IF;
END
$$;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

COMMIT;
