package assignment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"
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

func (ctrl *Controller) RegisterRoutes(camps *gin.RouterGroup) {
	runs := camps.Group("/:campId/sessions/:sessionId/assignment-runs")
	runs.POST("", ctrl.TriggerRun)
	runs.GET("", ctrl.ListRuns)
	runs.GET("/:runId", ctrl.GetRun)
	runs.DELETE("/:runId", ctrl.DeleteRun)
	runs.GET("/:runId/solutions/:solutionId", ctrl.GetSolution)
	runs.POST("/:runId/solutions/:solutionId/select", ctrl.SelectSolution)
}

type TriggerRunRequest struct {
	RunType       string                `json:"run_type"`
	MaxSolutions  *int                  `json:"max_solutions"`
	MaxIterations *int                  `json:"max_iterations"`
	Weights       *Weights              `json:"weights"`
	CamperWeights *CamperWeightsRequest `json:"camper_weights"`
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
	ID              string          `json:"id"`
	AssignmentRunID string          `json:"assignment_run_id"`
	SolutionIndex   int             `json:"solution_index"`
	Score           float64         `json:"score"`
	ScoreBreakdown  json.RawMessage `json:"score_breakdown"`
}

type SolutionDetailResponse struct {
	SolutionSummaryResponse
	Assignments  []AssignmentResponse  `json:"assignments"`
	Explanations []ExplanationResponse `json:"explanations"`
}

type AssignmentResponse struct {
	ID          string `json:"id"`
	CounselorID string `json:"counselor_id,omitempty"`
	CamperID    string `json:"camper_id,omitempty"`
	CabinID     string `json:"cabin_id"`
}

type ExplanationResponse struct {
	ID              string  `json:"id"`
	CounselorID     string  `json:"counselor_id,omitempty"`
	CamperID        string  `json:"camper_id,omitempty"`
	ExplanationType string  `json:"explanation_type"`
	ConstraintName  *string `json:"constraint_name"`
	Message         string  `json:"message"`
}

func (ctrl *Controller) TriggerRun(c *gin.Context) {
	campID := c.Param("campId")
	sessionID := c.Param("sessionId")

	var req TriggerRunRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	runType := req.RunType
	if runType == "" {
		runType = "counselor_cabin"
	}

	switch runType {
	case "counselor_cabin":
		cfg, err := buildSolverConfig(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		run, err := ctrl.svc.TriggerRun(c.Request.Context(), campID, sessionID, cfg)
		if err != nil {
			ctrl.handleTriggerError(c, campID, sessionID, err)
			return
		}

		c.JSON(http.StatusCreated, run)

	case "camper_cabin":
		cfg, err := buildCamperSolverConfig(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		run, err := ctrl.svc.TriggerCamperRun(c.Request.Context(), campID, sessionID, cfg)
		if err != nil {
			ctrl.handleTriggerError(c, campID, sessionID, err)
			return
		}

		c.JSON(http.StatusCreated, run)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown run_type: %s", runType)})
	}
}

func (ctrl *Controller) handleTriggerError(c *gin.Context, campID, sessionID string, err error) {
	if errors.Is(err, ErrNoSolutions) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "solver produced no valid solutions"})
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
	slog.
		With("camp_id", campID).
		With("session_id", sessionID).
		With("error", err).
		Error("error triggering assignment run")
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func (ctrl *Controller) ListRuns(c *gin.Context) {
	campID := c.Param("campId")
	sessionID := c.Param("sessionId")

	runs, err := ctrl.svc.ListRuns(c.Request.Context(), campID, sessionID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("session_id", sessionID).
			With("error", err).
			Error("error listing assignment runs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, runs)
}

func (ctrl *Controller) GetRun(c *gin.Context) {
	campID := c.Param("campId")
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
		slog.
			With("camp_id", campID).
			With("run_id", runID).
			With("error", err).
			Error("error getting assignment run")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, run)
}

func (ctrl *Controller) DeleteRun(c *gin.Context) {
	campID := c.Param("campId")
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
		slog.
			With("camp_id", campID).
			With("run_id", runID).
			With("error", err).
			Error("error deleting assignment run")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *Controller) GetSolution(c *gin.Context) {
	campID := c.Param("campId")
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
		slog.
			With("camp_id", campID).
			With("solution_id", solutionID).
			With("error", err).
			Error("error getting solution")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, sol)
}

func (ctrl *Controller) SelectSolution(c *gin.Context) {
	campID := c.Param("campId")
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
		slog.
			With("camp_id", campID).
			With("run_id", runID).
			With("solution_id", solutionID).
			With("error", err).
			Error("error selecting solution")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, run)
}

func buildSolverConfig(req TriggerRunRequest) (solver.SolverConfig, error) {
	cfg := solver.DefaultSolverConfig()

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
	if req.Weights != nil {
		if req.Weights.ReturningAgeGroup != nil {
			cfg.Weights.ReturningAgeGroup = *req.Weights.ReturningAgeGroup
		}
		if req.Weights.ReturningCabin != nil {
			cfg.Weights.ReturningCabin = *req.Weights.ReturningCabin
		}
		if req.Weights.CocounselorPreference != nil {
			cfg.Weights.CocounselorPreference = *req.Weights.CocounselorPreference
		}
		if req.Weights.AgeGroupPreference != nil {
			cfg.Weights.AgeGroupPreference = *req.Weights.AgeGroupPreference
		}
		if req.Weights.MultipleSeniors != nil {
			cfg.Weights.MultipleSeniors = *req.Weights.MultipleSeniors
		}
	}

	return cfg, nil
}

func buildCamperSolverConfig(req TriggerRunRequest) (solver.CamperSolverConfig, error) {
	cfg := solver.DefaultCamperSolverConfig()

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
	if req.CamperWeights != nil {
		if req.CamperWeights.FriendPreference != nil {
			cfg.Weights.FriendPreference = *req.CamperWeights.FriendPreference
		}
	}

	return cfg, nil
}
