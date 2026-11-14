package comments

import (
	"context"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/service/validation"
)

func (s *Service) LoadCommentTree(ctx context.Context, request domain.CommentTreeRequest) ([]domain.CommentTreePage, error) {
	for _, root := range request.Roots {
		if err := validation.ID("post ID", root.PostID); err != nil {
			return nil, err
		}
		if root.ParentCommentID != nil {
			if err := validation.ID("parent comment ID", *root.ParentCommentID); err != nil {
				return nil, err
			}
		}
	}
	seen := make(map[int]bool)
	var filter func([]domain.CommentSelection) ([]domain.CommentSelection, error)
	filter = func(selections []domain.CommentSelection) ([]domain.CommentSelection, error) {
		valid := make([]domain.CommentSelection, 0, len(selections))
		for _, selection := range selections {
			if selection.ID <= 0 || seen[selection.ID] {
				return nil, fmt.Errorf("%w: invalid comment selection ID", domain.ErrInvalidArgument)
			}
			seen[selection.ID] = true
			if validation.Page(selection.Page) != nil {
				continue
			}
			var err error
			selection.Replies, err = filter(selection.Replies)
			if err != nil {
				return nil, err
			}
			valid = append(valid, selection)
		}
		return valid, nil
	}
	var err error
	request.Selections, err = filter(request.Selections)
	if err != nil {
		return nil, err
	}
	if len(request.Roots) == 0 || len(request.Selections) == 0 {
		return nil, nil
	}
	return s.comments.LoadCommentTree(ctx, request)
}
