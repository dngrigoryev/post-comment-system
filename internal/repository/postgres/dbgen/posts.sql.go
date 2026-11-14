package dbgen

import (
	"context"
)

const createPost = `-- name: CreatePost :one
INSERT INTO posts (title, content, author, comments_permitted)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING id, title, content, author, comments_permitted, created_at
`

type CreatePostParams struct {
	Title             string
	Content           string
	AuthorID          int64
	CommentsPermitted bool
}

func (q *Queries) CreatePost(ctx context.Context, arg CreatePostParams) (Post, error) {
	row := q.db.QueryRow(ctx, createPost,
		arg.Title,
		arg.Content,
		arg.AuthorID,
		arg.CommentsPermitted,
	)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Content,
		&i.Author,
		&i.CommentsPermitted,
		&i.CreatedAt,
	)
	return i, err
}

const disableComments = `-- name: DisableComments :one
UPDATE posts
SET comments_permitted = FALSE
WHERE id = $1
  AND author = $2
RETURNING id, title, content, author, comments_permitted, created_at
`

type DisableCommentsParams struct {
	PostID int64
	UserID int64
}

func (q *Queries) DisableComments(ctx context.Context, arg DisableCommentsParams) (Post, error) {
	row := q.db.QueryRow(ctx, disableComments, arg.PostID, arg.UserID)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Content,
		&i.Author,
		&i.CommentsPermitted,
		&i.CreatedAt,
	)
	return i, err
}

const getPost = `-- name: GetPost :one
SELECT id, title, content, author, comments_permitted, created_at
FROM posts
WHERE id = $1
`

func (q *Queries) GetPost(ctx context.Context, postID int64) (Post, error) {
	row := q.db.QueryRow(ctx, getPost, postID)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Content,
		&i.Author,
		&i.CommentsPermitted,
		&i.CreatedAt,
	)
	return i, err
}

const getPostForShare = `-- name: GetPostForShare :one
SELECT id, title, content, author, comments_permitted, created_at
FROM posts
WHERE id = $1
FOR SHARE
`

func (q *Queries) GetPostForShare(ctx context.Context, postID int64) (Post, error) {
	row := q.db.QueryRow(ctx, getPostForShare, postID)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Content,
		&i.Author,
		&i.CommentsPermitted,
		&i.CreatedAt,
	)
	return i, err
}

const listPosts = `-- name: ListPosts :many
SELECT id, title, content, author, comments_permitted, created_at
FROM posts
WHERE id > $1
ORDER BY id ASC
LIMIT $2
`

type ListPostsParams struct {
	AfterID   int64
	PageLimit int32
}

func (q *Queries) ListPosts(ctx context.Context, arg ListPostsParams) ([]Post, error) {
	rows, err := q.db.Query(ctx, listPosts, arg.AfterID, arg.PageLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Post
	for rows.Next() {
		var i Post
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Content,
			&i.Author,
			&i.CommentsPermitted,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
