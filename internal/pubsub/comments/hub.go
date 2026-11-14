package comments

import (
	"context"
	"sync"

	"client-services/internal/domain"
)

const bufferSize = 64

type Hub struct {
	mu     sync.Mutex
	posts  map[int64]map[*subscription]struct{}
	closed bool
}

type subscription struct {
	comments chan domain.Comment
	stop     func() bool
}

func New() *Hub { return &Hub{posts: make(map[int64]map[*subscription]struct{})} }

func (h *Hub) SubscribeComments(ctx context.Context, postID int64) <-chan domain.Comment {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := &subscription{comments: make(chan domain.Comment, bufferSize)}
	if h.closed || ctx.Err() != nil {
		close(s.comments)
		return s.comments
	}
	if h.posts[postID] == nil {
		h.posts[postID] = make(map[*subscription]struct{})
	}
	h.posts[postID][s] = struct{}{}
	s.stop = context.AfterFunc(ctx, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.remove(postID, s)
	})
	return s.comments
}

func (h *Hub) PublishCommentAdded(comment domain.Comment) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.posts[comment.PostID] {
		copy := comment
		if comment.ParentCommentID != nil {
			id := *comment.ParentCommentID
			copy.ParentCommentID = &id
		}
		select {
		case s.comments <- copy:
		default:
			h.remove(comment.PostID, s)
		}
	}
}

func (h *Hub) remove(postID int64, s *subscription) {
	if _, ok := h.posts[postID][s]; !ok {
		return
	}
	delete(h.posts[postID], s)
	if len(h.posts[postID]) == 0 {
		delete(h.posts, postID)
	}
	s.stop()
	close(s.comments)
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for postID, subscribers := range h.posts {
		for s := range subscribers {
			h.remove(postID, s)
		}
	}
}
