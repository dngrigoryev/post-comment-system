package domain

import "time"

type Comment struct {
	ID              int64
	PostID          int64
	ParentCommentID *int64
	Content         string
	AuthorID        int64
	CreatedAt       time.Time
}

type CommentPage struct {
	Nodes    []Comment
	PageInfo PageInfo
}
