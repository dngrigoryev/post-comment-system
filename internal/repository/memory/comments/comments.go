package comments

import (
	"context"
	"fmt"
	"time"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/memory/internal/pagination"
	"client-services/internal/repository/memory/internal/store"
)

type CommentRepository struct{ state *store.State }

func New(state *store.State) *CommentRepository { return &CommentRepository{state: state} }

func (r *CommentRepository) AddComment(ctx context.Context, comment domain.Comment) (*domain.Comment, error) {
	r.state.PostsMu.RLock()
	defer r.state.PostsMu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	post, ok := r.state.Posts[comment.PostID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if !post.CommentsPermitted {
		return nil, domain.ErrCommentsDisabled
	}
	if _, ok := r.state.Users[comment.AuthorID]; !ok {
		return nil, fmt.Errorf("%w: invalid author reference", domain.ErrInvalidArgument)
	}
	comment = cloneComment(comment)
	r.state.CommentsMu.Lock()
	defer r.state.CommentsMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if comment.ParentCommentID != nil {
		parent, ok := r.state.Comments[*comment.ParentCommentID]
		if !ok || parent.PostID != comment.PostID {
			return nil, fmt.Errorf("%w: invalid parent reference", domain.ErrInvalidArgument)
		}
	}
	comment.ID = r.state.NextCommentID
	r.state.NextCommentID++
	comment.CreatedAt = time.Now().UTC()
	r.state.Comments[comment.ID] = comment
	if comment.ParentCommentID == nil {
		r.state.RootIDsByPost[comment.PostID] = append(r.state.RootIDsByPost[comment.PostID], comment.ID)
	} else {
		parentID := *comment.ParentCommentID
		r.state.ReplyIDsByParent[parentID] = append(r.state.ReplyIDsByParent[parentID], comment.ID)
	}
	result := cloneComment(comment)
	return &result, nil
}

func (r *CommentRepository) GetComment(ctx context.Context, commentID int64) (*domain.Comment, error) {
	r.state.CommentsMu.RLock()
	defer r.state.CommentsMu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	comment, ok := r.state.Comments[commentID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	result := cloneComment(comment)
	return &result, nil
}

func (r *CommentRepository) ListPostComments(ctx context.Context, postID int64, page domain.PageRequest) (*domain.CommentPage, error) {
	if err := pagination.ValidateSize(page.First); err != nil {
		return nil, err
	}
	afterID, err := cursor.DecodeComment(page.After, postID, nil)
	if err != nil {
		return nil, err
	}
	comments, err := r.commentSnapshot(ctx, postID, nil, afterID, page.First)
	if err != nil {
		return nil, err
	}
	return commentPage(ctx, comments, page.First, postID, nil)
}

func (r *CommentRepository) ListReplies(ctx context.Context, commentID int64, page domain.PageRequest) (*domain.CommentPage, error) {
	if err := pagination.ValidateSize(page.First); err != nil {
		return nil, err
	}
	parent, err := r.GetComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	afterID, err := cursor.DecodeComment(page.After, parent.PostID, &commentID)
	if err != nil {
		return nil, err
	}
	comments, err := r.commentSnapshot(ctx, parent.PostID, &commentID, afterID, page.First)
	if err != nil {
		return nil, err
	}
	return commentPage(ctx, comments, page.First, parent.PostID, &commentID)
}

func (r *CommentRepository) commentSnapshot(ctx context.Context, postID int64, parentCommentID *int64, afterID int64, first int) ([]domain.Comment, error) {
	r.state.CommentsMu.RLock()
	defer r.state.CommentsMu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	comments := r.commentSnapshotLocked(postID, parentCommentID, afterID, first)
	return comments, ctx.Err()
}

func (r *CommentRepository) commentSnapshotLocked(postID int64, parentCommentID *int64, afterID int64, first int) []domain.Comment {
	ids := r.state.RootIDsByPost[postID]
	if parentCommentID != nil {
		ids = r.state.ReplyIDsByParent[*parentCommentID]
	}
	ids = pagination.IDs(ids, afterID, first)
	comments := make([]domain.Comment, len(ids))
	for i, id := range ids {
		comments[i] = cloneComment(r.state.Comments[id])
	}
	return comments
}

func cloneComment(comment domain.Comment) domain.Comment {
	if comment.ParentCommentID != nil {
		id := *comment.ParentCommentID
		comment.ParentCommentID = &id
	}
	return comment
}
