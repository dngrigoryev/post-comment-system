package validation

import (
	"errors"
	"strings"
	"testing"

	"client-services/internal/domain"
)

func TestTextCountsCharacters(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		limit      int
		invalid    bool
	}{
		{"unicode boundary", strings.Repeat("🙂", 2000), 2000, false},
		{"unicode too long", strings.Repeat("я", 2001), 2000, true},
		{"blank", " \n\t", 2000, true}, {"invalid encoding", string([]byte{0xff}), 2000, true},
		{"unlimited content", strings.Repeat("a", 3000), 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Text("text", tc.text, tc.limit)
			if errors.Is(err, domain.ErrInvalidArgument) != tc.invalid {
				t.Fatal(err)
			}
		})
	}
}
func TestPageAndIdentity(t *testing.T) {
	empty := ""
	after := "opaque"
	for _, tc := range []struct {
		page    domain.PageRequest
		invalid bool
	}{
		{domain.PageRequest{First: 1}, false}, {domain.PageRequest{First: 100, After: &after}, false},
		{domain.PageRequest{First: 0}, true}, {domain.PageRequest{First: 101}, true}, {domain.PageRequest{First: 1, After: &empty}, true},
	} {
		if err := Page(tc.page); errors.Is(err, domain.ErrInvalidArgument) != tc.invalid {
			t.Fatalf("%+v: %v", tc.page, err)
		}
	}
	if ID("post", 1) != nil || User(1) != nil || !errors.Is(ID("post", 0), domain.ErrInvalidArgument) || !errors.Is(User(0), domain.ErrUnauthenticated) {
		t.Fatal("identity validation")
	}
}
