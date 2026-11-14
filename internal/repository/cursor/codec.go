package cursor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"client-services/internal/domain"
)

func encode(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decode(raw string, value any) error {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return fmt.Errorf("%w: invalid cursor encoding", domain.ErrInvalidArgument)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("%w: invalid cursor payload", domain.ErrInvalidArgument)
	}
	return nil
}
