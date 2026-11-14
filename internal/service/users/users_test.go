package users

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"client-services/internal/domain"
)

type repositoryStub struct {
	Repository
	batch func(context.Context, []int64) (map[int64]domain.User, error)
	get   func(context.Context, int64) (*domain.User, error)
}

func (r repositoryStub) GetUsersBatch(ctx context.Context, ids []int64) (map[int64]domain.User, error) {
	return r.batch(ctx, ids)
}
func (r repositoryStub) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	return r.get(ctx, id)
}

func TestBatchPreservesOrderAndIndividualErrors(t *testing.T) {
	calls := 0
	s := NewService(repositoryStub{batch: func(_ context.Context, ids []int64) (map[int64]domain.User, error) {
		calls++
		if !reflect.DeepEqual(ids, []int64{2, 1, 99, 1}) {
			t.Fatal(ids)
		}
		return map[int64]domain.User{1: {ID: 1}, 2: {ID: 2}}, nil
	}})
	users, errs := s.GetUsersBatch(t.Context(), []int64{2, 0, 1, 99, 1})
	if calls != 1 || len(users) != 5 || len(errs) != 5 || users[0].ID != 2 || users[2].ID != 1 || users[4].ID != 1 {
		t.Fatalf("%v %v", users, errs)
	}
	if !errors.Is(errs[1], domain.ErrInvalidArgument) || !errors.Is(errs[3], domain.ErrNotFound) || errs[0] != nil || errs[2] != nil || errs[4] != nil {
		t.Fatal(errs)
	}
	users[2].Name = "changed"
	if users[4].Name != "" {
		t.Fatal("duplicate results share mutable pointers")
	}
	users, errs = NewService(repositoryStub{}).GetUsersBatch(t.Context(), []int64{0, -1})
	if len(users) != 2 || errs[0] == nil || errs[1] == nil {
		t.Fatal("invalid IDs reached storage")
	}
}
func TestStorageErrorsAndGetUser(t *testing.T) {
	broken := errors.New("offline")
	s := NewService(repositoryStub{batch: func(context.Context, []int64) (map[int64]domain.User, error) { return nil, broken }, get: func(_ context.Context, id int64) (*domain.User, error) { return &domain.User{ID: id}, nil }})
	_, errs := s.GetUsersBatch(t.Context(), []int64{1, 0, 2})
	if !errors.Is(errs[0], broken) || !errors.Is(errs[2], broken) || !errors.Is(errs[1], domain.ErrInvalidArgument) {
		t.Fatal(errs)
	}
	u, err := s.GetUser(t.Context(), 8)
	if err != nil || u.ID != 8 {
		t.Fatalf("%v %v", u, err)
	}
	_, err = s.GetUser(t.Context(), 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	s = NewService(repositoryStub{get: func(context.Context, int64) (*domain.User, error) { return nil, broken }})
	_, err = s.GetUser(t.Context(), 1)
	if !errors.Is(err, broken) {
		t.Fatal(err)
	}
}
