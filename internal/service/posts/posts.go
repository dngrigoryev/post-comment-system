package posts

import (
	"context"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/service/validation"
)

const MaxTitleLength = 255

type Repository interface {
	CreatePost(ctx context.Context, post domain.Post) (*domain.Post, error)
	GetPost(ctx context.Context, postID int64) (*domain.Post, error)
	ListPosts(ctx context.Context, page domain.PageRequest) (*domain.PostPage, error)
	DisableComments(ctx context.Context, userID, postID int64) (*domain.Post, error)
}

type Service struct {
	posts Repository
}

func NewService(posts Repository) *Service {
	return &Service{posts: posts}
}

func (s *Service) CreatePost(ctx context.Context, userID int64, title, content string) (*domain.Post, error) {
	if err := validation.User(userID); err != nil {
		return nil, err
	}
	if err := validation.Text("title", title, MaxTitleLength); err != nil {
		return nil, err
	}
	if err := validation.Text("content", content, 0); err != nil {
		return nil, err
	}
	post, err := s.posts.CreatePost(ctx, domain.Post{
		Title:             title,
		Content:           content,
		AuthorID:          userID,
		CommentsPermitted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create post: %w", err)
	}
	return post, nil
}

func (s *Service) GetPost(ctx context.Context, postID int64) (*domain.Post, error) {
	if err := validation.ID("post ID", postID); err != nil {
		return nil, err
	}
	post, err := s.posts.GetPost(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("get post: %w", err)
	}
	return post, nil
}

func (s *Service) ListPosts(ctx context.Context, page domain.PageRequest) (*domain.PostPage, error) {
	if err := validation.Page(page); err != nil {
		return nil, err
	}
	posts, err := s.posts.ListPosts(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	return posts, nil
}

func (s *Service) DisableComments(ctx context.Context, userID, postID int64) (*domain.Post, error) {
	if err := validation.User(userID); err != nil {
		return nil, err
	}
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post.AuthorID != userID {
		return nil, domain.ErrForbidden
	}
	updated, err := s.posts.DisableComments(ctx, userID, postID)
	if err != nil {
		return nil, fmt.Errorf("disable comments: %w", err)
	}
	return updated, nil
}
