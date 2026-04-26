package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"camp-scheduler/internal/config"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	demoCampIDStr = "00000000-0000-0000-0000-000000000000"
	demoCampName  = "Demo Camp"
)

func pgUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		panic(fmt.Sprintf("invalid UUID %q: %v", s, err))
	}
	return u
}

func pgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func pgDate(year, month, day int) pgtype.Date {
	return pgtype.Date{
		Time:  time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	campID := pgUUID(demoCampIDStr)

	fmt.Println("Deleting existing demo camp data...")
	if err := deleteExisting(ctx, tx, campID); err != nil {
		fmt.Fprintf(os.Stderr, "error deleting existing data: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Creating demo camp data...")
	queries := db.New(tx)
	if err := seedData(ctx, tx, queries, campID); err != nil {
		fmt.Fprintf(os.Stderr, "error seeding data: %v\n", err)
		os.Exit(1)
	}

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error committing transaction: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Demo camp seeded successfully!")
	fmt.Println("To create the demo admin user, run `pnpm seed:super-admin` in the auth/ directory.")
}

func deleteExisting(ctx context.Context, tx pgx.Tx, campID pgtype.UUID) error {
	tables := []string{
		"assignment_run_selected_solutions",
		"activity_explanations",
		"activity_assignments",
		"activity_solutions",
		"camper_cabin_explanations",
		"camper_cabin_assignments",
		"camper_cabin_solutions",
		"counselor_cabin_explanations",
		"counselor_cabin_assignments",
		"counselor_cabin_solutions",
		"assignment_runs",
		"camper_friend_preferences",
		"camper_session_enrollments",
		"counselor_age_group_preferences",
		"counselor_cocounselor_preferences",
		"counselor_activity_preferences",
		"counselor_session_history",
		"counselor_certifications",
		"session_activities",
		"session_age_group_cabins",
		"session_time_slots",
		"session_age_groups",
	}

	for _, t := range tables {
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE camp_id = $1", t), campID); err != nil {
			return fmt.Errorf("error deleting from %s: %w", t, err)
		}
	}

	if _, err := tx.Exec(ctx, "UPDATE sessions SET previous_session = NULL WHERE camp_id = $1", campID); err != nil {
		return fmt.Errorf("error clearing session references: %w", err)
	}

	tables2 := []string{
		"sessions",
		"campers",
		"counselors",
		"activity_certifications",
		"activities",
		"certifications",
		"time_slots",
		"cabins",
		"age_groups",
		"seasons",
	}

	for _, t := range tables2 {
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE camp_id = $1", t), campID); err != nil {
			return fmt.Errorf("error deleting from %s: %w", t, err)
		}
	}

	if _, err := tx.Exec(ctx, "DELETE FROM camps WHERE id = $1", campID); err != nil {
		return fmt.Errorf("error deleting camp: %w", err)
	}

	return nil
}

type sessionConfig struct {
	s1AGs map[string]db.SessionAgeGroup
	s2AGs map[string]db.SessionAgeGroup
	s1TSs map[string]db.SessionTimeSlot
	s2TSs map[string]db.SessionTimeSlot
}

func seedData(ctx context.Context, tx pgx.Tx, q *db.Queries, campID pgtype.UUID) error {
	_, err := tx.Exec(ctx,
		"INSERT INTO camps (id, camp_name, camp_location, camp_enabled) VALUES ($1, $2, $3, true)",
		campID, demoCampName, pgText("Camp Demo, NH"))
	if err != nil {
		return fmt.Errorf("error creating camp: %w", err)
	}

	season, err := q.CreateSeason(ctx, db.CreateSeasonParams{
		CampID:     campID,
		SeasonName: "Summer 2026",
		StartDate:  pgDate(2026, 6, 15),
		EndDate:    pgDate(2026, 8, 15),
	})
	if err != nil {
		return fmt.Errorf("error creating season: %w", err)
	}

	s1, err := q.CreateSession(ctx, db.CreateSessionParams{
		CampID:          campID,
		SeasonID:        season.ID,
		SessionName:     "Session 1",
		PreviousSession: pgtype.UUID{Valid: false},
	})
	if err != nil {
		return fmt.Errorf("error creating session 1: %w", err)
	}

	s2, err := q.CreateSession(ctx, db.CreateSessionParams{
		CampID:          campID,
		SeasonID:        season.ID,
		SessionName:     "Session 2",
		PreviousSession: s1.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating session 2: %w", err)
	}

	ageGroups, err := createAgeGroups(ctx, q, campID)
	if err != nil {
		return err
	}

	cabins, err := createCabins(ctx, q, campID, ageGroups)
	if err != nil {
		return err
	}

	activities, err := createActivities(ctx, q, campID)
	if err != nil {
		return err
	}

	timeSlots, err := createTimeSlots(ctx, q, campID)
	if err != nil {
		return err
	}

	certs, err := createCertifications(ctx, q, campID)
	if err != nil {
		return err
	}

	if err := linkActivityCertifications(ctx, q, campID, activities, certs); err != nil {
		return err
	}

	cfg, err := configureSessions(ctx, q, campID, s1, s2, ageGroups, cabins, timeSlots, activities)
	if err != nil {
		return err
	}

	counselors, err := createCounselors(ctx, q, campID)
	if err != nil {
		return err
	}

	if err := rosterCounselors(ctx, q, campID, counselors, s1, s2); err != nil {
		return err
	}

	if err := linkCounselorCertifications(ctx, q, campID, counselors, certs); err != nil {
		return err
	}

	if err := setCounselorPreferences(ctx, q, campID, counselors, ageGroups, s1, s2, activities); err != nil {
		return err
	}

	if err := setCounselorHistory(ctx, q, campID, counselors, s1, ageGroups, cabins); err != nil {
		return err
	}

	campers, err := createCampers(ctx, q, campID, ageGroups)
	if err != nil {
		return err
	}

	if err := enrollCampers(ctx, q, campID, campers, s1, s2, cfg); err != nil {
		return err
	}

	if err := setFriendPreferences(ctx, q, campID, campers, s1, s2); err != nil {
		return err
	}

	// User accounts are managed by the auth-server (BetterAuth). Run
	// `pnpm seed:super-admin` in `auth/` to create the demo super-admin and
	// assign membership to the demo camp.
	return nil
}

func createAgeGroups(ctx context.Context, q *db.Queries, campID pgtype.UUID) (map[string]db.AgeGroup, error) {
	names := []string{"Bears", "Eagles", "Wolves"}
	m := make(map[string]db.AgeGroup)
	for _, name := range names {
		ag, err := q.CreateAgeGroup(ctx, db.CreateAgeGroupParams{
			CampID:       campID,
			AgeGroupName: name,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating age group %s: %w", name, err)
		}
		m[name] = ag
	}
	return m, nil
}

func createCabins(ctx context.Context, q *db.Queries, campID pgtype.UUID, ageGroups map[string]db.AgeGroup) (map[string]db.CreateCabinRow, error) {
	defs := []struct {
		name      string
		ageGroup  string
		size      int32
		counselors int32
		gender    string
	}{
		{"Pine Lodge", "Bears", 10, 2, "female"},
		{"Cedar Lodge", "Bears", 10, 2, "male"},
		{"Maple Lodge", "Eagles", 12, 2, "female"},
		{"Birch Lodge", "Eagles", 12, 2, "male"},
		{"Oak Lodge", "Wolves", 10, 2, "female"},
		{"Elm Lodge", "Wolves", 10, 2, "male"},
	}
	m := make(map[string]db.CreateCabinRow)
	for _, d := range defs {
		c, err := q.CreateCabin(ctx, db.CreateCabinParams{
			CampID:                    campID,
			DefaultAgeGroupID:         ageGroups[d.ageGroup].ID,
			CabinName:                 d.name,
			DefaultGroupSize:          d.size,
			DefaultRequiredCounselors: d.counselors,
			Gender:                    d.gender,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating cabin %s: %w", d.name, err)
		}
		m[d.name] = c
	}
	return m, nil
}

func createActivities(ctx context.Context, q *db.Queries, campID pgtype.UUID) (map[string]db.Activity, error) {
	names := []string{"Archery", "Swimming", "Arts & Crafts", "Nature Hiking", "Campfire Cooking", "Canoeing"}
	m := make(map[string]db.Activity)
	for _, name := range names {
		a, err := q.CreateActivity(ctx, db.CreateActivityParams{
			CampID:       campID,
			ActivityName: name,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating activity %s: %w", name, err)
		}
		m[name] = a
	}
	return m, nil
}

func createTimeSlots(ctx context.Context, q *db.Queries, campID pgtype.UUID) (map[string]db.TimeSlot, error) {
	defs := []struct {
		name string
		sort int32
	}{
		{"Morning 1", 0},
		{"Morning 2", 1},
		{"Afternoon 1", 2},
		{"Afternoon 2", 3},
	}
	m := make(map[string]db.TimeSlot)
	for _, d := range defs {
		ts, err := q.CreateTimeSlot(ctx, db.CreateTimeSlotParams{
			CampID:       campID,
			TimeSlotName: d.name,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating time slot %s: %w", d.name, err)
		}
		m[d.name] = ts
	}
	return m, nil
}

func createCertifications(ctx context.Context, q *db.Queries, campID pgtype.UUID) (map[string]db.Certification, error) {
	names := []string{"Lifeguard", "First Aid", "Archery Instructor"}
	m := make(map[string]db.Certification)
	for _, name := range names {
		c, err := q.CreateCertification(ctx, db.CreateCertificationParams{
			CampID:            campID,
			CertificationName: name,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating certification %s: %w", name, err)
		}
		m[name] = c
	}
	return m, nil
}

func linkActivityCertifications(ctx context.Context, q *db.Queries, campID pgtype.UUID, activities map[string]db.Activity, certs map[string]db.Certification) error {
	links := []struct {
		activity string
		cert     string
	}{
		{"Swimming", "Lifeguard"},
		{"Archery", "Archery Instructor"},
		{"Canoeing", "Lifeguard"},
	}
	for _, l := range links {
		_, err := q.CreateActivityCertification(ctx, db.CreateActivityCertificationParams{
			CampID:          campID,
			ActivityID:      activities[l.activity].ID,
			CertificationID: certs[l.cert].ID,
		})
		if err != nil {
			return fmt.Errorf("error linking %s→%s: %w", l.activity, l.cert, err)
		}
	}
	return nil
}

func configureSessions(ctx context.Context, q *db.Queries, campID pgtype.UUID, s1, s2 db.Session, ageGroups map[string]db.AgeGroup, cabins map[string]db.CreateCabinRow, timeSlots map[string]db.TimeSlot, activities map[string]db.Activity) (*sessionConfig, error) {
	s1AGs := make(map[string]db.SessionAgeGroup)
	s2AGs := make(map[string]db.SessionAgeGroup)

	for _, name := range []string{"Bears", "Eagles", "Wolves"} {
		sag1, err := q.CreateSessionAgeGroup(ctx, db.CreateSessionAgeGroupParams{
			CampID:     campID,
			SessionID:  s1.ID,
			AgeGroupID: ageGroups[name].ID,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating session 1 age group: %w", err)
		}
		s1AGs[name] = sag1

		sag2, err := q.CreateSessionAgeGroup(ctx, db.CreateSessionAgeGroupParams{
			CampID:     campID,
			SessionID:  s2.ID,
			AgeGroupID: ageGroups[name].ID,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating session 2 age group: %w", err)
		}
		s2AGs[name] = sag2
	}

	cabinDefs := []struct {
		name     string
		ageGroup string
	}{
		{"Pine Lodge", "Bears"},
		{"Cedar Lodge", "Bears"},
		{"Maple Lodge", "Eagles"},
		{"Birch Lodge", "Eagles"},
		{"Oak Lodge", "Wolves"},
		{"Elm Lodge", "Wolves"},
	}
	for _, cd := range cabinDefs {
		for _, sagMap := range []map[string]db.SessionAgeGroup{s1AGs, s2AGs} {
			_, err := q.CreateSessionAgeGroupCabin(ctx, db.CreateSessionAgeGroupCabinParams{
				CampID:             campID,
				SessionAgeGroupID:  sagMap[cd.ageGroup].ID,
				CabinID:            cabins[cd.name].ID,
				GroupSize:          9,
				RequiredCounselors: 2,
			})
			if err != nil {
				return nil, fmt.Errorf("error creating session age group cabin: %w", err)
			}
		}
	}

	s1TSs, err := createSessionTimeSlots(ctx, q, campID, s1, timeSlots)
	if err != nil {
		return nil, err
	}
	s2TSs, err := createSessionTimeSlots(ctx, q, campID, s2, timeSlots)
	if err != nil {
		return nil, err
	}

	if err := createSessionActivities(ctx, q, campID, s1TSs, s2TSs, activities); err != nil {
		return nil, err
	}

	return &sessionConfig{s1AGs: s1AGs, s2AGs: s2AGs, s1TSs: s1TSs, s2TSs: s2TSs}, nil
}

func createSessionTimeSlots(ctx context.Context, q *db.Queries, campID pgtype.UUID, session db.Session, timeSlots map[string]db.TimeSlot) (map[string]db.SessionTimeSlot, error) {
	defs := []struct {
		name string
		sort int32
	}{
		{"Morning 1", 0},
		{"Morning 2", 1},
		{"Afternoon 1", 2},
		{"Afternoon 2", 3},
	}
	m := make(map[string]db.SessionTimeSlot)
	for _, d := range defs {
		sts, err := q.CreateSessionTimeSlot(ctx, db.CreateSessionTimeSlotParams{
			CampID:     campID,
			SessionID:  session.ID,
			TimeSlotID: timeSlots[d.name].ID,
			SortOrder:  d.sort,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating session time slot: %w", err)
		}
		m[d.name] = sts
	}
	return m, nil
}

func createSessionActivities(ctx context.Context, q *db.Queries, campID pgtype.UUID, s1TSs, s2TSs map[string]db.SessionTimeSlot, activities map[string]db.Activity) error {
	type saDef struct {
		ts             map[string]db.SessionTimeSlot
		timeSlotName   string
		activityName   string
		capacity       int32
		reqCounselors  int32
	}

	s1Defs := []saDef{
		{s1TSs, "Morning 1", "Swimming", 20, 3},
		{s1TSs, "Morning 1", "Nature Hiking", 12, 2},
		{s1TSs, "Morning 1", "Archery", 8, 2},
		{s1TSs, "Morning 2", "Arts & Crafts", 15, 1},
		{s1TSs, "Morning 2", "Canoeing", 10, 2},
		{s1TSs, "Morning 2", "Campfire Cooking", 10, 1},
		{s1TSs, "Afternoon 1", "Swimming", 20, 2},
		{s1TSs, "Afternoon 1", "Archery", 8, 2},
		{s1TSs, "Afternoon 1", "Nature Hiking", 12, 2},
		{s1TSs, "Afternoon 2", "Canoeing", 10, 2},
		{s1TSs, "Afternoon 2", "Arts & Crafts", 15, 1},
		{s1TSs, "Afternoon 2", "Campfire Cooking", 10, 1},
	}

	// Session 2 intentionally has an unsolvable activity schedule: Morning 2
	// has both Swimming and Canoeing which both require Lifeguard certification
	// (4 Lifeguard slots needed), but only 3 counselors hold that certification.
	// This demonstrates what happens when hard constraints cannot be satisfied.
	s2Defs := []saDef{
		{s2TSs, "Morning 1", "Archery", 8, 2},
		{s2TSs, "Morning 1", "Swimming", 20, 3},
		{s2TSs, "Morning 1", "Arts & Crafts", 15, 1},
		{s2TSs, "Morning 2", "Nature Hiking", 12, 2},
		{s2TSs, "Morning 2", "Swimming", 20, 2},
		{s2TSs, "Morning 2", "Canoeing", 10, 2},
		{s2TSs, "Afternoon 1", "Archery", 8, 2},
		{s2TSs, "Afternoon 1", "Arts & Crafts", 15, 1},
		{s2TSs, "Afternoon 1", "Campfire Cooking", 10, 1},
		{s2TSs, "Afternoon 2", "Nature Hiking", 12, 2},
		{s2TSs, "Afternoon 2", "Arts & Crafts", 15, 1},
		{s2TSs, "Afternoon 2", "Campfire Cooking", 10, 1},
	}

	for _, d := range append(s1Defs, s2Defs...) {
		_, err := q.CreateSessionActivity(ctx, db.CreateSessionActivityParams{
			CampID:             campID,
			SessionTimeSlotID:  d.ts[d.timeSlotName].ID,
			ActivityID:         activities[d.activityName].ID,
			Capacity:           d.capacity,
			RequiredCounselors: d.reqCounselors,
		})
		if err != nil {
			return fmt.Errorf("error creating session activity %s in %s: %w", d.activityName, d.timeSlotName, err)
		}
	}
	return nil
}

func createCounselors(ctx context.Context, q *db.Queries, campID pgtype.UUID) (map[string]db.Counselor, error) {
	defs := []struct {
		first  string
		last   string
		junior bool
		gender string
	}{
		{"Sarah", "Johnson", false, "female"},
		{"Mike", "Chen", false, "male"},
		{"Emily", "Davis", false, "female"},
		{"James", "Wilson", false, "male"},
		{"Lisa", "Rodriguez", false, "female"},
		{"David", "Brown", false, "male"},
		{"Rachel", "Kim", false, "female"},
		{"Tom", "Anderson", false, "male"},
		{"Karen", "Martinez", false, "female"},
		{"Alex", "Thompson", true, "male"},
		{"Jordan", "Lee", true, "female"},
		{"Taylor", "White", true, "male"},
	}
	m := make(map[string]db.Counselor)
	for _, d := range defs {
		c, err := q.CreateCounselor(ctx, db.CreateCounselorParams{
			CampID:          campID,
			FirstName:       d.first,
			LastName:        d.last,
			JuniorCounselor: d.junior,
			Gender:          d.gender,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating counselor %s %s: %w", d.first, d.last, err)
		}
		m[d.first+" "+d.last] = db.Counselor{
			ID:              c.ID,
			CampID:          c.CampID,
			JuniorCounselor: c.JuniorCounselor,
			Archived:        c.Archived,
			Gender:          c.Gender,
			FirstName:       c.FirstName,
			LastName:        c.LastName,
		}
	}
	return m, nil
}

func linkCounselorCertifications(ctx context.Context, q *db.Queries, campID pgtype.UUID, counselors map[string]db.Counselor, certs map[string]db.Certification) error {
	links := []struct {
		counselor string
		cert      string
	}{
		{"Sarah Johnson", "Lifeguard"},
		{"Mike Chen", "Archery Instructor"},
		{"Emily Davis", "Lifeguard"},
		{"James Wilson", "First Aid"},
		{"Lisa Rodriguez", "Lifeguard"},
		{"Lisa Rodriguez", "First Aid"},
		{"David Brown", "Archery Instructor"},
	}
	for _, l := range links {
		_, err := q.CreateCounselorCertification(ctx, db.CreateCounselorCertificationParams{
			CampID:          campID,
			CounselorID:     counselors[l.counselor].ID,
			CertificationID:  certs[l.cert].ID,
		})
		if err != nil {
			return fmt.Errorf("error linking %s→%s: %w", l.counselor, l.cert, err)
		}
	}
	return nil
}

func setCounselorPreferences(ctx context.Context, q *db.Queries, campID pgtype.UUID, counselors map[string]db.Counselor, ageGroups map[string]db.AgeGroup, s1, s2 db.Session, activities map[string]db.Activity) error {
	type agPref struct {
		counselor string
		ageGroup  string
		rank      int32
	}
	s1AGPrefs := []agPref{
		{"Sarah Johnson", "Bears", 1},
		{"Mike Chen", "Wolves", 1},
		{"Emily Davis", "Eagles", 1},
		{"James Wilson", "Bears", 1},
		{"Lisa Rodriguez", "Eagles", 1},
		{"David Brown", "Wolves", 1},
		{"Rachel Kim", "Bears", 1},
		{"Tom Anderson", "Eagles", 1},
		{"Karen Martinez", "Wolves", 1},
	}
	s2AGPrefs := []agPref{
		{"Sarah Johnson", "Bears", 1},
		{"Mike Chen", "Eagles", 1},
		{"Emily Davis", "Eagles", 1},
		{"James Wilson", "Bears", 1},
		{"Lisa Rodriguez", "Eagles", 1},
	}

	for _, p := range s1AGPrefs {
		_, err := q.CreateCounselorAgeGroupPreference(ctx, db.CreateCounselorAgeGroupPreferenceParams{
			CampID:      campID,
			CounselorID: counselors[p.counselor].ID,
			SessionID:   s1.ID,
			AgeGroupID:  ageGroups[p.ageGroup].ID,
			Rank:        p.rank,
		})
		if err != nil {
			return fmt.Errorf("error creating counselor age group pref: %w", err)
		}
	}
	for _, p := range s2AGPrefs {
		_, err := q.CreateCounselorAgeGroupPreference(ctx, db.CreateCounselorAgeGroupPreferenceParams{
			CampID:      campID,
			CounselorID: counselors[p.counselor].ID,
			SessionID:   s2.ID,
			AgeGroupID:  ageGroups[p.ageGroup].ID,
			Rank:        p.rank,
		})
		if err != nil {
			return fmt.Errorf("error creating counselor age group pref: %w", err)
		}
	}

	type actPref struct {
		counselor string
		activity  string
		rank      int32
	}
	s1ActPrefs := []actPref{
		{"Sarah Johnson", "Swimming", 1},
		{"Mike Chen", "Archery", 1},
		{"Emily Davis", "Swimming", 1},
		{"Emily Davis", "Nature Hiking", 2},
		{"James Wilson", "Canoeing", 1},
		{"Lisa Rodriguez", "Canoeing", 1},
		{"David Brown", "Archery", 1},
		{"Rachel Kim", "Arts & Crafts", 1},
		{"Tom Anderson", "Nature Hiking", 1},
		{"Karen Martinez", "Campfire Cooking", 1},
	}
	s2ActPrefs := []actPref{
		{"Sarah Johnson", "Swimming", 1},
		{"Mike Chen", "Archery", 1},
		{"Mike Chen", "Nature Hiking", 2},
		{"Emily Davis", "Swimming", 1},
		{"James Wilson", "Canoeing", 1},
		{"Lisa Rodriguez", "Canoeing", 1},
		{"David Brown", "Archery", 1},
		{"Rachel Kim", "Arts & Crafts", 1},
	}

	for _, p := range s1ActPrefs {
		_, err := q.CreateCounselorActivityPreference(ctx, db.CreateCounselorActivityPreferenceParams{
			CampID:      campID,
			CounselorID: counselors[p.counselor].ID,
			SessionID:   s1.ID,
			ActivityID:  activities[p.activity].ID,
			Rank:        p.rank,
		})
		if err != nil {
			return fmt.Errorf("error creating counselor activity pref: %w", err)
		}
	}
	for _, p := range s2ActPrefs {
		_, err := q.CreateCounselorActivityPreference(ctx, db.CreateCounselorActivityPreferenceParams{
			CampID:      campID,
			CounselorID: counselors[p.counselor].ID,
			SessionID:   s2.ID,
			ActivityID:  activities[p.activity].ID,
			Rank:        p.rank,
		})
		if err != nil {
			return fmt.Errorf("error creating counselor activity pref: %w", err)
		}
	}

	type coPref struct {
		counselor string
		preferred string
		rank      int32
	}
	s1CoPrefs := []coPref{
		{"Sarah Johnson", "Emily Davis", 1},
		{"Emily Davis", "Sarah Johnson", 1},
		{"Mike Chen", "James Wilson", 1},
		{"James Wilson", "Mike Chen", 1},
		{"Rachel Kim", "Lisa Rodriguez", 1},
		{"Tom Anderson", "David Brown", 1},
	}
	s2CoPrefs := []coPref{
		{"Sarah Johnson", "Emily Davis", 1},
		{"Emily Davis", "Sarah Johnson", 1},
		{"James Wilson", "David Brown", 1},
		{"Rachel Kim", "Karen Martinez", 1},
	}

	for _, p := range s1CoPrefs {
		_, err := q.CreateCounselorCocounselorPreference(ctx, db.CreateCounselorCocounselorPreferenceParams{
			CampID:               campID,
			CounselorID:          counselors[p.counselor].ID,
			SessionID:            s1.ID,
			PreferredCounselorID: counselors[p.preferred].ID,
			Rank:                 p.rank,
		})
		if err != nil {
			return fmt.Errorf("error creating co-counselor pref: %w", err)
		}
	}
	for _, p := range s2CoPrefs {
		_, err := q.CreateCounselorCocounselorPreference(ctx, db.CreateCounselorCocounselorPreferenceParams{
			CampID:               campID,
			CounselorID:          counselors[p.counselor].ID,
			SessionID:            s2.ID,
			PreferredCounselorID: counselors[p.preferred].ID,
			Rank:                 p.rank,
		})
		if err != nil {
			return fmt.Errorf("error creating co-counselor pref: %w", err)
		}
	}

	return nil
}

func setCounselorHistory(ctx context.Context, q *db.Queries, campID pgtype.UUID, counselors map[string]db.Counselor, s1 db.Session, ageGroups map[string]db.AgeGroup, cabins map[string]db.CreateCabinRow) error {
	type history struct {
		counselor string
		ageGroup  string
		cabin     string
	}
	histories := []history{
		{"Sarah Johnson", "Bears", "Pine Lodge"},
		{"Mike Chen", "Wolves", "Elm Lodge"},
		{"Emily Davis", "Eagles", "Maple Lodge"},
		{"James Wilson", "Bears", "Cedar Lodge"},
	}
	for _, h := range histories {
		_, err := q.CreateCounselorSessionHistoryEntry(ctx, db.CreateCounselorSessionHistoryEntryParams{
			CampID:      campID,
			CounselorID: counselors[h.counselor].ID,
			SessionID:   s1.ID,
			AgeGroupID:  ageGroups[h.ageGroup].ID,
			CabinID:     cabins[h.cabin].ID,
		})
		if err != nil {
			return fmt.Errorf("error creating counselor history: %w", err)
		}
	}
	return nil
}

func createCampers(ctx context.Context, q *db.Queries, campID pgtype.UUID, ageGroups map[string]db.AgeGroup) (map[string]db.Camper, error) {
	type camperDef struct {
		first  string
		last   string
		gender string
	}
	// Each age group contributes 6 female and 6 male campers, matching the
	// one-female and one-male cabin per age group. Last names are realistic
	// surnames so the demo data exercises the new first_name/last_name
	// columns end-to-end.
	bears := []camperDef{
		{"Emma", "Patel", "female"}, {"Liam", "Nguyen", "male"},
		{"Olivia", "Garcia", "female"}, {"Noah", "Smith", "male"},
		{"Ava", "Cohen", "female"}, {"William", "Tran", "male"},
		{"Sophia", "Hughes", "female"}, {"Mason", "Reyes", "male"},
		{"Isabella", "Walker", "female"}, {"Lucas", "Park", "male"},
		{"Mia", "Foster", "female"}, {"Ethan", "Bailey", "male"},
	}
	eagles := []camperDef{
		{"Aiden", "Bennett", "male"}, {"Charlotte", "Wright", "female"},
		{"Benjamin", "Carter", "male"}, {"Harper", "Mitchell", "female"},
		{"Daniel", "Murphy", "male"}, {"Amelia", "Cooper", "female"},
		{"Henry", "Rivera", "male"}, {"Evelyn", "Stewart", "female"},
		{"Jack", "Morris", "male"}, {"Abigail", "Sanders", "female"},
		{"Owen", "Coleman", "male"}, {"Grace", "Hayes", "female"},
	}
	wolves := []camperDef{
		{"Logan", "Russell", "male"}, {"Chloe", "Brooks", "female"},
		{"Caleb", "Diaz", "male"}, {"Zoe", "Wood", "female"},
		{"Nathan", "Bell", "male"}, {"Lily", "Barnes", "female"},
		{"Ryan", "Powell", "male"}, {"Hannah", "Long", "female"},
		{"Connor", "Perry", "male"}, {"Maya", "Ross", "female"},
		{"Dylan", "Jenkins", "male"}, {"Stella", "Price", "female"},
	}

	all := append(append(bears, eagles...), wolves...)
	m := make(map[string]db.Camper)
	for _, d := range all {
		c, err := q.CreateCamper(ctx, db.CreateCamperParams{
			CampID:    campID,
			FirstName: d.first,
			LastName:  d.last,
			Gender:    d.gender,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating camper %s %s: %w", d.first, d.last, err)
		}
		m[d.first] = db.Camper{
			ID:        c.ID,
			CampID:    c.CampID,
			FirstName: c.FirstName,
			LastName:  c.LastName,
			Gender:    c.Gender,
		}
	}
	return m, nil
}

func rosterCounselors(ctx context.Context, q *db.Queries, campID pgtype.UUID, counselors map[string]db.Counselor, s1, s2 db.Session) error {
	for name, c := range counselors {
		for _, s := range []db.Session{s1, s2} {
			_, err := q.AddSessionCounselor(ctx, db.AddSessionCounselorParams{
				CampID:      campID,
				SessionID:   s.ID,
				CounselorID: c.ID,
			})
			if err != nil {
				return fmt.Errorf("error rostering counselor %s on session %s: %w", name, s.SessionName, err)
			}
		}
	}
	return nil
}

func enrollCampers(ctx context.Context, q *db.Queries, campID pgtype.UUID, campers map[string]db.Camper, s1, s2 db.Session, cfg *sessionConfig) error {
	agNames := []string{"Bears", "Eagles", "Wolves"}

	bears := []string{"Emma", "Liam", "Olivia", "Noah", "Ava", "William", "Sophia", "Mason", "Isabella", "Lucas", "Mia", "Ethan"}
	eagles := []string{"Aiden", "Charlotte", "Benjamin", "Harper", "Daniel", "Amelia", "Henry", "Evelyn", "Jack", "Abigail", "Owen", "Grace"}
	wolves := []string{"Logan", "Chloe", "Caleb", "Zoe", "Nathan", "Lily", "Ryan", "Hannah", "Connor", "Maya", "Dylan", "Stella"}

	ageGroupCampers := map[string][]string{
		"Bears":  bears,
		"Eagles": eagles,
		"Wolves": wolves,
	}

	for _, agName := range agNames {
		for _, name := range ageGroupCampers[agName] {
			_, err := q.CreateSessionEnrollment(ctx, db.CreateSessionEnrollmentParams{
				CampID:            campID,
				CamperID:          campers[name].ID,
				SessionID:         s1.ID,
				SessionAgeGroupID: cfg.s1AGs[agName].ID,
			})
			if err != nil {
				return fmt.Errorf("error enrolling %s in session 1: %w", name, err)
			}

			_, err = q.CreateSessionEnrollment(ctx, db.CreateSessionEnrollmentParams{
				CampID:            campID,
				CamperID:          campers[name].ID,
				SessionID:         s2.ID,
				SessionAgeGroupID: cfg.s2AGs[agName].ID,
			})
			if err != nil {
				return fmt.Errorf("error enrolling %s in session 2: %w", name, err)
			}
		}
	}
	return nil
}

func setFriendPreferences(ctx context.Context, q *db.Queries, campID pgtype.UUID, campers map[string]db.Camper, s1, s2 db.Session) error {
	type friendPref struct {
		camper   string
		friend   string
		rank     int32
	}

	s1Prefs := []friendPref{
		{"Emma", "Olivia", 1},
		{"Emma", "Sophia", 2},
		{"Olivia", "Emma", 1},
		{"Olivia", "Sophia", 2},
		{"Liam", "Noah", 1},
		{"Aiden", "Benjamin", 1},
		{"Aiden", "Daniel", 2},
		{"Charlotte", "Harper", 1},
		{"Logan", "Caleb", 1},
		{"Logan", "Nathan", 2},
	}

	s2Prefs := []friendPref{
		{"Emma", "Olivia", 1},
		{"Emma", "Ava", 2},
		{"Olivia", "Emma", 1},
		{"Olivia", "Sophia", 2},
		{"Liam", "Noah", 1},
		{"Liam", "Mason", 2},
		{"Aiden", "Benjamin", 1},
		{"Aiden", "Daniel", 2},
		{"Charlotte", "Harper", 1},
		{"Logan", "Caleb", 1},
		{"Logan", "Nathan", 2},
	}

	for _, p := range s1Prefs {
		_, err := q.CreateCamperFriendPreference(ctx, db.CreateCamperFriendPreferenceParams{
			CampID:            campID,
			CamperID:          campers[p.camper].ID,
			SessionID:         s1.ID,
			PreferredCamperID: campers[p.friend].ID,
			Rank:              p.rank,
		})
		if err != nil {
			return fmt.Errorf("error creating friend pref: %w", err)
		}
	}
	for _, p := range s2Prefs {
		_, err := q.CreateCamperFriendPreference(ctx, db.CreateCamperFriendPreferenceParams{
			CampID:            campID,
			CamperID:          campers[p.camper].ID,
			SessionID:         s2.ID,
			PreferredCamperID: campers[p.friend].ID,
			Rank:              p.rank,
		})
		if err != nil {
			return fmt.Errorf("error creating friend pref: %w", err)
		}
	}
	return nil
}

