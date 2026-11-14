package convert

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"client-services/internal/domain"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestIDAndErrorConversion(t *testing.T) {
	id, err := ID("post", "1099511627776")
	if err != nil || id != 1<<40 {
		t.Fatal(id, err)
	}
	for _, raw := range []string{"", "0", "-1", "1.1", "abc", "9223372036854775808"} {
		if _, err := ID("post", raw); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatal(raw, err)
		}
	}
	for _, tc := range []struct {
		err  error
		code string
	}{{domain.ErrInvalidArgument, "BAD_USER_INPUT"}, {domain.ErrUnauthenticated, "UNAUTHENTICATED"}, {domain.ErrForbidden, "FORBIDDEN"}, {domain.ErrNotFound, "NOT_FOUND"}, {domain.ErrCommentsDisabled, "COMMENTS_DISABLED"}, {errors.New("database password is secret"), "INTERNAL_SERVER_ERROR"}} {
		err := Error(fmt.Errorf("wrapped: %w", tc.err))
		var gql *gqlerror.Error
		if !errors.As(err, &gql) || gql.Extensions["code"] != tc.code {
			t.Fatal(err)
		}
		if tc.code == "INTERNAL_SERVER_ERROR" && gql.Message != "internal server error" {
			t.Fatal("storage details leaked")
		}
	}
	if Error(nil) != nil {
		t.Fatal("nil became an error")
	}
}
func TestEntitiesAndPages(t *testing.T) {
	now := time.Now()
	p := Post(&domain.Post{ID: 1 << 40, AuthorID: 7, Title: "title", Content: "text", CommentsPermitted: true, CreatedAt: now})
	if p.ID != "1099511627776" || p.AuthorID != 7 || !p.CommentsPermitted || !p.CreatedAt.Equal(now) {
		t.Fatal(p)
	}
	c := Comment(&domain.Comment{ID: 9, PostID: 2, AuthorID: 7, Content: "text", CreatedAt: now})
	if c.ID != "9" || c.PostID != 2 || c.AuthorID != 7 {
		t.Fatal(c)
	}
	u := User(&domain.User{ID: 7, Name: "Alice"})
	if u.ID != "7" || u.Name != "Alice" {
		t.Fatal(u)
	}
	posts := PostPage(&domain.PostPage{})
	comments := CommentPage(&domain.CommentPage{})
	if posts.Nodes == nil || comments.Nodes == nil || posts.PageInfo == nil || comments.PageInfo == nil {
		t.Fatal("empty pages must serialize as arrays")
	}
	if Post(nil) != nil || Comment(nil) != nil || User(nil) != nil || PostPage(nil) != nil || CommentPage(nil) != nil {
		t.Fatal("nil conversion")
	}
}
