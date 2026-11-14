package users

import (
	"context"

	"client-services/internal/domain"
	"client-services/internal/repository/postgres/dbgen"
	"client-services/internal/repository/postgres/internal/database"
)

type UserRepository struct{ queries *dbgen.Queries }

func New(db database.DB) *UserRepository { return &UserRepository{queries: dbgen.New(db)} }

func (r *UserRepository) GetUser(ctx context.Context, userID int64) (*domain.User, error) {
	row, err := r.queries.GetUser(ctx, userID)
	if err != nil {
		return nil, database.WrapError("get user", err)
	}
	return &domain.User{ID: row.ID, Name: row.Name}, nil
}

func (r *UserRepository) GetUsersBatch(ctx context.Context, userIDs []int64) (map[int64]domain.User, error) {
	users := make(map[int64]domain.User, len(userIDs))
	if len(userIDs) == 0 {
		return users, nil
	}
	rows, err := r.queries.GetUsersBatch(ctx, userIDs)
	if err != nil {
		return nil, database.WrapError("get users", err)
	}
	for _, row := range rows {
		users[row.ID] = domain.User{ID: row.ID, Name: row.Name}
	}
	return users, nil
}

func (r *UserRepository) EnsureUser(ctx context.Context, user domain.User, email string) error {
	err := r.queries.EnsureUser(ctx, dbgen.EnsureUserParams{UserID: user.ID, Name: user.Name, Email: email})
	if err != nil {
		return database.WrapError("initialize user", err)
	}
	return nil
}
