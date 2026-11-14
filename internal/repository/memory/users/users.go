package users

import (
	"context"

	"client-services/internal/domain"
	"client-services/internal/repository/memory/internal/store"
)

type UserRepository struct{ state *store.State }

func New(state *store.State) *UserRepository { return &UserRepository{state: state} }

func (r *UserRepository) GetUser(ctx context.Context, userID int64) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user, ok := r.state.Users[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &user, nil
}

func (r *UserRepository) GetUsersBatch(ctx context.Context, userIDs []int64) (map[int64]domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	users := make(map[int64]domain.User, len(userIDs))
	for _, id := range userIDs {
		if user, ok := r.state.Users[id]; ok {
			users[id] = user
		}
	}
	return users, nil
}
