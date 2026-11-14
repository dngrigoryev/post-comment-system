package postgres

import (
	"client-services/internal/repository/postgres/comments"
	"client-services/internal/repository/postgres/internal/database"
	"client-services/internal/repository/postgres/posts"
	"client-services/internal/repository/postgres/users"
)

type Repository struct {
	*posts.PostRepository
	*comments.CommentRepository
	*users.UserRepository
}

func New(db database.DB) *Repository {
	return &Repository{posts.New(db), comments.New(db), users.New(db)}
}
