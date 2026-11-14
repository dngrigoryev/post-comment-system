package convert

import (
	"strconv"

	"client-services/internal/domain"
	"client-services/internal/transport/graphql/model"
)

func Comment(comment *domain.Comment) *model.Comment {
	if comment == nil {
		return nil
	}
	return &model.Comment{
		ID:        strconv.FormatInt(comment.ID, 10),
		Content:   comment.Content,
		AuthorID:  comment.AuthorID,
		PostID:    comment.PostID,
		CreatedAt: comment.CreatedAt,
	}
}

func CommentPage(page *domain.CommentPage) *model.CommentPage {
	if page == nil {
		return nil
	}
	nodes := make([]*model.Comment, len(page.Nodes))
	for i := range page.Nodes {
		nodes[i] = Comment(&page.Nodes[i])
	}
	return &model.CommentPage{Nodes: nodes, PageInfo: pageInfo(page.PageInfo)}
}
