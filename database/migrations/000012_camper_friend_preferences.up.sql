BEGIN;

CREATE TABLE IF NOT EXISTS camper_friend_preferences (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    camper_id uuid not null,
    session_id uuid not null,
    preferred_camper_id uuid not null,
    rank integer not null,
    check (camper_id != preferred_camper_id),
    check (rank > 0),
    unique (camp_id, id),
    unique (camper_id, session_id, preferred_camper_id),
    unique (camper_id, session_id, rank),
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id),
    FOREIGN KEY (camp_id, preferred_camper_id) REFERENCES campers(camp_id, id)
);

COMMIT;
