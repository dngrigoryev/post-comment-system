package posts

import (
	"context"
	"errors"
	"fmt"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/postgres/dbgen"
	"client-services/internal/repository/postgres/internal/database"

	"github.com/jackc/pgx/v5"
)

type PostRepository struct{ queries *dbgen.Queries }

func New(db database.DB) *PostRepository { return &PostRepository{queries: dbgen.New(db)} }

func (r *PostRepository) CreatePost(ctx context.Context, post domain.Post) (*domain.Post, error) {
	row, err := r.queries.CreatePost(ctx, dbgen.CreatePostParams{
		Title:             post.Title,
		Content:           post.Content,
		AuthorID:          post.AuthorID,
		CommentsPermitted: post.CommentsPermitted,
	})
	if err != nil {
		return nil, database.WrapError("create post", err)
	}
	result := toDomainPost(row)
	return &result, nil
}

func (r *PostRepository) GetPost(ctx context.Context, postID int64) (*domain.Post, error) {
	row, err := r.queries.GetPost(ctx, postID)
	if err != nil {
		return nil, database.WrapError("get post", err)
	}
	result := toDomainPost(row)
	return &result, nil
}

func (r *PostRepository) ListPosts(ctx context.Context, page domain.PageRequest) (*domain.PostPage, error) {
	limit, err := database.PageLimit(page.First)
	if err != nil {
		return nil, err
	}
	afterID, err := cursor.DecodePost(page.After)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListPosts(ctx, dbgen.ListPostsParams{AfterID: afterID, PageLimit: limit})
	if err != nil {
		return nil, database.WrapError("list posts", err)
	}
	info := domain.PageInfo{HasNextPage: len(rows) > page.First}
	if info.HasNextPage {
		rows = rows[:page.First]
	}
	nodes := make([]domain.Post, len(rows))
	for i, row := range rows {
		nodes[i] = toDomainPost(row)
	}
	if len(nodes) > 0 {
		endCursor, err := cursor.EncodePost(nodes[len(nodes)-1].ID)
		if err != nil {
			return nil, fmt.Errorf("encode post page cursor: %w", err)
		}
		info.EndCursor = &endCursor
	}
	return &domain.PostPage{Nodes: nodes, PageInfo: info}, nil
}

func (r *PostRepository) DisableComments(ctx context.Context, userID, postID int64) (*domain.Post, error) {
	row, err := r.queries.DisableComments(ctx, dbgen.DisableCommentsParams{PostID: postID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		if _, lookupErr := r.queries.GetPost(ctx, postID); lookupErr != nil {
			return nil, database.WrapError("disable comments", lookupErr)
		}
		return nil, fmt.Errorf("disable comments: %w", domain.ErrForbidden)
	}
	if err != nil {
		return nil, database.WrapError("disable comments", err)
	}
	result := toDomainPost(row)
	return &result, nil
}
