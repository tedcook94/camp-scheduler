BEGIN;

CREATE SCHEMA IF NOT EXISTS app;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public;
SET search_path = app, public;

CREATE FUNCTION validate_enrollment_session_age_group() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM session_age_groups
        WHERE camp_id = NEW.camp_id
          AND session_id = NEW.session_id
          AND id = NEW.session_age_group_id
    ) THEN
        RAISE EXCEPTION
            'session_age_group_id % does not belong to session_id % for camp_id %',
            NEW.session_age_group_id,
            NEW.session_id,
            NEW.camp_id;
    END IF;

    RETURN NEW;
END;
$$;

CREATE TABLE camps (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_name text NOT NULL,
    camp_location text,
    camp_enabled boolean DEFAULT true NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_name)
);

CREATE TABLE activities (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    activity_name text NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, activity_name),
    UNIQUE (camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE age_groups (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    age_group_name text NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, age_group_name),
    UNIQUE (camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE campers (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    gender text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    CHECK ((gender = ANY (ARRAY['male'::text, 'female'::text]))),
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE certifications (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    certification_name text NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, certification_name),
    UNIQUE (camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE counselors (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    junior_counselor boolean DEFAULT false NOT NULL,
    gender text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    CHECK ((gender = ANY (ARRAY['male'::text, 'female'::text]))),
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE seasons (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    season_name text NOT NULL,
    start_date date NOT NULL,
    end_date date NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (camp_id, season_name),
    CHECK ((start_date <= end_date)),
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE time_slots (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    time_slot_name text NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (camp_id, time_slot_name),
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE cabins (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    default_age_group_id uuid NOT NULL,
    cabin_name text NOT NULL,
    default_group_size integer NOT NULL,
    default_required_counselors integer NOT NULL,
    gender text NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, cabin_name),
    UNIQUE (camp_id, id),
    CHECK ((gender = ANY (ARRAY['male'::text, 'female'::text]))),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, default_age_group_id) REFERENCES age_groups(camp_id, id)
);

CREATE TABLE activity_certifications (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    activity_id uuid NOT NULL,
    certification_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (activity_id, certification_id),
    FOREIGN KEY (camp_id, activity_id) REFERENCES activities(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, certification_id) REFERENCES certifications(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE counselor_certifications (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    certification_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (counselor_id, certification_id),
    FOREIGN KEY (camp_id, certification_id) REFERENCES certifications(camp_id, id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE sessions (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    season_id uuid NOT NULL,
    session_name text NOT NULL,
    previous_session uuid,
    archived boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (season_id, session_name),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, previous_session) REFERENCES sessions(camp_id, id),
    FOREIGN KEY (camp_id, season_id) REFERENCES seasons(camp_id, id)
);

CREATE TABLE assignment_runs (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_id uuid NOT NULL,
    run_type text NOT NULL,
    status text DEFAULT 'completed'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    is_stale boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (session_id, run_type),
    CHECK ((run_type = ANY (ARRAY['cabin'::text, 'activity_schedule'::text]))),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id)
);

CREATE TABLE camper_friend_preferences (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    camper_id uuid NOT NULL,
    session_id uuid NOT NULL,
    preferred_camper_id uuid NOT NULL,
    rank integer NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (camper_id, session_id, preferred_camper_id),
    UNIQUE (camper_id, session_id, rank),
    CHECK ((camper_id <> preferred_camper_id)),
    CHECK ((rank > 0)),
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, preferred_camper_id) REFERENCES campers(camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id)
);

CREATE TABLE counselor_activity_preferences (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    session_id uuid NOT NULL,
    activity_id uuid NOT NULL,
    rank integer NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (counselor_id, session_id, activity_id),
    CHECK ((rank > 0)),
    FOREIGN KEY (camp_id, activity_id) REFERENCES activities(camp_id, id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id)
);

CREATE TABLE counselor_age_group_preferences (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    session_id uuid NOT NULL,
    age_group_id uuid NOT NULL,
    rank integer NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (counselor_id, session_id, age_group_id),
    UNIQUE (counselor_id, session_id, rank),
    CHECK ((rank > 0)),
    FOREIGN KEY (camp_id, age_group_id) REFERENCES age_groups(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id)
);

CREATE TABLE counselor_cocounselor_preferences (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    session_id uuid NOT NULL,
    preferred_counselor_id uuid NOT NULL,
    rank integer NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (counselor_id, session_id, preferred_counselor_id),
    UNIQUE (counselor_id, session_id, rank),
    CHECK ((counselor_id <> preferred_counselor_id)),
    CHECK ((rank > 0)),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id, preferred_counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id)
);

CREATE TABLE counselor_session_history (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    session_id uuid NOT NULL,
    age_group_id uuid NOT NULL,
    cabin_id uuid,
    PRIMARY KEY (id),
    UNIQUE (counselor_id, session_id),
    FOREIGN KEY (camp_id, age_group_id) REFERENCES age_groups(camp_id, id),
    FOREIGN KEY (camp_id, cabin_id) REFERENCES cabins(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id)
);

CREATE TABLE session_age_groups (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_id uuid NOT NULL,
    age_group_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (session_id, age_group_id),
    FOREIGN KEY (camp_id, age_group_id) REFERENCES age_groups(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id)
);

CREATE TABLE session_counselors (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (session_id, counselor_id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE session_time_slots (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_id uuid NOT NULL,
    time_slot_id uuid NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (session_id, time_slot_id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id),
    FOREIGN KEY (camp_id, time_slot_id) REFERENCES time_slots(camp_id, id)
);

CREATE TABLE assignment_run_selected_solutions (
    camp_id uuid NOT NULL,
    run_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    solution_type text NOT NULL,
    session_id uuid NOT NULL,
    UNIQUE (camp_id, session_id, solution_type),
    UNIQUE (camp_id, run_id),
    CHECK ((solution_type = ANY (ARRAY['cabin'::text, 'activity_schedule'::text]))),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id),
    FOREIGN KEY (camp_id, run_id) REFERENCES assignment_runs(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE camper_cabin_solutions (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    assignment_run_id uuid NOT NULL,
    solution_index integer NOT NULL,
    score double precision NOT NULL,
    score_breakdown jsonb DEFAULT '[]'::jsonb NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (assignment_run_id, solution_index),
    UNIQUE (camp_id, id),
    FOREIGN KEY (camp_id, assignment_run_id) REFERENCES assignment_runs(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE counselor_activity_solutions (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    assignment_run_id uuid NOT NULL,
    solution_index integer NOT NULL,
    score double precision NOT NULL,
    score_breakdown jsonb DEFAULT '[]'::jsonb NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (assignment_run_id, solution_index),
    UNIQUE (camp_id, id),
    FOREIGN KEY (camp_id, assignment_run_id) REFERENCES assignment_runs(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE counselor_cabin_solutions (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    assignment_run_id uuid NOT NULL,
    solution_index integer NOT NULL,
    score double precision NOT NULL,
    score_breakdown jsonb DEFAULT '[]'::jsonb NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (assignment_run_id, solution_index),
    UNIQUE (camp_id, id),
    FOREIGN KEY (camp_id, assignment_run_id) REFERENCES assignment_runs(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id)
);

CREATE TABLE camper_session_enrollments (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    camper_id uuid NOT NULL,
    session_id uuid NOT NULL,
    session_age_group_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (camper_id, session_age_group_id),
    UNIQUE (camper_id, session_id),
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_age_group_id) REFERENCES session_age_groups(camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id)
);

CREATE TABLE session_cabins (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_age_group_id uuid NOT NULL,
    cabin_id uuid NOT NULL,
    group_size integer NOT NULL,
    required_counselors integer NOT NULL,
    gender text NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (session_age_group_id, cabin_id),
    CHECK ((gender = ANY (ARRAY['male'::text, 'female'::text]))),
    FOREIGN KEY (camp_id, cabin_id) REFERENCES cabins(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_age_group_id) REFERENCES session_age_groups(camp_id, id)
);

CREATE TABLE session_activities (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_time_slot_id uuid NOT NULL,
    activity_id uuid NOT NULL,
    capacity integer DEFAULT 1 NOT NULL,
    required_counselors integer DEFAULT 1 NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (session_time_slot_id, activity_id),
    CHECK ((capacity > 0)),
    CHECK ((required_counselors <= capacity)),
    CHECK ((required_counselors > 0)),
    FOREIGN KEY (camp_id, activity_id) REFERENCES activities(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_time_slot_id) REFERENCES session_time_slots(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE camper_cabin_explanations (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    camper_id uuid NOT NULL,
    explanation_type text NOT NULL,
    constraint_name text,
    message text NOT NULL,
    rank integer,
    PRIMARY KEY (id),
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES camper_cabin_solutions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_activity_unassigned (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (solution_id, counselor_id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_activity_solutions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_cabin_explanations (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    explanation_type text NOT NULL,
    constraint_name text,
    message text NOT NULL,
    rank integer,
    PRIMARY KEY (id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_cabin_solutions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_cabin_unassigned (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (solution_id, counselor_id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_cabin_solutions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE camper_cabin_assignments (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    camper_id uuid NOT NULL,
    session_cabin_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (solution_id, camper_id),
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES camper_cabin_solutions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, session_cabin_id) REFERENCES session_cabins(camp_id, id)
);

CREATE TABLE camper_cabin_overrides (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_id uuid NOT NULL,
    camper_id uuid NOT NULL,
    session_cabin_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (session_id, camper_id),
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_cabin_id) REFERENCES session_cabins(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_cabin_assignments (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    session_cabin_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (solution_id, counselor_id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id, session_cabin_id) REFERENCES session_cabins(camp_id, id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_cabin_solutions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_cabin_overrides (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    session_cabin_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (session_id, counselor_id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_cabin_id) REFERENCES session_cabins(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_activity_assignments (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    session_activity_id uuid NOT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_activity_id) REFERENCES session_activities(camp_id, id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_activity_solutions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_activity_explanations (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    explanation_type text NOT NULL,
    constraint_name text,
    message text NOT NULL,
    rank integer,
    session_activity_id uuid,
    PRIMARY KEY (id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_activity_id) REFERENCES session_activities(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_activity_solutions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_activity_overrides (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    session_id uuid NOT NULL,
    counselor_id uuid NOT NULL,
    session_activity_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (camp_id, id),
    UNIQUE (session_id, counselor_id, session_activity_id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, session_activity_id) REFERENCES session_activities(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE counselor_activity_unassigned_slots (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    camp_id uuid NOT NULL,
    unassigned_id uuid NOT NULL,
    session_time_slot_id uuid NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (unassigned_id, session_time_slot_id),
    FOREIGN KEY (camp_id, session_time_slot_id) REFERENCES session_time_slots(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id) REFERENCES camps(id),
    FOREIGN KEY (camp_id, unassigned_id) REFERENCES counselor_activity_unassigned(camp_id, id) ON DELETE CASCADE
);


CREATE INDEX camper_cabin_overrides_session_id_idx ON camper_cabin_overrides USING btree (session_id);
CREATE INDEX counselor_activity_overrides_session_id_idx ON counselor_activity_overrides USING btree (session_id);
CREATE INDEX counselor_cabin_overrides_session_id_idx ON counselor_cabin_overrides USING btree (session_id);
CREATE INDEX session_counselors_counselor_id_idx ON session_counselors USING btree (counselor_id);
CREATE INDEX session_counselors_session_id_idx ON session_counselors USING btree (session_id);

CREATE TRIGGER enrollment_session_age_group_consistency BEFORE INSERT OR UPDATE OF camp_id, session_id, session_age_group_id ON camper_session_enrollments FOR EACH ROW EXECUTE FUNCTION validate_enrollment_session_age_group();

COMMIT;
