package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"client-services/internal/config"
	"client-services/internal/domain"
	commentEvents "client-services/internal/pubsub/comments"
	"client-services/internal/repository/memory"
	"client-services/internal/repository/postgres"
	commentservice "client-services/internal/service/comments"
	postservice "client-services/internal/service/posts"
	userservice "client-services/internal/service/users"
	graphqltransport "client-services/internal/transport/graphql"
	"client-services/internal/transport/graphql/resolver"
	"client-services/internal/transport/http/middleware"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository interface {
	postservice.Repository
	commentservice.Repository
	userservice.Repository
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	repo, closeRepository, err := openRepository(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeRepository()

	events := commentEvents.New()
	defer events.Close()
	resolvers := &resolver.Resolver{
		PostService:    postservice.NewService(repo),
		UserService:    userservice.NewService(repo),
		CommentService: commentservice.NewService(repo, repo, events),
	}
	tokens := make(map[string]int64, len(cfg.AuthUsers))
	for _, user := range cfg.AuthUsers {
		tokens[user.Token] = user.ID
	}
	mux := http.NewServeMux()
	auth := middleware.NewAuthenticator(tokens)
	graph := graphqltransport.NewHandler(resolvers, auth.Authenticate)
	defer graph.Close()
	mux.Handle("/query", auth.Middleware(graph))
	mux.Handle("GET /{$}", playground.Handler("GraphQL", "/query"))
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       time.Minute,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()
	slog.Info("HTTP server starting", "address", cfg.HTTPAddr, "storage", cfg.Storage)
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown HTTP: %w", err)
	}
	return nil
}

func openRepository(ctx context.Context, cfg config.Config) (repository, func(), error) {
	users := make([]domain.User, len(cfg.AuthUsers))
	for i, user := range cfg.AuthUsers {
		users[i] = domain.User{ID: user.ID, Name: user.Name}
	}
	switch cfg.Storage {
	case config.StorageMemory:
		return memory.New(users...), func() {}, nil
	case config.StoragePostgres:
		connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		pool, err := pgxpool.New(connectCtx, cfg.DatabaseURL)
		if err != nil {
			return nil, nil, fmt.Errorf("create PostgreSQL pool: %w", err)
		}
		if err := pool.Ping(connectCtx); err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("connect PostgreSQL: %w", err)
		}
		repo := postgres.New(pool)
		for i, user := range cfg.AuthUsers {
			if err := repo.EnsureUser(connectCtx, users[i], user.Email); err != nil {
				pool.Close()
				return nil, nil, fmt.Errorf("initialize user %d: %w", user.ID, err)
			}
		}
		return repo, pool.Close, nil
	default:
		return nil, nil, fmt.Errorf("unsupported storage: %s", cfg.Storage)
	}
}
