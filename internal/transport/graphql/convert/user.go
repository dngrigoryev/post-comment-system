package convert

import (
	"strconv"

	"client-services/internal/domain"
	"client-services/internal/transport/graphql/model"
)

func User(user *domain.User) *model.User {
	if user == nil {
		return nil
	}
	return &model.User{
		ID:   strconv.FormatInt(user.ID, 10),
		Name: user.Name,
	}
}
