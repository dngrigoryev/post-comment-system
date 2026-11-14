package graphql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"client-services/internal/domain"
	"client-services/internal/repository/memory"
	commentservice "client-services/internal/service/comments"
	postservice "client-services/internal/service/posts"
	userservice "client-services/internal/service/users"
	graphqltransport "client-services/internal/transport/graphql"
	"client-services/internal/transport/graphql/resolver"
	"client-services/internal/transport/http/middleware"
)

type commentProbe struct {
	resolver.CommentService
	trees, roots, replies atomic.Int32
}

func (s *commentProbe) LoadCommentTree(ctx context.Context, r domain.CommentTreeRequest) ([]domain.CommentTreePage, error) {
	s.trees.Add(1)
	return s.CommentService.LoadCommentTree(ctx, r)
}
func (s *commentProbe) ListPostCommentsBatch(ctx context.Context, r []domain.PostCommentsRequest) ([]*domain.CommentPage, []error) {
	s.roots.Add(1)
	return s.CommentService.ListPostCommentsBatch(ctx, r)
}
func (s *commentProbe) ListRepliesBatch(ctx context.Context, r []domain.RepliesRequest) ([]*domain.CommentPage, []error) {
	s.replies.Add(1)
	return s.CommentService.ListRepliesBatch(ctx, r)
}

type userProbe struct {
	resolver.UserService
	calls atomic.Int32
}

func (s *userProbe) GetUsersBatch(ctx context.Context, ids []int64) ([]*domain.User, []error) {
	s.calls.Add(1)
	return s.UserService.GetUsersBatch(ctx, ids)
}

func request(t *testing.T, h http.Handler, query, token string, variables map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/query", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("HTTP %d: %s", rec.Code, rec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func object(v any) map[string]any { return v.(map[string]any) }
func nodes(v any) []any           { return object(v)["nodes"].([]any) }
func success(t *testing.T, r map[string]any) map[string]any {
	t.Helper()
	if r["errors"] != nil {
		t.Fatalf("GraphQL errors: %v", r["errors"])
	}
	return object(r["data"])
}
func errorCode(t *testing.T, r map[string]any, want string) {
	t.Helper()
	list, ok := r["errors"].([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("expected %s: %v", want, r)
	}
	if code := object(object(list[0])["extensions"])["code"]; code != want {
		t.Fatalf("got %v want %s", code, want)
	}
}
func handler(repo *memory.Repository) (http.Handler, *commentProbe, *userProbe) {
	c := &commentProbe{CommentService: commentservice.NewService(repo, repo)}
	u := &userProbe{UserService: userservice.NewService(repo)}
	h := graphqltransport.NewHandler(&resolver.Resolver{PostService: postservice.NewService(repo), CommentService: c, UserService: u})
	return middleware.Auth(map[string]int64{"owner": 1, "other": 2})(h), c, u
}

func TestMutationsAuthenticationAndCacheInvalidation(t *testing.T) {
	repo := memory.New(domain.User{ID: 1, Name: "Alice"}, domain.User{ID: 2, Name: "Bob"})
	h, _, _ := handler(repo)
	q := `mutation{createPost(input:{title:"post",content:"text"}){id}}`
	errorCode(t, request(t, h, q, "", nil), "UNAUTHENTICATED")
	data := success(t, request(t, h, q, "owner", nil))
	id := object(data["createPost"])["id"].(string)
	errorCode(t, request(t, h, fmt.Sprintf(`mutation{disableComments(postId:%q){id}}`, id), "other", nil), "FORBIDDEN")
	query := fmt.Sprintf(`mutation{ a:addComment(input:{postId:%q,content:"first"}){id author{id}} b:addComment(input:{postId:%q,content:"second"}){id} c:disableComments(postId:%q){comments(first:10){nodes{id content}}}}`, id, id, id)
	data = success(t, request(t, h, query, "owner", nil))
	if object(object(data["a"])["author"])["id"] != "1" || len(nodes(object(data["c"])["comments"])) != 2 {
		t.Fatal(data)
	}
	errorCode(t, request(t, h, fmt.Sprintf(`mutation{addComment(input:{postId:%q,content:"blocked"}){id}}`, id), "other", nil), "COMMENTS_DISABLED")
	data = success(t, request(t, h, `{post(id:"999"){id}}`, "", nil))
	if data["post"] != nil {
		t.Fatal(data)
	}
	errorCode(t, request(t, h, `{post(id:"bad"){id}}`, "", nil), "BAD_USER_INPUT")

	data = success(t, request(t, h, `mutation{a:createPost(input:{title:"fresh",content:"text"}){comments{nodes{id}}} b:addComment(input:{postId:"2",content:"new"}){id} c:disableComments(postId:"2"){comments{nodes{id}}}}`, "owner", nil))
	if len(nodes(object(data["a"])["comments"])) != 0 || len(nodes(object(data["c"])["comments"])) != 1 {
		t.Fatal(data)
	}
}

func TestPrefetchBatchesAllSelectedLevels(t *testing.T) {
	repo := memory.New(domain.User{ID: 1, Name: "Alice"}, domain.User{ID: 2, Name: "Bob"})
	for i := 0; i < 20; i++ {
		p, err := repo.CreatePost(t.Context(), domain.Post{AuthorID: 1, CommentsPermitted: true})
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 2; j++ {
			var parent *int64
			for level := 0; level < 4; level++ {
				c, err := repo.AddComment(t.Context(), domain.Comment{PostID: p.ID, ParentCommentID: parent, AuthorID: int64(level%2 + 1), Content: "text"})
				if err != nil {
					t.Fatal(err)
				}
				id := c.ID
				parent = &id
			}
		}
	}
	h, c, u := handler(repo)
	query := `query($n:Int!=2,$show:Boolean!=true){posts(first:20){nodes{id author{id} comments(first:$n){nodes{... C}}}}} fragment C on Comment{id author{id} replies(first:1) @include(if:$show){nodes{id author{id} replies(first:1){nodes{id author{id} replies(first:1){nodes{id author{id} replies(first:1){nodes{id}}}}}}}}}`
	for attempt := 1; attempt <= 2; attempt++ {
		data := success(t, request(t, h, query, "", nil))
		if len(nodes(data["posts"])) != 20 {
			t.Fatal(data)
		}
		for _, value := range nodes(data["posts"]) {
			p := object(value)
			if len(nodes(p["comments"])) != 2 {
				t.Fatal(p)
			}
			for _, root := range nodes(p["comments"]) {
				current := object(root)
				for level := 0; level < 4; level++ {
					if object(current["author"])["id"] != strconv.Itoa(level%2+1) {
						t.Fatal(current)
					}
					children := nodes(current["replies"])
					if level == 3 {
						if len(children) != 0 {
							t.Fatal(children)
						}
					} else {
						if len(children) != 1 {
							t.Fatal(children)
						}
						current = object(children[0])
					}
				}
			}
		}
		if c.trees.Load() != int32(attempt) || u.calls.Load() != int32(attempt) || c.roots.Load() != 0 || c.replies.Load() != 0 {
			t.Fatalf("tree=%d users=%d root batches=%d reply batches=%d", c.trees.Load(), u.calls.Load(), c.roots.Load(), c.replies.Load())
		}
	}

	data := success(t, request(t, h, `{post(id:"1"){a:comments(first:1){nodes{id}} b:comments(first:1){nodes{id replies(first:1){nodes{id}}}} c:comments(first:2){nodes{id}}}}`, "", nil))
	p := object(data["post"])
	if len(nodes(p["a"])) != 1 || len(nodes(p["b"])) != 1 || len(nodes(p["c"])) != 2 || len(nodes(object(nodes(p["b"])[0])["replies"])) != 1 {
		t.Fatal(p)
	}
	errorCode(t, request(t, h, `{post(id:"1"){title comments{nodes{id replies(first:0){nodes{id}}}}}}`, "", nil), "BAD_USER_INPUT")
}
