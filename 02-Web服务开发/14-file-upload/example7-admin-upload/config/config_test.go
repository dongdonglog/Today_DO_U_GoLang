package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testConfig = `
server:
  host: 127.0.0.1
  port: 8080

jwt:
  access_secret: ""
  refresh_secret: ""
  access_token_expire: 15m
  refresh_token_expire: 168h

upload:
  max_size: 10485760
  allowed_types:
    - image/jpeg
    - image/png
    - application/pdf
  storage_path: ./uploads

log:
  level: info
`

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestLoadSecretsFromEnv(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, testConfig)
	t.Setenv("JWT_ACCESS_SECRET", strings.Repeat("a", 32))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("b", 32))

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.JWT.AccessSecret != strings.Repeat("a", 32) {
		t.Fatalf("expected access secret from env")
	}
	if cfg.JWT.RefreshSecret != strings.Repeat("b", 32) {
		t.Fatalf("expected refresh secret from env")
	}
}

func TestLoadRejectsWeakSecret(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, strings.Replace(testConfig, `access_secret: ""`, `access_secret: "secret"`, 1))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("b", 32))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "too weak") {
		t.Fatalf("expected weak secret error, got %v", err)
	}
}
