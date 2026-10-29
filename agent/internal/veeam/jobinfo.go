package veeam

import (
	"context"
	"regexp"
	"strings"
)

var (
	clockPattern     = regexp.MustCompile(`\b([01]?\d|2[0-3]):([0-5]\d)\b`)
	dayPattern       = regexp.MustCompile(`(?i)\bday\s*(?:of\s*(?:the\s*)?month)?\s*[:=]?\s*(\d{1,2})\b`)
	ordinalPattern   = regexp.MustCompile(`(?i)\b(\d{1,2})(?:st|nd|rd|th)\b`)
	beforeDayPattern = regexp.MustCompile(`(?i)\b(\d{1,2})\s*days?\b`)
)

// JobInfo 是 veeamconfig job info 的解析结果。
type JobInfo struct {
	ID          string
	Name        string
	Repository  string
	Schedule    string
	ActiveFull  string
	Retention   string
	Compression string
	JobType     string
	Objects     []string
}

// JobInfo 解析 veeamconfig job info 的键值与分节文本。
func (c *Client) JobInfo(ctx context.Context, jobID string) (JobInfo, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return JobInfo{}, nil
	}
	output, err := c.run(ctx, "job", "info", "--id", jobID)
	if err != nil {
		return JobInfo{}, err
	}
	return parseJobInfo(output), nil
}

func parseJobInfo(output string) JobInfo {
	info := JobInfo{}
	values := make(map[string]string, 16)
	sections := make(map[string][]string, 4)
	section := ""
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isSeparator(trimmed) || strings.HasPrefix(trimmed, "Backup job") {
			continue
		}
		if name, ok := sectionName(trimmed); ok {
			section = name
			continue
		}
		key, value, ok := splitJobInfoLine(trimmed)
		if !ok {
			if section != "" {
				sections[section] = append(sections[section], trimmed)
			}
			continue
		}
		values[normalizeJobInfoKey(key)] = value
		if strings.Contains(normalizeJobInfoKey(key), "activefull") {
			section = ""
			continue
		}
		if section != "" {
			sections[section] = append(sections[section], trimmed)
		}
	}
	info.ID = values["id"]
	info.Name = values["name"]
	info.JobType = values["type"]
	info.Repository = firstValue(values, "repositoryname", "repository")
	info.Compression = firstValue(values, "compression", "compressionlevel")
	info.Retention = normalizeRetention(firstValue(values, "maxpoints", "restorepoints", "retention"))
	info.Schedule = humanizeSchedule(jobInfoScheduleLines(values, sections))
	info.ActiveFull = humanizeActiveFull(jobInfoActiveFullLines(values, sections))
	info.Objects = collectObjects(sections["objects"])
	return info
}

func sectionName(line string) (string, bool) {
	if !strings.HasSuffix(line, ":") {
		return "", false
	}
	name := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(line, ":")))
	switch {
	case strings.Contains(name, "objects"):
		return "objects", true
	case strings.Contains(name, "full"):
		return "activefull", true
	case strings.Contains(name, "schedule"):
		return "schedule", true
	}
	return "", true
}

func splitJobInfoLine(line string) (string, string, bool) {
	index := strings.Index(line, ":")
	if index <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:index])
	value := strings.TrimSpace(line[index+1:])
	if key == "" || value == "" {
		return "", "", false
	}
	return key, value, true
}

func normalizeJobInfoKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.ReplaceAll(key, " ", "")
}

func firstValue(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func normalizeRetention(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if match := dayPattern.FindStringSubmatch(value); match != nil {
		return match[1]
	}
	fields := strings.Fields(value)
	for _, field := range fields {
		if isAllDigits(field) {
			return field
		}
	}
	return value
}

func humanizeSchedule(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	text := strings.ToLower(strings.Join(lines, " "))
	clock := clockText(text)
	weekday := weekdayText(text)
	switch {
	case strings.Contains(text, "disabled"), strings.Contains(text, "not scheduled"):
		return ""
	case strings.Contains(text, "monthly"), strings.Contains(text, "every month"):
		if day := dayText(text); day != "" {
			return joinParts("每月 "+day+" 日", clock)
		}
	case strings.Contains(text, "weekly"), strings.Contains(text, "every week"):
		if weekday != "" {
			return joinParts("每周"+weekday, clock)
		}
	case strings.Contains(text, "daily"), strings.Contains(text, "every day"):
		return joinParts("每天", clock)
	case strings.Contains(text, "hourly"):
		return "每小时"
	}
	if clock != "" {
		return "每天 " + clock
	}
	return ""
}

func humanizeActiveFull(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	text := strings.ToLower(strings.Join(lines, " "))
	switch {
	case strings.Contains(text, "disabled"), strings.Contains(text, "off"):
		return "未启用"
	case strings.Contains(text, "monthly"), strings.Contains(text, "every month"):
		if day := dayText(text); day != "" {
			return "每月 " + day + " 日"
		}
		return "每月"
	case strings.Contains(text, "weekly"), strings.Contains(text, "every week"):
		if weekday := weekdayText(text); weekday != "" {
			return "每周" + weekday
		}
		return "每周"
	}
	return ""
}

func collectObjects(lines []string) []string {
	objects := make([]string, 0, len(lines))
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if index := strings.LastIndex(value, ":"); index >= 0 {
			value = strings.TrimSpace(value[index+1:])
		}
		value = strings.Trim(value, "\"'")
		if value == "" || strings.EqualFold(value, "enabled") || strings.EqualFold(value, "disabled") {
			continue
		}
		if !strings.HasPrefix(value, "/") && !strings.Contains(value, "\\") && !strings.Contains(value, "/") {
			value = "/dev/" + value
		}
		objects = append(objects, value)
	}
	return objects
}

func clockText(text string) string {
	match := clockPattern.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	hour := match[1]
	if len(hour) == 1 {
		hour = "0" + hour
	}
	return hour + ":" + match[2]
}

func dayText(text string) string {
	if match := dayPattern.FindStringSubmatch(text); match != nil {
		return match[1]
	}
	if match := ordinalPattern.FindStringSubmatch(text); match != nil {
		return match[1]
	}
	if match := beforeDayPattern.FindStringSubmatch(text); match != nil {
		return match[1]
	}
	return ""
}

func weekdayText(text string) string {
	weekdays := []struct {
		key   string
		label string
	}{
		{"monday", "一"}, {"tuesday", "二"}, {"wednesday", "三"}, {"thursday", "四"},
		{"friday", "五"}, {"saturday", "六"}, {"sunday", "日"},
	}
	for _, item := range weekdays {
		if strings.Contains(text, item.key) {
			return item.label
		}
	}
	return ""
}

func joinParts(prefix, clock string) string {
	if clock == "" {
		return prefix
	}
	return prefix + " " + clock
}

func isAllDigits(value string) bool {
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

func jobInfoScheduleLines(values map[string]string, sections map[string][]string) []string {
	if lines := sections["schedule"]; len(lines) > 0 {
		return lines
	}
	if value := values["schedule"]; value != "" {
		return []string{value}
	}
	return nil
}

func jobInfoActiveFullLines(values map[string]string, sections map[string][]string) []string {
	if lines := sections["activefull"]; len(lines) > 0 {
		return lines
	}
	lines := make([]string, 0, 2)
	for key, value := range values {
		if strings.Contains(key, "activefull") && value != "" {
			lines = append(lines, value)
		}
	}
	return lines
}
