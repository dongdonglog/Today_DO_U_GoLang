package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const baseConfig = `
app:
  name: admin-api
  env: dev
  version: 1.0.0

server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 5s
  write_timeout: 10s

database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ""
  name: admin_db
  max_open_conns: 100
  max_idle_conns: 10

redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0

log:
  level: info
  format: json
  output: stdout
`

func writeConfig(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
		t.Fatalf("write config %s: %v", name, err)
	}
}

func TestLoadBaseConfig(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.yaml", baseConfig)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Name != "admin-api" || cfg.Server.Port != 8080 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.yaml", baseConfig)

	t.Setenv("APP_ENV", "test")
	t.Setenv("ADMIN_API_SERVER_PORT", "9090")
	t.Setenv("DB_PASSWORD", "from-env")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Env != "test" {
		t.Fatalf("expected env test, got %s", cfg.App.Env)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.Password != "from-env" {
		t.Fatalf("expected password from env")
	}
}

func TestProdRequiresDatabasePassword(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.yaml", baseConfig)
	writeConfig(t, dir, "config.prod.yaml", `
app:
  env: prod
`)

	t.Setenv("APP_ENV", "prod")

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "database password is required") {
		t.Fatalf("expected prod password error, got %v", err)
	}
}

func TestInvalidPortFailsValidation(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.yaml", strings.Replace(baseConfig, "port: 8080", "port: 70000", 1))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "invalid server port") {
		t.Fatalf("expected invalid port error, got %v", err)
	}
}

func TestMaskedDSNDoesNotLeakPassword(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "secret-value",
		Name:     "admin_db",
	}

	masked := cfg.MaskedDSN()
	if strings.Contains(masked, "secret-value") {
		t.Fatalf("masked DSN leaked password: %s", masked)
	}
	if !strings.Contains(masked, "******") {
		t.Fatalf("masked DSN did not include mask: %s", masked)
	}
}
