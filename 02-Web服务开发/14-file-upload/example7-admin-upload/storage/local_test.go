package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidFilenameAllowsSafeRelativePath(t *testing.T) {
	if !IsValidFilename("2026/07/08/avatar.jpg") {
		t.Fatalf("expected date-based relative path to be valid")
	}
}

func TestIsValidFilenameRejectsTraversal(t *testing.T) {
	cases := []string{
		"../etc/passwd",
		"2026/../../etc/passwd",
		"/etc/passwd",
		`2026\..\secret`,
	}

	for _, tc := range cases {
		if IsValidFilename(tc) {
			t.Fatalf("expected %q to be invalid", tc)
		}
	}
}

func TestGetRejectsPathTraversal(t *testing.T) {
	s := NewLocalStorage(t.TempDir(), []string{"text/plain; charset=utf-8"}, 1024)

	if _, err := s.Get("../etc/passwd"); err == nil {
		t.Fatalf("expected traversal path rejected")
	}
}

func TestGetAllowsSavedRelativePath(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir, []string{"text/plain; charset=utf-8"}, 1024)

	relativePath := "2026/07/08/test.txt"
	fullPath := filepath.Join(dir, relativePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := s.Get(relativePath)
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	if !strings.HasSuffix(got, filepath.FromSlash(relativePath)) {
		t.Fatalf("unexpected path: %s", got)
	}
}
