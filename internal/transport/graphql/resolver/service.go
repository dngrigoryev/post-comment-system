package resolver

import (
	"context"

	"client-services/internal/domain"
	"client-services/internal/transport/graphql/loader"
)

type PostService interface {
	CreatePost(ctx context.Context, userID int64, title, content string) (*domain.Post, error)
	GetPost(ctx context.Context, postID int64) (*domain.Post, error)
	ListPosts(ctx context.Context, page domain.PageRequest) (*domain.PostPage, error)
	DisableComments(ctx context.Context, userID, postID int64) (*domain.Post, error)
}

type CommentService interface {
	loader.CommentService
	AddComment(ctx context.Context, userID, postID int64, parentCommentID *int64, content string) (*domain.Comment, error)
}

type UserService interface{ loader.UserService }
