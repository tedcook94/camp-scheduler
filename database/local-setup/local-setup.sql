CREATE USER camp_scheduler PASSWORD 'p@ss123';
CREATE DATABASE camp_scheduler WITH OWNER camp_scheduler;

\connect camp_scheduler

ALTER ROLE camp_scheduler SUPERUSER;
