package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	StorageMemory   = "memory"
	StoragePostgres = "postgres"
)

type Config struct {
	Storage     string
	HTTPAddr    string
	DatabaseURL string
	AuthUsers   []AuthUser
}

type AuthUser struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Token string `json:"token"`
}

func Load() (Config, error) {
	cfg := Config{
		Storage:     envOrDefault("STORAGE", StorageMemory),
		HTTPAddr:    envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	switch cfg.Storage {
	case StorageMemory:
	case StoragePostgres:
		if cfg.DatabaseURL == "" {
			return Config{}, fmt.Errorf("DATABASE_URL is required when STORAGE=postgres")
		}
	default:
		return Config{}, fmt.Errorf("STORAGE must be memory or postgres")
	}
	users, err := loadAuthUsers(os.Getenv("AUTH_USERS"))
	if err != nil {
		return Config{}, err
	}
	cfg.AuthUsers = users
	return cfg, nil
}

func loadAuthUsers(raw string) ([]AuthUser, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("AUTH_USERS is required")
	}
	var users []AuthUser
	if err := json.Unmarshal([]byte(raw), &users); err != nil {
		return nil, fmt.Errorf("AUTH_USERS must be a JSON array of users: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("AUTH_USERS must contain at least one user")
	}
	ids := make(map[int64]bool, len(users))
	tokens := make(map[string]bool, len(users))
	for i, user := range users {
		if user.ID <= 0 || strings.TrimSpace(user.Name) == "" || strings.TrimSpace(user.Email) == "" {
			return nil, fmt.Errorf("AUTH_USERS[%d] requires a positive id, name and email", i)
		}
		parts := strings.Fields(user.Token)
		if len(parts) != 1 || parts[0] != user.Token {
			return nil, fmt.Errorf("AUTH_USERS[%d] requires a non-empty token without whitespace", i)
		}
		if ids[user.ID] {
			return nil, fmt.Errorf("AUTH_USERS contains duplicate user id %d", user.ID)
		}
		if tokens[user.Token] {
			return nil, fmt.Errorf("AUTH_USERS contains duplicate tokens")
		}
		ids[user.ID] = true
		tokens[user.Token] = true
	}
	return users, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
