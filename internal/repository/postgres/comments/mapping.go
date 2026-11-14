package comments

import (
	"client-services/internal/domain"
	"client-services/internal/repository/postgres/dbgen"
)

func toDomainComment(row dbgen.Comment) domain.Comment {
	var parentID *int64
	if row.ParentComment.Valid {
		id := row.ParentComment.Int64
		parentID = &id
	}
	return domain.Comment{
		ID:              row.ID,
		PostID:          row.Post,
		ParentCommentID: parentID,
		Content:         row.Content,
		AuthorID:        row.Author,
		CreatedAt:       row.CreatedAt.Time,
	}
}
