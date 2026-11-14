-- name: CreatePost :one
INSERT INTO posts (title, content, author, comments_permitted)
VALUES (
    sqlc.arg(title),
    sqlc.arg(content),
    sqlc.arg(author_id),
    sqlc.arg(comments_permitted)
)
RETURNING id, title, content, author, comments_permitted, created_at;

-- name: GetPost :one
SELECT id, title, content, author, comments_permitted, created_at
FROM posts
WHERE id = sqlc.arg(post_id);

-- name: GetPostForShare :one
SELECT id, title, content, author, comments_permitted, created_at
FROM posts
WHERE id = sqlc.arg(post_id)
FOR SHARE;

-- name: ListPosts :many
SELECT id, title, content, author, comments_permitted, created_at
FROM posts
WHERE id > sqlc.arg(after_id)
ORDER BY id ASC
LIMIT sqlc.arg(page_limit);

-- name: DisableComments :one
UPDATE posts
SET comments_permitted = FALSE
WHERE id = sqlc.arg(post_id)
  AND author = sqlc.arg(user_id)
RETURNING id, title, content, author, comments_permitted, created_at;
