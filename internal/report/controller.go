package report

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	svc *Service
}

func NewController(svc *Service) *Controller {
	return &Controller{svc: svc}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	reports := rg.Group("/sessions/:sessionId/reports")
	reports.GET("/cabin", ctrl.GetCabin)
	reports.GET("/activities", ctrl.GetActivities)
}

type exportParams struct {
	format            string
	includeUnassigned bool
}

// parseExportParams reads the format + include_unassigned query params
// shared by both report endpoints. Format is required; include_unassigned
// defaults to true to preserve backwards-compatible behaviour.
func parseExportParams(c *gin.Context) (exportParams, error) {
	format := c.Query("format")
	if format != "csv" && format != "pdf" {
		return exportParams{}, fmt.Errorf("format must be 'csv' or 'pdf'")
	}
	include := true
	if v := c.Query("include_unassigned"); v != "" {
		switch v {
		case "true", "1":
			include = true
		case "false", "0":
			include = false
		default:
			return exportParams{}, fmt.Errorf("include_unassigned must be 'true', 'false', '1', or '0'")
		}
	}
	return exportParams{format: format, includeUnassigned: include}, nil
}

func (ctrl *Controller) GetCabin(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	params, err := parseExportParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := ctrl.svc.GetCabinReport(c.Request.Context(), campID, sessionID)
	if err != nil {
		ctrl.handleError(c, log, sessionID, "cabin", err)
		return
	}

	slug := FilenameSlug(report.Session.Name)
	date := time.Now().UTC().Format("2006-01-02")

	// Render to a buffer first so a mid-render failure surfaces as a clean
	// JSON error rather than a truncated download with a 200 status.
	var buf bytes.Buffer
	var contentType, filename string
	switch params.format {
	case "csv":
		filename = fmt.Sprintf("cabin-report-%s-%s.csv", slug, date)
		contentType = "text/csv; charset=utf-8"
		err = WriteCabinCSV(&buf, report, CSVOptions{IncludeUnassigned: params.includeUnassigned})
	case "pdf":
		filename = fmt.Sprintf("cabin-report-%s-%s.pdf", slug, date)
		contentType = "application/pdf"
		err = WriteCabinPDF(&buf, report, PDFOptions{IncludeUnassigned: params.includeUnassigned})
	}
	if err != nil {
		ctrl.handleError(c, log, sessionID, "cabin", err)
		return
	}

	writeDownload(c, log, contentType, filename, buf.Bytes())
}

func (ctrl *Controller) GetActivities(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	params, err := parseExportParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := ctrl.svc.GetActivityReport(c.Request.Context(), campID, sessionID)
	if err != nil {
		ctrl.handleError(c, log, sessionID, "activity", err)
		return
	}

	slug := FilenameSlug(report.Session.Name)
	date := time.Now().UTC().Format("2006-01-02")

	var buf bytes.Buffer
	var contentType, filename string
	switch params.format {
	case "csv":
		filename = fmt.Sprintf("activity-report-%s-%s.csv", slug, date)
		contentType = "text/csv; charset=utf-8"
		err = WriteActivityCSV(&buf, report, CSVOptions{IncludeUnassigned: params.includeUnassigned})
	case "pdf":
		filename = fmt.Sprintf("activity-report-%s-%s.pdf", slug, date)
		contentType = "application/pdf"
		err = WriteActivityPDF(&buf, report, PDFOptions{IncludeUnassigned: params.includeUnassigned})
	}
	if err != nil {
		ctrl.handleError(c, log, sessionID, "activity", err)
		return
	}

	writeDownload(c, log, contentType, filename, buf.Bytes())
}

// writeDownload sets standard download headers and streams a fully-formed
// payload. Callers must only invoke this once they are certain rendering
// succeeded.
func writeDownload(c *gin.Context, log *slog.Logger, contentType, filename string, body []byte) {
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Length", strconv.Itoa(len(body)))
	if _, err := c.Writer.Write(body); err != nil {
		log.With("error", err).Error("error streaming report response")
	}
}

func (ctrl *Controller) handleError(c *gin.Context, log *slog.Logger, sessionID, kind string, err error) {
	switch {
	case errors.Is(err, ErrNoSelectedSolution):
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("no selected %s assignment for this session", kind),
		})
		return
	case errors.Is(err, ErrSessionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	case errors.Is(err, ErrCampNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
		return
	case api.IsBadInput(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.
		With("session_id", sessionID).
		With("report", kind).
		With("error", err).
		Error("error generating report")
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
