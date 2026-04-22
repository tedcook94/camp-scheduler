package solver

import (
	"fmt"
	"sort"
	"strings"
)

func ExplainActivity(snapshot ActivitySnapshot, solution ActivitySolution) ActivityExplanation {
	slotsByID := indexActivitySlots(snapshot)

	reasonsByAssignment := buildActivityReasonMap(solution.Score.Breakdown)

	// Build counselor -> list of slot IDs they're assigned to.
	counselorSlots := make(map[string][]string)
	for slotID, counselorIDs := range solution.Assignment.SlotCounselors {
		for _, cID := range counselorIDs {
			counselorSlots[cID] = append(counselorSlots[cID], slotID)
		}
	}

	var assignments []ActivityAssignmentExplanation
	for slotID, counselorIDs := range solution.Assignment.SlotCounselors {
		for _, cID := range counselorIDs {
			key := cID + "|" + slotID
			reasons := reasonsByAssignment[key]
			if len(reasons) == 0 {
				continue
			}

			assignments = append(assignments, ActivityAssignmentExplanation{
				CounselorID: cID,
				SlotID:      slotID,
				Reasons:     reasons,
			})
		}
	}

	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].SlotID != assignments[j].SlotID {
			return assignments[i].SlotID < assignments[j].SlotID
		}
		return assignments[i].CounselorID < assignments[j].CounselorID
	})

	var unmet []ActivityUnmetPreference
	eligibleUnmet, ineligible := findUnmetActivityPreferences(snapshot, counselorSlots, slotsByID)
	unmet = append(unmet, eligibleUnmet...)
	unmet = append(unmet, findUnassignedCounselors(snapshot, counselorSlots)...)

	sort.Slice(unmet, func(i, j int) bool {
		if unmet[i].CounselorID != unmet[j].CounselorID {
			return unmet[i].CounselorID < unmet[j].CounselorID
		}
		if unmet[i].Constraint != unmet[j].Constraint {
			return unmet[i].Constraint < unmet[j].Constraint
		}
		return unmet[i].Message < unmet[j].Message
	})

	sort.Slice(ineligible, func(i, j int) bool {
		if ineligible[i].CounselorID != ineligible[j].CounselorID {
			return ineligible[i].CounselorID < ineligible[j].CounselorID
		}
		return ineligible[i].Message < ineligible[j].Message
	})

	return ActivityExplanation{
		Assignments:           assignments,
		UnmetPreferences:      unmet,
		IneligiblePreferences: ineligible,
	}
}

// buildActivityReasonMap keys reasons by "counselorID|slotID" so that
// explanations are accurate per assignment, not just per counselor.
func buildActivityReasonMap(breakdown []ScoreComponent) map[string][]AssignmentReason {
	reasons := make(map[string][]AssignmentReason)
	for _, c := range breakdown {
		if c.CounselorID == "" {
			continue
		}
		key := c.CounselorID + "|" + c.SlotID
		reasons[key] = append(reasons[key], AssignmentReason{
			Constraint: c.Constraint,
			Rank:       c.Rank,
			Message:    fmt.Sprintf("%s (+%.1f)", c.Message, c.Score),
		})
	}
	return reasons
}

// findUnmetActivityPreferences partitions a counselor's unmet preferences
// into two buckets: those they were eligible for but not assigned to, and
// those they could never be assigned to because they lack a required
// certification. The two buckets are aggregated independently: when a
// bucket contains every preference, it produces a single combined message;
// otherwise it produces one message per preference.
func findUnmetActivityPreferences(snapshot ActivitySnapshot, counselorSlots map[string][]string, slotsByID map[string]ActivitySlot) ([]ActivityUnmetPreference, []ActivityUnmetPreference) {
	activityNames := make(map[string]string)
	for _, slot := range snapshot.Slots {
		activityNames[slot.ActivityID] = slot.ActivityName
	}

	// Build counselor -> set of activity IDs they're assigned to.
	counselorActivities := make(map[string]map[string]bool)
	for cID, slotIDs := range counselorSlots {
		for _, slotID := range slotIDs {
			slot := slotsByID[slotID]
			if counselorActivities[cID] == nil {
				counselorActivities[cID] = make(map[string]bool)
			}
			counselorActivities[cID][slot.ActivityID] = true
		}
	}

	counselorsByID := indexActivityCounselors(snapshot)

	// For each (counselor, activity) compute whether the counselor could
	// be assigned to ANY slot of that activity in the session, and which
	// certification IDs they would be missing if not.
	type ineligibilityInfo struct {
		eligible       bool
		noSlots        bool
		missingCertIDs []string
	}
	checkEligibility := func(c ActivityCounselor, activityID string) ineligibilityInfo {
		// Iterate every slot for this activity. The counselor is eligible
		// if any slot's required certs are all satisfied. When ineligible,
		// report the cert gap from the slot with the fewest missing certs
		// (the "easiest" gap to fix). If the activity has no slots in
		// this session at all, flag it separately so the message reflects
		// that rather than producing a blank cert list.
		var bestMissing []string
		bestMissingFound := false
		sawSlot := false
		for _, slot := range snapshot.Slots {
			if slot.ActivityID != activityID {
				continue
			}
			sawSlot = true
			missing := []string{}
			for _, certID := range slot.RequiredCertifications {
				if !c.Certifications[certID] {
					missing = append(missing, certID)
				}
			}
			if len(missing) == 0 {
				return ineligibilityInfo{eligible: true}
			}
			if !bestMissingFound || len(missing) < len(bestMissing) {
				bestMissing = missing
				bestMissingFound = true
			}
		}
		if !sawSlot {
			return ineligibilityInfo{eligible: false, noSlots: true}
		}
		return ineligibilityInfo{eligible: false, missingCertIDs: bestMissing}
	}

	activityLabel := func(id string) string {
		if name, ok := activityNames[id]; ok && name != "" {
			return name
		}
		return id
	}

	certNameLabel := func(id string) string {
		if snapshot.CertificationNames != nil {
			if name, ok := snapshot.CertificationNames[id]; ok && name != "" {
				return name
			}
		}
		return id
	}

	missingCertList := func(ids []string) string {
		names := make([]string, len(ids))
		for i, id := range ids {
			names[i] = certNameLabel(id)
		}
		return strings.Join(names, ", ")
	}

	missingCertPhrase := func(ids []string) string {
		if len(ids) == 1 {
			return fmt.Sprintf("missing required certification %s", certNameLabel(ids[0]))
		}
		return fmt.Sprintf("missing required certifications %s", missingCertList(ids))
	}

	missingCertShort := func(ids []string) string {
		return fmt.Sprintf("missing %s", missingCertList(ids))
	}

	var eligibleUnmet, ineligible []ActivityUnmetPreference
	for counselorID, prefs := range snapshot.ActivityPreferences {
		assigned := counselorActivities[counselorID]

		// Determine the best (lowest numeric) rank the counselor actually
		// got among their preferences. Preferences at or above that rank
		// are considered met; only strictly higher-ranked (lower numeric
		// rank) preferences count as real misses.
		bestMetRank := 0 // 0 = none met
		for _, pref := range prefs {
			if pref.Rank <= 0 {
				continue
			}
			if assigned[pref.TargetID] && (bestMetRank == 0 || pref.Rank < bestMetRank) {
				bestMetRank = pref.Rank
			}
		}

		counselor, hasCounselor := counselorsByID[counselorID]

		type prefWithMiss struct {
			pref    RankedPreference
			missing []string
			noSlots bool
		}
		var eligibleBucket []RankedPreference
		var ineligibleBucket []prefWithMiss

		for _, pref := range prefs {
			if pref.Rank <= 0 {
				continue
			}
			// A preference counts as unmet only if no equal-or-higher-rank
			// preference was met.
			if bestMetRank != 0 && pref.Rank >= bestMetRank {
				continue
			}
			if assigned[pref.TargetID] {
				continue
			}
			if !hasCounselor {
				eligibleBucket = append(eligibleBucket, pref)
				continue
			}
			info := checkEligibility(counselor, pref.TargetID)
			if info.eligible {
				eligibleBucket = append(eligibleBucket, pref)
			} else {
				ineligibleBucket = append(ineligibleBucket, prefWithMiss{pref: pref, missing: info.missingCertIDs, noSlots: info.noSlots})
			}
		}

		if len(eligibleBucket) == 0 && len(ineligibleBucket) == 0 {
			continue
		}

		// Count the preferences that were *candidates* for being unmet
		// (rank strictly better than any met preference). Aggregation
		// collapses to a single row only when a bucket covers all of them.
		candidateCount := 0
		for _, pref := range prefs {
			if pref.Rank <= 0 {
				continue
			}
			if bestMetRank != 0 && pref.Rank >= bestMetRank {
				continue
			}
			candidateCount++
		}

		sort.Slice(eligibleBucket, func(i, j int) bool {
			return eligibleBucket[i].Rank < eligibleBucket[j].Rank
		})
		sort.Slice(ineligibleBucket, func(i, j int) bool {
			return ineligibleBucket[i].pref.Rank < ineligibleBucket[j].pref.Rank
		})

		// Eligible-unmet aggregation: single row when this bucket covers
		// ALL candidate preferences. Otherwise individual rows.
		aggregateEligible := len(eligibleBucket) > 0 && len(eligibleBucket) == candidateCount
		if aggregateEligible {
			parts := make([]string, len(eligibleBucket))
			for i, p := range eligibleBucket {
				parts[i] = activityLabel(p.TargetID)
			}
			eligibleUnmet = append(eligibleUnmet, ActivityUnmetPreference{
				CounselorID: counselorID,
				Constraint:  "activity_preference",
				Message: fmt.Sprintf(
					"not assigned to any preferred activity (%s)",
					strings.Join(parts, ", "),
				),
			})
		} else {
			for _, pref := range eligibleBucket {
				eligibleUnmet = append(eligibleUnmet, ActivityUnmetPreference{
					CounselorID: counselorID,
					Constraint:  "activity_preference",
					Rank:        pref.Rank,
					Message: fmt.Sprintf(
						"not assigned to preferred activity %s (rank %d)",
						activityLabel(pref.TargetID), pref.Rank,
					),
				})
			}
		}

		// Ineligible aggregation mirrors the same rule.
		aggregateIneligible := len(ineligibleBucket) > 0 && len(ineligibleBucket) == candidateCount
		if aggregateIneligible {
			parts := make([]string, len(ineligibleBucket))
			for i, pm := range ineligibleBucket {
				if pm.noSlots {
					parts[i] = fmt.Sprintf("%s — not scheduled", activityLabel(pm.pref.TargetID))
				} else {
					parts[i] = fmt.Sprintf("%s — %s",
						activityLabel(pm.pref.TargetID), missingCertShort(pm.missing))
				}
			}
			ineligible = append(ineligible, ActivityUnmetPreference{
				CounselorID: counselorID,
				Constraint:  "activity_preference_ineligible",
				Message: fmt.Sprintf(
					"cannot be assigned to any preferred activity (%s)",
					strings.Join(parts, ", "),
				),
			})
		} else {
			for _, pm := range ineligibleBucket {
				var msg string
				if pm.noSlots {
					msg = fmt.Sprintf(
						"cannot be assigned to preferred activity %s (rank %d) — no slots scheduled this session",
						activityLabel(pm.pref.TargetID), pm.pref.Rank,
					)
				} else {
					msg = fmt.Sprintf(
						"cannot be assigned to preferred activity %s (rank %d) — %s",
						activityLabel(pm.pref.TargetID), pm.pref.Rank, missingCertPhrase(pm.missing),
					)
				}
				ineligible = append(ineligible, ActivityUnmetPreference{
					CounselorID: counselorID,
					Constraint:  "activity_preference_ineligible",
					Rank:        pm.pref.Rank,
					Message:     msg,
				})
			}
		}
	}
	return eligibleUnmet, ineligible
}

func findUnassignedCounselors(snapshot ActivitySnapshot, counselorSlots map[string][]string) []ActivityUnmetPreference {
	var unmet []ActivityUnmetPreference
	for _, c := range snapshot.Counselors {
		if len(counselorSlots[c.ID]) == 0 {
			unmet = append(unmet, ActivityUnmetPreference{
				CounselorID: c.ID,
				Constraint:  "unassigned",
				Message: fmt.Sprintf(
					"counselor %q not assigned to any activity",
					c.Name,
				),
			})
		}
	}
	return unmet
}
