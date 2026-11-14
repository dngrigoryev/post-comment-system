-- name: AddComment :one
INSERT INTO comments (post, parent_comment, content, author)
VALUES (
    sqlc.arg(post_id),
    sqlc.narg(parent_comment_id),
    sqlc.arg(content),
    sqlc.arg(author_id)
)
RETURNING id, post, parent_comment, content, author, created_at;

-- name: GetComment :one
SELECT id, post, parent_comment, content, author, created_at
FROM comments
WHERE id = sqlc.arg(comment_id);

-- name: ListPostComments :many
SELECT id, post, parent_comment, content, author, created_at
FROM comments
WHERE post = sqlc.arg(post_id)
  AND parent_comment IS NULL
  AND id > sqlc.arg(after_id)
ORDER BY id ASC
LIMIT sqlc.arg(page_limit);

-- name: ListPostCommentsBatch :many
WITH requests AS (
    SELECT GENERATE_SUBSCRIPTS(sqlc.arg(post_ids)::bigint[], 1) AS request_index,
           UNNEST(sqlc.arg(post_ids)) AS post_id,
           UNNEST(sqlc.arg(after_ids)::bigint[]) AS after_id,
           UNNEST(sqlc.arg(page_limits)::integer[]) AS page_limit
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
ORDER BY r.request_index, c.id;

-- name: ListRepliesBatch :many
WITH requests AS (
    SELECT GENERATE_SUBSCRIPTS(sqlc.arg(parent_ids)::bigint[], 1) AS request_index,
           UNNEST(sqlc.arg(parent_ids)) AS parent_id,
           UNNEST(sqlc.arg(post_ids)::bigint[]) AS post_id,
           UNNEST(sqlc.arg(after_ids)::bigint[]) AS after_id,
           UNNEST(sqlc.arg(page_limits)::integer[]) AS page_limit
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
ORDER BY r.request_index, c.id;

-- name: ListReplies :many
SELECT id, post, parent_comment, content, author, created_at
FROM comments
WHERE parent_comment = sqlc.arg(parent_comment_id)
  AND id > sqlc.arg(after_id)
ORDER BY id ASC
LIMIT sqlc.arg(page_limit);
