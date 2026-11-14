package comments

import (
	"context"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/memory/internal/pagination"
)

type commentTreePageSnapshot struct {
	root        domain.CommentTreeRoot
	selectionID int
	first       int
	comments    []domain.Comment
}

func (r *CommentRepository) LoadCommentTree(ctx context.Context, request domain.CommentTreeRequest) ([]domain.CommentTreePage, error) {
	snapshots, err := r.commentTreeSnapshot(ctx, request)
	if err != nil {
		return nil, err
	}
	result := make([]domain.CommentTreePage, 0, len(snapshots))
	for _, snapshot := range snapshots {
		root := snapshot.root
		page, err := commentPage(ctx, snapshot.comments, snapshot.first, root.PostID, root.ParentCommentID)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.CommentTreePage{
			SelectionID: snapshot.selectionID, PostID: root.PostID, ParentCommentID: root.ParentCommentID, Page: page,
		})
	}
	return result, nil
}

func (r *CommentRepository) commentTreeSnapshot(ctx context.Context, request domain.CommentTreeRequest) ([]commentTreePageSnapshot, error) {
	r.state.PostsMu.RLock()
	defer r.state.PostsMu.RUnlock()
	r.state.CommentsMu.RLock()
	defer r.state.CommentsMu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	type task struct {
		root      domain.CommentTreeRoot
		selection domain.CommentSelection
	}
	var pending []task
	for _, root := range request.Roots {
		if _, ok := r.state.Posts[root.PostID]; !ok {
			continue
		}
		if root.ParentCommentID != nil {
			parent, ok := r.state.Comments[*root.ParentCommentID]
			if !ok || parent.PostID != root.PostID {
				continue
			}
		}
		for _, selection := range request.Selections {
			pending = append(pending, task{root, selection})
		}
	}
	var snapshots []commentTreePageSnapshot
	for len(pending) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		root, selection := current.root, current.selection
		if pagination.ValidateSize(selection.Page.First) != nil {
			continue
		}
		afterID, err := cursor.DecodeComment(selection.Page.After, root.PostID, root.ParentCommentID)
		if err != nil {
			continue
		}
		comments := r.commentSnapshotLocked(root.PostID, root.ParentCommentID, afterID, selection.Page.First)
		snapshots = append(snapshots, commentTreePageSnapshot{
			root: root, selectionID: selection.ID, first: selection.Page.First, comments: comments,
		})

		for _, comment := range comments[:min(selection.Page.First, len(comments))] {
			for _, child := range selection.Replies {
				id := comment.ID
				pending = append(pending, task{domain.CommentTreeRoot{PostID: root.PostID, ParentCommentID: &id}, child})
			}
		}
	}
	return snapshots, ctx.Err()
}
