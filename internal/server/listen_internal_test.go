package server

import "testing"

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
