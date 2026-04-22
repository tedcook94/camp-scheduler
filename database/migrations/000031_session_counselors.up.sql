BEGIN;

CREATE TABLE IF NOT EXISTS session_counselors (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_id uuid not null,
    counselor_id uuid not null,
    unique (session_id, counselor_id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS session_counselors_session_id_idx ON session_counselors (session_id);
CREATE INDEX IF NOT EXISTS session_counselors_counselor_id_idx ON session_counselors (counselor_id);

-- Backfill: every existing session gets all currently enabled counselors of its camp.
INSERT INTO session_counselors (camp_id, session_id, counselor_id)
SELECT s.camp_id, s.id, c.id
FROM sessions s
JOIN counselors c ON c.camp_id = s.camp_id
WHERE c.counselor_enabled = true
ON CONFLICT (session_id, counselor_id) DO NOTHING;

COMMIT;
