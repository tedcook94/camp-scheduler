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
	slug := slugify(req.Name)
	if slug == "" {
		// Refuse to create a camp whose name slugs to "" because the
		// auth-server organization id/slug invariant would be unrecoverable.
		return camp.CampResponse{}, api.BadInput("camp name must contain at least one alphanumeric character")
	}

	c, err := svc.queries.CreateCamp(ctx, db.CreateCampParams{
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
	})
	if err != nil {
		return camp.CampResponse{}, fmt.Errorf("error creating camp: %w", err)
	}

	resp := camp.ToCampResponse(c)
	if svc.orgSync != nil {
		if err := svc.orgSync.CreateOrg(ctx, resp.ID, req.Name, slug); err != nil {
			// There is no reconciliation path between the camps table and
			// auth-server organizations, so a sync failure must be undone
			// rather than left as silent drift. Best effort: delete the
			// just-created camp row. If the rollback also fails, surface
			// both errors so an operator can repair manually.
			slog.
				With("camp_id", resp.ID).
				With("error", err).
				Error("error syncing camp to auth-server organization; rolling back camp row")
			if _, delErr := svc.queries.DeleteCamp(ctx, c.ID); delErr != nil {
				slog.
					With("camp_id", resp.ID).
					With("error", delErr).
					Error("error rolling back camp row after auth-server sync failure")
				return camp.CampResponse{}, fmt.Errorf(
					"error syncing camp %s to auth-server: %w (rollback also failed: %v)",
					resp.ID, err, delErr,
				)
			}
			return camp.CampResponse{}, fmt.Errorf("error syncing camp %s to auth-server: %w", resp.ID, err)
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
			return fmt.Errorf("camp %s deleted from database but failed to delete auth-server organization: %w", id, err)
		}
	}
	return nil
}

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := slugRegex.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}

// AddMember assigns a user to a camp's organization in the auth-server.
// The role is forwarded to the auth-server unchanged; callers are responsible
// for supplying a default when none is provided by the request.
func (svc *Service) AddMember(ctx context.Context, userID, campID, role string) error {
	if svc.orgSync == nil {
		return errors.New("auth-server sync is not configured")
	}
	if _, err := api.ParseUUID(userID); err != nil {
		return err
	}
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}
	if _, err := svc.queries.GetCamp(ctx, uid); err != nil {
		return fmt.Errorf("error getting camp %s: %w", campID, err)
	}
	return svc.orgSync.AddMember(ctx, userID, campID, role)
}

// RemoveMember removes a user from a camp's organization in the auth-server.
func (svc *Service) RemoveMember(ctx context.Context, userID, campID string) error {
	if svc.orgSync == nil {
		return errors.New("auth-server sync is not configured")
	}
	if _, err := api.ParseUUID(userID); err != nil {
		return err
	}
	if _, err := api.ParseUUID(campID); err != nil {
		return err
	}
	return svc.orgSync.RemoveMember(ctx, userID, campID)
}

// UpdateUser patches a user's profile fields via the auth-server.
func (svc *Service) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) error {
	if svc.orgSync == nil {
		return errors.New("auth-server sync is not configured")
	}
	if _, err := api.ParseUUID(userID); err != nil {
		return err
	}
	return svc.orgSync.UpdateUser(ctx, userID, UpdateUserRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	})
}

// SetUserRole updates a user's global role via the auth-server.
func (svc *Service) SetUserRole(ctx context.Context, userID, role string) error {
	if svc.orgSync == nil {
		return errors.New("auth-server sync is not configured")
	}
	if _, err := api.ParseUUID(userID); err != nil {
		return err
	}
	if role != "user" && role != "super_admin" {
		return api.BadInput("role must be 'user' or 'super_admin'")
	}
	return svc.orgSync.SetUserRole(ctx, userID, role)
}
