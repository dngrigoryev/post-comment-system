package comments

import (
	"context"
	"errors"
	"strings"
	"testing"

	"client-services/internal/domain"
)

type postReaderFunc func(context.Context, int64) (*domain.Post, error)

func (f postReaderFunc) GetPost(ctx context.Context, id int64) (*domain.Post, error) {
	return f(ctx, id)
}

type repositoryStub struct {
	Repository
	add        func(context.Context, domain.Comment) (*domain.Comment, error)
	get        func(context.Context, int64) (*domain.Comment, error)
	roots      func(context.Context, int64, domain.PageRequest) (*domain.CommentPage, error)
	replies    func(context.Context, int64, domain.PageRequest) (*domain.CommentPage, error)
	rootBatch  func(context.Context, []domain.PostCommentsRequest) ([]*domain.CommentPage, []error)
	replyBatch func(context.Context, []domain.RepliesRequest) ([]*domain.CommentPage, []error)
	tree       func(context.Context, domain.CommentTreeRequest) ([]domain.CommentTreePage, error)
}

func (r repositoryStub) AddComment(ctx context.Context, c domain.Comment) (*domain.Comment, error) {
	return r.add(ctx, c)
}
func (r repositoryStub) GetComment(ctx context.Context, id int64) (*domain.Comment, error) {
	return r.get(ctx, id)
}
func (r repositoryStub) ListPostComments(ctx context.Context, id int64, p domain.PageRequest) (*domain.CommentPage, error) {
	return r.roots(ctx, id, p)
}
func (r repositoryStub) ListReplies(ctx context.Context, id int64, p domain.PageRequest) (*domain.CommentPage, error) {
	return r.replies(ctx, id, p)
}
func (r repositoryStub) ListPostCommentsBatch(ctx context.Context, p []domain.PostCommentsRequest) ([]*domain.CommentPage, []error) {
	return r.rootBatch(ctx, p)
}
func (r repositoryStub) ListRepliesBatch(ctx context.Context, p []domain.RepliesRequest) ([]*domain.CommentPage, []error) {
	return r.replyBatch(ctx, p)
}
func (r repositoryStub) LoadCommentTree(ctx context.Context, p domain.CommentTreeRequest) ([]domain.CommentTreePage, error) {
	return r.tree(ctx, p)
}

func TestAddCommentBusinessRules(t *testing.T) {
	parent := int64(9)
	badParent := int64(0)
	broken := errors.New("offline")
	for _, tc := range []struct {
		name                               string
		user, post                         int64
		parent                             *int64
		text                               string
		disabled, foreign                  bool
		postErr, parentErr, writeErr, want error
	}{
		{name: "root", user: 1, post: 2, text: "text"},
		{name: "reply", user: 1, post: 2, parent: &parent, text: strings.Repeat("я", 2000)},
		{name: "anonymous", post: 2, text: "text", want: domain.ErrUnauthenticated},
		{name: "invalid post", user: 1, text: "text", want: domain.ErrInvalidArgument},
		{name: "invalid parent", user: 1, post: 2, parent: &badParent, text: "text", want: domain.ErrInvalidArgument},
		{name: "blank", user: 1, post: 2, text: " ", want: domain.ErrInvalidArgument},
		{name: "too long", user: 1, post: 2, text: strings.Repeat("🙂", 2001), want: domain.ErrInvalidArgument},
		{name: "disabled", user: 1, post: 2, text: "text", disabled: true, want: domain.ErrCommentsDisabled},
		{name: "foreign parent", user: 1, post: 2, text: "text", parent: &parent, foreign: true, want: domain.ErrInvalidArgument},
		{name: "missing post", user: 1, post: 2, text: "text", postErr: domain.ErrNotFound, want: domain.ErrNotFound},
		{name: "missing parent", user: 1, post: 2, text: "text", parent: &parent, parentErr: domain.ErrNotFound, want: domain.ErrNotFound},
		{name: "write error", user: 1, post: 2, text: "text", writeErr: broken, want: broken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			repo := repositoryStub{
				get: func(_ context.Context, id int64) (*domain.Comment, error) {
					postID := tc.post
					if tc.foreign {
						postID++
					}
					return &domain.Comment{ID: id, PostID: postID}, tc.parentErr
				},
				add: func(_ context.Context, c domain.Comment) (*domain.Comment, error) {
					writes++
					if c.AuthorID != tc.user || c.PostID != tc.post || c.Content != tc.text || c.ParentCommentID != tc.parent {
						t.Fatalf("wrong comment %+v", c)
					}
					c.ID = 10
					return &c, tc.writeErr
				},
			}
			s := NewService(repo, postReaderFunc(func(_ context.Context, id int64) (*domain.Post, error) {
				return &domain.Post{ID: id, CommentsPermitted: !tc.disabled}, tc.postErr
			}))
			c, err := s.AddComment(t.Context(), tc.user, tc.post, tc.parent, tc.text)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want == nil {
				if writes != 1 || c.ID != 10 {
					t.Fatal("comment not stored")
				}
			} else if tc.writeErr == nil && writes != 0 {
				t.Fatal("invalid comment stored")
			}
		})
	}
}

func TestReadPages(t *testing.T) {
	broken := errors.New("offline")
	for _, reply := range []bool{false, true} {
		for _, storageErr := range []error{nil, broken} {
			repo := repositoryStub{
				get: func(context.Context, int64) (*domain.Comment, error) { return &domain.Comment{ID: 1}, nil },
				roots: func(_ context.Context, id int64, p domain.PageRequest) (*domain.CommentPage, error) {
					return &domain.CommentPage{Nodes: []domain.Comment{{ID: 3}}}, storageErr
				},
				replies: func(_ context.Context, id int64, p domain.PageRequest) (*domain.CommentPage, error) {
					return &domain.CommentPage{Nodes: []domain.Comment{{ID: 3}}}, storageErr
				},
			}
			s := NewService(repo, postReaderFunc(func(context.Context, int64) (*domain.Post, error) { return &domain.Post{}, nil }))
			read := s.ListPostComments
			if reply {
				read = s.ListReplies
			}
			page, err := read(t.Context(), 1, domain.PageRequest{First: 2})
			if !errors.Is(err, storageErr) {
				t.Fatal(err)
			}
			if err == nil && page.Nodes[0].ID != 3 {
				t.Fatal(page)
			}
			_, err = read(t.Context(), 0, domain.PageRequest{First: 2})
			if !errors.Is(err, domain.ErrInvalidArgument) {
				t.Fatal(err)
			}
			_, err = read(t.Context(), 1, domain.PageRequest{First: 0})
			if !errors.Is(err, domain.ErrInvalidArgument) {
				t.Fatal(err)
			}
		}
	}
	s := NewService(repositoryStub{get: func(context.Context, int64) (*domain.Comment, error) { return nil, domain.ErrNotFound }}, postReaderFunc(func(context.Context, int64) (*domain.Post, error) { return nil, domain.ErrNotFound }))
	_, err := s.ListPostComments(t.Context(), 1, domain.PageRequest{First: 1})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	_, err = s.ListReplies(t.Context(), 1, domain.PageRequest{First: 1})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestBatchIsolatesInvalidRequests(t *testing.T) {
	repo := repositoryStub{
		rootBatch: func(_ context.Context, r []domain.PostCommentsRequest) ([]*domain.CommentPage, []error) {
			if len(r) != 2 || r[0].PostID != 1 || r[1].PostID != 2 {
				t.Fatal(r)
			}
			return []*domain.CommentPage{{Nodes: []domain.Comment{{ID: 10}}}, nil}, []error{nil, domain.ErrNotFound}
		},
		replyBatch: func(_ context.Context, r []domain.RepliesRequest) ([]*domain.CommentPage, []error) {
			if len(r) != 1 || r[0].ParentCommentID != 3 {
				t.Fatal(r)
			}
			return []*domain.CommentPage{{}}, []error{nil}
		},
	}
	s := NewService(repo, nil)
	pages, errs := s.ListPostCommentsBatch(t.Context(), []domain.PostCommentsRequest{{PostID: 1, Page: domain.PageRequest{First: 1}}, {PostID: 0, Page: domain.PageRequest{First: 1}}, {PostID: 1, Page: domain.PageRequest{First: 101}}, {PostID: 2, Page: domain.PageRequest{First: 1}}})
	if len(pages) != 4 || pages[0].Nodes[0].ID != 10 || errs[0] != nil || !errors.Is(errs[1], domain.ErrInvalidArgument) || !errors.Is(errs[2], domain.ErrInvalidArgument) || !errors.Is(errs[3], domain.ErrNotFound) {
		t.Fatal(pages, errs)
	}
	_, errs = s.ListRepliesBatch(t.Context(), []domain.RepliesRequest{{PostID: 1, ParentCommentID: 3, Page: domain.PageRequest{First: 1}}, {PostID: 1, ParentCommentID: 0, Page: domain.PageRequest{First: 1}}, {PostID: 0, ParentCommentID: 3, Page: domain.PageRequest{First: 1}}})
	if errs[0] != nil || !errors.Is(errs[1], domain.ErrInvalidArgument) || !errors.Is(errs[2], domain.ErrInvalidArgument) {
		t.Fatal(errs)
	}
	pages, errs = s.ListPostCommentsBatch(t.Context(), nil)
	if len(pages) != 0 || len(errs) != 0 {
		t.Fatal("empty batch")
	}
}

func TestTreePlanPrunesInvalidBranches(t *testing.T) {
	s := NewService(repositoryStub{tree: func(_ context.Context, r domain.CommentTreeRequest) ([]domain.CommentTreePage, error) {
		if len(r.Selections) != 1 || r.Selections[0].ID != 1 || len(r.Selections[0].Replies) != 1 || r.Selections[0].Replies[0].ID != 3 {
			t.Fatalf("bad filtered plan %+v", r)
		}
		return []domain.CommentTreePage{{SelectionID: 3}}, nil
	}}, nil)
	r := domain.CommentTreeRequest{Roots: []domain.CommentTreeRoot{{PostID: 1}}, Selections: []domain.CommentSelection{{ID: 1, Page: domain.PageRequest{First: 1}, Replies: []domain.CommentSelection{{ID: 2, Page: domain.PageRequest{First: 0}}, {ID: 3, Page: domain.PageRequest{First: 1}}}}}}
	pages, err := s.LoadCommentTree(t.Context(), r)
	if err != nil || len(pages) != 1 {
		t.Fatal(pages, err)
	}
	if len(r.Selections[0].Replies) != 2 {
		t.Fatal("caller plan was mutated")
	}
	r.Selections[0].Replies[1].ID = 1
	_, err = s.LoadCommentTree(t.Context(), r)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	r.Roots[0].PostID = 0
	_, err = s.LoadCommentTree(t.Context(), r)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	pages, err = NewService(repositoryStub{}, nil).LoadCommentTree(t.Context(), domain.CommentTreeRequest{})
	if err != nil || len(pages) != 0 {
		t.Fatal(pages, err)
	}
}
