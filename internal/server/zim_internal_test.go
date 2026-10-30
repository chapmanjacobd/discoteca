package server

import "testing"

func TestValidateKiwixConfig(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"disabled", 0, false},
		{"default port", 8181, false},
		{"maximum port", 65535, false},
		{"negative port", -1, true},
		{"above maximum", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Server{KiwixPort: tt.port}
			err := cmd.validateKiwixConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateKiwixConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
