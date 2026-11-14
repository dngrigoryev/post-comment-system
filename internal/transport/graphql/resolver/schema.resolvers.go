package resolver

import (
	"client-services/internal/domain"
	"client-services/internal/transport/graphql/convert"
	"client-services/internal/transport/graphql/generated"
	"client-services/internal/transport/graphql/loader"
	"client-services/internal/transport/graphql/model"
	"client-services/internal/transport/http/middleware"
	"context"
	"errors"
)

func (r *commentResolver) Author(ctx context.Context, obj *model.Comment) (*model.User, error) {
	user, err := loader.User(ctx, obj.AuthorID)
	if err != nil {
		return nil, convert.Error(err)
	}
	return convert.User(user), nil
}

func (r *commentResolver) Replies(ctx context.Context, obj *model.Comment, first int, after *string) (*model.CommentPage, error) {
	id, err := convert.ID("comment ID", obj.ID)
	if err != nil {
		return nil, convert.Error(err)
	}
	page, err := loader.Replies(ctx, obj.PostID, id, first, after)
	if err != nil {
		return nil, convert.Error(err)
	}
	return convert.CommentPage(page), nil
}

func (r *mutationResolver) CreatePost(ctx context.Context, input model.CreatePostInput) (*model.Post, error) {
	userID, err := middleware.RetrieveUserID(ctx)
	if err != nil {
		return nil, convert.Error(err)
	}
	post, err := r.PostService.CreatePost(ctx, userID, input.Title, input.Content)
	if err != nil {
		return nil, convert.Error(err)
	}
	loader.Clear(ctx)
	loader.PrefetchPost(ctx, post)
	return convert.Post(post), nil
}

func (r *mutationResolver) AddComment(ctx context.Context, input model.AddCommentInput) (*model.Comment, error) {
	userID, err := middleware.RetrieveUserID(ctx)
	if err != nil {
		return nil, convert.Error(err)
	}
	postID, err := convert.ID("post ID", input.PostID)
	if err != nil {
		return nil, convert.Error(err)
	}
	var parentID *int64
	if input.ParentCommentID != nil {
		id, err := convert.ID("parent comment ID", *input.ParentCommentID)
		if err != nil {
			return nil, convert.Error(err)
		}
		parentID = &id
	}
	comment, err := r.CommentService.AddComment(ctx, userID, postID, parentID, input.Content)
	if err != nil {
		return nil, convert.Error(err)
	}
	loader.Clear(ctx)
	loader.PrefetchComment(ctx, comment)
	return convert.Comment(comment), nil
}

func (r *mutationResolver) DisableComments(ctx context.Context, postID string) (*model.Post, error) {
	userID, err := middleware.RetrieveUserID(ctx)
	if err != nil {
		return nil, convert.Error(err)
	}
	id, err := convert.ID("post ID", postID)
	if err != nil {
		return nil, convert.Error(err)
	}
	post, err := r.PostService.DisableComments(ctx, userID, id)
	if err != nil {
		return nil, convert.Error(err)
	}
	loader.Clear(ctx)
	loader.PrefetchPost(ctx, post)
	return convert.Post(post), nil
}

func (r *postResolver) Author(ctx context.Context, obj *model.Post) (*model.User, error) {
	user, err := loader.User(ctx, obj.AuthorID)
	if err != nil {
		return nil, convert.Error(err)
	}
	return convert.User(user), nil
}

func (r *postResolver) Comments(ctx context.Context, obj *model.Post, first int, after *string) (*model.CommentPage, error) {
	id, err := convert.ID("post ID", obj.ID)
	if err != nil {
		return nil, convert.Error(err)
	}
	page, err := loader.PostComments(ctx, id, first, after)
	if err != nil {
		return nil, convert.Error(err)
	}
	return convert.CommentPage(page), nil
}

func (r *queryResolver) Posts(ctx context.Context, first int, after *string) (*model.PostPage, error) {
	page, err := r.PostService.ListPosts(ctx, domain.PageRequest{First: first, After: after})
	if err != nil {
		return nil, convert.Error(err)
	}
	loader.PrefetchPostPage(ctx, page)
	return convert.PostPage(page), nil
}

func (r *queryResolver) Post(ctx context.Context, id string) (*model.Post, error) {
	postID, err := convert.ID("post ID", id)
	if err != nil {
		return nil, convert.Error(err)
	}
	post, err := r.PostService.GetPost(ctx, postID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, convert.Error(err)
	}
	loader.PrefetchPost(ctx, post)
	return convert.Post(post), nil
}

func (r *subscriptionResolver) CommentAdded(ctx context.Context, postID string) (<-chan *model.Comment, error) {
	id, err := convert.ID("post ID", postID)
	if err != nil {
		return nil, convert.Error(err)
	}
	ctx, cancel := context.WithCancel(ctx)
	comments, err := r.CommentService.SubscribeComments(ctx, id)
	if err != nil {
		cancel()
		return nil, convert.Error(err)
	}
	result := make(chan *model.Comment)
	go func() {
		defer close(result)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case comment, ok := <-comments:
				if !ok {
					return
				}
				select {
				case result <- convert.Comment(&comment):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return result, nil
}

func (r *Resolver) Comment() generated.CommentResolver { return &commentResolver{r} }

func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

func (r *Resolver) Post() generated.PostResolver { return &postResolver{r} }

func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type (
	commentResolver      struct{ *Resolver }
	mutationResolver     struct{ *Resolver }
	postResolver         struct{ *Resolver }
	queryResolver        struct{ *Resolver }
	subscriptionResolver struct{ *Resolver }
)
