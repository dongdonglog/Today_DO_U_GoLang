package main

import (
	"strings"
	"testing"
)

func TestParseLogLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantNil  bool
		wantLvl  string
		wantMsg  string
	}{
		{
			name:    "ERROR 日志",
			line:    "2024-01-01 10:00:00 ERROR ERROR_TIMEOUT: connection timeout",
			wantNil: false,
			wantLvl: "ERROR",
			wantMsg: "ERROR_TIMEOUT: connection timeout",
		},
		{
			name:    "WARN 日志",
			line:    "2024-01-01 10:00:00 WARN WARN_SLOW: slow query",
			wantNil: false,
			wantLvl: "WARN",
			wantMsg: "WARN_SLOW: slow query",
		},
		{
			name:    "INFO 日志",
			line:    "2024-01-01 10:00:00 INFO INFO_START: server started",
			wantNil: false,
			wantLvl: "INFO",
			wantMsg: "INFO_START: server started",
		},
		{
			name:    "太短的行",
			line:    "short",
			wantNil: true,
		},
		{
			name:    "无效格式",
			line:    "2024-01-01 10:00:00 UNKNOWN something",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLogLine(tt.line)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseLogLine() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ParseLogLine() = nil, want non-nil")
			}
			if got.Level != tt.wantLvl {
				t.Errorf("Level = %v, want %v", got.Level, tt.wantLvl)
			}
			if got.Message != tt.wantMsg {
				t.Errorf("Message = %v, want %v", got.Message, tt.wantMsg)
			}
		})
	}
}

func TestAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()

	// 测试数据
	logs := `2024-01-01 10:00:00 ERROR ERROR_TIMEOUT: timeout
2024-01-01 10:00:01 ERROR ERROR_TIMEOUT: timeout
2024-01-01 10:00:02 ERROR ERROR_CONNECTION: connection failed
2024-01-01 10:00:03 INFO INFO_START: started
2024-01-01 10:00:04 ERROR ERROR_AUTH: auth failed
2024-01-01 10:00:05 ERROR ERROR_TIMEOUT: timeout`

	reader := strings.NewReader(logs)
	if err := analyzer.AnalyzeReader(reader); err != nil {
		t.Fatalf("AnalyzeReader() error = %v", err)
	}

	// 验证总错误数
	if got := analyzer.TotalErrors(); got != 5 {
		t.Errorf("TotalErrors() = %v, want 5", got)
	}

	// 验证 Top 2
	top2 := analyzer.TopN(2)
	if len(top2) != 2 {
		t.Fatalf("TopN(2) returned %d items, want 2", len(top2))
	}

	// ERROR_TIMEOUT 应该最多（3 次）
	if top2[0].ErrorType != "ERROR_TIMEOUT" {
		t.Errorf("Top 1 = %v, want ERROR_TIMEOUT", top2[0].ErrorType)
	}
	if top2[0].Count != 3 {
		t.Errorf("Top 1 count = %v, want 3", top2[0].Count)
	}
}

func TestExtractErrorType(t *testing.T) {
	tests := []struct {
		message string
		want    string
	}{
		{"ERROR_TIMEOUT: connection timeout", "ERROR_TIMEOUT"},
		{"ERROR_CONNECTION: failed", "ERROR_CONNECTION"},
		{"no colon", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractErrorType(tt.message)
		if got != tt.want {
			t.Errorf("extractErrorType(%q) = %q, want %q", tt.message, got, tt.want)
		}
	}
}
