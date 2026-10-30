package server

import (
	"os"
	"testing"
)

func TestListenAddress(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{"default host", "", 5555, "127.0.0.1:5555"},
		{"loopback", "127.0.0.1", 8080, "127.0.0.1:8080"},
		{"ipv6 loopback", "::1", 8080, "[::1]:8080"},
		{"all interfaces", "0.0.0.0", 9000, "0.0.0.0:9000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Server{Host: tt.host, Port: tt.port}
			if got := cmd.listenAddress(); got != tt.want {
				t.Fatalf("listenAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateListenConfig(t *testing.T) {
	t.Run("loopback does not need a token", func(t *testing.T) {
		cmd := &Server{Host: "127.0.0.1"}
		if err := cmd.validateListenConfig(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non-loopback requires a token", func(t *testing.T) {
		t.Setenv("DISCO_API_TOKEN", "")
		cmd := &Server{Host: "0.0.0.0"}
		if err := cmd.validateListenConfig(); err == nil {
			t.Fatal("expected an error for an unauthenticated remote bind")
		}
	})

	t.Run("non-loopback with a token is allowed", func(t *testing.T) {
		t.Setenv("DISCO_API_TOKEN", "secret")
		cmd := &Server{Host: "0.0.0.0"}
		if err := cmd.validateListenConfig(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("localhost is loopback", func(t *testing.T) {
		t.Setenv("DISCO_API_TOKEN", "")
		cmd := &Server{Host: "localhost"}
		if err := cmd.validateListenConfig(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestLoopbackHost(t *testing.T) {
	for host, want := range map[string]bool{
		"":            true,
		"localhost":   true,
		"127.0.0.1":   true,
		"127.0.0.2":   true,
		"::1":         true,
		"[::1]":       true,
		"0.0.0.0":     false,
		"192.168.1.5": false,
		"example.com": false,
	} {
		if got := isLoopbackHost(host); got != want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestSetupAPITokenPrintsOnlyGeneratedTokens(t *testing.T) {
	t.Run("uses environment token", func(t *testing.T) {
		t.Setenv("DISCO_API_TOKEN", "from-env")
		cmd := &Server{}
		cmd.setupAPIToken()
		if cmd.APIToken != "from-env" {
			t.Fatalf("expected environment token, got %q", cmd.APIToken)
		}
	})

	t.Run("generates a token when unset", func(t *testing.T) {
		t.Setenv("DISCO_API_TOKEN", "")
		cmd := &Server{}
		cmd.setupAPIToken()
		if cmd.APIToken == "" {
			t.Fatal("expected a generated token")
		}
		if cmd.APIToken == os.Getenv("DISCO_API_TOKEN") {
			t.Fatal("generated token should not come from the environment")
		}
	})
}
