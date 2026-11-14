package users

import (
	"context"
	"errors"
	"testing"

	"client-services/internal/domain"
	"client-services/internal/repository/memory/internal/store"
)

func TestImmutableUsersAndBatch(t *testing.T) {
	user := domain.User{ID: 1, Name: "original"}
	s := store.New(user)
	r := New(s)
	user.Name = "input changed"

	s.PostsMu.Lock()
	s.CommentsMu.Lock()
	u, err := r.GetUser(t.Context(), 1)
	s.CommentsMu.Unlock()
	s.PostsMu.Unlock()
	if err != nil || u.Name != "original" {
		t.Fatal(u, err)
	}
	u.Name = "output changed"
	batch, err := r.GetUsersBatch(t.Context(), []int64{1, 999, 1})
	if err != nil || len(batch) != 1 || batch[1].Name != "original" {
		t.Fatal(batch, err)
	}
	delete(batch, 1)
	u, err = r.GetUser(t.Context(), 1)
	if err != nil || u.Name != "original" {
		t.Fatal(u, err)
	}
	_, err = r.GetUser(t.Context(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = r.GetUsersBatch(ctx, []int64{1})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, err = r.GetUser(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
