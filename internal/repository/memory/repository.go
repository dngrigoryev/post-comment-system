package memory

import (
	"client-services/internal/domain"
	"client-services/internal/repository/memory/comments"
	"client-services/internal/repository/memory/internal/store"
	"client-services/internal/repository/memory/posts"
	"client-services/internal/repository/memory/users"
)

type Repository struct {
	*posts.PostRepository
	*comments.CommentRepository
	*users.UserRepository
}

func New(initialUsers ...domain.User) *Repository {
	state := store.New(initialUsers...)
	return &Repository{posts.New(state), comments.New(state), users.New(state)}
}
