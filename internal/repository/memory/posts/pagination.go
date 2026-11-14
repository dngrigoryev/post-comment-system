package posts

import (
	"context"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
)

func postPage(ctx context.Context, posts []domain.Post, first int) (*domain.PostPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info := domain.PageInfo{HasNextPage: len(posts) > first}
	nodes := make([]domain.Post, min(first, len(posts)))
	copy(nodes, posts)
	if len(nodes) > 0 {
		endCursor, err := cursor.EncodePost(nodes[len(nodes)-1].ID)
		if err != nil {
			return nil, fmt.Errorf("encode post page cursor: %w", err)
		}
		info.EndCursor = &endCursor
	}
	return &domain.PostPage{Nodes: nodes, PageInfo: info}, nil
}
