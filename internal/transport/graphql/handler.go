package graphql

import (
	"context"
	"net/http"
	"time"

	"client-services/internal/transport/graphql/generated"
	"client-services/internal/transport/graphql/loader"
	"client-services/internal/transport/graphql/resolver"

	gql "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"
)

type Handler struct {
	server http.Handler
	ctx    context.Context
	cancel context.CancelFunc
}

func (h *Handler) Close() { h.cancel() }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.ctx.Err() != nil {
		http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get("Upgrade") != "" {
		ctx, cancel := context.WithCancel(r.Context())
		stop := context.AfterFunc(h.ctx, cancel)
		defer stop()
		defer cancel()
		r = r.WithContext(ctx)
	}
	h.server.ServeHTTP(w, r)
}

func NewHandler(resolvers *resolver.Resolver, authenticate func(context.Context, string) (context.Context, error)) *Handler {
	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: resolvers}))
	srv.AddTransport(transport.Websocket{
		Implementation:        websocketImplementation{},
		InitTimeout:           5 * time.Second,
		KeepAlivePingInterval: 15 * time.Second,
		PingPongInterval:      15 * time.Second,
		InitFunc:              websocketInit(authenticate),
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](100))
	srv.Use(extension.Introspection{})
	srv.AroundOperations(func(ctx context.Context, next gql.OperationHandler) gql.ResponseHandler {
		return next(loader.WithContext(ctx, resolvers.UserService, resolvers.CommentService))
	})
	srv.AroundResponses(func(ctx context.Context, next gql.ResponseHandler) *gql.Response {
		if gql.HasOperationContext(ctx) && gql.GetOperationContext(ctx).Operation.Operation == ast.Subscription {
			ctx = loader.WithContext(ctx, resolvers.UserService, resolvers.CommentService)
		}
		return next(ctx)
	})
	ctx, cancel := context.WithCancel(context.Background())
	return &Handler{server: srv, ctx: ctx, cancel: cancel}
}
