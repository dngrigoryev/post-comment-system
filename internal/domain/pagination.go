package domain

type PageRequest struct {
	First int
	After *string
}

type PageInfo struct {
	EndCursor   *string
	HasNextPage bool
}

type PostCommentsRequest struct {
	PostID int64
	Page   PageRequest
}

type RepliesRequest struct {
	PostID          int64
	ParentCommentID int64
	Page            PageRequest
}
