package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/camp"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("camp not found")

type Service struct {
	queries *db.Queries
	orgSync *OrgSyncer
}

// NewService constructs the camp admin service. If orgSync is non-nil,
// camp create/delete will mirror to the auth-server's organization table.
func NewService(queries *db.Queries, orgSync *OrgSyncer) *Service {
	return &Service{queries: queries, orgSync: orgSync}
}

func (svc *Service) List(ctx context.Context) ([]camp.CampResponse, error) {
	camps, err := svc.queries.ListCamps(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing camps: %w", err)
	}

	result := make([]camp.CampResponse, len(camps))
	for i, c := range camps {
		result[i] = camp.ToCampResponse(c)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, id string) (camp.CampResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return camp.CampResponse{}, err
	}

	c, err := svc.queries.GetCamp(ctx, uid)
	if err != nil {
		return camp.CampResponse{}, fmt.Errorf("error getting camp %s: %w", id, err)
	}

	return camp.ToCampResponse(c), nil
}

func (svc *Service) Create(ctx context.Context, req CreateCampRequest) (camp.CampResponse, error) {
	c, err := svc.queries.CreateCamp(ctx, db.CreateCampParams{
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
	})
	if err != nil {
		return camp.CampResponse{}, fmt.Errorf("error creating camp: %w", err)
	}

	resp := camp.ToCampResponse(c)
	if svc.orgSync != nil {
		if err := svc.orgSync.CreateOrg(ctx, resp.ID, req.Name, slugify(req.Name)); err != nil {
			// Org sync failure is logged but not fatal: a reconciliation job
			// can re-sync, and the camp row is the source of truth.
			slog.
				With("camp_id", resp.ID).
				With("error", err).
				Error("error syncing camp to auth-server organization")
		}
	}
	return resp, nil
}

func (svc *Service) Update(ctx context.Context, id string, req UpdateCampRequest) (camp.CampResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return camp.CampResponse{}, err
	}

	c, err := svc.queries.UpdateCamp(ctx, db.UpdateCampParams{
		ID:           uid,
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
		CampEnabled:  api.ToPgBool(req.Enabled),
	})
	if err != nil {
		return camp.CampResponse{}, fmt.Errorf("error updating camp %s: %w", id, err)
	}

	return camp.ToCampResponse(c), nil
}

func (svc *Service) Delete(ctx context.Context, id string) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteCamp(ctx, uid)
	if err != nil {
		return fmt.Errorf("error deleting camp %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if svc.orgSync != nil {
		if err := svc.orgSync.DeleteOrg(ctx, id); err != nil {
			slog.
				With("camp_id", id).
				With("error", err).
				Error("error deleting camp organization in auth-server")
		}
	}
	return nil
}

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := slugRegex.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}
