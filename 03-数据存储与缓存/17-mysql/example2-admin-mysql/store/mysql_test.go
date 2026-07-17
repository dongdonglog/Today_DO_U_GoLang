package store

import (
	"database/sql"
	"testing"

	"github.com/go-book/mysql/example2-admin-mysql/model"
)

func TestBuildListQueryUsesExactEmailFilter(t *testing.T) {
	status := 1
	query := buildListQuery(&model.ListUsersRequest{
		Page:   2,
		Size:   20,
		Email:  "alice@example.com",
		Status: &status,
	})

	wantCount := "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND active_email = ? AND status = ?"
	if query.CountSQL != wantCount {
		t.Fatalf("unexpected count sql: %s", query.CountSQL)
	}
	if len(query.Args) != 2 || query.Args[0] != "alice@example.com" || query.Args[1] != status {
		t.Fatalf("unexpected args: %#v", query.Args)
	}
	if query.Page != 2 || query.Size != 20 {
		t.Fatalf("unexpected pagination: page=%d size=%d", query.Page, query.Size)
	}
	if got := query.ListArgs[len(query.ListArgs)-2]; got != 20 {
		t.Fatalf("unexpected limit arg: %#v", got)
	}
	if got := query.ListArgs[len(query.ListArgs)-1]; got != 20 {
		t.Fatalf("unexpected offset arg: %#v", got)
	}
}

func TestBuildListQueryDefaultsPagination(t *testing.T) {
	query := buildListQuery(&model.ListUsersRequest{})

	if query.Page != 1 || query.Size != 10 {
		t.Fatalf("unexpected default pagination: page=%d size=%d", query.Page, query.Size)
	}
	if got := query.ListArgs[len(query.ListArgs)-2]; got != 10 {
		t.Fatalf("unexpected default limit: %#v", got)
	}
	if got := query.ListArgs[len(query.ListArgs)-1]; got != 0 {
		t.Fatalf("unexpected default offset: %#v", got)
	}
}

func TestNullStringPtr(t *testing.T) {
	if got := nullStringPtr(sql.NullString{}); got != nil {
		t.Fatalf("expected nil pointer, got %#v", got)
	}

	got := nullStringPtr(sql.NullString{String: "13800138000", Valid: true})
	if got == nil || *got != "13800138000" {
		t.Fatalf("unexpected string pointer: %#v", got)
	}
}

func TestNullableString(t *testing.T) {
	if got := nullableString(" "); got != nil {
		t.Fatalf("expected nil for blank string, got %#v", got)
	}
	if got := nullableString("13800138000"); got != "13800138000" {
		t.Fatalf("unexpected nullable string: %#v", got)
	}
}
