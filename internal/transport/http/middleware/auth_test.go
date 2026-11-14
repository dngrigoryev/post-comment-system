package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"client-services/internal/domain"
)

func TestBearerAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers []string
		status  int
		id      int64
	}{
		{"public", nil, 200, 0}, {"owner", []string{"Bearer secret"}, 200, 1}, {"case insensitive", []string{"bearer other"}, 200, 2},
		{"unknown", []string{"Bearer invalid"}, 401, 0}, {"wrong scheme", []string{"Basic secret"}, 401, 0}, {"empty", []string{""}, 401, 0}, {"missing token", []string{"Bearer"}, 401, 0}, {"extra token", []string{"Bearer secret extra"}, 401, 0}, {"duplicate", []string{"Bearer secret", "Bearer other"}, 401, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			tokens := map[string]int64{"secret": 1, "other": 2}
			h := Auth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				id, err := RetrieveUserID(r.Context())
				if id != tc.id {
					t.Fatalf("user=%d want=%d", id, tc.id)
				}
				if tc.id == 0 && !errors.Is(err, domain.ErrUnauthenticated) {
					t.Fatal(err)
				}
				if tc.id > 0 && err != nil {
					t.Fatal(err)
				}
				w.WriteHeader(200)
			}))
			tokens["secret"] = 999
			req := httptest.NewRequest("POST", "/query", nil)
			for _, v := range tc.headers {
				req.Header.Add("Authorization", v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.status || called != (tc.status == 200) {
				t.Fatalf("status=%d called=%v", rec.Code, called)
			}
			if tc.status == 401 && (rec.Header().Get("WWW-Authenticate") != "Bearer" || strings.Contains(rec.Body.String(), "secret")) {
				t.Fatal("invalid unauthorized response")
			}
		})
	}
}
func TestInvalidContextUser(t *testing.T) {
	if _, err := RetrieveUserID(WithUserID(t.Context(), 0)); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatal(err)
	}
}
