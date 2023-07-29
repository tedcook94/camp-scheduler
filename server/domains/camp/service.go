package camp

import (
	"camp-scheduler/server/repository"
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Service struct {
	repo   *repository.Repository[Camp]
	logger logrus.FieldLogger
}

func NewService(
	repo *repository.Repository[Camp],
	logger logrus.FieldLogger,
) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) Create(ctx context.Context, camp Camp) (Camp, error) {
	s.logger.Debug("saving new camp")

	err := s.repo.Create(ctx, &camp)
	if err != nil {
		s.logger.WithError(err).Error("failed to save camp")
		return Camp{}, errors.Wrap(err, "failed to save camp")
	}

	return camp, nil
}

func (s *Service) Get(ctx context.Context) ([]Camp, error) {
	s.logger.Debug("getting camps")

	camps, err := s.repo.Get(ctx)
	if err != nil {
		s.logger.WithError(err).Error("failed to fetch camps")
		return nil, errors.Wrap(err, "failed to fetch camps")
	}

	return camps, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Camp, error) {
	log := s.logger.WithFields(logrus.Fields{
		"campID": id,
	})
	log.Debug("getting camp by id")

	camp, err := s.repo.FindOne(ctx, id)
	if err != nil {
		log.WithError(err).Error("failed to fetch camp")
		return Camp{}, errors.Wrap(err, "failed to fetch camp")
	}

	return camp, err
}

func (s *Service) Update(ctx context.Context, camp Camp) (Camp, error) {
	s.logger.Debug("updating camp")

	err := s.repo.Update(ctx, &camp)
	if err != nil {
		s.logger.WithError(err).Error("failed to save camp")
		return Camp{}, errors.Wrap(err, "failed to save camp")
	}
	return camp, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	log := s.logger.WithFields(logrus.Fields{
		"campID": id,
	})
	log.Debug("deleting camp")

	err := s.repo.Delete(ctx, id)
	if err != nil {
		log.WithError(err).Error("failed to delete camp")
		return errors.Wrap(err, "integrations: failed to delete camp")
	}

	return nil
}
