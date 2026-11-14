package graphql

import (
	"net/http"

	"client-services/internal/transport/graphql/generated"
	"client-services/internal/transport/graphql/loader"
	"client-services/internal/transport/graphql/resolver"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"
)

func NewHandler(resolvers *resolver.Resolver) http.Handler {
	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: resolvers}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](100))
	srv.Use(extension.Introspection{})
	return loader.Middleware(resolvers.UserService, resolvers.CommentService)(srv)
}
