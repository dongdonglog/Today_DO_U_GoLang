package main

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string
	Level     string
	Message   string
}

// ErrorStats 错误统计
type ErrorStats struct {
	ErrorType string
	Count     int
}

// ParseLogLine 解析日志行
// 格式: "2024-01-01 12:00:00 ERROR ERROR_TIMEOUT: connection timeout"
func ParseLogLine(line string) *LogEntry {
	if len(line) < 20 {
		return nil
	}

	// 简单解析：假设格式固定
	// 实际项目中应该用正则表达式
	timestamp := line[:19]
	rest := line[20:]

	// 查找级别
	var level string
	if len(rest) >= 5 && rest[:5] == "ERROR" {
		level = "ERROR"
		rest = rest[6:] // 跳过 "ERROR "
	} else if len(rest) >= 4 && rest[:4] == "WARN" {
		level = "WARN"
		rest = rest[5:]
	} else if len(rest) >= 4 && rest[:4] == "INFO" {
		level = "INFO"
		rest = rest[5:]
	} else {
		return nil
	}

	return &LogEntry{
		Timestamp: timestamp,
		Level:     level,
		Message:   rest,
	}
}
