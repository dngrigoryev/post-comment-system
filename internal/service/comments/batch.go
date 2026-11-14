package comments

import (
	"context"

	"client-services/internal/domain"
	"client-services/internal/service/validation"
)

func (s *Service) ListPostCommentsBatch(ctx context.Context, requests []domain.PostCommentsRequest) ([]*domain.CommentPage, []error) {
	return batchCommentPages(requests, func(request domain.PostCommentsRequest) error {
		if err := validation.ID("post ID", request.PostID); err != nil {
			return err
		}
		return validation.Page(request.Page)
	}, func(valid []domain.PostCommentsRequest) ([]*domain.CommentPage, []error) {
		return s.comments.ListPostCommentsBatch(ctx, valid)
	})
}

func (s *Service) ListRepliesBatch(ctx context.Context, requests []domain.RepliesRequest) ([]*domain.CommentPage, []error) {
	return batchCommentPages(requests, func(request domain.RepliesRequest) error {
		if err := validation.ID("post ID", request.PostID); err != nil {
			return err
		}
		if err := validation.ID("parent comment ID", request.ParentCommentID); err != nil {
			return err
		}
		return validation.Page(request.Page)
	}, func(valid []domain.RepliesRequest) ([]*domain.CommentPage, []error) {
		return s.comments.ListRepliesBatch(ctx, valid)
	})
}

func batchCommentPages[T any](requests []T, validate func(T) error, fetch func([]T) ([]*domain.CommentPage, []error)) ([]*domain.CommentPage, []error) {
	pages := make([]*domain.CommentPage, len(requests))
	errs := make([]error, len(requests))
	valid := make([]T, 0, len(requests))
	indices := make([]int, 0, len(requests))
	for i, request := range requests {
		errs[i] = validate(request)
		if errs[i] == nil {
			valid = append(valid, request)
			indices = append(indices, i)
		}
	}
	if len(valid) == 0 {
		return pages, errs
	}
	loaded, loadErrors := fetch(valid)
	for i, original := range indices {
		pages[original], errs[original] = loaded[i], loadErrors[i]
	}
	return pages, errs
}
