package loader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"client-services/internal/domain"
)

type usersFunc func(context.Context, []int64) ([]*domain.User, []error)

func (f usersFunc) GetUsersBatch(ctx context.Context, ids []int64) ([]*domain.User, []error) {
	return f(ctx, ids)
}

func TestBatchingCacheAndClear(t *testing.T) {
	calls := 0
	name := "before"
	users := usersFunc(func(_ context.Context, ids []int64) ([]*domain.User, []error) {
		calls++
		values := make([]*domain.User, len(ids))
		for i, id := range ids {
			values[i] = &domain.User{ID: id, Name: name}
		}
		return values, make([]error, len(ids))
	})
	l := newLoaders(users, nil)
	ctx := context.WithValue(t.Context(), contextKey{}, l)
	values, err := l.users.LoadAll(ctx, []int64{3, 1, 3, 2})
	if err != nil || len(values) != 4 || values[0].ID != 3 || values[1].ID != 1 || calls != 1 {
		t.Fatal(values, err, calls)
	}
	name = "after"
	u, err := User(ctx, 3)
	if err != nil || u.Name != "before" || calls != 1 {
		t.Fatal(u, err, calls)
	}
	Clear(ctx)
	u, err = User(ctx, 3)
	if err != nil || u.Name != "after" || calls != 2 {
		t.Fatal(u, err, calls)
	}
	if _, err := User(t.Context(), 1); err == nil {
		t.Fatal("missing loaders accepted")
	}
}

func TestRequestIsolation(t *testing.T) {
	calls := 0
	users := usersFunc(func(_ context.Context, ids []int64) ([]*domain.User, []error) {
		calls++
		return []*domain.User{{ID: ids[0], Name: fmt.Sprint(calls)}}, []error{nil}
	})
	h := Middleware(users, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := User(r.Context(), 1)
		if err != nil {
			t.Error(err)
			return
		}
		_, err = User(r.Context(), 1)
		if err != nil {
			t.Error(err)
		}
		fmt.Fprint(w, u.Name)
	}))
	for i := 1; i <= 2; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/query", nil))
		if rec.Body.String() != fmt.Sprint(i) {
			t.Fatal(rec.Body.String())
		}
	}
	if calls != 2 {
		t.Fatal("cache crossed request boundaries")
	}
}
func TestPageKeyUsesCursorValues(t *testing.T) {
	a, b := "same", "same"
	if newPageKey(2, &a) != newPageKey(2, &b) {
		t.Fatal("pointer addresses affect cache key")
	}
	empty := ""
	if newPageKey(2, nil) == newPageKey(2, &empty) {
		t.Fatal("omitted cursor collides with invalid empty cursor")
	}
	key := newPageKey(2, &a)
	p := key.request()
	if p.First != 2 || p.After == nil || *p.After != a {
		t.Fatal(p)
	}
}

type commentStub struct {
	CommentService
	roots   func(context.Context, []domain.PostCommentsRequest) ([]*domain.CommentPage, []error)
	replies func(context.Context, []domain.RepliesRequest) ([]*domain.CommentPage, []error)
}

func (s commentStub) ListPostCommentsBatch(ctx context.Context, r []domain.PostCommentsRequest) ([]*domain.CommentPage, []error) {
	return s.roots(ctx, r)
}
func (s commentStub) ListRepliesBatch(ctx context.Context, r []domain.RepliesRequest) ([]*domain.CommentPage, []error) {
	return s.replies(ctx, r)
}

func TestCommentPageBatchesAndInvalidation(t *testing.T) {
	rootCalls, replyCalls := 0, 0
	comments := commentStub{
		roots: func(_ context.Context, r []domain.PostCommentsRequest) ([]*domain.CommentPage, []error) {
			rootCalls++
			pages := make([]*domain.CommentPage, len(r))
			for i, request := range r {
				pages[i] = &domain.CommentPage{Nodes: []domain.Comment{{ID: request.PostID, Content: fmt.Sprint(rootCalls)}}}
			}
			return pages, make([]error, len(r))
		},
		replies: func(_ context.Context, r []domain.RepliesRequest) ([]*domain.CommentPage, []error) {
			replyCalls++
			pages := make([]*domain.CommentPage, len(r))
			for i, request := range r {
				pages[i] = &domain.CommentPage{Nodes: []domain.Comment{{ID: request.ParentCommentID, Content: fmt.Sprint(replyCalls)}}}
			}
			return pages, make([]error, len(r))
		},
	}
	users := usersFunc(func(context.Context, []int64) ([]*domain.User, []error) {
		t.Error("unexpected user lookup while loading comment pages")
		return nil, nil
	})
	l := newLoaders(users, comments)
	ctx := context.WithValue(t.Context(), contextKey{}, l)
	values, err := l.postComments.LoadAll(ctx, []postCommentsKey{{PostID: 1, pageKey: newPageKey(2, nil)}, {PostID: 2, pageKey: newPageKey(2, nil)}})
	if err != nil || rootCalls != 1 || values[1].Nodes[0].ID != 2 {
		t.Fatal(values, err, rootCalls)
	}
	_, err = l.replies.LoadAll(ctx, []repliesKey{{PostID: 1, ParentCommentID: 10, pageKey: newPageKey(2, nil)}, {PostID: 1, ParentCommentID: 20, pageKey: newPageKey(2, nil)}})
	if err != nil || replyCalls != 1 {
		t.Fatal(err, replyCalls)
	}
	_, err = PostComments(ctx, 1, 2, nil)
	if err != nil || rootCalls != 1 {
		t.Fatal(err, rootCalls)
	}
	_, err = Replies(ctx, 1, 10, 2, nil)
	if err != nil || replyCalls != 1 {
		t.Fatal(err, replyCalls)
	}
	Clear(ctx)
	page, err := PostComments(ctx, 1, 2, nil)
	if err != nil || page.Nodes[0].Content != "2" {
		t.Fatal(page, err)
	}
	page, err = Replies(ctx, 1, 10, 2, nil)
	if err != nil || page.Nodes[0].Content != "2" {
		t.Fatal(page, err)
	}
	if _, err := PostComments(t.Context(), 1, 2, nil); err == nil {
		t.Fatal("missing loaders accepted")
	}
	if _, err := Replies(t.Context(), 1, 10, 2, nil); err == nil {
		t.Fatal("missing loaders accepted")
	}
}
