package comments

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/testutil/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func commentValues(id, post int64, parent pgtype.Int8) []any {
	return []any{id, post, parent, "text", int64(7), pgtype.Timestamptz{Time: time.Unix(123, 0), Valid: true}}
}
func batchValues(index int32, parent, id, post int64, p pgtype.Int8) []any {
	return append([]any{index, parent}, commentValues(id, post, p)...)
}

func TestAddCommentTransaction(t *testing.T) {
	broken := errors.New("offline")
	for _, tc := range []struct {
		name                                  string
		disabled                              bool
		lookupErr, insertErr, commitErr, want error
	}{
		{name: "root"}, {name: "disabled", disabled: true, want: domain.ErrCommentsDisabled},
		{name: "missing post", lookupErr: pgx.ErrNoRows, want: domain.ErrNotFound},
		{name: "foreign parent", insertErr: &pgconn.PgError{Code: "23503"}, want: domain.ErrInvalidArgument},
		{name: "insert failed", insertErr: broken, want: broken}, {name: "commit failed", commitErr: broken, want: broken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []string
			tx := &pgtest.Tx{DB: &pgtest.DB{}}
			tx.CommitFunc = func(context.Context) error { events = append(events, "commit"); return tc.commitErr }
			tx.RollbackFunc = func(context.Context) error { events = append(events, "rollback"); return nil }
			tx.RowFunc = func(_ context.Context, s string, a ...any) pgx.Row {
				if strings.Contains(s, "GetPostForShare") {
					events = append(events, "lock post")
					return pgtest.Row{Values: []any{int64(2), "post", "text", int64(7), !tc.disabled, pgtype.Timestamptz{}}, Err: tc.lookupErr}
				}
				events = append(events, "insert")
				if a[0] != int64(2) || a[1] != (pgtype.Int8{}) || a[2] != "text" || a[3] != int64(7) {
					t.Fatalf("arguments %v", a)
				}
				return pgtest.Row{Values: commentValues(10, 2, pgtype.Int8{}), Err: tc.insertErr}
			}
			db := &pgtest.DB{BeginFunc: func(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
				if o.IsoLevel != pgx.ReadCommitted {
					t.Fatal(o)
				}
				events = append(events, "begin")
				return tx, nil
			}}
			c, err := New(db).AddComment(t.Context(), domain.Comment{PostID: 2, Content: "text", AuthorID: 7})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			want := []string{"begin", "lock post"}
			if !tc.disabled && tc.lookupErr == nil {
				want = append(want, "insert")
				if tc.insertErr == nil {
					want = append(want, "commit")
				}
			}
			want = append(want, "rollback")
			if !reflect.DeepEqual(events, want) {
				t.Fatalf("events %v want %v", events, want)
			}
			if err == nil && (c.ID != 10 || c.ParentCommentID != nil || c.CreatedAt.Unix() != 123) {
				t.Fatal(c)
			}
		})
	}
	db := &pgtest.DB{BeginFunc: func(context.Context, pgx.TxOptions) (pgx.Tx, error) { return nil, broken }}
	_, err := New(db).AddComment(t.Context(), domain.Comment{})
	if !errors.Is(err, broken) {
		t.Fatal(err)
	}
}

func TestReadsAndScopedPagination(t *testing.T) {
	parent := pgtype.Int8{Int64: 9, Valid: true}
	db := &pgtest.DB{
		RowFunc: func(context.Context, string, ...any) pgx.Row {
			return pgtest.Row{Values: commentValues(9, 2, pgtype.Int8{})}
		},
		QueryFunc: func(_ context.Context, s string, a ...any) (pgx.Rows, error) {
			if a[1] != int64(0) || a[2] != int32(2) {
				t.Fatalf("arguments %v", a)
			}
			if strings.Contains(s, "ListReplies") {
				if a[0] != parent {
					t.Fatalf("arguments %v", a)
				}
				return pgtest.NewRows(commentValues(10, 2, parent), commentValues(11, 2, parent)), nil
			}
			return pgtest.NewRows(commentValues(9, 2, pgtype.Int8{})), nil
		},
	}
	r := New(db)
	c, err := r.GetComment(t.Context(), 9)
	if err != nil || c.ID != 9 || c.PostID != 2 || c.ParentCommentID != nil {
		t.Fatal(c, err)
	}
	page, err := r.ListReplies(t.Context(), 9, domain.PageRequest{First: 1})
	if err != nil || len(page.Nodes) != 1 || !page.PageInfo.HasNextPage || *page.Nodes[0].ParentCommentID != 9 {
		t.Fatal(page, err)
	}
	decoded, err := cursor.DecodeComment(page.PageInfo.EndCursor, 2, &parent.Int64)
	if err != nil || decoded != 10 {
		t.Fatal(decoded, err)
	}
	_, err = r.ListPostComments(t.Context(), 2, domain.PageRequest{First: 1, After: page.PageInfo.EndCursor})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	page, err = r.ListPostComments(t.Context(), 2, domain.PageRequest{First: 1})
	if err != nil || len(page.Nodes) != 1 || page.PageInfo.HasNextPage {
		t.Fatal(page, err)
	}
}

func TestBatchAlignmentAndEmptyPages(t *testing.T) {
	invalid := "bad cursor"
	calls := 0
	db := &pgtest.DB{QueryFunc: func(_ context.Context, s string, a ...any) (pgx.Rows, error) {
		calls++
		if !reflect.DeepEqual(a[0], []int64{1, 3, 999}) || !reflect.DeepEqual(a[2], []int32{2, 3, 2}) {
			t.Fatalf("arguments %v", a)
		}
		return pgtest.NewRows(batchValues(1, 1, 11, 1, pgtype.Int8{}), batchValues(1, 1, 12, 1, pgtype.Int8{}), batchValues(2, 3, 0, 0, pgtype.Int8{}), batchValues(3, 0, 0, 0, pgtype.Int8{})), nil
	}}
	r := New(db)
	pages, errs := r.ListPostCommentsBatch(t.Context(), []domain.PostCommentsRequest{{PostID: 1, Page: domain.PageRequest{First: 1}}, {PostID: 2, Page: domain.PageRequest{First: 1, After: &invalid}}, {PostID: 3, Page: domain.PageRequest{First: 2}}, {PostID: 999, Page: domain.PageRequest{First: 1}}})
	if calls != 1 || len(pages) != 4 || errs[0] != nil || !pages[0].PageInfo.HasNextPage || pages[0].Nodes[0].ID != 11 || !errors.Is(errs[1], domain.ErrInvalidArgument) || errs[2] != nil || pages[2].Nodes == nil || len(pages[2].Nodes) != 0 || !errors.Is(errs[3], domain.ErrNotFound) {
		t.Fatal(pages, errs)
	}
	db.QueryFunc = func(_ context.Context, s string, a ...any) (pgx.Rows, error) {
		if !reflect.DeepEqual(a[0], []int64{9}) || !reflect.DeepEqual(a[1], []int64{1}) {
			t.Fatalf("arguments %v", a)
		}
		return pgtest.NewRows(batchValues(1, 9, 10, 1, pgtype.Int8{Int64: 9, Valid: true})), nil
	}
	pages, errs = r.ListRepliesBatch(t.Context(), []domain.RepliesRequest{{PostID: 1, ParentCommentID: 9, Page: domain.PageRequest{First: 1}}})
	if errs[0] != nil || *pages[0].Nodes[0].ParentCommentID != 9 {
		t.Fatal(pages, errs)
	}
	broken := errors.New("offline")
	db.QueryFunc = func(context.Context, string, ...any) (pgx.Rows, error) { return nil, broken }
	_, errs = r.ListRepliesBatch(t.Context(), []domain.RepliesRequest{{PostID: 1, ParentCommentID: 9, Page: domain.PageRequest{First: 1}}})
	if !errors.Is(errs[0], broken) {
		t.Fatal(errs)
	}
}

func TestTreePlanAndPageMapping(t *testing.T) {
	bad := "bad"
	calls := 0
	db := &pgtest.DB{QueryFunc: func(_ context.Context, s string, a ...any) (pgx.Rows, error) {
		calls++
		if !reflect.DeepEqual(a[0], []int32{1, 3}) || !reflect.DeepEqual(a[1], []int32{0, 1}) || !reflect.DeepEqual(a[6], []int64{2}) || !reflect.DeepEqual(a[7], []int64{0}) {
			t.Fatalf("arguments %v", a)
		}
		row := func(sel int32, parent, id int64) []any {
			return []any{sel, int64(2), parent, id, pgtype.Int8{Int64: parent, Valid: parent != 0}, "text", int64(7), pgtype.Timestamptz{}}
		}
		return pgtest.NewRows(row(1, 0, 10), row(1, 0, 11), row(3, 10, 12)), nil
	}}
	r := New(db)
	pages, err := r.LoadCommentTree(t.Context(), domain.CommentTreeRequest{Roots: []domain.CommentTreeRoot{{PostID: 2}}, Selections: []domain.CommentSelection{{ID: 1, Page: domain.PageRequest{First: 1}, Replies: []domain.CommentSelection{{ID: 2, Page: domain.PageRequest{First: 1, After: &bad}}, {ID: 3, Page: domain.PageRequest{First: 1}}}}}})
	if err != nil || calls != 1 || len(pages) != 2 || pages[0].ParentCommentID != nil || len(pages[0].Page.Nodes) != 1 || !pages[0].Page.PageInfo.HasNextPage || *pages[1].ParentCommentID != 10 || pages[1].Page.Nodes[0].ID != 12 {
		t.Fatal(pages, err)
	}
}
