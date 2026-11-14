package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"client-services/internal/domain"
	commentEvents "client-services/internal/pubsub/comments"
	"client-services/internal/repository/memory"
	commentservice "client-services/internal/service/comments"
	postservice "client-services/internal/service/posts"
	userservice "client-services/internal/service/users"
	graphqltransport "client-services/internal/transport/graphql"
	"client-services/internal/transport/graphql/resolver"
	"client-services/internal/transport/http/middleware"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type subscriptionProbe struct {
	resolver.CommentService
	ready, cancelled chan int64
}

func (s *subscriptionProbe) SubscribeComments(ctx context.Context, id int64) (<-chan domain.Comment, error) {
	stream, err := s.CommentService.SubscribeComments(ctx, id)
	if err == nil {
		context.AfterFunc(ctx, func() { s.cancelled <- id })
		s.ready <- id
	}
	return stream, err
}

type namedUsers struct {
	resolver.UserService
	name atomic.Value
}

func (u *namedUsers) GetUsersBatch(ctx context.Context, ids []int64) ([]*domain.User, []error) {
	users, errs := u.UserService.GetUsersBatch(ctx, ids)
	for _, user := range users {
		if user != nil && user.ID == 1 {
			user.Name = u.name.Load().(string)
		}
	}
	return users, errs
}

func subscriptionFixture(t *testing.T) (http.Handler, *httptest.Server, *graphqltransport.Handler, *subscriptionProbe, *namedUsers) {
	t.Helper()
	repo := memory.New(domain.User{ID: 1, Name: "Alice"}, domain.User{ID: 2, Name: "Bob"})
	for i := 0; i < 2; i++ {
		if _, err := repo.CreatePost(t.Context(), domain.Post{AuthorID: 1, CommentsPermitted: true}); err != nil {
			t.Fatal(err)
		}
	}
	hub := commentEvents.New()
	t.Cleanup(hub.Close)
	comments := &subscriptionProbe{CommentService: commentservice.NewService(repo, repo, hub), ready: make(chan int64, 32), cancelled: make(chan int64, 32)}
	users := &namedUsers{UserService: userservice.NewService(repo)}
	users.name.Store("Alice")
	auth := middleware.NewAuthenticator(map[string]int64{"owner": 1, "other": 2})
	graph := graphqltransport.NewHandler(&resolver.Resolver{PostService: postservice.NewService(repo), CommentService: comments, UserService: users}, auth.Authenticate)
	t.Cleanup(graph.Close)
	h := auth.Middleware(graph)
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return h, server, graph, comments, users
}

type wsFrame struct {
	Type, ID string
	Payload  json.RawMessage
	Err      error `json:"-"`
}
type wsClient struct {
	conn   *websocket.Conn
	frames chan wsFrame
}

func dial(t *testing.T, server *httptest.Server, protocol string, headers http.Header) *wsClient {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/query", &websocket.DialOptions{Subprotocols: []string{protocol}, HTTPHeader: headers})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	client := &wsClient{conn: conn, frames: make(chan wsFrame, 32)}
	go func() {
		for {
			var frame wsFrame
			err := wsjson.Read(t.Context(), conn, &frame)
			if err != nil {
				frame.Err = err
			}
			select {
			case client.frames <- frame:
			case <-t.Context().Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return client
}
func (c *wsClient) send(t *testing.T, typ, id string, payload any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, c.conn, map[string]any{"type": typ, "id": id, "payload": payload}); err != nil {
		t.Fatal(err)
	}
}
func (c *wsClient) next(t *testing.T) wsFrame {
	t.Helper()
	for {
		select {
		case frame := <-c.frames:
			if frame.Type == "ka" {
				continue
			}
			if frame.Type == "ping" {
				c.send(t, "pong", "", nil)
				continue
			}
			return frame
		case <-time.After(5 * time.Second):
			t.Fatal("missing WebSocket frame")
			return wsFrame{}
		}
	}
}
func (c *wsClient) init(t *testing.T, payload any) {
	t.Helper()
	c.send(t, "connection_init", "", payload)
	frame := c.next(t)
	if frame.Type != "connection_ack" || frame.Err != nil {
		t.Fatalf("connection init: %+v", frame)
	}
}
func waitPost(t *testing.T, ch <-chan int64, want int64) {
	t.Helper()
	select {
	case id := <-ch:
		if id != want {
			t.Fatal(id, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("subscription lifecycle did not progress")
	}
}
func streamData(t *testing.T, frame wsFrame, id, typ string) map[string]any {
	t.Helper()
	if frame.Err != nil || frame.ID != id || frame.Type != typ {
		t.Fatalf("unexpected frame %+v", frame)
	}
	var result map[string]any
	if err := json.Unmarshal(frame.Payload, &result); err != nil {
		t.Fatal(err)
	}
	return success(t, result)
}

func TestWebSocketSubscriptionsFanoutFreshCacheAndCancellation(t *testing.T) {
	h, server, graph, comments, users := subscriptionFixture(t)
	client := dial(t, server, "graphql-transport-ws", nil)
	client.init(t, map[string]any{})
	for _, id := range []string{"1", "2"} {
		client.send(t, "subscribe", id, map[string]any{"query": fmt.Sprintf(`subscription{commentAdded(postId:%q){id content author{id name} replies(first:1){nodes{id}}}}`, id)})
		if id == "1" {
			waitPost(t, comments.ready, 1)
		} else {
			waitPost(t, comments.ready, 2)
		}
	}
	add := func(post, content, parent string) string {
		input := fmt.Sprintf(`postId:%q,content:%q`, post, content)
		if parent != "" {
			input += fmt.Sprintf(`,parentCommentId:%q`, parent)
		}
		data := success(t, request(t, h, `mutation{addComment(input:{`+input+`}){id}}`, "owner", nil))
		return object(data["addComment"])["id"].(string)
	}
	root := add("1", "root", "")
	event := object(streamData(t, client.next(t), "1", "next")["commentAdded"])
	if event["id"] != root || object(event["author"])["name"] != "Alice" || len(nodes(event["replies"])) != 0 {
		t.Fatal(event)
	}
	other := add("2", "other post", "")
	event = object(streamData(t, client.next(t), "2", "next")["commentAdded"])
	if event["id"] != other {
		t.Fatal("wrong post received an event")
	}
	users.name.Store("Updated")
	reply := add("1", "reply", root)
	event = object(streamData(t, client.next(t), "1", "next")["commentAdded"])
	if event["id"] != reply || object(event["author"])["name"] != "Updated" {
		t.Fatal("subscription reused an earlier payload's loader cache", event)
	}
	client.send(t, "complete", "1", nil)
	waitPost(t, comments.cancelled, 1)
	frame := client.next(t)
	if frame.Type != "complete" || frame.ID != "1" {
		t.Fatalf("cancel response %+v", frame)
	}
	add("1", "after cancellation", "")
	remaining := add("2", "still subscribed", "")
	event = object(streamData(t, client.next(t), "2", "next")["commentAdded"])
	if event["id"] != remaining {
		t.Fatal(event)
	}
	graph.Close()
	frame = client.next(t)
	if frame.Err == nil {
		t.Fatalf("handler close left connection open: %+v", frame)
	}
	waitPost(t, comments.cancelled, 2)
}

func TestLegacyWebSocketProtocolAndInitAuthentication(t *testing.T) {
	h, server, _, comments, _ := subscriptionFixture(t)
	client := dial(t, server, "graphql-ws", nil)
	client.init(t, map[string]any{"Authorization": "Bearer owner"})
	client.send(t, "start", "s", map[string]any{"query": `subscription{commentAdded(postId:"1"){id}}`})
	waitPost(t, comments.ready, 1)
	result := success(t, request(t, h, `mutation{addComment(input:{postId:"1",content:"legacy"}){id}}`, "owner", nil))
	data := streamData(t, client.next(t), "s", "data")
	if object(data["commentAdded"])["id"] != object(result["addComment"])["id"] {
		t.Fatal(data)
	}
	client.send(t, "start", "m", map[string]any{"query": `mutation{createPost(input:{title:"WebSocket",content:"text"}){author{id}}}`})
	data = streamData(t, client.next(t), "m", "data")
	if object(object(data["createPost"])["author"])["id"] != "1" {
		t.Fatal(data)
	}
	client.send(t, "stop", "s", nil)
	waitPost(t, comments.cancelled, 1)
}

func TestSubscriptionValidationAndInitRejections(t *testing.T) {
	_, server, _, _, _ := subscriptionFixture(t)
	for _, tc := range []struct{ id, code string }{{"bad", "BAD_USER_INPUT"}, {"999", "NOT_FOUND"}} {
		client := dial(t, server, "graphql-transport-ws", nil)
		client.init(t, nil)
		client.send(t, "subscribe", "s", map[string]any{"query": fmt.Sprintf(`subscription{commentAdded(postId:%q){id}}`, tc.id)})
		frame := client.next(t)
		if frame.Type != "next" && frame.Type != "error" {
			t.Fatalf("unexpected frame %+v", frame)
		}
		var payload any
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if frame.Type == "error" {
			payload = map[string]any{"errors": payload}
		}
		errorCode(t, object(payload), tc.code)
	}
	for _, payload := range []any{map[string]any{"Authorization": "Bearer invalid"}, map[string]any{"Authorization": 12}, map[string]any{"Authorization": "Bearer owner", "authorization": "Bearer owner"}} {
		client := dial(t, server, "graphql-transport-ws", nil)
		client.send(t, "connection_init", "", payload)
		frame := client.next(t)
		if frame.Type == "connection_ack" {
			t.Fatal("invalid auth accepted")
		}
	}
	client := dial(t, server, "graphql-transport-ws", http.Header{"Authorization": []string{"Bearer owner"}})
	client.send(t, "connection_init", "", map[string]any{"authorization": "Bearer other"})
	if frame := client.next(t); frame.Type == "connection_ack" {
		t.Fatal("conflicting HTTP and WebSocket users accepted")
	}
}
