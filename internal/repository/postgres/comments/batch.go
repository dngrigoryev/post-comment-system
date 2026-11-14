package comments

import (
	"context"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/postgres/dbgen"
	"client-services/internal/repository/postgres/internal/database"
)

type commentBatchRequest struct {
	postID   int64
	parentID *int64
	page     domain.PageRequest
}

func (r *CommentRepository) ListPostCommentsBatch(ctx context.Context, requests []domain.PostCommentsRequest) ([]*domain.CommentPage, []error) {
	batch := make([]commentBatchRequest, len(requests))
	for i, request := range requests {
		batch[i] = commentBatchRequest{postID: request.PostID, page: request.Page}
	}
	return r.listCommentPagesBatch(ctx, batch, false)
}

func (r *CommentRepository) ListRepliesBatch(ctx context.Context, requests []domain.RepliesRequest) ([]*domain.CommentPage, []error) {
	batch := make([]commentBatchRequest, len(requests))
	for i, request := range requests {
		parentID := request.ParentCommentID
		batch[i] = commentBatchRequest{postID: request.PostID, parentID: &parentID, page: request.Page}
	}
	return r.listCommentPagesBatch(ctx, batch, true)
}

func (r *CommentRepository) listCommentPagesBatch(ctx context.Context, requests []commentBatchRequest, replies bool) ([]*domain.CommentPage, []error) {
	pages := make([]*domain.CommentPage, len(requests))
	errs := make([]error, len(requests))
	indices := make([]int, 0, len(requests))
	postIDs := make([]int64, 0, len(requests))
	parentIDs := make([]int64, 0, len(requests))
	afterIDs := make([]int64, 0, len(requests))
	limits := make([]int32, 0, len(requests))
	for i, request := range requests {
		limit, err := database.PageLimit(request.page.First)
		if err != nil {
			errs[i] = err
			continue
		}
		afterID, err := cursor.DecodeComment(request.page.After, request.postID, request.parentID)
		if err != nil {
			errs[i] = err
			continue
		}
		indices = append(indices, i)
		postIDs = append(postIDs, request.postID)
		afterIDs = append(afterIDs, afterID)
		limits = append(limits, limit)
		if replies {
			parentIDs = append(parentIDs, *request.parentID)
		}
	}
	if len(indices) == 0 {
		return pages, errs
	}
	var rows []dbgen.ListPostCommentsBatchRow
	var err error
	if replies {
		var replyRows []dbgen.ListRepliesBatchRow
		replyRows, err = r.queries.ListRepliesBatch(ctx, dbgen.ListRepliesBatchParams{
			ParentIds: parentIDs, PostIds: postIDs, AfterIds: afterIDs, PageLimits: limits,
		})
		for _, row := range replyRows {
			rows = append(rows, dbgen.ListPostCommentsBatchRow(row))
		}
	} else {
		rows, err = r.queries.ListPostCommentsBatch(ctx, dbgen.ListPostCommentsBatchParams{
			PostIds: postIDs, AfterIds: afterIDs, PageLimits: limits,
		})
	}
	if err != nil {
		for _, i := range indices {
			errs[i] = database.WrapError("list comment pages", err)
		}
		return pages, errs
	}
	comments := make([][]dbgen.Comment, len(indices))
	for _, row := range rows {
		position := int(row.RequestIndex) - 1
		original := indices[position]
		if row.ParentID == 0 {
			errs[original] = domain.ErrNotFound
		} else if row.ID != 0 {
			comments[position] = append(comments[position], dbgen.Comment{
				ID: row.ID, Post: row.Post, ParentComment: row.ParentComment,
				Content: row.Content, Author: row.Author, CreatedAt: row.CreatedAt,
			})
		}
	}
	for position, original := range indices {
		if errs[original] != nil {
			continue
		}
		request := requests[original]
		pages[original], errs[original] = commentPage(comments[position], request.page.First, request.postID, request.parentID)
	}
	return pages, errs
}
