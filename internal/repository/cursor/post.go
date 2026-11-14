package cursor

import (
	"fmt"

	"client-services/internal/domain"
)

const postKind = "post"

type postCursor struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

func EncodePost(id int64) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("%w: invalid post cursor ID", domain.ErrInvalidArgument)
	}
	return encode(postCursor{Kind: postKind, ID: id})
}

func DecodePost(after *string) (int64, error) {
	if after == nil {
		return 0, nil
	}
	var value postCursor
	if err := decode(*after, &value); err != nil {
		return 0, err
	}
	if value.Kind != postKind {
		return 0, fmt.Errorf("%w: expected a post cursor", domain.ErrInvalidArgument)
	}
	if value.ID <= 0 {
		return 0, fmt.Errorf("%w: invalid post cursor ID", domain.ErrInvalidArgument)
	}
	return value.ID, nil
}
