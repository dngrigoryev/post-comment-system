package comments

import (
	"context"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/service/validation"
)

type Events interface {
	PublishCommentAdded(domain.Comment)
	SubscribeComments(context.Context, int64) <-chan domain.Comment
}

func (s *Service) SubscribeComments(ctx context.Context, postID int64) (<-chan domain.Comment, error) {
	if err := validation.ID("post ID", postID); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.posts.GetPost(ctx, postID); err != nil {
		return nil, fmt.Errorf("get subscription's post: %w", err)
	}
	return s.events.SubscribeComments(ctx, postID), nil
}
