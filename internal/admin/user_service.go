package admin

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"golang.org/x/crypto/bcrypt"
)

var ErrUserNotFound = errors.New("user not found")

type UserService struct {
	queries *db.Queries
}

func NewUserService(queries *db.Queries) *UserService {
	return &UserService{queries: queries}
}

type UserResponse struct {
	ID        string  `json:"id"`
	CampID    *string `json:"camp_id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Role      string  `json:"role"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type CreateUserRequest struct {
	CampID    *string `json:"camp_id"`
	Username  string  `json:"username" binding:"required"`
	Email     string  `json:"email" binding:"required,email"`
	Password  string  `json:"password" binding:"required,min=8"`
	FirstName string  `json:"first_name" binding:"required"`
	LastName  string  `json:"last_name" binding:"required"`
	Role      string  `json:"role" binding:"required,oneof=admin super_admin"`
}

type UpdateUserRequest struct {
	CampID    *string `json:"camp_id"`
	Username  string  `json:"username" binding:"required"`
	Email     string  `json:"email" binding:"required,email"`
	FirstName string  `json:"first_name" binding:"required"`
	LastName  string  `json:"last_name" binding:"required"`
	Role      string  `json:"role" binding:"required,oneof=admin super_admin"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

func (svc *UserService) List(ctx context.Context) ([]UserResponse, error) {
	users, err := svc.queries.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}

	result := make([]UserResponse, len(users))
	for i, u := range users {
		result[i] = toUserResponseFromListRow(u)
	}
	return result, nil
}

func (svc *UserService) ListByCamp(ctx context.Context, campID string) ([]UserResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	users, err := svc.queries.ListUsersByCamp(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing users for camp %s: %w", campID, err)
	}

	result := make([]UserResponse, len(users))
	for i, u := range users {
		result[i] = toUserResponseFromCampRow(u)
	}
	return result, nil
}

func (svc *UserService) GetByID(ctx context.Context, id string) (UserResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return UserResponse{}, err
	}

	u, err := svc.queries.GetUserByID(ctx, uid)
	if err != nil {
		return UserResponse{}, fmt.Errorf("error getting user %s: %w", id, err)
	}

	return toUserResponse(u), nil
}

func (svc *UserService) Create(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return UserResponse{}, fmt.Errorf("error hashing password: %w", err)
	}

	campID, err := api.ToPgUUID(req.CampID)
	if err != nil {
		return UserResponse{}, err
	}

	u, err := svc.queries.CreateUser(ctx, db.CreateUserParams{
		CampID:       campID,
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
	})
	if err != nil {
		return UserResponse{}, fmt.Errorf("error creating user: %w", err)
	}

	return toUserResponse(u), nil
}

func (svc *UserService) Update(ctx context.Context, id string, req UpdateUserRequest) (UserResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return UserResponse{}, err
	}

	campID, err := api.ToPgUUID(req.CampID)
	if err != nil {
		return UserResponse{}, err
	}

	u, err := svc.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:        uid,
		CampID:    campID,
		Username:  req.Username,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
	})
	if err != nil {
		return UserResponse{}, fmt.Errorf("error updating user %s: %w", id, err)
	}

	return toUserResponseFromUpdateRow(u), nil
}

// TODO: when token revocation is implemented, this should also revoke all
// refresh tokens for the user so that existing sessions are invalidated.
func (svc *UserService) UpdatePassword(ctx context.Context, id string, req UpdatePasswordRequest) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %w", err)
	}

	rows, err := svc.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           uid,
		PasswordHash: string(hash),
	})
	if err != nil {
		return fmt.Errorf("error updating password for user %s: %w", id, err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (svc *UserService) Delete(ctx context.Context, id string) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteUser(ctx, uid)
	if err != nil {
		return fmt.Errorf("error deleting user %s: %w", id, err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

func toUserResponse(u db.User) UserResponse {
	return UserResponse{
		ID:        api.UUIDToString(u.ID),
		CampID:    api.UUIDToStringPtr(u.CampID),
		Username:  u.Username,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: u.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toUserResponseFromListRow(u db.ListUsersRow) UserResponse {
	return UserResponse{
		ID:        api.UUIDToString(u.ID),
		CampID:    api.UUIDToStringPtr(u.CampID),
		Username:  u.Username,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: u.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toUserResponseFromCampRow(u db.ListUsersByCampRow) UserResponse {
	return UserResponse{
		ID:        api.UUIDToString(u.ID),
		CampID:    api.UUIDToStringPtr(u.CampID),
		Username:  u.Username,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: u.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toUserResponseFromUpdateRow(u db.UpdateUserRow) UserResponse {
	return UserResponse{
		ID:        api.UUIDToString(u.ID),
		CampID:    api.UUIDToStringPtr(u.CampID),
		Username:  u.Username,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: u.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
