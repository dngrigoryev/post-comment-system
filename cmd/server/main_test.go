package main

import (
	"testing"

	"client-services/internal/config"
	"client-services/internal/domain"
)

func TestOpenMemoryRepository(t *testing.T) {
	r, closeRepo, err := openRepository(t.Context(), config.Config{Storage: config.StorageMemory, AuthUsers: []config.AuthUser{{ID: 7, Name: "Alice"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer closeRepo()
	u, err := r.GetUser(t.Context(), 7)
	if err != nil || u.Name != "Alice" {
		t.Fatal(u, err)
	}
	p, err := r.CreatePost(t.Context(), domain.Post{AuthorID: 7, CommentsPermitted: true})
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.AddComment(t.Context(), domain.Comment{PostID: p.ID, AuthorID: 7})
	if err != nil || c.PostID != p.ID {
		t.Fatal(c, err)
	}
	if _, _, err := openRepository(t.Context(), config.Config{Storage: "unknown"}); err == nil {
		t.Fatal("unknown storage accepted")
	}
}
