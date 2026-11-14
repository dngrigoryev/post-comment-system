package domain

import "time"

type Post struct {
	ID                int64
	Title             string
	Content           string
	AuthorID          int64
	CommentsPermitted bool
	CreatedAt         time.Time
}

type PostPage struct {
	Nodes    []Post
	PageInfo PageInfo
}
