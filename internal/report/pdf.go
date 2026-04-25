package report

import (
	"embed"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// PDFOptions controls optional sections of the PDF output.
type PDFOptions struct {
	IncludeUnassigned bool
}

const (
	pdfMarginLR   = 12.7 // 0.5 inch in mm
	pdfMarginTop  = 12.7
	pdfMarginBot  = 14.0
	pdfHeaderH    = 18.0 // reserved space at top of body for the page header
	pdfBodyOffset = pdfMarginTop + pdfHeaderH

	pdfLineH = 5.0 // standard line height for body text rows

	pdfFontFamily = "DejaVu"
)

// fontFS bundles the DejaVu Sans family used for all PDF rendering. We
// embed the TTFs so the binary is self-contained and we get full Unicode
// support (Latin Extended, Cyrillic, Vietnamese, etc.) — names from real
// camp rosters often contain characters outside cp1252.
//
//go:embed fonts/DejaVuSans.ttf fonts/DejaVuSans-Bold.ttf fonts/DejaVuSans-Oblique.ttf fonts/DejaVuSans-BoldOblique.ttf
var fontFS embed.FS

// cellKey indexes the activity grid by activity name + time slot name.
type cellKey struct{ act, slot string }

// docCtx wraps the pdf handle. With AddUTF8FontFromBytes the library
// accepts UTF-8 strings natively, so no translator wrapping is needed.
type docCtx struct {
	pdf *fpdf.Fpdf
}

// WriteCabinPDF renders a cabin report as a US Letter portrait PDF.
func WriteCabinPDF(w io.Writer, r *CabinReport, opts PDFOptions) error {
	doc := newPDF("P", r.Camp.Name, r.Session.Name, formatGeneratedAt(r.GeneratedAt))
	doc.pdf.AddPage()
	writeTitle(doc, "Cabin Assignments")

	// Iterate sorted cabins (already age-group then cabin alpha) and emit a
	// large heading whenever the age group changes.
	currentAgeGroup := ""
	for i := range r.Cabins {
		g := r.Cabins[i]
		if g.AgeGroupName != currentAgeGroup {
			if currentAgeGroup != "" {
				doc.pdf.Ln(3)
			}
			ensureSpace(doc.pdf, 22)
			writeAgeGroupHeading(doc, g.AgeGroupName)
			currentAgeGroup = g.AgeGroupName
		}
		renderCabinTable(doc, currentAgeGroup, g)
	}

	if opts.IncludeUnassigned {
		hasUnassigned := len(r.Unassigned.Counselors) > 0 || len(r.Unassigned.Campers) > 0
		if hasUnassigned {
			doc.pdf.Ln(3)
			ensureSpace(doc.pdf, 22)
			writeAgeGroupHeading(doc, "Unassigned")
			if len(r.Unassigned.Counselors) > 0 {
				writeSubHeading(doc, "Counselors")
				for _, c := range r.Unassigned.Counselors {
					writeListItem(doc, displayName(c.FirstName, c.LastName, c.Name))
				}
			}
			if len(r.Unassigned.Campers) > 0 {
				writeSubHeading(doc, "Campers")
				for _, c := range r.Unassigned.Campers {
					writeListItem(doc, displayName(c.FirstName, c.LastName, c.Name))
				}
			}
		}
	}

	if doc.pdf.Err() {
		return fmt.Errorf("error rendering cabin pdf: %w", doc.pdf.Error())
	}
	return doc.pdf.Output(w)
}

// renderCabinTable draws a single cabin as a two-column (Counselors |
// Campers) bordered table that spans the body width. Page breaks within a
// cabin re-emit the age-group label, cabin name, and column header so the
// reader never loses context.
func renderCabinTable(doc *docCtx, ageGroup string, g CabinGroup) {
	pdf := doc.pdf
	pageW, _ := pdf.GetPageSize()
	bodyW := pageW - 2*pdfMarginLR
	colW := bodyW / 2

	ensureSpace(pdf, 20)
	drawCabinTableHeader(doc, ageGroup, g.CabinName, colW, false)

	if len(g.Counselors) == 0 && len(g.Campers) == 0 {
		pdf.SetFont(pdfFontFamily, "I", 10)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(bodyW, 7, "No one assigned", "1", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		return
	}

	pdf.SetFont(pdfFontFamily, "", 10)
	rows := len(g.Counselors)
	if len(g.Campers) > rows {
		rows = len(g.Campers)
	}
	for i := 0; i < rows; i++ {
		var leftName, rightName string
		if i < len(g.Counselors) {
			leftName = displayName(g.Counselors[i].FirstName, g.Counselors[i].LastName, g.Counselors[i].Name)
		}
		if i < len(g.Campers) {
			rightName = displayName(g.Campers[i].FirstName, g.Campers[i].LastName, g.Campers[i].Name)
		}
		drawCabinRow(doc, ageGroup, g.CabinName, colW, leftName, rightName)
	}
}

// drawCabinTableHeader emits the cabin's title block: an age-group
// reminder (compact on continuation pages), the cabin name (with
// "(continued)" suffix when continuing), and the COUNSELORS|CAMPERS
// column header row.
func drawCabinTableHeader(doc *docCtx, ageGroup, cabinName string, colW float64, continuation bool) {
	pdf := doc.pdf

	if continuation {
		pdf.SetFont(pdfFontFamily, "B", 11)
		pdf.SetTextColor(80, 80, 80)
		pdf.CellFormat(0, 5, ageGroup, "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}

	pdf.SetFont(pdfFontFamily, "B", 12)
	pdf.SetTextColor(0, 0, 0)
	title := cabinName
	if continuation {
		title = cabinName + " (continued)"
	}
	pdf.CellFormat(0, 6, title, "", 1, "L", false, 0, "")
	pdf.Ln(0.5)

	pdf.SetFont(pdfFontFamily, "B", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.SetDrawColor(180, 180, 180)
	pdf.SetTextColor(60, 60, 60)
	pdf.CellFormat(colW, 6, "COUNSELORS", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colW, 6, "CAMPERS", "1", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont(pdfFontFamily, "", 10)
}

// drawCabinRow renders a single Counselors|Campers row with matched
// heights so multi-line wrapped names don't desync the columns. If the
// row won't fit on the current page, it forces a break and re-emits the
// cabin's table header before drawing the row.
func drawCabinRow(doc *docCtx, ageGroup, cabinName string, colW float64, left, right string) {
	pdf := doc.pdf
	lineH := pdfLineH
	leftLines := wrapLines(pdf, left, colW-2)
	rightLines := wrapLines(pdf, right, colW-2)
	rows := len(leftLines)
	if len(rightLines) > rows {
		rows = len(rightLines)
	}
	if rows == 0 {
		rows = 1
	}
	rowH := float64(rows) * lineH

	_, pageH := pdf.GetPageSize()
	if pdf.GetY()+rowH > pageH-pdfMarginBot {
		pdf.AddPage()
		drawCabinTableHeader(doc, ageGroup, cabinName, colW, true)
	}

	x := pdf.GetX()
	y := pdf.GetY()

	pdf.Rect(x, y, colW, rowH, "D")
	pdf.Rect(x+colW, y, colW, rowH, "D")

	pdf.SetXY(x+1, y)
	for _, ln := range leftLines {
		pdf.CellFormat(colW-2, lineH, ln, "", 2, "L", false, 0, "")
	}
	pdf.SetXY(x+colW+1, y)
	for _, ln := range rightLines {
		pdf.CellFormat(colW-2, lineH, ln, "", 2, "L", false, 0, "")
	}

	pdf.SetXY(x, y+rowH)
}

// wrapLines splits text into lines that fit within maxW given the current
// font. Returns nil for empty input so callers can still produce a row.
func wrapLines(pdf *fpdf.Fpdf, text string, maxW float64) []string {
	if text == "" {
		return nil
	}
	lines := pdf.SplitLines([]byte(text), maxW)
	out := make([]string, 0, len(lines))
	for _, b := range lines {
		out = append(out, string(b))
	}
	return out
}

// WriteActivityPDF renders an activity report as a single landscape grid:
// activities (rows, alphabetical) by time slots (columns, sorted).
func WriteActivityPDF(w io.Writer, r *ActivityReport, opts PDFOptions) error {
	doc := newPDF("L", r.Camp.Name, r.Session.Name, formatGeneratedAt(r.GeneratedAt))
	doc.pdf.AddPage()
	writeTitle(doc, "Activity Assignments")

	// Pivot the report into a sparse map keyed by (activity, timeSlot).
	// Slot/activity sets are derived from r.TimeSlots (which the builder
	// seeds from the configured schedule), so configured-but-unstaffed
	// rows/columns still render with empty cells.
	cells := make(map[cellKey][]string)
	activitySet := make(map[string]struct{})
	slotOrder := []string{}

	for _, ts := range r.TimeSlots {
		slotOrder = append(slotOrder, ts.Name)
		for _, act := range ts.Activities {
			activitySet[act.Name] = struct{}{}
			if len(act.Counselors) == 0 {
				continue
			}
			key := cellKey{act: act.Name, slot: ts.Name}
			for _, c := range act.Counselors {
				cells[key] = append(cells[key], displayName(c.FirstName, c.LastName, c.Name))
			}
		}
	}

	activities := make([]string, 0, len(activitySet))
	for a := range activitySet {
		activities = append(activities, a)
	}
	sort.Strings(activities)

	if len(slotOrder) > 0 || len(activities) > 0 {
		renderActivityGrid(doc, activities, slotOrder, cells)
	} else {
		writeMuted(doc, "No activity schedule configured")
	}

	if opts.IncludeUnassigned && len(r.Unassigned) > 0 {
		doc.pdf.Ln(4)
		ensureSpace(doc.pdf, 22)
		writeAgeGroupHeading(doc, "Unassigned")
		for _, u := range r.Unassigned {
			line := displayName(u.FirstName, u.LastName, u.Name)
			if len(u.MissingTimeSlots) > 0 {
				line = fmt.Sprintf("%s — missing: %s", line, strings.Join(u.MissingTimeSlots, ", "))
			}
			writeListItem(doc, line)
		}
	}

	if doc.pdf.Err() {
		return fmt.Errorf("error rendering activity pdf: %w", doc.pdf.Error())
	}
	return doc.pdf.Output(w)
}

// renderActivityGrid draws the activities-by-timeslots table.
func renderActivityGrid(doc *docCtx, activities, slots []string, cells map[cellKey][]string) {
	pdf := doc.pdf
	pageW, _ := pdf.GetPageSize()
	bodyW := pageW - 2*pdfMarginLR
	const actColW = 50.0
	slotColW := (bodyW - actColW) / float64(len(slots))

	drawHeader := func() {
		pdf.SetFont(pdfFontFamily, "B", 9)
		pdf.SetFillColor(240, 240, 240)
		pdf.SetDrawColor(180, 180, 180)
		pdf.SetTextColor(60, 60, 60)
		pdf.CellFormat(actColW, 7, "ACTIVITY", "1", 0, "L", true, 0, "")
		for _, s := range slots {
			pdf.CellFormat(slotColW, 7, strings.ToUpper(s), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetTextColor(0, 0, 0)
	}

	drawHeader()
	pdf.SetFont(pdfFontFamily, "", 9)
	lineH := pdfLineH

	for _, a := range activities {
		actLines := wrapLines(pdf, a, actColW-2)
		if len(actLines) == 0 {
			actLines = []string{a}
		}
		maxLines := len(actLines)
		cellTexts := make([][]string, len(slots))
		for i, s := range slots {
			names := cells[cellKey{act: a, slot: s}]
			lines := []string{}
			for _, n := range names {
				wrapped := wrapLines(pdf, n, slotColW-2)
				if len(wrapped) == 0 {
					wrapped = []string{n}
				}
				lines = append(lines, wrapped...)
			}
			cellTexts[i] = lines
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}
		rowH := float64(maxLines) * lineH

		_, pageH := pdf.GetPageSize()
		if pdf.GetY()+rowH > pageH-pdfMarginBot {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont(pdfFontFamily, "", 9)
		}

		x := pdf.GetX()
		y := pdf.GetY()

		pdf.Rect(x, y, actColW, rowH, "D")
		pdf.SetFont(pdfFontFamily, "B", 9)
		pdf.SetXY(x+1, y)
		for _, ln := range actLines {
			pdf.CellFormat(actColW-2, lineH, ln, "", 2, "L", false, 0, "")
		}
		pdf.SetFont(pdfFontFamily, "", 9)

		for i := range slots {
			cellX := x + actColW + float64(i)*slotColW
			pdf.Rect(cellX, y, slotColW, rowH, "D")
			pdf.SetXY(cellX+1, y)
			for _, ln := range cellTexts[i] {
				pdf.CellFormat(slotColW-2, lineH, ln, "", 2, "L", false, 0, "")
			}
		}

		pdf.SetXY(x, y+rowH)
	}
}

// newPDF wires up document defaults, registers the embedded DejaVu Sans
// font family, and configures the per-page header/footer.
func newPDF(orientation, campName, sessionName, generatedAt string) *docCtx {
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: orientation,
		UnitStr:        "mm",
		SizeStr:        "Letter",
	})
	pdf.SetMargins(pdfMarginLR, pdfMarginTop, pdfMarginLR)
	pdf.SetAutoPageBreak(true, pdfMarginBot)

	for _, f := range []struct {
		style string
		path  string
	}{
		{"", "fonts/DejaVuSans.ttf"},
		{"B", "fonts/DejaVuSans-Bold.ttf"},
		{"I", "fonts/DejaVuSans-Oblique.ttf"},
		{"BI", "fonts/DejaVuSans-BoldOblique.ttf"},
	} {
		data, err := fontFS.ReadFile(f.path)
		if err != nil {
			pdf.SetError(fmt.Errorf("error reading embedded font %s: %w", f.path, err))
			continue
		}
		pdf.AddUTF8FontFromBytes(pdfFontFamily, f.style, data)
	}

	pdf.SetHeaderFunc(func() {
		pageW, _ := pdf.GetPageSize()
		pdf.SetY(pdfMarginTop)
		pdf.SetFont(pdfFontFamily, "B", 10)
		pdf.CellFormat(0, 5, campName, "", 1, "L", false, 0, "")
		pdf.SetFont(pdfFontFamily, "", 9)
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(0, 4, fmt.Sprintf("%s · Generated %s", sessionName, generatedAt), "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetDrawColor(200, 200, 200)
		y := pdf.GetY() + 1
		pdf.Line(pdfMarginLR, y, pageW-pdfMarginLR, y)
		pdf.SetDrawColor(0, 0, 0)
		pdf.SetY(pdfBodyOffset)
	})

	pdf.SetFooterFunc(func() {
		pdf.SetY(-pdfMarginBot + 4)
		pdf.SetFont(pdfFontFamily, "", 8)
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(0, 4, fmt.Sprintf("Page %d of {nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})
	pdf.AliasNbPages("{nb}")

	return &docCtx{pdf: pdf}
}

func writeTitle(doc *docCtx, title string) {
	doc.pdf.SetFont(pdfFontFamily, "B", 16)
	doc.pdf.CellFormat(0, 8, title, "", 1, "L", false, 0, "")
	doc.pdf.Ln(2)
}

// writeAgeGroupHeading is the large per-section heading used for cabin
// age-group breaks and the unassigned section.
func writeAgeGroupHeading(doc *docCtx, text string) {
	pdf := doc.pdf
	pageW, _ := pdf.GetPageSize()
	pdf.SetFont(pdfFontFamily, "B", 16)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 8, text, "", 1, "L", false, 0, "")
	pdf.SetDrawColor(120, 120, 120)
	y := pdf.GetY()
	pdf.Line(pdfMarginLR, y, pageW-pdfMarginLR, y)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Ln(3)
}

func writeSubHeading(doc *docCtx, text string) {
	doc.pdf.SetFont(pdfFontFamily, "B", 10)
	doc.pdf.SetTextColor(80, 80, 80)
	doc.pdf.CellFormat(0, 5, strings.ToUpper(text), "", 1, "L", false, 0, "")
	doc.pdf.SetTextColor(0, 0, 0)
}

func writeListItem(doc *docCtx, text string) {
	doc.pdf.SetFont(pdfFontFamily, "", 10)
	doc.pdf.CellFormat(4, 5, "", "", 0, "L", false, 0, "")
	doc.pdf.CellFormat(0, 5, text, "", 1, "L", false, 0, "")
}

func writeMuted(doc *docCtx, text string) {
	doc.pdf.SetFont(pdfFontFamily, "I", 10)
	doc.pdf.SetTextColor(120, 120, 120)
	doc.pdf.CellFormat(4, 5, "", "", 0, "L", false, 0, "")
	doc.pdf.CellFormat(0, 5, text, "", 1, "L", false, 0, "")
	doc.pdf.SetTextColor(0, 0, 0)
}

// ensureSpace forces a page break when the next block (height mm) would
// run into the bottom margin. Avoids orphaned section headings at the
// very bottom of a page.
func ensureSpace(pdf *fpdf.Fpdf, blockHeight float64) {
	_, pageH := pdf.GetPageSize()
	if pdf.GetY()+blockHeight > pageH-pdfMarginBot {
		pdf.AddPage()
	}
}

func formatGeneratedAt(t time.Time) string {
	return t.Format("Jan 2, 2006")
}
