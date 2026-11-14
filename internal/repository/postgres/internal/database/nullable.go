package database

import "github.com/jackc/pgx/v5/pgtype"

func NullableID(id *int64) pgtype.Int8 {
	if id == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *id, Valid: true}
}
