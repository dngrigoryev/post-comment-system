package posts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/testutil/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func postValues(id int64, enabled bool) []any {
	return []any{id, "title", "content", int64(7), enabled, pgtype.Timestamptz{Time: time.Unix(123, 0), Valid: true}}
}
func TestCreateAndGetPost(t *testing.T) {
	db := &pgtest.DB{RowFunc: func(_ context.Context, s string, a ...any) pgx.Row {
		if strings.Contains(s, "CreatePost") {
			if len(a) != 4 || a[0] != "title" || a[2] != int64(7) || a[3] != true {
				t.Fatalf("arguments %v", a)
			}
		} else if a[0] != int64(1<<40) {
			t.Fatalf("arguments %v", a)
		}
		return pgtest.Row{Values: postValues(1<<40, true)}
	}}
	r := New(db)
	p, err := r.CreatePost(t.Context(), domain.Post{Title: "title", Content: "content", AuthorID: 7, CommentsPermitted: true})
	if err != nil || p.ID != 1<<40 || p.AuthorID != 7 || !p.CommentsPermitted || p.CreatedAt.Unix() != 123 {
		t.Fatal(p, err)
	}
	p, err = r.GetPost(t.Context(), p.ID)
	if err != nil || p.Title != "title" {
		t.Fatal(p, err)
	}
	db.RowFunc = func(context.Context, string, ...any) pgx.Row { return pgtest.Row{Err: pgx.ErrNoRows} }
	_, err = r.GetPost(t.Context(), 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}
func TestListPostsKeysetAndExtraRow(t *testing.T) {
	after, _ := cursor.EncodePost(1 << 40)
	db := &pgtest.DB{QueryFunc: func(_ context.Context, s string, a ...any) (pgx.Rows, error) {
		if a[0] != int64(1<<40) || a[1] != int32(3) {
			t.Fatalf("arguments %v", a)
		}
		return pgtest.NewRows(postValues(1<<40+1, true), postValues(1<<40+2, true), postValues(1<<40+3, true)), nil
	}}
	r := New(db)
	page, err := r.ListPosts(t.Context(), domain.PageRequest{First: 2, After: &after})
	if err != nil || len(page.Nodes) != 2 || !page.PageInfo.HasNextPage {
		t.Fatal(page, err)
	}
	id, err := cursor.DecodePost(page.PageInfo.EndCursor)
	if err != nil || id != 1<<40+2 {
		t.Fatal(id, err)
	}
	db.QueryFunc = func(context.Context, string, ...any) (pgx.Rows, error) { return pgtest.NewRows(), nil }
	page, err = r.ListPosts(t.Context(), domain.PageRequest{First: 2})
	if err != nil || page.Nodes == nil || len(page.Nodes) != 0 || page.PageInfo.EndCursor != nil {
		t.Fatal(page, err)
	}
	_, err = r.ListPosts(t.Context(), domain.PageRequest{First: 0})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
}
func TestDisableDistinguishesMissingAndForbidden(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		updateErr, lookupErr, want error
	}{{name: "owner"}, {name: "other owner", updateErr: pgx.ErrNoRows, want: domain.ErrForbidden}, {name: "missing", updateErr: pgx.ErrNoRows, lookupErr: pgx.ErrNoRows, want: domain.ErrNotFound}} {
		t.Run(tc.name, func(t *testing.T) {
			db := &pgtest.DB{RowFunc: func(_ context.Context, s string, a ...any) pgx.Row {
				if strings.Contains(s, "DisableComments") {
					if a[0] != int64(10) || a[1] != int64(7) {
						t.Fatalf("arguments %v", a)
					}
					return pgtest.Row{Values: postValues(10, false), Err: tc.updateErr}
				}
				return pgtest.Row{Values: postValues(10, true), Err: tc.lookupErr}
			}}
			p, err := New(db).DisableComments(t.Context(), 7, 10)
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			if err == nil && p.CommentsPermitted {
				t.Fatal("comments still enabled")
			}
		})
	}
}
