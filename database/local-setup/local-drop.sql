DROP DATABASE IF EXISTS camp_scheduler;

DO
$$BEGIN
IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'camp_scheduler') THEN
    EXECUTE 'DROP OWNED BY camp_scheduler';
END IF;
END$$;

DROP USER IF EXISTS camp_scheduler;
