package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, "development")
	}

	if cfg.Port != 8000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8000)
	}
}

func TestLoadCustomValues(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("PORT", "9000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, "production")
	}

	if cfg.Port != 9000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9000)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "not a number", port: "abc"},
		{name: "zero", port: "0"},
		{name: "negative", port: "-1"},
		{name: "too large", port: "65536"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", tt.port)

			_, err := Load()
			if err == nil {
				t.Errorf("Load() expected an error for PORT=%q", tt.port)
			}
		})
	}
}

func TestLoadDoesNotModifyEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("PORT", "8080")

	_, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if got := os.Getenv("APP_ENV"); got != "test" {
		t.Errorf("APP_ENV = %q, want %q", got, "test")
	}
}
