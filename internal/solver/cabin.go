package solver

import (
	"context"
	"fmt"
	"sort"

	"camp-scheduler/internal/db"
)

// CabinSnapshot bundles the counselor and camper data needed to assign both
// populations to cabins simultaneously, with a shared total capacity per
// cabin (counselors + campers).
type CabinSnapshot struct {
	SessionID string
	Counselor SessionSnapshot
	Camper    CamperCabinSnapshot
}

// CabinSolverConfig configures the combined cabin solver. The counselor and
// camper sub-solvers reuse their existing weights.
type CabinSolverConfig struct {
	MaxSolutions    int
	MaxIterations   int
	CounselorConfig SolverConfig
	CamperConfig    CamperSolverConfig
}

func DefaultCabinSolverConfig() CabinSolverConfig {
	return CabinSolverConfig{
		MaxSolutions:    5,
		MaxIterations:   100_000,
		CounselorConfig: DefaultSolverConfig(),
		CamperConfig:    DefaultCamperSolverConfig(),
	}
}

// CabinSolution pairs a counselor placement with a camper placement that
// shares the same cabins. The pair is ranked by the combined score.
type CabinSolution struct {
	Counselor  Solution
	Camper     CamperSolution
	TotalScore float64
}

func BuildCabinSnapshot(ctx context.Context, queries *db.Queries, campID, sessionID string) (CabinSnapshot, error) {
	counselorSnap, err := BuildSnapshot(ctx, queries, campID, sessionID)
	if err != nil {
		return CabinSnapshot{}, fmt.Errorf("error building counselor snapshot: %w", err)
	}

	camperSnap, err := BuildCamperCabinSnapshot(ctx, queries, campID, sessionID)
	if err != nil {
		return CabinSnapshot{}, fmt.Errorf("error building camper snapshot: %w", err)
	}

	return CabinSnapshot{
		SessionID: sessionID,
		Counselor: counselorSnap,
		Camper:    camperSnap,
	}, nil
}

// SolveCabin assigns counselors and campers to cabins together, enforcing
// the shared per-cabin capacity (counselors + campers <= cabin.Capacity).
//
// Strategy: run the counselor solver first to produce a pool of counselor
// placements, then for each one derive a camper snapshot whose per-cabin
// capacity is reduced by the counselor occupancy and run the camper solver
// on it. Each (counselor, camper) pair is scored as the sum of the two
// sub-scores. Pairs are then sorted by combined score and trimmed to
// MaxSolutions.
//
// To avoid the cliff where a counselor solution that ranks just below
// MaxSolutions on its own would have produced the best combined score,
// the counselor solver is over-sampled (4x the requested output, minimum
// 20) and the camper solver returns multiple alternatives per counselor
// solution. All combinations are scored before trimming.
func SolveCabin(snapshot CabinSnapshot, config CabinSolverConfig) []CabinSolution {
	if config.MaxSolutions <= 0 {
		return nil
	}

	counselorCfg := config.CounselorConfig
	counselorSampleTarget := config.MaxSolutions * 4
	if counselorSampleTarget < 20 {
		counselorSampleTarget = 20
	}
	if counselorCfg.MaxSolutions <= 0 || counselorCfg.MaxSolutions < counselorSampleTarget {
		counselorCfg.MaxSolutions = counselorSampleTarget
	}
	if counselorCfg.MaxIterations <= 0 {
		counselorCfg.MaxIterations = config.MaxIterations
	}
	camperCfg := config.CamperConfig
	camperPerCounselor := config.MaxSolutions
	if camperPerCounselor < 3 {
		camperPerCounselor = 3
	}
	if camperCfg.MaxSolutions <= 0 || camperCfg.MaxSolutions < camperPerCounselor {
		camperCfg.MaxSolutions = camperPerCounselor
	}
	if camperCfg.MaxIterations <= 0 {
		camperCfg.MaxIterations = config.MaxIterations
	}

	counselorSolutions := Solve(snapshot.Counselor, counselorCfg)
	if len(counselorSolutions) == 0 {
		return nil
	}

	return pairAndRankCabinSolutions(counselorSolutions, snapshot.Camper, camperCfg, camperPerCounselor, config.MaxSolutions)
}

// pairAndRankCabinSolutions runs the camper sub-solver against each counselor
// candidate (after reducing per-cabin capacity for the counselors already
// placed), enumerates every (counselor, camper) combination, then sorts and
// trims to maxSolutions. Extracted from SolveCabin so the joint-scoring
// behaviour can be exercised with hand-crafted counselor solutions in tests.
func pairAndRankCabinSolutions(
	counselorSolutions []Solution,
	camperSnapshot CamperCabinSnapshot,
	camperCfg CamperSolverConfig,
	camperPerCounselor int,
	maxSolutions int,
) []CabinSolution {
	pairs := make([]CabinSolution, 0, len(counselorSolutions)*camperPerCounselor)
	for _, cSol := range counselorSolutions {
		camperSnap := reduceCapacityForCounselors(camperSnapshot, cSol.Assignment)
		camperSolutions := SolveCamperCabin(camperSnap, camperCfg)
		if len(camperSolutions) == 0 {
			continue
		}
		for _, mSol := range camperSolutions {
			pairs = append(pairs, CabinSolution{
				Counselor:  cSol,
				Camper:     mSol,
				TotalScore: cSol.Score.Total + mSol.Score.Total,
			})
		}
	}

	if len(pairs) == 0 {
		return nil
	}

	sortPairsByScoreDesc(pairs)
	if len(pairs) > maxSolutions {
		pairs = pairs[:maxSolutions]
	}
	return pairs
}

// reduceCapacityForCounselors returns a copy of the camper snapshot whose
// cabin capacities have been reduced by the number of counselors placed in
// each cabin by the counselor solution.
func reduceCapacityForCounselors(snap CamperCabinSnapshot, counselorAssignment Assignment) CamperCabinSnapshot {
	out := snap
	out.Cabins = make([]CamperCabin, len(snap.Cabins))
	for i, c := range snap.Cabins {
		used := len(counselorAssignment.CabinCounselors[c.ID])
		remaining := c.Capacity - used
		if remaining < 0 {
			remaining = 0
		}
		c.Capacity = remaining
		out.Cabins[i] = c
	}
	return out
}

func sortPairsByScoreDesc(pairs []CabinSolution) {
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].TotalScore > pairs[j].TotalScore
	})
}
