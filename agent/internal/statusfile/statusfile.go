// Package statusfile 解析备份状态文件中的时间、大小、退出码与状态行。
package statusfile

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var sizePattern = regexp.MustCompile(`(?i)^([0-9]+(?:[.,][0-9]+)?)\s*([a-z]*)$`)

// Info 是状态文件解析结果。
type Info struct {
	Start            time.Time
	HasStart         bool
	End              time.Time
	HasEnd           bool
	Time             time.Time
	HasTime          bool
	SizeBytes        int64
	HasSize          bool
	TransferredBytes int64
	HasTransferred   bool
	ExitCode         int
	HasExit          bool
	Status           string
}

// ParseFile 读取并解析状态文件。
func ParseFile(path string) (Info, error) {
	file, err := os.Open(path)
	if err != nil {
		return Info{}, fmt.Errorf("open status file %s: %w", path, err)
	}
	defer file.Close()
	info, err := Parse(file)
	if err != nil {
		return Info{}, fmt.Errorf("parse status file %s: %w", path, err)
	}
	return info, nil
}

// Parse 从文本流解析状态行，无法识别的行会被忽略。
func Parse(reader io.Reader) (Info, error) {
	data, err := io.ReadAll(io.LimitReader(reader, 1<<20))
	if err != nil {
		return Info{}, err
	}
	var info Info
	if parsed, ok := parseJSON(data); ok {
		info = parsed
	}
	lines := make([]string, 0, 16)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		lines = append(lines, line)
		key, value, ok := splitLine(line)
		if !ok {
			continue
		}
		applyLine(&info, key, value)
	}
	if err := scanner.Err(); err != nil {
		return Info{}, err
	}
	if !info.HasStart && !info.HasEnd && !info.HasTime {
		applyFreeformTimes(&info, lines)
	}
	if info.Status == "" {
		info.Status = freeformStatus(lines)
	}
	return info, nil
}

func parseJSON(data []byte) (Info, bool) {
	trimmed := strings.TrimSpace(string(data))
	if !strings.HasPrefix(trimmed, "{") {
		return Info{}, false
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return Info{}, false
	}
	flat := make(map[string]any, len(raw))
	flattenJSON("", raw, flat, 0)
	var info Info
	for key, value := range flat {
		switch typed := value.(type) {
		case string:
			applyLine(&info, key, typed)
		case float64:
			applyLine(&info, key, strconv.FormatFloat(typed, 'f', -1, 64))
		}
	}
	return info, true
}

func flattenJSON(prefix string, raw map[string]any, flat map[string]any, depth int) {
	if depth > 3 {
		return
	}
	for key, value := range raw {
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}
		if nested, ok := value.(map[string]any); ok {
			flattenJSON(name, nested, flat, depth+1)
			continue
		}
		flat[name] = value
	}
}

func applyFreeformTimes(info *Info, lines []string) {
	times := make([]time.Time, 0, 2)
	for _, line := range lines {
		if parsed, ok := parseTime(strings.Trim(line, "[]()")); ok {
			times = append(times, parsed)
			continue
		}
		if parsed, ok := parseTime(leadingTimestamp(line)); ok {
			times = append(times, parsed)
		}
	}
	if len(times) > 0 {
		info.Start, info.HasStart = times[0], true
	}
	if len(times) > 1 {
		info.End, info.HasEnd = times[1], true
	}
}

func leadingTimestamp(line string) string {
	if len(line) > 19 {
		return strings.TrimSpace(line[:19])
	}
	return strings.TrimSpace(line)
}

func freeformStatus(lines []string) string {
	for _, line := range lines {
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "success"), strings.Contains(lower, "succeeded"),
			strings.Contains(lower, "completed"), strings.Contains(lower, "finished"):
			return "success"
		case strings.Contains(lower, "failed"), strings.Contains(lower, "failure"):
			return "failed"
		case strings.Contains(lower, "running"), strings.Contains(lower, "in progress"):
			return "running"
		}
	}
	return ""
}

func splitLine(line string) (string, string, bool) {
	var key, value string
	if index := strings.Index(line, "="); index >= 0 {
		key = line[:index]
		value = line[index+1:]
	} else if index := strings.Index(line, ":"); index >= 0 {
		key = line[:index]
		value = line[index+1:]
	} else {
		return "", "", false
	}
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func normalizeKey(key string) string {
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", "\"", "", "'", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(key)))
}

func applyLine(info *Info, key, value string) {
	switch normalizeKey(key) {
	case "start", "starttime":
		if parsed, ok := parseTime(value); ok {
			info.Start, info.HasStart = parsed, true
		}
	case "end", "endtime":
		if parsed, ok := parseTime(value); ok {
			info.End, info.HasEnd = parsed, true
		}
	case "time", "timestamp", "date":
		if parsed, ok := parseTime(value); ok {
			info.Time, info.HasTime = parsed, true
		}
	case "size", "processedsize", "totalsize":
		if parsed, err := ParseSize(value); err == nil {
			info.SizeBytes, info.HasSize = parsed, true
		}
	case "transferred", "transferredsize":
		if parsed, err := ParseSize(value); err == nil {
			info.TransferredBytes, info.HasTransferred = parsed, true
		}
	case "exit", "exitcode", "code", "result":
		if parsed, err := parseInt(value); err == nil {
			info.ExitCode, info.HasExit = parsed, true
		}
	case "status", "state":
		info.Status = normalizeStatus(value)
	}
}

func parseTime(value string) (time.Time, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Time{}, false
	}
	if isDigits(text) {
		stamp, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return time.Time{}, false
		}
		if len(text) >= 13 {
			return time.UnixMilli(stamp), true
		}
		return time.Unix(stamp, 0), true
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02 15:04",
		"2006/01/02 15:04",
		"01/02/2006 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func parseInt(value string) (int, error) {
	text := strings.TrimSpace(value)
	if parsed, err := strconv.ParseInt(text, 0, 64); err == nil {
		return int(parsed), nil
	}
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(parsed), nil
}

// ParseSize 解析带单位的文件大小，支持 KB、MB、GB、TB 与二进制单位。
func ParseSize(text string) (int64, error) {
	value := strings.TrimSpace(strings.ToLower(text))
	if value == "" {
		return 0, fmt.Errorf("empty size")
	}

	matches := sizePattern.FindStringSubmatch(value)
	if matches == nil {
		return 0, fmt.Errorf("invalid size %q", text)
	}
	number, err := strconv.ParseFloat(normalizeDecimal(matches[1]), 64)
	if err != nil || number < 0 {
		return 0, fmt.Errorf("invalid size %q", text)
	}
	unit := strings.ToLower(matches[2])
	multiplier := float64(1)
	switch unit {
	case "", "b", "byte", "bytes":
		multiplier = 1
	case "k", "kb", "kib":
		multiplier = 1024
	case "m", "mb", "mib":
		multiplier = 1024 * 1024
	case "g", "gb", "gib":
		multiplier = 1024 * 1024 * 1024
	case "t", "tb", "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unsupported size unit %q", matches[2])
	}
	return int64(number*multiplier + 0.5), nil
}

// FormatSize 把字节数格式化为便于展示的大小文本。
func FormatSize(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}
	const (
		kilobyte = int64(1024)
		megabyte = 1024 * kilobyte
		gigabyte = 1024 * megabyte
		terabyte = 1024 * gigabyte
	)
	switch {
	case bytes >= terabyte:
		return formatUnit(float64(bytes)/float64(terabyte), "TB")
	case bytes >= gigabyte:
		return formatUnit(float64(bytes)/float64(gigabyte), "GB")
	case bytes >= megabyte:
		return formatUnit(float64(bytes)/float64(megabyte), "MB")
	case bytes >= kilobyte:
		return formatUnit(float64(bytes)/float64(kilobyte), "KB")
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatUnit(value float64, unit string) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return text + " " + unit
}

func normalizeDecimal(value string) string {
	if !strings.Contains(value, ",") {
		return value
	}
	if strings.Count(value, ",") == 1 && !strings.Contains(value, ".") {
		parts := strings.SplitN(value, ",", 2)
		if len(parts[1]) == 3 {
			return parts[0] + parts[1]
		}
		return parts[0] + "." + parts[1]
	}
	return strings.ReplaceAll(value, ",", "")
}

func normalizeStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "success", "succeeded", "ok", "completed", "done", "finished":
		return "success"
	case "failed", "failure", "error", "fail":
		return "failed"
	case "running", "inprogress", "in-progress", "started", "working":
		return "running"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
