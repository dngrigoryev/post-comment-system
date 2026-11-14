package users

import (
	"context"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/service/validation"
)

type Repository interface {
	GetUser(ctx context.Context, userID int64) (*domain.User, error)
	GetUsersBatch(ctx context.Context, userIDs []int64) (map[int64]domain.User, error)
}

type Service struct{ users Repository }

func NewService(users Repository) *Service { return &Service{users: users} }

func (s *Service) GetUsersBatch(ctx context.Context, userIDs []int64) ([]*domain.User, []error) {
	values := make([]*domain.User, len(userIDs))
	errs := make([]error, len(userIDs))
	validIDs := make([]int64, 0, len(userIDs))
	for i, id := range userIDs {
		errs[i] = validation.ID("user ID", id)
		if errs[i] == nil {
			validIDs = append(validIDs, id)
		}
	}
	if len(validIDs) == 0 {
		return values, errs
	}
	users, err := s.users.GetUsersBatch(ctx, validIDs)
	for i, id := range userIDs {
		if errs[i] != nil {
			continue
		}
		if err != nil {
			errs[i] = fmt.Errorf("get users: %w", err)
		} else if user, ok := users[id]; ok {
			values[i] = &user
		} else {
			errs[i] = domain.ErrNotFound
		}
	}
	return values, errs
}

func (s *Service) GetUser(ctx context.Context, userID int64) (*domain.User, error) {
	if err := validation.ID("user ID", userID); err != nil {
		return nil, err
	}
	user, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}
