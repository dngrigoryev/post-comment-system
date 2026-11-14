package posts

import (
	"context"
	"errors"
	"testing"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/memory/internal/store"
)

func TestIndexedPaginationAndCopies(t *testing.T) {
	s := store.New(domain.User{ID: 1})
	r := New(s)
	for i := 0; i < 5; i++ {
		p, err := r.CreatePost(t.Context(), domain.Post{ID: 999, Title: "post", Content: "text", AuthorID: 1, CommentsPermitted: true})
		if err != nil || p.ID != int64(i+1) || p.CreatedAt.IsZero() {
			t.Fatal(p, err)
		}
		p.Title = "changed"
	}
	var after *string
	var ids []int64
	for {
		page, err := r.ListPosts(t.Context(), domain.PageRequest{First: 2, After: after})
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range page.Nodes {
			if p.Title != "post" {
				t.Fatal("returned post aliases storage")
			}
			ids = append(ids, p.ID)
		}
		after = page.PageInfo.EndCursor
		if !page.PageInfo.HasNextPage {
			break
		}
	}
	if len(ids) != 5 {
		t.Fatal(ids)
	}
	for i, id := range ids {
		if id != int64(i+1) {
			t.Fatal(ids)
		}
	}
	page, err := r.ListPosts(t.Context(), domain.PageRequest{First: 2, After: after})
	if err != nil || page.Nodes == nil || len(page.Nodes) != 0 || page.PageInfo.EndCursor != nil || page.PageInfo.HasNextPage {
		t.Fatal(page, err)
	}
	large, _ := cursor.EncodePost(1 << 40)
	page, err = r.ListPosts(t.Context(), domain.PageRequest{First: 2, After: &large})
	if err != nil || len(page.Nodes) != 0 {
		t.Fatal(page, err)
	}
	p, err := r.GetPost(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	p.Content = "changed"
	p, _ = r.GetPost(t.Context(), 1)
	if p.Content != "text" {
		t.Fatal("get aliases storage")
	}
}
func TestErrorsAndOwnership(t *testing.T) {
	r := New(store.New(domain.User{ID: 1}, domain.User{ID: 2}))
	p, err := r.CreatePost(t.Context(), domain.Post{AuthorID: 1, CommentsPermitted: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.DisableComments(t.Context(), 2, p.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatal(err)
	}
	p, err = r.DisableComments(t.Context(), 1, p.ID)
	if err != nil || p.CommentsPermitted {
		t.Fatal(p, err)
	}
	_, err = r.GetPost(t.Context(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	_, err = r.DisableComments(t.Context(), 1, 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	_, err = r.CreatePost(t.Context(), domain.Post{AuthorID: 99})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	_, err = r.ListPosts(t.Context(), domain.PageRequest{First: 0})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	bad := "invalid"
	_, err = r.ListPosts(t.Context(), domain.PageRequest{First: 1, After: &bad})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = r.CreatePost(ctx, domain.Post{AuthorID: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, err = r.ListPosts(ctx, domain.PageRequest{First: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
