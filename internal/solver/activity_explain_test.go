package solver

import (
	"strings"
	"testing"
)

func TestExplainActivityIneligibility(t *testing.T) {
	// Two slots for the same activity "Canoeing" requiring "Lifeguard"
	// across two time slots, plus one slot for "Arts" with no certs.
	canoeMorning := ActivitySlot{
		ID: "slot-canoe-morning", ActivityID: "act-canoe", ActivityName: "Canoeing",
		TimeSlotID: "ts-morning", TimeSlotName: "Morning",
		RequiredCounselors: 1, Capacity: 2,
		RequiredCertifications: []string{"cert-lifeguard"},
	}
	canoeAfternoon := ActivitySlot{
		ID: "slot-canoe-afternoon", ActivityID: "act-canoe", ActivityName: "Canoeing",
		TimeSlotID: "ts-afternoon", TimeSlotName: "Afternoon",
		RequiredCounselors: 1, Capacity: 2,
		RequiredCertifications: []string{"cert-lifeguard"},
	}
	artsMorning := ActivitySlot{
		ID: "slot-arts-morning", ActivityID: "act-arts", ActivityName: "Arts",
		TimeSlotID: "ts-morning", TimeSlotName: "Morning",
		RequiredCounselors: 1, Capacity: 5,
	}
	swimMorning := ActivitySlot{
		ID: "slot-swim-morning", ActivityID: "act-swim", ActivityName: "Swimming",
		TimeSlotID: "ts-morning", TimeSlotName: "Morning",
		RequiredCounselors: 1, Capacity: 2,
		RequiredCertifications: []string{"cert-lifeguard"},
	}

	t.Run("eligible-but-unmet stays in UnmetPreferences", func(t *testing.T) {
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{artsMorning, canoeMorning},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "Karen", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {{TargetID: "act-arts", Rank: 1}},
			},
			CertificationNames: map[string]string{"cert-lifeguard": "Lifeguard"},
		}
		// Karen IS eligible for Arts (no certs needed) but not assigned.
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{}},
		}

		exp := ExplainActivity(snapshot, solution)

		if len(exp.IneligiblePreferences) != 0 {
			t.Errorf("expected 0 ineligible, got %d: %v", len(exp.IneligiblePreferences), exp.IneligiblePreferences)
		}
		// Should be in unmet (also includes "unassigned counselor"), so at
		// least one entry should be activity_preference.
		foundUnmet := false
		for _, u := range exp.UnmetPreferences {
			if u.Constraint == "activity_preference" {
				foundUnmet = true
				if !strings.Contains(u.Message, "Arts") {
					t.Errorf("expected message to mention Arts, got %q", u.Message)
				}
			}
		}
		if !foundUnmet {
			t.Errorf("expected an activity_preference entry, got %v", exp.UnmetPreferences)
		}
	})

	t.Run("ineligible single missing cert names the cert", func(t *testing.T) {
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{canoeMorning, canoeAfternoon},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "James", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {{TargetID: "act-canoe", Rank: 1}},
			},
			CertificationNames: map[string]string{"cert-lifeguard": "Lifeguard"},
		}
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{}},
		}

		exp := ExplainActivity(snapshot, solution)

		if len(exp.IneligiblePreferences) != 1 {
			t.Fatalf("expected 1 ineligible, got %d: %v", len(exp.IneligiblePreferences), exp.IneligiblePreferences)
		}
		ip := exp.IneligiblePreferences[0]
		if ip.Constraint != "activity_preference_ineligible" {
			t.Errorf("expected constraint activity_preference_ineligible, got %q", ip.Constraint)
		}
		// Single ineligible covering all prefs => aggregated message form.
		want := "cannot be assigned to any preferred activity (Canoeing — missing Lifeguard)"
		if ip.Message != want {
			t.Errorf("message:\n got  %q\n want %q", ip.Message, want)
		}

		// The eligible-unmet bucket must NOT also contain Canoeing.
		for _, u := range exp.UnmetPreferences {
			if u.Constraint == "activity_preference" && strings.Contains(u.Message, "Canoeing") {
				t.Errorf("Canoeing leaked into unmet preferences: %q", u.Message)
			}
		}
	})

	t.Run("multiple missing certs listed", func(t *testing.T) {
		// One slot needs both Lifeguard AND First Aid.
		dualSlot := ActivitySlot{
			ID: "slot-dual", ActivityID: "act-rescue", ActivityName: "Rescue",
			TimeSlotID: "ts-morning", TimeSlotName: "Morning",
			RequiredCounselors: 1, Capacity: 1,
			RequiredCertifications: []string{"cert-lifeguard", "cert-firstaid"},
		}
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{dualSlot},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "Tom", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {{TargetID: "act-rescue", Rank: 1}},
			},
			CertificationNames: map[string]string{
				"cert-lifeguard": "Lifeguard",
				"cert-firstaid":  "First Aid",
			},
		}
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{}},
		}

		exp := ExplainActivity(snapshot, solution)

		if len(exp.IneligiblePreferences) != 1 {
			t.Fatalf("expected 1 ineligible, got %d: %v", len(exp.IneligiblePreferences), exp.IneligiblePreferences)
		}
		msg := exp.IneligiblePreferences[0].Message
		if !strings.Contains(msg, "Lifeguard") || !strings.Contains(msg, "First Aid") {
			t.Errorf("expected message to list both certs, got %q", msg)
		}
		// Single pref → aggregated form uses "missing X, Y".
		if !strings.Contains(msg, "missing Lifeguard, First Aid") {
			t.Errorf("expected 'missing Lifeguard, First Aid', got %q", msg)
		}
	})

	t.Run("eligible if one slot variant has satisfiable certs", func(t *testing.T) {
		// Two slots for the same activity: one needs Lifeguard, the other
		// has no requirements. Counselor lacks Lifeguard but is eligible
		// via the second slot.
		hardSlot := ActivitySlot{
			ID: "slot-canoe-1", ActivityID: "act-canoe", ActivityName: "Canoeing",
			TimeSlotID: "ts-morning", TimeSlotName: "Morning",
			RequiredCounselors: 1, Capacity: 1,
			RequiredCertifications: []string{"cert-lifeguard"},
		}
		easySlot := ActivitySlot{
			ID: "slot-canoe-2", ActivityID: "act-canoe", ActivityName: "Canoeing",
			TimeSlotID: "ts-afternoon", TimeSlotName: "Afternoon",
			RequiredCounselors: 1, Capacity: 1,
		}
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{hardSlot, easySlot},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "Karen", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {{TargetID: "act-canoe", Rank: 1}},
			},
			CertificationNames: map[string]string{"cert-lifeguard": "Lifeguard"},
		}
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{}},
		}

		exp := ExplainActivity(snapshot, solution)

		if len(exp.IneligiblePreferences) != 0 {
			t.Errorf("expected 0 ineligible (eligible via easy slot), got %v", exp.IneligiblePreferences)
		}
	})

	t.Run("mixed buckets render as individual rows in each bucket", func(t *testing.T) {
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{canoeMorning, artsMorning},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "James", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {
					{TargetID: "act-canoe", Rank: 1}, // ineligible
					{TargetID: "act-arts", Rank: 2},  // eligible
				},
			},
			CertificationNames: map[string]string{"cert-lifeguard": "Lifeguard"},
		}
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{}},
		}

		exp := ExplainActivity(snapshot, solution)

		// Ineligible bucket: not all-or-nothing, so individual row.
		if len(exp.IneligiblePreferences) != 1 {
			t.Fatalf("expected 1 ineligible row, got %d: %v", len(exp.IneligiblePreferences), exp.IneligiblePreferences)
		}
		ip := exp.IneligiblePreferences[0]
		want := "cannot be assigned to preferred activity Canoeing (rank 1) — missing required certification Lifeguard"
		if ip.Message != want {
			t.Errorf("ineligible message:\n got  %q\n want %q", ip.Message, want)
		}

		// Eligible-unmet bucket: should have an Arts row.
		foundArts := false
		for _, u := range exp.UnmetPreferences {
			if u.Constraint == "activity_preference" && strings.Contains(u.Message, "Arts") {
				foundArts = true
				if strings.Contains(u.Message, "any preferred") {
					t.Errorf("expected per-pref form, got aggregated: %q", u.Message)
				}
			}
		}
		if !foundArts {
			t.Errorf("expected eligible-unmet entry for Arts, got %v", exp.UnmetPreferences)
		}
	})

	t.Run("all ineligible aggregates", func(t *testing.T) {
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{canoeMorning, swimMorning},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "James", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {
					{TargetID: "act-canoe", Rank: 1},
					{TargetID: "act-swim", Rank: 2},
				},
			},
			CertificationNames: map[string]string{"cert-lifeguard": "Lifeguard"},
		}
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{}},
		}

		exp := ExplainActivity(snapshot, solution)

		if len(exp.IneligiblePreferences) != 1 {
			t.Fatalf("expected single aggregated ineligible row, got %d: %v",
				len(exp.IneligiblePreferences), exp.IneligiblePreferences)
		}
		msg := exp.IneligiblePreferences[0].Message
		if !strings.HasPrefix(msg, "cannot be assigned to any preferred activity") {
			t.Errorf("expected aggregated form, got %q", msg)
		}
		if !strings.Contains(msg, "Canoeing") || !strings.Contains(msg, "Swimming") {
			t.Errorf("expected both activities listed, got %q", msg)
		}
		// Activities should be ordered by rank: Canoeing (rank 1) before Swimming (rank 2).
		if strings.Index(msg, "Canoeing") > strings.Index(msg, "Swimming") {
			t.Errorf("expected Canoeing before Swimming (rank order), got %q", msg)
		}
	})

	t.Run("preferred activity has no slots in session", func(t *testing.T) {
		// Counselor prefers act-ghost which is not scheduled this session
		// (snapshot.Slots only contains an unrelated activity).
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{artsMorning},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "Karen", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {{TargetID: "act-ghost", Rank: 1}},
			},
			CertificationNames: map[string]string{},
		}
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{}},
		}

		exp := ExplainActivity(snapshot, solution)

		if len(exp.IneligiblePreferences) != 1 {
			t.Fatalf("expected 1 ineligible row, got %d: %v", len(exp.IneligiblePreferences), exp.IneligiblePreferences)
		}
		ip := exp.IneligiblePreferences[0]
		if ip.Constraint != "activity_preference_ineligible" {
			t.Errorf("expected constraint activity_preference_ineligible, got %q", ip.Constraint)
		}
		// Single pref => aggregated form, "X — not scheduled".
		want := "cannot be assigned to any preferred activity (act-ghost — not scheduled)"
		if ip.Message != want {
			t.Errorf("message:\n got  %q\n want %q", ip.Message, want)
		}
		// Regression guard: must not produce the empty-cert text.
		if strings.Contains(ip.Message, "missing required certification") {
			t.Errorf("did not expect missing-cert text for no-slots case, got %q", ip.Message)
		}
	})
}

// TestExplainActivityBestMetRank covers the rule that a preference is only
// reported as unmet when no equal-or-higher-rank preference was satisfied.
// Lower-ranked unmet preferences are suppressed when a higher-ranked one
// was met, since the counselor got something at least as good.
func TestExplainActivityBestMetRank(t *testing.T) {
	swimMorning := ActivitySlot{
		ID: "slot-swim", ActivityID: "act-swim", ActivityName: "Swimming",
		TimeSlotID: "ts-morning", TimeSlotName: "Morning",
		RequiredCounselors: 1, Capacity: 2,
	}
	hikeMorning := ActivitySlot{
		ID: "slot-hike", ActivityID: "act-hike", ActivityName: "Nature Hiking",
		TimeSlotID: "ts-morning", TimeSlotName: "Morning",
		RequiredCounselors: 1, Capacity: 2,
	}
	artsMorning := ActivitySlot{
		ID: "slot-arts", ActivityID: "act-arts", ActivityName: "Arts",
		TimeSlotID: "ts-morning", TimeSlotName: "Morning",
		RequiredCounselors: 1, Capacity: 2,
	}

	t.Run("lower unmet suppressed when higher met", func(t *testing.T) {
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{swimMorning, hikeMorning},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "Emily", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {
					{TargetID: "act-swim", Rank: 1},
					{TargetID: "act-hike", Rank: 2},
				},
			},
		}
		// Emily got her rank-1 (Swimming). Rank-2 should NOT be reported.
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{
				"slot-swim": {"c1"},
			}},
		}

		exp := ExplainActivity(snapshot, solution)

		for _, u := range exp.UnmetPreferences {
			if u.Constraint == "activity_preference" {
				t.Errorf("did not expect any activity_preference unmet entry, got %q", u.Message)
			}
		}
	})

	t.Run("higher unmet reported when only lower met", func(t *testing.T) {
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{swimMorning, hikeMorning},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "Emily", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {
					{TargetID: "act-swim", Rank: 1},
					{TargetID: "act-hike", Rank: 2},
				},
			},
		}
		// Emily got her rank-2 (Hiking). Rank-1 (Swimming) is still a real miss.
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{
				"slot-hike": {"c1"},
			}},
		}

		exp := ExplainActivity(snapshot, solution)

		var swimUnmet *ActivityUnmetPreference
		for i := range exp.UnmetPreferences {
			u := &exp.UnmetPreferences[i]
			if u.Constraint == "activity_preference" && strings.Contains(u.Message, "Swimming") {
				swimUnmet = u
				break
			}
		}
		if swimUnmet == nil {
			t.Fatalf("expected Swimming reported as unmet, got %v", exp.UnmetPreferences)
		}
		// Hiking (the met one) must not appear as unmet.
		for _, u := range exp.UnmetPreferences {
			if u.Constraint == "activity_preference" && strings.Contains(u.Message, "Hiking") {
				t.Errorf("did not expect Hiking in unmet preferences: %q", u.Message)
			}
		}
	})

	t.Run("aggregation uses candidates above met", func(t *testing.T) {
		// Three preferences: Swim (1), Hike (2), Arts (3). Emily got Arts
		// (rank 3). Both Swim and Hike are real misses. Since they cover
		// ALL prefs above the met rank, the eligible bucket aggregates.
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{swimMorning, hikeMorning, artsMorning},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "Emily", Certifications: map[string]bool{}},
			},
			ActivityPreferences: map[string][]RankedPreference{
				"c1": {
					{TargetID: "act-swim", Rank: 1},
					{TargetID: "act-hike", Rank: 2},
					{TargetID: "act-arts", Rank: 3},
				},
			},
		}
		solution := ActivitySolution{
			Assignment: ActivityAssignment{SlotCounselors: map[string][]string{
				"slot-arts": {"c1"},
			}},
		}

		exp := ExplainActivity(snapshot, solution)

		var aggregated *ActivityUnmetPreference
		for i := range exp.UnmetPreferences {
			u := &exp.UnmetPreferences[i]
			if u.Constraint == "activity_preference" && strings.Contains(u.Message, "any preferred activity") {
				aggregated = u
				break
			}
		}
		if aggregated == nil {
			t.Fatalf("expected aggregated 'any preferred activity' row, got %v", exp.UnmetPreferences)
		}
		if !strings.Contains(aggregated.Message, "Swimming") || !strings.Contains(aggregated.Message, "Nature Hiking") {
			t.Errorf("expected both Swimming and Nature Hiking in aggregated message, got %q", aggregated.Message)
		}
		if strings.Contains(aggregated.Message, "Arts") {
			t.Errorf("did not expect Arts (the met preference) in aggregated message, got %q", aggregated.Message)
		}
	})
}
