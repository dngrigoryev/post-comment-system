package loader

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"client-services/internal/domain"

	"github.com/vikstrous/dataloadgen"
)

type UserService interface {
	GetUsersBatch(ctx context.Context, ids []int64) ([]*domain.User, []error)
}

type CommentService interface {
	LoadCommentTree(ctx context.Context, request domain.CommentTreeRequest) ([]domain.CommentTreePage, error)
	ListPostCommentsBatch(ctx context.Context, requests []domain.PostCommentsRequest) ([]*domain.CommentPage, []error)
	ListRepliesBatch(ctx context.Context, requests []domain.RepliesRequest) ([]*domain.CommentPage, []error)
}

type loaders struct {
	userService    UserService
	commentService CommentService
	users          *dataloadgen.Loader[int64, *domain.User]
	postComments   *dataloadgen.Loader[postCommentsKey, *domain.CommentPage]
	replies        *dataloadgen.Loader[repliesKey, *domain.CommentPage]
}

type pageKey struct {
	First    int
	After    string
	HasAfter bool
}

type postCommentsKey struct {
	PostID int64
	pageKey
}

type repliesKey struct {
	PostID          int64
	ParentCommentID int64
	pageKey
}

func newPageKey(first int, after *string) pageKey {
	key := pageKey{First: first, HasAfter: after != nil}
	if after != nil {
		key.After = *after
	}
	return key
}

func (key pageKey) request() domain.PageRequest {
	page := domain.PageRequest{First: key.First}
	if key.HasAfter {
		page.After = &key.After
	}
	return page
}

func newLoaders(users UserService, comments CommentService) *loaders {
	options := []dataloadgen.Option{
		dataloadgen.WithWait(time.Millisecond),
		dataloadgen.WithBatchCapacity(100),
	}
	return &loaders{
		userService:    users,
		commentService: comments,
		users:          dataloadgen.NewLoader(users.GetUsersBatch, options...),
		postComments: dataloadgen.NewLoader(func(ctx context.Context, keys []postCommentsKey) ([]*domain.CommentPage, []error) {
			requests := make([]domain.PostCommentsRequest, len(keys))
			for i, key := range keys {
				requests[i] = domain.PostCommentsRequest{PostID: key.PostID, Page: key.request()}
			}
			return comments.ListPostCommentsBatch(ctx, requests)
		}, options...),
		replies: dataloadgen.NewLoader(func(ctx context.Context, keys []repliesKey) ([]*domain.CommentPage, []error) {
			requests := make([]domain.RepliesRequest, len(keys))
			for i, key := range keys {
				requests[i] = domain.RepliesRequest{PostID: key.PostID, ParentCommentID: key.ParentCommentID, Page: key.request()}
			}
			return comments.ListRepliesBatch(ctx, requests)
		}, options...),
	}
}

type contextKey struct{}

func WithContext(ctx context.Context, users UserService, comments CommentService) context.Context {
	return context.WithValue(ctx, contextKey{}, newLoaders(users, comments))
}

func Middleware(users UserService, comments CommentService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithContext(r.Context(), users, comments)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func fromContext(ctx context.Context) (*loaders, error) {
	value, ok := ctx.Value(contextKey{}).(*loaders)
	if !ok {
		return nil, fmt.Errorf("GraphQL loaders are not initialized")
	}
	return value, nil
}

func User(ctx context.Context, id int64) (*domain.User, error) {
	value, err := fromContext(ctx)
	if err != nil {
		return nil, err
	}
	return value.users.Load(ctx, id)
}

func PostComments(ctx context.Context, postID int64, first int, after *string) (*domain.CommentPage, error) {
	value, err := fromContext(ctx)
	if err != nil {
		return nil, err
	}
	return value.postComments.Load(ctx, postCommentsKey{PostID: postID, pageKey: newPageKey(first, after)})
}

func Replies(ctx context.Context, postID, parentCommentID int64, first int, after *string) (*domain.CommentPage, error) {
	value, err := fromContext(ctx)
	if err != nil {
		return nil, err
	}
	return value.replies.Load(ctx, repliesKey{PostID: postID, ParentCommentID: parentCommentID, pageKey: newPageKey(first, after)})
}

func Clear(ctx context.Context) {
	if value, ok := ctx.Value(contextKey{}).(*loaders); ok {
		value.users.ClearAll()
		value.postComments.ClearAll()
		value.replies.ClearAll()
	}
}
