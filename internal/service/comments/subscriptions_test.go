package comments

import (
	"context"
	"errors"
	"testing"

	"client-services/internal/domain"
)

func TestPublishOnlyAfterSuccessfulStore(t *testing.T) {
	broken := errors.New("commit failed")
	for _, storeErr := range []error{nil, broken} {
		saved, published := false, 0
		r := repositoryStub{add: func(context.Context, domain.Comment) (*domain.Comment, error) {
			saved = storeErr == nil
			return &domain.Comment{ID: 20, PostID: 1, AuthorID: 2}, storeErr
		}}
		events := eventsStub{publish: func(c domain.Comment) {
			published++
			if !saved || c.ID != 20 || c.PostID != 1 {
				t.Fatal("event published before a successful repository return")
			}
		}}
		s := NewService(r, postReaderFunc(func(context.Context, int64) (*domain.Post, error) {
			return &domain.Post{ID: 1, CommentsPermitted: true}, nil
		}), events)
		_, err := s.AddComment(t.Context(), 2, 1, nil, "text")
		if !errors.Is(err, storeErr) {
			t.Fatal(err)
		}
		want := 0
		if storeErr == nil {
			want = 1
		}
		if published != want {
			t.Fatal(published)
		}
	}
}
func TestSubscriptionValidatesExistingPost(t *testing.T) {
	for _, tc := range []struct {
		post         int64
		lookup, want error
	}{{1, nil, nil}, {0, nil, domain.ErrInvalidArgument}, {2, domain.ErrNotFound, domain.ErrNotFound}} {
		calls := 0
		stream := make(chan domain.Comment)
		s := NewService(nil, postReaderFunc(func(context.Context, int64) (*domain.Post, error) { return &domain.Post{}, tc.lookup }), eventsStub{subscribe: func(_ context.Context, id int64) <-chan domain.Comment {
			calls++
			if id != tc.post {
				t.Fatal(id)
			}
			return stream
		}})
		got, err := s.SubscribeComments(t.Context(), tc.post)
		if !errors.Is(err, tc.want) {
			t.Fatal(err)
		}
		if tc.want == nil && (got != stream || calls != 1) {
			t.Fatal("subscription not registered")
		}
		if tc.want != nil && calls != 0 {
			t.Fatal("invalid subscription registered")
		}
	}
}
