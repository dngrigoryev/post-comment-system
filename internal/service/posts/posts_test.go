package posts

import (
	"context"
	"errors"
	"strings"
	"testing"

	"client-services/internal/domain"
)

type repositoryStub struct {
	Repository
	create  func(context.Context, domain.Post) (*domain.Post, error)
	get     func(context.Context, int64) (*domain.Post, error)
	list    func(context.Context, domain.PageRequest) (*domain.PostPage, error)
	disable func(context.Context, int64, int64) (*domain.Post, error)
}

func (r repositoryStub) CreatePost(ctx context.Context, p domain.Post) (*domain.Post, error) {
	return r.create(ctx, p)
}
func (r repositoryStub) GetPost(ctx context.Context, id int64) (*domain.Post, error) {
	return r.get(ctx, id)
}
func (r repositoryStub) ListPosts(ctx context.Context, p domain.PageRequest) (*domain.PostPage, error) {
	return r.list(ctx, p)
}
func (r repositoryStub) DisableComments(ctx context.Context, user, id int64) (*domain.Post, error) {
	return r.disable(ctx, user, id)
}

func TestCreatePost(t *testing.T) {
	ctx := t.Context()
	repo := repositoryStub{create: func(gotCtx context.Context, p domain.Post) (*domain.Post, error) {
		if gotCtx != ctx || p.AuthorID != 7 || p.Title != "Заголовок" || p.Content != "Текст" || !p.CommentsPermitted || p.ID != 0 {
			t.Fatalf("unexpected post: %+v", p)
		}
		p.ID = 10
		return &p, nil
	}}
	p, err := NewService(repo).CreatePost(ctx, 7, "Заголовок", "Текст")
	if err != nil || p.ID != 10 {
		t.Fatalf("post=%v err=%v", p, err)
	}
	for _, tc := range []struct {
		name           string
		user           int64
		title, content string
		want           error
	}{
		{"anonymous", 0, "title", "text", domain.ErrUnauthenticated},
		{"blank title", 7, "  ", "text", domain.ErrInvalidArgument},
		{"long title", 7, strings.Repeat("я", 256), "text", domain.ErrInvalidArgument},
		{"blank content", 7, "title", "\n", domain.ErrInvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewService(repositoryStub{}).CreatePost(t.Context(), tc.user, tc.title, tc.content)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

func TestDisableCommentsOwnership(t *testing.T) {
	for _, tc := range []struct {
		name         string
		user, author int64
		lookup, want error
		write        bool
	}{
		{"owner", 7, 7, nil, nil, true}, {"other user", 8, 7, nil, domain.ErrForbidden, false},
		{"missing", 7, 7, domain.ErrNotFound, domain.ErrNotFound, false},
		{"anonymous", 0, 7, nil, domain.ErrUnauthenticated, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			repo := repositoryStub{
				get: func(_ context.Context, id int64) (*domain.Post, error) {
					return &domain.Post{ID: id, AuthorID: tc.author}, tc.lookup
				},
				disable: func(_ context.Context, user, id int64) (*domain.Post, error) {
					called = true
					if user != tc.user || id != 10 {
						t.Fatal("wrong arguments")
					}
					return &domain.Post{ID: id}, nil
				},
			}
			_, err := NewService(repo).DisableComments(t.Context(), tc.user, 10)
			if !errors.Is(err, tc.want) || called != tc.write {
				t.Fatalf("err=%v write=%v", err, called)
			}
		})
	}
}

func TestReadAndRepositoryErrors(t *testing.T) {
	broken := errors.New("storage unavailable")
	s := NewService(repositoryStub{
		create: func(context.Context, domain.Post) (*domain.Post, error) { return nil, broken },
		get:    func(context.Context, int64) (*domain.Post, error) { return nil, broken },
		list:   func(context.Context, domain.PageRequest) (*domain.PostPage, error) { return nil, broken },
	})
	_, err := s.CreatePost(t.Context(), 1, "title", "text")
	if !errors.Is(err, broken) {
		t.Fatal(err)
	}
	_, err = s.GetPost(t.Context(), 1)
	if !errors.Is(err, broken) {
		t.Fatal(err)
	}
	_, err = s.GetPost(t.Context(), 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	_, err = s.ListPosts(t.Context(), domain.PageRequest{First: 1})
	if !errors.Is(err, broken) {
		t.Fatal(err)
	}
	_, err = s.ListPosts(t.Context(), domain.PageRequest{First: 101})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	s = NewService(repositoryStub{list: func(_ context.Context, p domain.PageRequest) (*domain.PostPage, error) {
		if p.First != 2 {
			t.Fatal(p)
		}
		return &domain.PostPage{Nodes: []domain.Post{{ID: 1}}}, nil
	}})
	page, err := s.ListPosts(t.Context(), domain.PageRequest{First: 2})
	if err != nil || len(page.Nodes) != 1 {
		t.Fatalf("%v %v", page, err)
	}
}
