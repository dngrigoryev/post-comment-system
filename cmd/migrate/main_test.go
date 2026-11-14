package main

import "testing"

func TestRequiresDatabaseURL(t *testing.T) {
	if run(t.Context(), "") == nil {
		t.Fatal("empty DATABASE_URL accepted")
	}
}
