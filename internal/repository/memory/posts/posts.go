package posts

import (
	"context"
	"fmt"
	"time"

	"client-services/internal/domain"
	"client-services/internal/repository/cursor"
	"client-services/internal/repository/memory/internal/pagination"
	"client-services/internal/repository/memory/internal/store"
)

type PostRepository struct{ state *store.State }

func New(state *store.State) *PostRepository { return &PostRepository{state: state} }

func (r *PostRepository) CreatePost(ctx context.Context, post domain.Post) (*domain.Post, error) {
	r.state.PostsMu.Lock()
	defer r.state.PostsMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, ok := r.state.Users[post.AuthorID]; !ok {
		return nil, fmt.Errorf("%w: invalid author reference", domain.ErrInvalidArgument)
	}
	post.ID = r.state.NextPostID
	r.state.NextPostID++
	post.CreatedAt = time.Now().UTC()
	r.state.Posts[post.ID] = post
	r.state.PostIDs = append(r.state.PostIDs, post.ID)
	return &post, nil
}

func (r *PostRepository) GetPost(ctx context.Context, postID int64) (*domain.Post, error) {
	r.state.PostsMu.RLock()
	defer r.state.PostsMu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	post, ok := r.state.Posts[postID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &post, nil
}

func (r *PostRepository) ListPosts(ctx context.Context, page domain.PageRequest) (*domain.PostPage, error) {
	if err := pagination.ValidateSize(page.First); err != nil {
		return nil, err
	}
	afterID, err := cursor.DecodePost(page.After)
	if err != nil {
		return nil, err
	}
	posts, err := r.postSnapshot(ctx, afterID, page.First)
	if err != nil {
		return nil, err
	}
	return postPage(ctx, posts, page.First)
}

func (r *PostRepository) postSnapshot(ctx context.Context, afterID int64, first int) ([]domain.Post, error) {
	r.state.PostsMu.RLock()
	defer r.state.PostsMu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ids := pagination.IDs(r.state.PostIDs, afterID, first)
	posts := make([]domain.Post, len(ids))
	for i, id := range ids {
		posts[i] = r.state.Posts[id]
	}
	return posts, ctx.Err()
}

func (r *PostRepository) DisableComments(ctx context.Context, userID, postID int64) (*domain.Post, error) {
	r.state.PostsMu.Lock()
	defer r.state.PostsMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	post, ok := r.state.Posts[postID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if post.AuthorID != userID {
		return nil, domain.ErrForbidden
	}
	post.CommentsPermitted = false
	r.state.Posts[postID] = post
	return &post, nil
}
