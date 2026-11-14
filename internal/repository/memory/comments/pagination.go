package comments

import (
	"context"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
)

func commentPage(ctx context.Context, comments []domain.Comment, first int, postID int64, parentCommentID *int64) (*domain.CommentPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info := domain.PageInfo{HasNextPage: len(comments) > first}
	nodes := make([]domain.Comment, min(first, len(comments)))
	copy(nodes, comments)
	if len(nodes) > 0 {
		endCursor, err := cursor.EncodeComment(nodes[len(nodes)-1].ID, postID, parentCommentID)
		if err != nil {
			return nil, fmt.Errorf("encode comment page cursor: %w", err)
		}
		info.EndCursor = &endCursor
	}
	return &domain.CommentPage{Nodes: nodes, PageInfo: info}, nil
}
