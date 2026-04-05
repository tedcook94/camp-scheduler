DROP TRIGGER IF EXISTS enrollment_session_age_group_consistency ON camper_session_enrollments;
DROP FUNCTION IF EXISTS validate_enrollment_session_age_group();
DROP TABLE IF EXISTS camper_session_enrollments;
