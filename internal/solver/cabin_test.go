package solver

import (
	"testing"
)

func TestSortPairsByScoreDesc(t *testing.T) {
	pairs := []CabinSolution{
		{TotalScore: 1.0},
		{TotalScore: 5.0},
		{TotalScore: 3.0},
		{TotalScore: 4.0},
		{TotalScore: 2.0},
	}
	sortPairsByScoreDesc(pairs)
	want := []float64{5.0, 4.0, 3.0, 2.0, 1.0}
	for i, p := range pairs {
		if p.TotalScore != want[i] {
			t.Errorf("position %d: got %v, want %v", i, p.TotalScore, want[i])
		}
	}
}

// TestSolveCabinPairsSortedByCombinedScore verifies that the combined cabin
// solver returns pairs sorted by counselor+camper combined score (rather
// than by counselor-only score). With no preferences and identical cabins,
// the sub-scores are deterministic; we simply assert the descending-by-
// combined-score invariant on the returned slice.
func TestSolveCabinPairsSortedByCombinedScore(t *testing.T) {
	counselors := SessionSnapshot{
		Cabins: []Cabin{
			{ID: "c1", Name: "Pine", AgeGroupID: "ag1", AgeGroupName: "Bears", RequiredCounselors: 1, Capacity: 4, Gender: "female"},
			{ID: "c2", Name: "Oak", AgeGroupID: "ag1", AgeGroupName: "Bears", RequiredCounselors: 1, Capacity: 4, Gender: "female"},
		},
		Counselors: []Counselor{
			{ID: "sr1", Name: "S1", IsJunior: false, Gender: "female"},
			{ID: "sr2", Name: "S2", IsJunior: false, Gender: "female"},
		},
	}
	campers := CamperCabinSnapshot{
		Cabins: []CamperCabin{
			{ID: "c1", Name: "Pine", AgeGroupID: "ag1", AgeGroupName: "Bears", Capacity: 4, Gender: "female"},
			{ID: "c2", Name: "Oak", AgeGroupID: "ag1", AgeGroupName: "Bears", Capacity: 4, Gender: "female"},
		},
		Campers: []Camper{
			{ID: "ca1", Name: "C1", AgeGroupID: "ag1", Gender: "female"},
			{ID: "ca2", Name: "C2", AgeGroupID: "ag1", Gender: "female"},
		},
	}

	cfg := DefaultCabinSolverConfig()
	cfg.MaxSolutions = 5

	pairs := SolveCabin(CabinSnapshot{
		SessionID: "s1",
		Counselor: counselors,
		Camper:    campers,
	}, cfg)

	if len(pairs) == 0 {
		t.Fatal("expected at least one pair")
	}
	if len(pairs) > cfg.MaxSolutions {
		t.Errorf("expected at most %d pairs, got %d", cfg.MaxSolutions, len(pairs))
	}
	for i := 1; i < len(pairs); i++ {
		if pairs[i].TotalScore > pairs[i-1].TotalScore {
			t.Errorf("pairs not sorted by combined score desc at %d: %v > %v", i, pairs[i].TotalScore, pairs[i-1].TotalScore)
		}
		expected := pairs[i].Counselor.Score.Total + pairs[i].Camper.Score.Total
		if pairs[i].TotalScore != expected {
			t.Errorf("pair %d TotalScore %v != counselor %v + camper %v", i, pairs[i].TotalScore, pairs[i].Counselor.Score.Total, pairs[i].Camper.Score.Total)
		}
	}
}

// TestPairAndRankCabinSolutionsBeatsCounselorFirst is a white-box regression
// test for the over-sampling fix in SolveCabin. It hand-crafts two counselor
// candidates with very different cabin placements:
//
//   - solnA scores higher on the counselor side but stuffs both seniors into
//     "Big" cabin, leaving "Big" with only 2 camper slots. The 3 friend-loving
//     campers split 2/1 and only one mutual pair stays together.
//   - solnB scores lower on the counselor side but spreads the seniors across
//     cabins, leaving "Big" with 3 camper slots. All 3 campers cluster in
//     "Big", satisfying every friend preference.
//
// With the over-sampling fix, pairAndRankCabinSolutions enumerates BOTH
// (counselor, camper) combinations and ranks by combined score, so solnB+its
// camper layout wins. Without the fix (counselor-first truncation to top-1),
// only solnA would be paired and the test would fail.
func TestPairAndRankCabinSolutionsBeatsCounselorFirst(t *testing.T) {
	cabinBig := CamperCabin{
		ID: "big", Name: "Big", AgeGroupID: "ag1", AgeGroupName: "Bears",
		Capacity: 4, Gender: "female",
	}
	cabinSmall := CamperCabin{
		ID: "small", Name: "Small", AgeGroupID: "ag1", AgeGroupName: "Bears",
		Capacity: 2, Gender: "female",
	}
	camperSnap := CamperCabinSnapshot{
		Cabins: []CamperCabin{cabinBig, cabinSmall},
		Campers: []Camper{
			{ID: "ca1", Name: "C1", AgeGroupID: "ag1", Gender: "female"},
			{ID: "ca2", Name: "C2", AgeGroupID: "ag1", Gender: "female"},
			{ID: "ca3", Name: "C3", AgeGroupID: "ag1", Gender: "female"},
		},
		// Friend pref clique: every camper ranks the other two.
		FriendPreferences: map[string][]RankedPreference{
			"ca1": {{TargetID: "ca2", Rank: 1}, {TargetID: "ca3", Rank: 2}},
			"ca2": {{TargetID: "ca1", Rank: 1}, {TargetID: "ca3", Rank: 2}},
			"ca3": {{TargetID: "ca1", Rank: 1}, {TargetID: "ca2", Rank: 2}},
		},
	}

	// solnA: both seniors in Big -> Big has 2 camper slots, Small has 1.
	// Campers split 2/1; only one of the 6 directional friend prefs at
	// rank 1 stays in the same cabin (ca1<->ca2). Camper score is low.
	// Counselor-side score is moderate (15).
	solnA := Solution{
		Assignment: Assignment{
			CabinCounselors: map[string][]string{
				"big":   {"sr1", "sr2"},
				"small": {},
			},
		},
		Score: ScoreResult{Total: 15},
	}
	// solnB: one senior per cabin -> Big has 3 camper slots, Small has 1.
	// All 3 campers cluster in Big, every friend pref is satisfied.
	// Counselor-side score is 0 (no cocounselor satisfaction).
	solnB := Solution{
		Assignment: Assignment{
			CabinCounselors: map[string][]string{
				"big":   {"sr1"},
				"small": {"sr2"},
			},
		},
		Score: ScoreResult{Total: 0},
	}

	camperCfg := DefaultCamperSolverConfig()
	camperCfg.MaxSolutions = 5
	camperCfg.MaxIterations = 100_000

	pairs := pairAndRankCabinSolutions(
		[]Solution{solnA, solnB},
		camperSnap,
		camperCfg,
		5, // camperPerCounselor
		5, // maxSolutions
	)

	if len(pairs) == 0 {
		t.Fatal("expected at least one paired solution")
	}

	// The top pair must use solnB (the lower-counselor-score candidate
	// that yields the better camper layout). Without over-sampling, the
	// solver would only pair solnA and we'd never see solnB's camper win.
	top := pairs[0]
	if top.Counselor.Score.Total != solnB.Score.Total {
		t.Fatalf("over-sampling regression: top pair used counselor solution with score %.2f, "+
			"expected the lower-counselor-score alternative (%.2f) which leaves better camper capacity. "+
			"Counselor-first truncation may have crept back in.",
			top.Counselor.Score.Total, solnB.Score.Total)
	}

	// Sanity: combined score additivity.
	const eps = 1e-9
	for i, p := range pairs {
		want := p.Counselor.Score.Total + p.Camper.Score.Total
		if diff := p.TotalScore - want; diff > eps || diff < -eps {
			t.Errorf("pair %d TotalScore %v != counselor %v + camper %v",
				i, p.TotalScore, p.Counselor.Score.Total, p.Camper.Score.Total)
		}
	}

	// Sanity: pairs sorted by combined score desc.
	for i := 1; i < len(pairs); i++ {
		if pairs[i].TotalScore > pairs[i-1].TotalScore+eps {
			t.Errorf("pairs not sorted at %d: %v > %v", i, pairs[i].TotalScore, pairs[i-1].TotalScore)
		}
	}
}
