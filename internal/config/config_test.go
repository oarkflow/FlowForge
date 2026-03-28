package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()
	if cfg == nil {
		t.Fatal("Load() should return non-nil config")
	}
	if cfg.Port != "8082" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8082")
	}
	if cfg.DatabasePath != "data/flowforge.db" {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, "data/flowforge.db")
	}
	if cfg.JWTSecret != "change-me-in-production" {
		t.Errorf("JWTSecret = %q, want default", cfg.JWTSecret)
	}
	if cfg.JWTExpiration != 15*time.Minute {
		t.Errorf("JWTExpiration = %v, want %v", cfg.JWTExpiration, 15*time.Minute)
	}
	if cfg.RefreshExpiration != 7*24*time.Hour {
		t.Errorf("RefreshExpiration = %v, want %v", cfg.RefreshExpiration, 7*24*time.Hour)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.MaxUploadSize != 50*1024*1024 {
		t.Errorf("MaxUploadSize = %d, want %d", cfg.MaxUploadSize, 50*1024*1024)
	}
	if !cfg.EmbeddedWorker {
		t.Error("EmbeddedWorker should default to true")
	}
	if cfg.AllowedOrigins != "*" {
		t.Errorf("AllowedOrigins = %q, want %q", cfg.AllowedOrigins, "*")
	}
}

func TestLoad_OverrideFromEnv(t *testing.T) {
	// Set env var with FLOWFORGE prefix
	os.Setenv("FLOWFORGE_PORT", "9090")
	os.Setenv("FLOWFORGE_LOG_LEVEL", "debug")
	os.Setenv("FLOWFORGE_EMBEDDED_WORKER", "false")
	defer func() {
		os.Unsetenv("FLOWFORGE_PORT")
		os.Unsetenv("FLOWFORGE_LOG_LEVEL")
		os.Unsetenv("FLOWFORGE_EMBEDDED_WORKER")
	}()

	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q (from env)", cfg.Port, "9090")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q (from env)", cfg.LogLevel, "debug")
	}
}

func TestLoad_VaultDefaults(t *testing.T) {
	cfg := Load()
	if cfg.VaultAuthMethod != "token" {
		t.Errorf("VaultAuthMethod = %q, want %q", cfg.VaultAuthMethod, "token")
	}
	if cfg.VaultMountPath != "secret" {
		t.Errorf("VaultMountPath = %q, want %q", cfg.VaultMountPath, "secret")
	}
	if cfg.VaultPrefix != "flowforge" {
		t.Errorf("VaultPrefix = %q, want %q", cfg.VaultPrefix, "flowforge")
	}
}

func TestLoad_AWSDefaults(t *testing.T) {
	cfg := Load()
	if cfg.AWSSecretsPrefix != "flowforge/" {
		t.Errorf("AWSSecretsPrefix = %q, want %q", cfg.AWSSecretsPrefix, "flowforge/")
	}
}

func TestLoad_GCPDefaults(t *testing.T) {
	cfg := Load()
	if cfg.GCPSecretsPrefix != "flowforge-" {
		t.Errorf("GCPSecretsPrefix = %q, want %q", cfg.GCPSecretsPrefix, "flowforge-")
	}
}

func TestLoad_SecretRotationCheckInterval(t *testing.T) {
	cfg := Load()
	if cfg.SecretRotationCheckInterval != 1*time.Hour {
		t.Errorf("SecretRotationCheckInterval = %v, want %v", cfg.SecretRotationCheckInterval, 1*time.Hour)
	}
}
