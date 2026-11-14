-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    email VARCHAR(255) NOT NULL CHECK (email LIKE '%@%.%')
);

CREATE TABLE posts (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    author BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comments_permitted BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    post BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_comment BIGINT,
    content VARCHAR(2000) NOT NULL,
    author BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (post, id),
    FOREIGN KEY (post, parent_comment) REFERENCES comments(post, id) ON DELETE CASCADE
);

CREATE INDEX ON comments (post, parent_comment, id);
CREATE INDEX ON comments (parent_comment, id);

-- +goose Down
DROP TABLE comments;
DROP TABLE posts;
DROP TABLE users;
