package store

import (
	"sync"

	"client-services/internal/domain"
)

type State struct {
	Users map[int64]domain.User

	PostsMu    sync.RWMutex
	Posts      map[int64]domain.Post
	PostIDs    []int64
	NextPostID int64

	CommentsMu       sync.RWMutex
	Comments         map[int64]domain.Comment
	RootIDsByPost    map[int64][]int64
	ReplyIDsByParent map[int64][]int64
	NextCommentID    int64
}

func New(users ...domain.User) *State {
	r := &State{
		Users:            make(map[int64]domain.User, len(users)),
		Posts:            make(map[int64]domain.Post),
		Comments:         make(map[int64]domain.Comment),
		RootIDsByPost:    make(map[int64][]int64),
		ReplyIDsByParent: make(map[int64][]int64),
		NextPostID:       1,
		NextCommentID:    1,
	}
	for _, user := range users {
		r.Users[user.ID] = user
	}
	return r
}
