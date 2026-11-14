package cursor

import (
	"fmt"

	"client-services/internal/domain"
)

const commentKind = "comment"

type commentCursor struct {
	Kind            string `json:"kind"`
	ID              int64  `json:"id"`
	PostID          int64  `json:"post"`
	ParentCommentID *int64 `json:"parent"`
}

func EncodeComment(id, postID int64, parentCommentID *int64) (string, error) {
	value := commentCursor{Kind: commentKind, ID: id, PostID: postID, ParentCommentID: parentCommentID}
	if !value.valid() {
		return "", fmt.Errorf("%w: invalid comment cursor values", domain.ErrInvalidArgument)
	}
	return encode(value)
}

func DecodeComment(after *string, postID int64, parentCommentID *int64) (int64, error) {
	value, err := ParseComment(after)
	if err != nil {
		return 0, err
	}
	if after == nil {
		return 0, nil
	}
	if value.PostID != postID || !sameParent(value.ParentCommentID, parentCommentID) {
		return 0, fmt.Errorf("%w: cursor belongs to a different list", domain.ErrInvalidArgument)
	}
	return value.ID, nil
}

type CommentPosition struct {
	ID              int64
	PostID          int64
	ParentCommentID *int64
}

func ParseComment(after *string) (CommentPosition, error) {
	if after == nil {
		return CommentPosition{}, nil
	}
	var value commentCursor
	if err := decode(*after, &value); err != nil {
		return CommentPosition{}, err
	}
	if value.Kind != commentKind {
		return CommentPosition{}, fmt.Errorf("%w: expected a comment cursor", domain.ErrInvalidArgument)
	}
	if !value.valid() {
		return CommentPosition{}, fmt.Errorf("%w: invalid comment cursor values", domain.ErrInvalidArgument)
	}
	return CommentPosition{ID: value.ID, PostID: value.PostID, ParentCommentID: value.ParentCommentID}, nil
}

func (c commentCursor) valid() bool {
	return c.ID > 0 && c.PostID > 0 && (c.ParentCommentID == nil || *c.ParentCommentID > 0)
}

func sameParent(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
