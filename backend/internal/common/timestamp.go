package common

import (
	"errors"
	"strings"
	"time"
)

// TimestampLayout 统一时间格式：UTC、固定三位毫秒、Z 结尾，定长便于按字符串比较先后
const TimestampLayout = "2006-01-02T15:04:05.000Z"

// SQLTimestampNow SQLite 侧生成统一格式当前时间的表达式
const SQLTimestampNow = "strftime('%Y-%m-%dT%H:%M:%fZ','now')"

// NowTimestamp 返回统一格式的当前时间
func NowTimestamp() string {
	return time.Now().UTC().Format(TimestampLayout)
}

// FormatTimestamp 按统一格式格式化时间
func FormatTimestamp(value time.Time) string {
	return value.UTC().Format(TimestampLayout)
}

// ParseTimestamp 解析统一格式，兼容历史 RFC3339 文本
func ParseTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	if parsed, err := time.Parse(TimestampLayout, value); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

// TimestampUnix 解析统一格式并返回 Unix 秒，无法解析时返回 0
func TimestampUnix(value string) int64 {
	parsed, err := ParseTimestamp(value)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}
