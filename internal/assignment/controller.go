package assignment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"
	"camp-scheduler/internal/solver"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Controller struct {
	svc *Service
}

func NewController(svc *Service) *Controller {
	return &Controller{svc: svc}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	runs := rg.Group("/sessions/:sessionId/assignment-runs")
	runs.POST("", ctrl.TriggerRun)
	runs.GET("", ctrl.ListRuns)
	runs.GET("/:runId", ctrl.GetRun)
	runs.DELETE("/:runId", ctrl.DeleteRun)
	runs.GET("/:runId/solutions/:solutionId", ctrl.GetSolution)
	runs.POST("/:runId/solutions/:solutionId/select", ctrl.SelectSolution)

	byId := rg.Group("/assignment-runs")
	byId.GET("/:runId", ctrl.GetRunById)
}

type TriggerRunRequest struct {
	RunType         string                  `json:"run_type"`
	MaxSolutions    *int                    `json:"max_solutions"`
	MaxIterations   *int                    `json:"max_iterations"`
	Weights         *Weights                `json:"weights"`
	CamperWeights   *CamperWeightsRequest   `json:"camper_weights"`
	ActivityWeights *ActivityWeightsRequest `json:"activity_weights"`
}

type Weights struct {
	ReturningAgeGroup     *float64 `json:"returning_age_group"`
	ReturningCabin        *float64 `json:"returning_cabin"`
	CocounselorPreference *float64 `json:"cocounselor_preference"`
	AgeGroupPreference    *float64 `json:"age_group_preference"`
	MultipleSeniors       *float64 `json:"multiple_seniors"`
}

type CamperWeightsRequest struct {
	FriendPreference *float64 `json:"friend_preference"`
}

type ActivityWeightsRequest struct {
	ActivityPreference *float64 `json:"activity_preference"`
}

type RunResponse struct {
	ID                 string  `json:"id"`
	CampID             string  `json:"camp_id"`
	SessionID          string  `json:"session_id"`
	RunType            string  `json:"run_type"`
	Status             string  `json:"status"`
	SelectedSolutionID *string `json:"selected_solution_id"`
	CreatedAt          string  `json:"created_at"`
}

type RunDetailResponse struct {
	RunResponse
	Solutions []SolutionSummaryResponse `json:"solutions"`
}

type SolutionSummaryResponse struct {
	ID                   string          `json:"id"`
	CamperSolutionID     string          `json:"camper_solution_id,omitempty"`
	AssignmentRunID      string          `json:"assignment_run_id"`
	SolutionIndex        int             `json:"solution_index"`
	Score                float64         `json:"score"`
	ScoreBreakdown       json.RawMessage `json:"score_breakdown"`
	CamperScoreBreakdown json.RawMessage `json:"camper_score_breakdown,omitempty"`
}

type SolutionDetailResponse struct {
	SolutionSummaryResponse
	Assignments          []AssignmentResponse           `json:"assignments"`
	Explanations         []ExplanationResponse          `json:"explanations"`
	UnassignedCounselors []UnassignedCounselorResponse  `json:"unassigned_counselors"`
}

// UnassignedCounselorResponse identifies a counselor that the solver could
// not place. For cabin runs, MissingTimeSlots is empty (the counselor has
// no cabin assignment at all). For activity runs, MissingTimeSlots lists
// the time slots the counselor was not assigned to despite having at least
// one eligible activity slot in that time slot.
type UnassignedCounselorResponse struct {
	CounselorID      string                  `json:"counselor_id"`
	CounselorName    string                  `json:"counselor_name"`
	MissingTimeSlots []UnassignedTimeSlotRef `json:"missing_time_slots,omitempty"`
}

type UnassignedTimeSlotRef struct {
	SessionTimeSlotID string `json:"session_time_slot_id"`
	TimeSlotName      string `json:"time_slot_name"`
}

type AssignmentResponse struct {
	ID                string `json:"id"`
	CounselorID       string `json:"counselor_id,omitempty"`
	CounselorName     string `json:"counselor_name,omitempty"`
	CamperID          string `json:"camper_id,omitempty"`
	CamperName        string `json:"camper_name,omitempty"`
	CabinID           string `json:"cabin_id,omitempty"`
	CabinName         string `json:"cabin_name,omitempty"`
	AgeGroupName      string `json:"age_group_name,omitempty"`
	SessionActivityID string `json:"session_activity_id,omitempty"`
	ActivityName      string `json:"activity_name,omitempty"`
	TimeSlotName      string `json:"time_slot_name,omitempty"`
	SortOrder         int32  `json:"sort_order,omitempty"`
}

type ExplanationResponse struct {
	ID              string  `json:"id"`
	CounselorID     string  `json:"counselor_id,omitempty"`
	CounselorName   string  `json:"counselor_name,omitempty"`
	CamperID        string  `json:"camper_id,omitempty"`
	CamperName      string  `json:"camper_name,omitempty"`
	ExplanationType string  `json:"explanation_type"`
	ConstraintName  *string `json:"constraint_name"`
	Rank            *int32  `json:"rank,omitempty"`
	Message         string  `json:"message"`
}

func (ctrl *Controller) TriggerRun(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req TriggerRunRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	runType := req.RunType
	if runType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_type is required"})
		return
	}

	switch runType {
	case "cabin":
		cfg, err := buildCabinSolverConfig(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		run, err := ctrl.svc.TriggerCabinRun(c.Request.Context(), campID, sessionID, cfg)
		if err != nil {
			ctrl.handleTriggerError(c, log, sessionID, err)
			return
		}

		c.JSON(http.StatusCreated, run)

	case "activity_schedule":
		cfg, err := buildActivitySolverConfig(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		run, err := ctrl.svc.TriggerActivityRun(c.Request.Context(), campID, sessionID, cfg)
		if err != nil {
			ctrl.handleTriggerError(c, log, sessionID, err)
			return
		}

		c.JSON(http.StatusCreated, run)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown run_type: %s (must be 'cabin' or 'activity_schedule')", runType)})
	}
}

func (ctrl *Controller) handleTriggerError(c *gin.Context, log *slog.Logger, sessionID string, err error) {
	var preconditionErr *PreconditionError
	if errors.As(err, &preconditionErr) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": preconditionErr.Error()})
		return
	}
	var camperErr *CamperUnassignedError
	if errors.As(err, &camperErr) {
		campers := make([]gin.H, len(camperErr.Campers))
		for i, cm := range camperErr.Campers {
			campers[i] = gin.H{"id": cm.ID, "name": cm.Name}
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":              camperErr.Error(),
			"unassigned_campers": campers,
		})
		return
	}
	if errors.Is(err, ErrNoSolutions) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "The solver could not find any valid assignments. This usually means constraints are too restrictive — check cabin requirements, counselor availability, or camper enrollment."})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	if api.IsBadInput(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.
		With("session_id", sessionID).
		With("error", err).
		Error("error triggering assignment run")
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func (ctrl *Controller) ListRuns(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	runs, err := ctrl.svc.ListRuns(c.Request.Context(), campID, sessionID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error listing assignment runs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, runs)
}

func (ctrl *Controller) GetRun(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	runID := c.Param("runId")

	run, err := ctrl.svc.GetRun(c.Request.Context(), campID, runID)
	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "assignment run not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("run_id", runID).
			With("error", err).
			Error("error getting assignment run")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, run)
}

func (ctrl *Controller) GetRunById(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	runID := c.Param("runId")

	run, err := ctrl.svc.GetRun(c.Request.Context(), campID, runID)
	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "assignment run not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("run_id", runID).
			With("error", err).
			Error("error getting assignment run")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, run)
}

func (ctrl *Controller) DeleteRun(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	runID := c.Param("runId")

	err := ctrl.svc.DeleteRun(c.Request.Context(), campID, runID)
	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "assignment run not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("run_id", runID).
			With("error", err).
			Error("error deleting assignment run")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *Controller) GetSolution(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	runID := c.Param("runId")
	solutionID := c.Param("solutionId")

	sol, err := ctrl.svc.GetSolution(c.Request.Context(), campID, runID, solutionID)
	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "assignment run not found"})
			return
		}
		if errors.Is(err, ErrSolutionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "solution not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("solution_id", solutionID).
			With("error", err).
			Error("error getting solution")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, sol)
}

func (ctrl *Controller) SelectSolution(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	runID := c.Param("runId")
	solutionID := c.Param("solutionId")

	run, err := ctrl.svc.SelectSolution(c.Request.Context(), campID, runID, solutionID)
	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "assignment run not found"})
			return
		}
		if errors.Is(err, ErrSolutionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "solution not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("run_id", runID).
			With("solution_id", solutionID).
			With("error", err).
			Error("error selecting solution")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, run)
}

func buildCabinSolverConfig(req TriggerRunRequest) (solver.CabinSolverConfig, error) {
	cfg := solver.DefaultCabinSolverConfig()

	if req.MaxSolutions != nil {
		if *req.MaxSolutions <= 0 {
			return cfg, fmt.Errorf("max_solutions must be greater than 0")
		}
		cfg.MaxSolutions = *req.MaxSolutions
		cfg.CounselorConfig.MaxSolutions = *req.MaxSolutions
	}
	if req.MaxIterations != nil {
		if *req.MaxIterations <= 0 {
			return cfg, fmt.Errorf("max_iterations must be greater than 0")
		}
		cfg.MaxIterations = *req.MaxIterations
		cfg.CounselorConfig.MaxIterations = *req.MaxIterations
		cfg.CamperConfig.MaxIterations = *req.MaxIterations
	}
	if req.Weights != nil {
		if req.Weights.ReturningAgeGroup != nil {
			cfg.CounselorConfig.Weights.ReturningAgeGroup = *req.Weights.ReturningAgeGroup
		}
		if req.Weights.ReturningCabin != nil {
			cfg.CounselorConfig.Weights.ReturningCabin = *req.Weights.ReturningCabin
		}
		if req.Weights.CocounselorPreference != nil {
			cfg.CounselorConfig.Weights.CocounselorPreference = *req.Weights.CocounselorPreference
		}
		if req.Weights.AgeGroupPreference != nil {
			cfg.CounselorConfig.Weights.AgeGroupPreference = *req.Weights.AgeGroupPreference
		}
		if req.Weights.MultipleSeniors != nil {
			cfg.CounselorConfig.Weights.MultipleSeniors = *req.Weights.MultipleSeniors
		}
	}
	if req.CamperWeights != nil {
		if req.CamperWeights.FriendPreference != nil {
			cfg.CamperConfig.Weights.FriendPreference = *req.CamperWeights.FriendPreference
		}
	}

	return cfg, nil
}

func buildActivitySolverConfig(req TriggerRunRequest) (solver.ActivitySolverConfig, error) {
	cfg := solver.DefaultActivitySolverConfig()

	if req.MaxSolutions != nil {
		if *req.MaxSolutions <= 0 {
			return cfg, fmt.Errorf("max_solutions must be greater than 0")
		}
		cfg.MaxSolutions = *req.MaxSolutions
	}
	if req.MaxIterations != nil {
		if *req.MaxIterations <= 0 {
			return cfg, fmt.Errorf("max_iterations must be greater than 0")
		}
		cfg.MaxIterations = *req.MaxIterations
	}
	if req.ActivityWeights != nil {
		if req.ActivityWeights.ActivityPreference != nil {
			cfg.Weights.ActivityPreference = *req.ActivityWeights.ActivityPreference
		}
	}

	return cfg, nil
}
