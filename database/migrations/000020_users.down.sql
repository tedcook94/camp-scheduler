BEGIN;

SELECT cron.unschedule('cleanup-expired-refresh-tokens');

DROP EXTENSION IF EXISTS pg_cron;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

COMMIT;
