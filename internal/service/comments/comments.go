package comments

import (
	"context"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/service/validation"
)

const MaxCommentLength = 2000

type PostReader interface {
	GetPost(ctx context.Context, postID int64) (*domain.Post, error)
}

type Repository interface {
	LoadCommentTree(ctx context.Context, request domain.CommentTreeRequest) ([]domain.CommentTreePage, error)
	AddComment(ctx context.Context, comment domain.Comment) (*domain.Comment, error)
	GetComment(ctx context.Context, commentID int64) (*domain.Comment, error)
	ListPostComments(ctx context.Context, postID int64, page domain.PageRequest) (*domain.CommentPage, error)
	ListReplies(ctx context.Context, commentID int64, page domain.PageRequest) (*domain.CommentPage, error)
	ListPostCommentsBatch(ctx context.Context, requests []domain.PostCommentsRequest) ([]*domain.CommentPage, []error)
	ListRepliesBatch(ctx context.Context, requests []domain.RepliesRequest) ([]*domain.CommentPage, []error)
}

type Service struct {
	comments Repository
	posts    PostReader
}

func NewService(comments Repository, posts PostReader) *Service {
	return &Service{comments: comments, posts: posts}
}

func (s *Service) AddComment(ctx context.Context, userID, postID int64, parentCommentID *int64, content string) (*domain.Comment, error) {
	if err := validation.User(userID); err != nil {
		return nil, err
	}
	if err := validation.ID("post ID", postID); err != nil {
		return nil, err
	}
	if err := validation.Text("content", content, MaxCommentLength); err != nil {
		return nil, err
	}
	if parentCommentID != nil {
		if err := validation.ID("parent comment ID", *parentCommentID); err != nil {
			return nil, err
		}
	}
	post, err := s.posts.GetPost(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("get comment's post: %w", err)
	}
	if !post.CommentsPermitted {
		return nil, domain.ErrCommentsDisabled
	}
	if parentCommentID != nil {
		parent, err := s.comments.GetComment(ctx, *parentCommentID)
		if err != nil {
			return nil, fmt.Errorf("get parent comment: %w", err)
		}
		if parent.PostID != postID {
			return nil, fmt.Errorf("%w: parent comment belongs to another post", domain.ErrInvalidArgument)
		}
	}
	comment, err := s.comments.AddComment(ctx, domain.Comment{
		PostID:          postID,
		ParentCommentID: parentCommentID,
		Content:         content,
		AuthorID:        userID,
	})
	if err != nil {
		return nil, fmt.Errorf("add comment: %w", err)
	}
	return comment, nil
}

func (s *Service) ListPostComments(ctx context.Context, postID int64, page domain.PageRequest) (*domain.CommentPage, error) {
	if err := validation.ID("post ID", postID); err != nil {
		return nil, err
	}
	if err := validation.Page(page); err != nil {
		return nil, err
	}
	if _, err := s.posts.GetPost(ctx, postID); err != nil {
		return nil, fmt.Errorf("get comments' post: %w", err)
	}
	comments, err := s.comments.ListPostComments(ctx, postID, page)
	if err != nil {
		return nil, fmt.Errorf("list post comments: %w", err)
	}
	return comments, nil
}

func (s *Service) ListReplies(ctx context.Context, commentID int64, page domain.PageRequest) (*domain.CommentPage, error) {
	if err := validation.ID("comment ID", commentID); err != nil {
		return nil, err
	}
	if err := validation.Page(page); err != nil {
		return nil, err
	}
	if _, err := s.comments.GetComment(ctx, commentID); err != nil {
		return nil, fmt.Errorf("get replies' parent: %w", err)
	}
	comments, err := s.comments.ListReplies(ctx, commentID, page)
	if err != nil {
		return nil, fmt.Errorf("list replies: %w", err)
	}
	return comments, nil
}
