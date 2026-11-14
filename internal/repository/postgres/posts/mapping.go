package posts

import (
	"client-services/internal/domain"
	"client-services/internal/repository/postgres/dbgen"
)

func toDomainPost(row dbgen.Post) domain.Post {
	return domain.Post{
		ID:                row.ID,
		Title:             row.Title,
		Content:           row.Content,
		AuthorID:          row.Author,
		CommentsPermitted: row.CommentsPermitted,
		CreatedAt:         row.CreatedAt.Time,
	}
}
