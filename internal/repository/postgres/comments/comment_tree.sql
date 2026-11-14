-- name: LoadCommentTree :many
WITH RECURSIVE selections AS (
    SELECT UNNEST(sqlc.arg(selection_ids)::integer[]) AS selection_id,
           UNNEST(sqlc.arg(parent_selection_ids)::integer[]) AS parent_selection_id,
           UNNEST(sqlc.arg(page_sizes)::integer[]) AS page_size,
           UNNEST(sqlc.arg(after_ids)::bigint[]) AS after_id,
           UNNEST(sqlc.arg(cursor_post_ids)::bigint[]) AS cursor_post_id,
           UNNEST(sqlc.arg(cursor_parent_ids)::bigint[]) AS cursor_parent_id
), roots AS (
    SELECT UNNEST(sqlc.arg(post_ids)::bigint[]) AS post_id,
           UNNEST(sqlc.arg(parent_ids)::bigint[]) AS parent_id
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
ORDER BY selection_id, post_id, parent_id, id;
