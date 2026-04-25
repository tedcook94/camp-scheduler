package report

import (
	"encoding/csv"
	"io"
	"strings"
)

// CSVOptions controls optional sections of the CSV output.
type CSVOptions struct {
	IncludeUnassigned bool
}

// WriteCabinCSV emits a flat one-row-per-person CSV. Each cabin's
// counselors and campers are emitted in their group order; an unassigned
// section follows after a blank separator row when included. Names are
// split into Last Name and First Name columns so spreadsheets can sort
// or filter on either part.
func WriteCabinCSV(w io.Writer, r *CabinReport, opts CSVOptions) error {
	cw := csv.NewWriter(w)
	// defer guards early-error returns; explicit Flush+Error below covers success path.
	defer cw.Flush()

	header := []string{"Age Group", "Cabin", "Role", "Last Name", "First Name"}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, g := range r.Cabins {
		for _, c := range g.Counselors {
			last, first := splitForCSV(c.FirstName, c.LastName, c.Name)
			if err := cw.Write([]string{g.AgeGroupName, g.CabinName, "Counselor", last, first}); err != nil {
				return err
			}
		}
		for _, c := range g.Campers {
			last, first := splitForCSV(c.FirstName, c.LastName, c.Name)
			if err := cw.Write([]string{g.AgeGroupName, g.CabinName, "Camper", last, first}); err != nil {
				return err
			}
		}
	}

	if opts.IncludeUnassigned {
		hasUnassigned := len(r.Unassigned.Counselors) > 0 || len(r.Unassigned.Campers) > 0
		if hasUnassigned {
			if err := cw.Write([]string{}); err != nil {
				return err
			}
			for _, c := range r.Unassigned.Counselors {
				last, first := splitForCSV(c.FirstName, c.LastName, c.Name)
				if err := cw.Write([]string{"", "", "Unassigned Counselor", last, first}); err != nil {
					return err
				}
			}
			for _, c := range r.Unassigned.Campers {
				last, first := splitForCSV(c.FirstName, c.LastName, c.Name)
				if err := cw.Write([]string{"", "", "Unassigned Camper", last, first}); err != nil {
					return err
				}
			}
		}
	}

	cw.Flush()
	return cw.Error()
}

// WriteActivityCSV emits a flat one-row-per-counselor-placement CSV. An
// unassigned section follows; for partially-unassigned counselors each
// missed time slot becomes its own row so the data is filterable by slot.
// A final "Empty Activities" section lists configured (slot, activity)
// pairs that received no counselors so directors can see coverage gaps at
// a glance — always emitted, regardless of IncludeUnassigned.
func WriteActivityCSV(w io.Writer, r *ActivityReport, opts CSVOptions) error {
	cw := csv.NewWriter(w)
	// defer guards early-error returns; explicit Flush+Error below covers success path.
	defer cw.Flush()

	header := []string{"Time Slot", "Activity", "Last Name", "First Name"}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, ts := range r.TimeSlots {
		for _, act := range ts.Activities {
			for _, c := range act.Counselors {
				last, first := splitForCSV(c.FirstName, c.LastName, c.Name)
				if err := cw.Write([]string{ts.Name, act.Name, last, first}); err != nil {
					return err
				}
			}
		}
	}

	if opts.IncludeUnassigned && len(r.Unassigned) > 0 {
		if err := cw.Write([]string{}); err != nil {
			return err
		}
		for _, u := range r.Unassigned {
			last, first := splitForCSV(u.FirstName, u.LastName, u.Name)
			if len(u.MissingTimeSlots) == 0 {
				if err := cw.Write([]string{"", "unassigned", last, first}); err != nil {
					return err
				}
				continue
			}
			for _, ts := range u.MissingTimeSlots {
				if err := cw.Write([]string{ts, "unassigned", last, first}); err != nil {
					return err
				}
			}
		}
	}

	// Empty activities are always emitted: this is a configuration-coverage
	// signal, not an assignment-noise toggle.
	type emptyPair struct{ slot, activity string }
	var empties []emptyPair
	for _, ts := range r.TimeSlots {
		for _, act := range ts.Activities {
			if len(act.Counselors) == 0 {
				empties = append(empties, emptyPair{slot: ts.Name, activity: act.Name})
			}
		}
	}
	if len(empties) > 0 {
		if err := cw.Write([]string{}); err != nil {
			return err
		}
		if err := cw.Write([]string{"Empty Activities", "", "", ""}); err != nil {
			return err
		}
		for _, e := range empties {
			if err := cw.Write([]string{e.slot, e.activity, "", ""}); err != nil {
				return err
			}
		}
	}

	cw.Flush()
	return cw.Error()
}

// FilenameSlug normalises a name for use in a Content-Disposition filename.
// Lowercases, swaps non-alphanumeric runs for "-", trims leading/trailing
// hyphens, and falls back to "report" when empty.
func FilenameSlug(name string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevHyphen = false
		default:
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return "report"
	}
	return out
}
