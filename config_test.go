package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSecretsLoadsPasswordFile(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(secretPath, []byte("super-secret\n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	cfg := Config{
		Auth: CfgAuth{
			Username:     "admin",
			PasswordFile: secretPath,
		},
	}

	if err := cfg.resolveSecrets(); err != nil {
		t.Fatalf("resolve secrets: %v", err)
	}

	if cfg.Auth.Password != "super-secret" {
		t.Fatalf("expected trimmed password, got %q", cfg.Auth.Password)
	}
}

func TestResolveSecretsKeepsInlinePasswordWithoutPasswordFile(t *testing.T) {
	cfg := Config{
		Auth: CfgAuth{
			Username: "admin",
			Password: "inline-secret",
		},
	}

	if err := cfg.resolveSecrets(); err != nil {
		t.Fatalf("resolve secrets: %v", err)
	}

	if cfg.Auth.Password != "inline-secret" {
		t.Fatalf("expected inline password to remain unchanged, got %q", cfg.Auth.Password)
	}
}

func TestResolveSecretsRejectsEmptyPasswordFile(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(secretPath, []byte("\n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	cfg := Config{
		Auth: CfgAuth{
			Username:     "admin",
			PasswordFile: secretPath,
		},
	}

	if err := cfg.resolveSecrets(); err == nil {
		t.Fatal("expected empty password file to fail")
	}
}
