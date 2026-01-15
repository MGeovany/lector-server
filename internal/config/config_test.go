package config

import "testing"

const defaultMaxFileSize int64 = 50 * 1024 * 1024

func TestNewConfig_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("MAX_FILE_SIZE", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SUPABASE_URL", "")
	t.Setenv("SUPABASE_ANON_KEY", "")
	t.Setenv("JWT_SECRET", "")

	cfg := NewConfig()

	if cfg.GetServerPort() != "8080" {
		t.Fatalf("expected default server port 8080, got %s", cfg.GetServerPort())
	}
	if cfg.GetMaxFileSize() != defaultMaxFileSize {
		t.Fatalf("expected default max file size %d, got %d", defaultMaxFileSize, cfg.GetMaxFileSize())
	}
	if cfg.GetLogLevel() != "info" {
		t.Fatalf("expected default log level info, got %s", cfg.GetLogLevel())
	}
	if cfg.GetSupabaseURL() != "" {
		t.Fatalf("expected default supabase url empty, got %s", cfg.GetSupabaseURL())
	}
	if cfg.GetSupabaseKey() != "" {
		t.Fatalf("expected default supabase key empty, got %s", cfg.GetSupabaseKey())
	}
	if cfg.GetJWTSecret() != "your-secret-key-change-in-production" {
		t.Fatalf("expected default jwt secret, got %s", cfg.GetJWTSecret())
	}
}

func TestNewConfig_Overrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("SERVER_PORT", "7070")
	t.Setenv("MAX_FILE_SIZE", "12345")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SUPABASE_URL", "http://localhost:54321")
	t.Setenv("SUPABASE_ANON_KEY", "test-key")
	t.Setenv("JWT_SECRET", "secret")

	cfg := NewConfig()

	if cfg.GetServerPort() != "9090" {
		t.Fatalf("expected server port 9090, got %s", cfg.GetServerPort())
	}
	if cfg.GetMaxFileSize() != 12345 {
		t.Fatalf("expected max file size 12345, got %d", cfg.GetMaxFileSize())
	}
	if cfg.GetLogLevel() != "debug" {
		t.Fatalf("expected log level debug, got %s", cfg.GetLogLevel())
	}
	if cfg.GetSupabaseURL() != "http://localhost:54321" {
		t.Fatalf("expected supabase url http://localhost:54321, got %s", cfg.GetSupabaseURL())
	}
	if cfg.GetSupabaseKey() != "test-key" {
		t.Fatalf("expected supabase key test-key, got %s", cfg.GetSupabaseKey())
	}
	if cfg.GetJWTSecret() != "secret" {
		t.Fatalf("expected jwt secret secret, got %s", cfg.GetJWTSecret())
	}
}

func TestNewConfig_Fallbacks(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("SERVER_PORT", "9091")
	t.Setenv("MAX_FILE_SIZE", "not-a-number")

	cfg := NewConfig()

	if cfg.GetServerPort() != "9091" {
		t.Fatalf("expected server port 9091, got %s", cfg.GetServerPort())
	}
	if cfg.GetMaxFileSize() != defaultMaxFileSize {
		t.Fatalf("expected default max file size %d, got %d", defaultMaxFileSize, cfg.GetMaxFileSize())
	}
}
