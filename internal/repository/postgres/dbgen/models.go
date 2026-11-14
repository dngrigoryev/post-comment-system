package dbgen

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Comment struct {
	ID            int64
	Post          int64
	ParentComment pgtype.Int8
	Content       string
	Author        int64
	CreatedAt     pgtype.Timestamptz
}

type Post struct {
	ID                int64
	Title             string
	Content           string
	Author            int64
	CommentsPermitted bool
	CreatedAt         pgtype.Timestamptz
}

type User struct {
	ID    int64
	Name  string
	Email string
}
