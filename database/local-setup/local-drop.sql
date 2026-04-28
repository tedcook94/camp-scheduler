DROP DATABASE IF EXISTS camp_scheduler;
DROP DATABASE IF EXISTS camp_scheduler_test;

DO
$$BEGIN
IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_user') THEN
    EXECUTE 'DROP OWNED BY app_user';
END IF;
IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'auth_user') THEN
    EXECUTE 'DROP OWNED BY auth_user';
END IF;
IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'camp_scheduler') THEN
    EXECUTE 'DROP OWNED BY camp_scheduler';
END IF;
END$$;

DROP USER IF EXISTS app_user;
DROP USER IF EXISTS auth_user;
DROP USER IF EXISTS camp_scheduler;
