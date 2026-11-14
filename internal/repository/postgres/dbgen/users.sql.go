package dbgen

import (
	"context"
)

const ensureUser = `-- name: EnsureUser :exec
INSERT INTO users (id, name, email)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO NOTHING
`

type EnsureUserParams struct {
	UserID int64
	Name   string
	Email  string
}

func (q *Queries) EnsureUser(ctx context.Context, arg EnsureUserParams) error {
	_, err := q.db.Exec(ctx, ensureUser, arg.UserID, arg.Name, arg.Email)
	return err
}

const getUser = `-- name: GetUser :one
SELECT id, name
FROM users
WHERE id = $1
`

type GetUserRow struct {
	ID   int64
	Name string
}

func (q *Queries) GetUser(ctx context.Context, userID int64) (GetUserRow, error) {
	row := q.db.QueryRow(ctx, getUser, userID)
	var i GetUserRow
	err := row.Scan(&i.ID, &i.Name)
	return i, err
}

const getUsersBatch = `-- name: GetUsersBatch :many
SELECT id, name
FROM users
WHERE id = ANY($1::bigint[])
`

type GetUsersBatchRow struct {
	ID   int64
	Name string
}

func (q *Queries) GetUsersBatch(ctx context.Context, userIds []int64) ([]GetUsersBatchRow, error) {
	rows, err := q.db.Query(ctx, getUsersBatch, userIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetUsersBatchRow
	for rows.Next() {
		var i GetUsersBatchRow
		if err := rows.Scan(&i.ID, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
