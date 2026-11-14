package convert

import (
	"strconv"

	"client-services/internal/domain"
	"client-services/internal/transport/graphql/model"
)

func Post(post *domain.Post) *model.Post {
	if post == nil {
		return nil
	}
	return &model.Post{
		ID:                strconv.FormatInt(post.ID, 10),
		Title:             post.Title,
		Content:           post.Content,
		AuthorID:          post.AuthorID,
		CommentsPermitted: post.CommentsPermitted,
		CreatedAt:         post.CreatedAt,
	}
}

func PostPage(page *domain.PostPage) *model.PostPage {
	if page == nil {
		return nil
	}
	nodes := make([]*model.Post, len(page.Nodes))
	for i := range page.Nodes {
		nodes[i] = Post(&page.Nodes[i])
	}
	return &model.PostPage{Nodes: nodes, PageInfo: pageInfo(page.PageInfo)}
}
