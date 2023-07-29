package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

type Repository[T any] struct {
	db bun.IDB
}

func New[T any](db bun.IDB) *Repository[T] {
	return &Repository[T]{db}
}

func (r *Repository[T]) Create(ctx context.Context, model *T) error {
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

func (r *Repository[T]) Get(ctx context.Context) ([]T, error) {
	models := make([]T, 0)
	err := r.db.NewSelect().Model(&models).Scan(ctx)
	return models, err
}

func (r *Repository[T]) Find(
	ctx context.Context,
	limit int,
	offset int,
	ids ...string,
) ([]T, error) {
	var models []T
	err := r.db.NewSelect().Model(&models).
		Where(`id in (?)`, bun.In(ids)).
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	return models, err
}

func (r *Repository[T]) FindOne(ctx context.Context, id string) (T, error) {
	var model T
	err := r.db.NewSelect().Model(&model).
		Where(`id = ?`, id).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return model, err
}

func (r *Repository[T]) FindWhere(
	ctx context.Context,
	limit int,
	offset int,
	query string,
	queryArgs ...any,
) ([]T, error) {
	var models []T
	err := r.db.NewSelect().Model(&models).
		Where(query, queryArgs...).
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	return models, err
}

func (r *Repository[T]) Update(ctx context.Context, model *T) error {
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

func (r *Repository[T]) Delete(ctx context.Context, id string) error {
	result, err := r.db.NewDelete().Model((*T)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}
	if result, err := result.RowsAffected(); err != nil {
		return err
	} else if result < 1 {
		return ErrNotFound
	}
	return err
}
