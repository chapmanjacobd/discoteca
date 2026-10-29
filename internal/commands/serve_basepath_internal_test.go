package commands

import (
	"context"
	"testing"
	"time"
)

func TestNormalizeBasePath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"/", ""},
		{"disco", "/disco"},
		{"/disco/", "/disco"},
		{" /disco ", "/disco"},
		{"/disco, /other", "/disco"},
		{"/disco/../other", ""},
		{"//evil.example", "/evil.example"},
		{"../up", ""},
		{"/bad\\path", ""},
		{"/bad\"quote", ""},
	}

	for _, tt := range tests {
		if got := normalizeBasePath(tt.in); got != tt.want {
			t.Errorf("normalizeBasePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestInjectBasePath(t *testing.T) {
	html := []byte("<html><head><title>x</title></head><body></body></html>")
	got := string(injectBasePath(html, "/disco"))
	want := `<html><head><title>x</title><script>window.__DISCO_BASE_PATH__="/disco";</script></head><body></body></html>`
	if got != want {
		t.Errorf("injectBasePath() = %q, want %q", got, want)
	}

	if string(injectBasePath(html, "")) != string(html) {
		t.Error("injectBasePath with an empty prefix should not modify the document")
	}
}

func TestAcquireSearchSlot(t *testing.T) {
	cmd := &ServeCmd{MaxConcurrentSearches: 1}

	release, err := cmd.acquireSearchSlot(context.Background())
	if err != nil {
		t.Fatalf("first slot should be available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, waitErr := cmd.acquireSearchSlot(ctx); waitErr == nil {
		t.Fatal("second slot should wait until the first is released")
	}

	release()

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	release, err = cmd.acquireSearchSlot(ctx)
	if err != nil {
		t.Fatalf("slot should be available after release: %v", err)
	}
	release()
}

func TestAcquireSearchSlotUnlimited(t *testing.T) {
	cmd := &ServeCmd{MaxConcurrentSearches: 0}
	for range 10 {
		release, err := cmd.acquireSearchSlot(context.Background())
		if err != nil {
			t.Fatalf("unlimited searches should not block: %v", err)
		}
		release()
	}
}
