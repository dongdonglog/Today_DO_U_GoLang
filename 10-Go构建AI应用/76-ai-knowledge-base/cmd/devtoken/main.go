package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"example.com/go-book/76-ai-knowledge-base/internal/auth"
	"example.com/go-book/76-ai-knowledge-base/internal/config"
)

func main() {
	tenant := flag.String("tenant", "demo-tenant", "tenant ID for the local token")
	rolesArg := flag.String("roles", "knowledge_reader", "comma-separated knowledge roles")
	ttl := flag.Duration("ttl", 15*time.Minute, "token lifetime (maximum 24h)")
	flag.Parse()

	if err := config.LoadDotEnv(".env"); err != nil {
		fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	if len(cfg.JWTSecret) < 32 {
		fatal(fmt.Errorf("KB_JWT_SECRET must contain at least 32 bytes"))
	}
	roles := make([]string, 0)
	for _, role := range strings.Split(*rolesArg, ",") {
		role = strings.TrimSpace(role)
		if role == "knowledge_admin" || role == "knowledge_reader" {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		fatal(fmt.Errorf("roles must include knowledge_admin or knowledge_reader"))
	}
	token, err := auth.IssueToken([]byte(cfg.JWTSecret), cfg.JWTIssuer, strings.TrimSpace(*tenant), roles, *ttl)
	if err != nil {
		fatal(err)
	}
	fmt.Println(token)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
