package comments

import (
	"context"
	"sync"
	"testing"
	"time"

	"client-services/internal/domain"
)

func receive(t *testing.T, ch <-chan domain.Comment) domain.Comment {
	t.Helper()
	select {
	case c, ok := <-ch:
		if !ok {
			t.Fatal("subscription closed")
		}
		return c
	case <-time.After(2 * time.Second):
		t.Fatal("missing event")
		return domain.Comment{}
	}
}
func TestFanoutFilteringAndCopyIsolation(t *testing.T) {
	h := New()
	defer h.Close()
	a := h.SubscribeComments(t.Context(), 1)
	b := h.SubscribeComments(t.Context(), 1)
	other := h.SubscribeComments(t.Context(), 2)
	parent := int64(7)
	h.PublishCommentAdded(domain.Comment{ID: 10, PostID: 1, ParentCommentID: &parent, Content: "text"})
	parent = 999
	one, two := receive(t, a), receive(t, b)
	if one.ID != 10 || two.ID != 10 || *one.ParentCommentID != 7 || *two.ParentCommentID != 7 {
		t.Fatal(one, two)
	}
	*one.ParentCommentID = 888
	if *two.ParentCommentID != 7 {
		t.Fatal("subscriber payloads share pointers")
	}
	select {
	case <-other:
		t.Fatal("event sent to another post")
	default:
	}
}
func TestCancellationAndClose(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(t.Context())
	ch := h.SubscribeComments(ctx, 1)
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unexpected event")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled subscription not removed")
	}
	closedCtx, cancel := context.WithCancel(t.Context())
	cancel()
	closed := h.SubscribeComments(closedCtx, 1)
	if _, ok := <-closed; ok {
		t.Fatal("cancelled context accepted")
	}
	ch = h.SubscribeComments(t.Context(), 1)
	h.Close()
	h.Close()
	if _, ok := <-ch; ok {
		t.Fatal("hub close did not close subscription")
	}
	if _, ok := <-h.SubscribeComments(t.Context(), 1); ok {
		t.Fatal("subscription after hub close")
	}
	h.PublishCommentAdded(domain.Comment{PostID: 1})
}
func TestSlowSubscriberDoesNotBlockOthers(t *testing.T) {
	h := New()
	defer h.Close()
	slow := h.SubscribeComments(t.Context(), 1)
	fast := h.SubscribeComments(t.Context(), 1)
	for i := 0; i < bufferSize+2; i++ {
		h.PublishCommentAdded(domain.Comment{ID: int64(i + 1), PostID: 1})
		if got := receive(t, fast); got.ID != int64(i+1) {
			t.Fatal(got)
		}
	}
	count := 0
	for range slow {
		count++
	}
	if count != bufferSize {
		t.Fatalf("slow subscriber queued %d comments", count)
	}
}
func TestConcurrentPublishSubscribeCancel(t *testing.T) {
	h := New()
	defer h.Close()
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(t.Context())
			ch := h.SubscribeComments(ctx, 1)
			cancel()
			for range ch {
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				h.PublishCommentAdded(domain.Comment{PostID: 1})
			}
		}()
	}
	wg.Wait()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.posts) != 0 {
		t.Fatal("cancelled subscriptions retained")
	}
}
