package convert

import (
	"client-services/internal/domain"
	"client-services/internal/transport/graphql/model"
)

func pageInfo(info domain.PageInfo) *model.PageInfo {
	return &model.PageInfo{EndCursor: info.EndCursor, HasNextPage: info.HasNextPage}
}
