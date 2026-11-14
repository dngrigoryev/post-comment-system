package config

import (
	"strings"
	"testing"
)

const testUsers = `[{"id":1,"name":"Alice","email":"alice@example.com","token":"secret"}]`

func TestLoadStorageSelection(t *testing.T) {
	t.Setenv("AUTH_USERS", testUsers)
	t.Setenv("STORAGE", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	c, err := Load()
	if err != nil || c.Storage != StorageMemory || c.HTTPAddr != ":8080" || c.AuthUsers[0].ID != 1 {
		t.Fatal(c, err)
	}
	t.Setenv("STORAGE", "postgres")
	if _, err := Load(); err == nil {
		t.Fatal("missing DATABASE_URL accepted")
	}
	t.Setenv("DATABASE_URL", "postgres://local/db")
	t.Setenv("HTTP_ADDR", ":9000")
	c, err = Load()
	if err != nil || c.Storage != StoragePostgres || c.HTTPAddr != ":9000" {
		t.Fatal(c, err)
	}
	t.Setenv("STORAGE", "unknown")
	if _, err := Load(); err == nil {
		t.Fatal("unknown storage accepted")
	}
}
func TestAuthUserValidation(t *testing.T) {
	for _, raw := range []string{"", "null", "[]", "{", `[{"id":0,"name":"a","email":"a@b.c","token":"secret"}]`, `[{"id":1,"name":" ","email":"a@b.c","token":"secret"}]`, `[{"id":1,"name":"a","email":"","token":"secret"}]`, `[{"id":1,"name":"a","email":"a@b.c","token":" bad "}]`, `[{"id":1,"name":"a","email":"a@b.c","token":"two words"}]`, strings.TrimSuffix(testUsers, "]") + `,{"id":1,"name":"b","email":"b@b.c","token":"other"}]`, strings.TrimSuffix(testUsers, "]") + `,{"id":2,"name":"b","email":"b@b.c","token":"secret"}]`} {
		users, err := loadAuthUsers(raw)
		if err == nil {
			t.Fatalf("accepted invalid users: %+v", users)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatal("token leaked in validation error")
		}
	}
}
