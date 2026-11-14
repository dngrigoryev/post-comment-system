package dbgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const loadCommentTree = `-- name: LoadCommentTree :many
WITH RECURSIVE selections AS (
    SELECT UNNEST($1::integer[]) AS selection_id,
           UNNEST($2::integer[]) AS parent_selection_id,
           UNNEST($3::integer[]) AS page_size,
           UNNEST($4::bigint[]) AS after_id,
           UNNEST($5::bigint[]) AS cursor_post_id,
           UNNEST($6::bigint[]) AS cursor_parent_id
), roots AS (
    SELECT UNNEST($7::bigint[]) AS post_id,
           UNNEST($8::bigint[]) AS parent_id
), tree AS (
    SELECT s.selection_id, s.page_size, r.post_id, r.parent_id,
           COALESCE(c.item_index, 0) AS item_index,
           COALESCE(c.id, 0) AS id, c.parent_comment,
           COALESCE(c.content, '') AS content, COALESCE(c.author, 0) AS author,
           COALESCE(c.created_at, NOW()) AS created_at
    FROM roots r
    JOIN posts p ON p.id = r.post_id
    JOIN selections s ON s.parent_selection_id = 0
    LEFT JOIN comments parent ON parent.id = r.parent_id AND parent.post = r.post_id
    LEFT JOIN LATERAL (
        SELECT id, parent_comment, content, author, created_at,
               ROW_NUMBER() OVER (ORDER BY id) AS item_index
        FROM comments
        WHERE post = r.post_id
          AND (r.parent_id = 0 AND parent_comment IS NULL OR parent_comment = NULLIF(r.parent_id, 0))
          AND id > s.after_id
        ORDER BY id
        LIMIT s.page_size + 1
    ) c ON TRUE
    WHERE (r.parent_id = 0 OR parent.id IS NOT NULL)
      AND (s.after_id = 0 OR s.cursor_post_id = r.post_id AND s.cursor_parent_id = r.parent_id)

    UNION ALL

    SELECT s.selection_id, s.page_size, t.post_id, t.id AS parent_id,
           COALESCE(c.item_index, 0) AS item_index,
           COALESCE(c.id, 0) AS id, c.parent_comment,
           COALESCE(c.content, '') AS content, COALESCE(c.author, 0) AS author,
           COALESCE(c.created_at, NOW()) AS created_at
    FROM tree t
    JOIN selections s ON s.parent_selection_id = t.selection_id
    LEFT JOIN LATERAL (
        SELECT id, parent_comment, content, author, created_at,
               ROW_NUMBER() OVER (ORDER BY id) AS item_index
        FROM comments
        WHERE post = t.post_id AND parent_comment = t.id AND id > s.after_id
        ORDER BY id
        LIMIT s.page_size + 1
    ) c ON TRUE
    WHERE t.id <> 0 AND t.item_index <= t.page_size
      AND (s.after_id = 0 OR s.cursor_post_id = t.post_id AND s.cursor_parent_id = t.id)
)
SELECT selection_id::integer, post_id::bigint, parent_id::bigint,
       id, parent_comment, content, author, created_at
FROM tree
ORDER BY selection_id, post_id, parent_id, id
`

type LoadCommentTreeParams struct {
	SelectionIds       []int32
	ParentSelectionIds []int32
	PageSizes          []int32
	AfterIds           []int64
	CursorPostIds      []int64
	CursorParentIds    []int64
	PostIds            []int64
	ParentIds          []int64
}

type LoadCommentTreeRow struct {
	SelectionID   int32
	PostID        int64
	ParentID      int64
	ID            int64
	ParentComment pgtype.Int8
	Content       string
	Author        int64
	CreatedAt     pgtype.Timestamptz
}

func (q *Queries) LoadCommentTree(ctx context.Context, arg LoadCommentTreeParams) ([]LoadCommentTreeRow, error) {
	rows, err := q.db.Query(ctx, loadCommentTree,
		arg.SelectionIds,
		arg.ParentSelectionIds,
		arg.PageSizes,
		arg.AfterIds,
		arg.CursorPostIds,
		arg.CursorParentIds,
		arg.PostIds,
		arg.ParentIds,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []LoadCommentTreeRow
	for rows.Next() {
		var i LoadCommentTreeRow
		if err := rows.Scan(
			&i.SelectionID,
			&i.PostID,
			&i.ParentID,
			&i.ID,
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
