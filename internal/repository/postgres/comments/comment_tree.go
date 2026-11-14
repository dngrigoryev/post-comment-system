package comments

import (
	"context"
	"fmt"
	"math"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/postgres/dbgen"
	"client-services/internal/repository/postgres/internal/database"
)

func (r *CommentRepository) LoadCommentTree(ctx context.Context, request domain.CommentTreeRequest) ([]domain.CommentTreePage, error) {
	params := dbgen.LoadCommentTreeParams{}
	selections := make(map[int]domain.CommentSelection)
	var flatten func([]domain.CommentSelection, int) error
	flatten = func(nodes []domain.CommentSelection, parent int) error {
		for _, node := range nodes {
			if node.ID <= 0 || node.ID > math.MaxInt32 {
				return fmt.Errorf("%w: invalid comment selection ID", domain.ErrInvalidArgument)
			}
			limit, err := database.PageLimit(node.Page.First)
			if err != nil {
				continue
			}
			position, err := cursor.ParseComment(node.Page.After)
			if err != nil {

				continue
			}
			params.SelectionIds = append(params.SelectionIds, int32(node.ID))
			params.ParentSelectionIds = append(params.ParentSelectionIds, int32(parent))
			params.PageSizes = append(params.PageSizes, limit-1)
			params.AfterIds = append(params.AfterIds, position.ID)
			params.CursorPostIds = append(params.CursorPostIds, position.PostID)
			var parentID int64
			if position.ParentCommentID != nil {
				parentID = *position.ParentCommentID
			}
			params.CursorParentIds = append(params.CursorParentIds, parentID)
			selections[node.ID] = node
			if err := flatten(node.Replies, node.ID); err != nil {
				return err
			}
		}
		return nil
	}
	if err := flatten(request.Selections, 0); err != nil {
		return nil, err
	}
	for _, root := range request.Roots {
		params.PostIds = append(params.PostIds, root.PostID)
		var parentID int64
		if root.ParentCommentID != nil {
			parentID = *root.ParentCommentID
		}
		params.ParentIds = append(params.ParentIds, parentID)
	}
	if len(params.SelectionIds) == 0 || len(params.PostIds) == 0 {
		return nil, nil
	}
	rows, err := r.queries.LoadCommentTree(ctx, params)
	if err != nil {
		return nil, database.WrapError("load comment tree", err)
	}
	var result []domain.CommentTreePage
	for start := 0; start < len(rows); {
		first := rows[start]
		end := start
		var comments []dbgen.Comment
		for end < len(rows) && rows[end].SelectionID == first.SelectionID && rows[end].PostID == first.PostID && rows[end].ParentID == first.ParentID {
			row := rows[end]
			if row.ID != 0 {
				comments = append(comments, dbgen.Comment{
					ID: row.ID, Post: row.PostID, ParentComment: row.ParentComment,
					Content: row.Content, Author: row.Author, CreatedAt: row.CreatedAt,
				})
			}
			end++
		}
		var parentID *int64
		if first.ParentID != 0 {
			id := first.ParentID
			parentID = &id
		}
		page, err := commentPage(comments, selections[int(first.SelectionID)].Page.First, first.PostID, parentID)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.CommentTreePage{
			SelectionID: int(first.SelectionID), PostID: first.PostID, ParentCommentID: parentID, Page: page,
		})
		start = end
	}
	return result, nil
}
