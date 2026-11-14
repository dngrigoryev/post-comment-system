package cursor

import (
	"errors"
	"testing"

	"client-services/internal/domain"
)

func TestPostCursorRoundTrip(t *testing.T) {
	for _, id := range []int64{1, 1 << 40} {
		encoded, err := EncodePost(id)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodePost(&encoded)
		if err != nil || decoded != id {
			t.Fatal(decoded, err)
		}
	}
	if id, err := DecodePost(nil); id != 0 || err != nil {
		t.Fatal(id, err)
	}
	for _, id := range []int64{0, -1} {
		if _, err := EncodePost(id); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{"", "!", "bnVsbA", "e30", "W10"} {
		if _, err := DecodePost(&raw); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatal(raw, err)
		}
	}
}
func TestCommentCursorScope(t *testing.T) {
	parent := int64(7)
	other := int64(8)
	for _, p := range []*int64{nil, &parent} {
		encoded, err := EncodeComment(1<<40, 2, p)
		if err != nil {
			t.Fatal(err)
		}
		id, err := DecodeComment(&encoded, 2, p)
		if err != nil || id != 1<<40 {
			t.Fatal(id, err)
		}
		for _, scope := range []struct {
			post   int64
			parent *int64
		}{{3, p}, {2, &other}} {
			if _, err := DecodeComment(&encoded, scope.post, scope.parent); !errors.Is(err, domain.ErrInvalidArgument) {
				t.Fatal(err)
			}
		}
		position, err := ParseComment(&encoded)
		if err != nil || position.ID != id || position.PostID != 2 {
			t.Fatal(position, err)
		}
	}
	encoded, _ := EncodeComment(10, 2, &parent)
	if _, err := DecodeComment(&encoded, 2, nil); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	post, _ := EncodePost(10)
	if _, err := DecodeComment(&post, 2, nil); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	root, _ := EncodeComment(10, 2, nil)
	if _, err := DecodePost(&root); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatal(err)
	}
	zero := int64(0)
	for _, tc := range []struct {
		id, post int64
		parent   *int64
	}{{0, 2, nil}, {1, 0, nil}, {1, 2, &zero}} {
		if _, err := EncodeComment(tc.id, tc.post, tc.parent); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
	if id, err := DecodeComment(nil, 2, nil); id != 0 || err != nil {
		t.Fatal(id, err)
	}
}
func FuzzCursorDecoding(f *testing.F) {
	valid, _ := EncodePost(100)
	comment, _ := EncodeComment(100, 2, nil)
	for _, seed := range []string{"", "garbage", "bnVsbA", valid, comment} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		id, err := DecodePost(&raw)
		if err == nil && id <= 0 {
			t.Fatal("nonpositive post ID")
		}
		if err != nil && !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatal(err)
		}
		id, err = DecodeComment(&raw, 2, nil)
		if err == nil && id <= 0 {
			t.Fatal("nonpositive comment ID")
		}
		if err != nil && !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatal(err)
		}
	})
}
