package dbgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const addComment = `-- name: AddComment :one
INSERT INTO comments (post, parent_comment, content, author)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING id, post, parent_comment, content, author, created_at
`

type AddCommentParams struct {
	PostID          int64
	ParentCommentID pgtype.Int8
	Content         string
	AuthorID        int64
}

func (q *Queries) AddComment(ctx context.Context, arg AddCommentParams) (Comment, error) {
	row := q.db.QueryRow(ctx, addComment,
		arg.PostID,
		arg.ParentCommentID,
		arg.Content,
		arg.AuthorID,
	)
	var i Comment
	err := row.Scan(
		&i.ID,
		&i.Post,
		&i.ParentComment,
		&i.Content,
		&i.Author,
		&i.CreatedAt,
	)
	return i, err
}

const getComment = `-- name: GetComment :one
SELECT id, post, parent_comment, content, author, created_at
FROM comments
WHERE id = $1
`

func (q *Queries) GetComment(ctx context.Context, commentID int64) (Comment, error) {
	row := q.db.QueryRow(ctx, getComment, commentID)
	var i Comment
	err := row.Scan(
		&i.ID,
		&i.Post,
		&i.ParentComment,
		&i.Content,
		&i.Author,
		&i.CreatedAt,
	)
	return i, err
}

const listPostComments = `-- name: ListPostComments :many
SELECT id, post, parent_comment, content, author, created_at
FROM comments
WHERE post = $1
  AND parent_comment IS NULL
  AND id > $2
ORDER BY id ASC
LIMIT $3
`

type ListPostCommentsParams struct {
	PostID    int64
	AfterID   int64
	PageLimit int32
}

func (q *Queries) ListPostComments(ctx context.Context, arg ListPostCommentsParams) ([]Comment, error) {
	rows, err := q.db.Query(ctx, listPostComments, arg.PostID, arg.AfterID, arg.PageLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Comment
	for rows.Next() {
		var i Comment
		if err := rows.Scan(
			&i.ID,
			&i.Post,
			&i.ParentComment,
			&i.Content,
			&i.Author,
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

const listPostCommentsBatch = `-- name: ListPostCommentsBatch :many
WITH requests AS (
    SELECT GENERATE_SUBSCRIPTS($1::bigint[], 1) AS request_index,
           UNNEST($1) AS post_id,
           UNNEST($2::bigint[]) AS after_id,
           UNNEST($3::integer[]) AS page_limit
)
SELECT r.request_index, COALESCE(p.id, 0) AS parent_id,
       COALESCE(c.id, 0) AS id, COALESCE(c.post, 0) AS post,
       c.parent_comment, COALESCE(c.content, '') AS content,
       COALESCE(c.author, 0) AS author, COALESCE(c.created_at, NOW()) AS created_at
FROM requests r
LEFT JOIN posts p ON p.id = r.post_id
LEFT JOIN LATERAL (
    SELECT id, post, parent_comment, content, author, created_at
    FROM comments
    WHERE post = p.id AND parent_comment IS NULL AND id > r.after_id
    ORDER BY id
    LIMIT r.page_limit
) c ON TRUE
ORDER BY r.request_index, c.id
`

type ListPostCommentsBatchParams struct {
	PostIds    []int64
	AfterIds   []int64
	PageLimits []int32
}

type ListPostCommentsBatchRow struct {
	RequestIndex  int32
	ParentID      int64
	ID            int64
	Post          int64
	ParentComment pgtype.Int8
	Content       string
	Author        int64
	CreatedAt     pgtype.Timestamptz
}

func (q *Queries) ListPostCommentsBatch(ctx context.Context, arg ListPostCommentsBatchParams) ([]ListPostCommentsBatchRow, error) {
	rows, err := q.db.Query(ctx, listPostCommentsBatch, arg.PostIds, arg.AfterIds, arg.PageLimits)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPostCommentsBatchRow
	for rows.Next() {
		var i ListPostCommentsBatchRow
		if err := rows.Scan(
			&i.RequestIndex,
			&i.ParentID,
			&i.ID,
			&i.Post,
			&i.ParentComment,
			&i.Content,
			&i.Author,
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

const listReplies = `-- name: ListReplies :many
SELECT id, post, parent_comment, content, author, created_at
FROM comments
WHERE parent_comment = $1
  AND id > $2
ORDER BY id ASC
LIMIT $3
`

type ListRepliesParams struct {
	ParentCommentID pgtype.Int8
	AfterID         int64
	PageLimit       int32
}

func (q *Queries) ListReplies(ctx context.Context, arg ListRepliesParams) ([]Comment, error) {
	rows, err := q.db.Query(ctx, listReplies, arg.ParentCommentID, arg.AfterID, arg.PageLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Comment
	for rows.Next() {
		var i Comment
		if err := rows.Scan(
			&i.ID,
			&i.Post,
			&i.ParentComment,
			&i.Content,
			&i.Author,
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

const listRepliesBatch = `-- name: ListRepliesBatch :many
WITH requests AS (
    SELECT GENERATE_SUBSCRIPTS($1::bigint[], 1) AS request_index,
           UNNEST($1) AS parent_id,
           UNNEST($2::bigint[]) AS post_id,
           UNNEST($3::bigint[]) AS after_id,
           UNNEST($4::integer[]) AS page_limit
)
SELECT r.request_index, COALESCE(p.id, 0) AS parent_id,
       COALESCE(c.id, 0) AS id, COALESCE(c.post, 0) AS post,
       c.parent_comment, COALESCE(c.content, '') AS content,
       COALESCE(c.author, 0) AS author, COALESCE(c.created_at, NOW()) AS created_at
FROM requests r
LEFT JOIN comments p ON p.id = r.parent_id AND p.post = r.post_id
LEFT JOIN LATERAL (
    SELECT id, post, parent_comment, content, author, created_at
    FROM comments
    WHERE parent_comment = p.id AND id > r.after_id
    ORDER BY id
    LIMIT r.page_limit
) c ON TRUE
ORDER BY r.request_index, c.id
`

type ListRepliesBatchParams struct {
	ParentIds  []int64
	PostIds    []int64
	AfterIds   []int64
	PageLimits []int32
}

type ListRepliesBatchRow struct {
	RequestIndex  int32
	ParentID      int64
	ID            int64
	Post          int64
	ParentComment pgtype.Int8
	Content       string
	Author        int64
	CreatedAt     pgtype.Timestamptz
}

func (q *Queries) ListRepliesBatch(ctx context.Context, arg ListRepliesBatchParams) ([]ListRepliesBatchRow, error) {
	rows, err := q.db.Query(ctx, listRepliesBatch,
		arg.ParentIds,
		arg.PostIds,
		arg.AfterIds,
		arg.PageLimits,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListRepliesBatchRow
	for rows.Next() {
		var i ListRepliesBatchRow
		if err := rows.Scan(
			&i.RequestIndex,
			&i.ParentID,
			&i.ID,
			&i.Post,
			&i.ParentComment,
			&i.Content,
			&i.Author,
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
