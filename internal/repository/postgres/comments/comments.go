package comments

import (
	"context"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/postgres/dbgen"
	"client-services/internal/repository/postgres/internal/database"

	"github.com/jackc/pgx/v5"
)

type CommentRepository struct {
	db      database.DB
	queries *dbgen.Queries
}

func New(db database.DB) *CommentRepository {
	return &CommentRepository{db: db, queries: dbgen.New(db)}
}

func (r *CommentRepository) AddComment(ctx context.Context, comment domain.Comment) (*domain.Comment, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, database.WrapError("begin add comment", err)
	}
	defer tx.Rollback(ctx)
	queries := r.queries.WithTx(tx)
	post, err := queries.GetPostForShare(ctx, comment.PostID)
	if err != nil {
		return nil, database.WrapError("get comment's post", err)
	}
	if !post.CommentsPermitted {
		return nil, domain.ErrCommentsDisabled
	}
	row, err := queries.AddComment(ctx, dbgen.AddCommentParams{
		PostID:          comment.PostID,
		ParentCommentID: database.NullableID(comment.ParentCommentID),
		Content:         comment.Content,
		AuthorID:        comment.AuthorID,
	})
	if err != nil {
		return nil, database.WrapError("add comment", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, database.WrapError("commit add comment", err)
	}
	result := toDomainComment(row)
	return &result, nil
}

func (r *CommentRepository) GetComment(ctx context.Context, commentID int64) (*domain.Comment, error) {
	row, err := r.queries.GetComment(ctx, commentID)
	if err != nil {
		return nil, database.WrapError("get comment", err)
	}
	result := toDomainComment(row)
	return &result, nil
}

func (r *CommentRepository) ListPostComments(ctx context.Context, postID int64, page domain.PageRequest) (*domain.CommentPage, error) {
	limit, err := database.PageLimit(page.First)
	if err != nil {
		return nil, err
	}
	afterID, err := cursor.DecodeComment(page.After, postID, nil)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListPostComments(ctx, dbgen.ListPostCommentsParams{
		PostID: postID, AfterID: afterID, PageLimit: limit,
	})
	if err != nil {
		return nil, database.WrapError("list post comments", err)
	}
	return commentPage(rows, page.First, postID, nil)
}

func (r *CommentRepository) ListReplies(ctx context.Context, commentID int64, page domain.PageRequest) (*domain.CommentPage, error) {
	limit, err := database.PageLimit(page.First)
	if err != nil {
		return nil, err
	}
	parent, err := r.queries.GetComment(ctx, commentID)
	if err != nil {
		return nil, database.WrapError("get replies' parent", err)
	}
	afterID, err := cursor.DecodeComment(page.After, parent.Post, &commentID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListReplies(ctx, dbgen.ListRepliesParams{
		ParentCommentID: database.NullableID(&commentID), AfterID: afterID, PageLimit: limit,
	})
	if err != nil {
		return nil, database.WrapError("list replies", err)
	}
	return commentPage(rows, page.First, parent.Post, &commentID)
}

func commentPage(rows []dbgen.Comment, first int, postID int64, parentCommentID *int64) (*domain.CommentPage, error) {
	info := domain.PageInfo{HasNextPage: len(rows) > first}
	if info.HasNextPage {
		rows = rows[:first]
	}
	nodes := make([]domain.Comment, len(rows))
	for i, row := range rows {
		nodes[i] = toDomainComment(row)
	}
	if len(nodes) > 0 {
		endCursor, err := cursor.EncodeComment(nodes[len(nodes)-1].ID, postID, parentCommentID)
		if err != nil {
			return nil, fmt.Errorf("encode comment page cursor: %w", err)
		}
		info.EndCursor = &endCursor
	}
	return &domain.CommentPage{Nodes: nodes, PageInfo: info}, nil
}
