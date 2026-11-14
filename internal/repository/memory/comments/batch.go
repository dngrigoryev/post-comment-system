package comments

import (
	"context"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/memory/internal/pagination"
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
	afterIDs := make([]int64, len(requests))
	for i, request := range requests {
		if err := pagination.ValidateSize(request.page.First); err != nil {
			errs[i] = err
		} else {
			afterIDs[i], errs[i] = cursor.DecodeComment(request.page.After, request.postID, request.parentID)
		}
	}
	comments := r.batchCommentSnapshot(ctx, requests, afterIDs, errs, replies)
	for i, request := range requests {
		if errs[i] == nil {
			pages[i], errs[i] = commentPage(ctx, comments[i], request.page.First, request.postID, request.parentID)
		}
	}
	return pages, errs
}

func (r *CommentRepository) batchCommentSnapshot(ctx context.Context, requests []commentBatchRequest, afterIDs []int64, errs []error, replies bool) [][]domain.Comment {
	if !replies {
		r.state.PostsMu.RLock()
		defer r.state.PostsMu.RUnlock()
	}
	r.state.CommentsMu.RLock()
	defer r.state.CommentsMu.RUnlock()
	comments := make([][]domain.Comment, len(requests))
	for i, request := range requests {
		if errs[i] != nil {
			continue
		}
		if err := ctx.Err(); err != nil {
			errs[i] = err
			continue
		}
		if replies {
			parent, ok := r.state.Comments[*request.parentID]
			if !ok || parent.PostID != request.postID {
				errs[i] = domain.ErrNotFound
				continue
			}
		} else if _, ok := r.state.Posts[request.postID]; !ok {
			errs[i] = domain.ErrNotFound
			continue
		}
		comments[i] = r.commentSnapshotLocked(request.postID, request.parentID, afterIDs[i], request.page.First)
	}
	return comments
}
