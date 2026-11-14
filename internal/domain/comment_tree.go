package domain

type CommentSelection struct {
	ID      int
	Page    PageRequest
	Replies []CommentSelection
}

type CommentTreeRoot struct {
	PostID          int64
	ParentCommentID *int64
}

type CommentTreeRequest struct {
	Roots      []CommentTreeRoot
	Selections []CommentSelection
}

type CommentTreePage struct {
	SelectionID     int
	PostID          int64
	ParentCommentID *int64
	Page            *CommentPage
}
