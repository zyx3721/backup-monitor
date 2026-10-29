// Package cron 解析 5 段 cron 表达式并计算最近的计划时刻。
package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type field uint64

type cronFields struct {
	minute field
	hour   field
	day    field
	month  field
	week   field
}

// Schedule 是解析后的 cron 计划。
type Schedule struct {
	raw    string
	parts  [5]string
	fields cronFields
}

var macros = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

const searchMinutes = 7 * 24 * 60

// Parse 解析 5 段 cron 表达式，日与星期同时限定时按 AND 语义匹配以与服务端判定一致。
func Parse(expr string) (Schedule, error) {
	raw := strings.TrimSpace(expr)
	if raw == "" {
		return Schedule{}, fmt.Errorf("empty cron expression")
	}
	expanded, ok := Expand(raw)
	if !ok {
		return Schedule{}, fmt.Errorf("invalid cron expression %q", expr)
	}
	parts := strings.Fields(expanded)
	if len(parts) != 5 {
		return Schedule{}, fmt.Errorf("invalid cron expression %q: expected 5 fields", expr)
	}

	ranges := [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	var parsed [5]field
	for index, part := range parts {
		value, err := parseField(part, ranges[index][0], ranges[index][1])
		if err != nil {
			return Schedule{}, fmt.Errorf("invalid cron expression %q: field %d: %w", expr, index+1, err)
		}
		parsed[index] = value
	}

	week := parsed[4]
	if week&(1<<7) != 0 {
		week = (week &^ (1 << 7)) | 1
	}
	if week == 0 {
		return Schedule{}, fmt.Errorf("invalid cron expression %q: weekday field is empty", expr)
	}

	return Schedule{
		raw:   raw,
		parts: [5]string{parts[0], parts[1], parts[2], parts[3], parts[4]},
		fields: cronFields{
			minute: parsed[0],
			hour:   parsed[1],
			day:    parsed[2],
			month:  parsed[3],
			week:   week,
		},
	}, nil
}

// Validate 判断 cron 表达式是否合法。
func Validate(expr string) bool {
	_, err := Parse(expr)
	return err == nil
}

// Expand 展开受支持的 cron 宏，失败时返回 false。
func Expand(expr string) (string, bool) {
	raw := strings.ToLower(strings.TrimSpace(expr))
	if !strings.HasPrefix(raw, "@") {
		return expr, true
	}
	value, ok := macros[raw]
	return value, ok
}

// Matches 判断时间是否命中计划。
func (s Schedule) Matches(t time.Time) bool {
	return s.fields.minute.has(t.Minute()) &&
		s.fields.hour.has(t.Hour()) &&
		s.fields.day.has(t.Day()) &&
		s.fields.month.has(int(t.Month())) &&
		s.fields.week.has(int(t.Weekday()))
}

// GetLastScheduledTime 返回不晚于 now 的最近一次计划时刻。
func GetLastScheduledTime(expr string, now time.Time) (time.Time, bool) {
	schedule, err := Parse(expr)
	if err != nil {
		return time.Time{}, false
	}
	cursor := now.Truncate(time.Minute)
	for index := 0; index <= searchMinutes; index++ {
		if schedule.Matches(cursor) {
			return cursor, true
		}
		cursor = cursor.Add(-time.Minute)
	}
	return time.Time{}, false
}

// GetNextScheduledTime 返回晚于 after 的下一次计划时刻。
func GetNextScheduledTime(expr string, after time.Time) (time.Time, bool) {
	schedule, err := Parse(expr)
	if err != nil {
		return time.Time{}, false
	}
	cursor := after.Truncate(time.Minute).Add(time.Minute)
	for index := 0; index < searchMinutes; index++ {
		if schedule.Matches(cursor) {
			return cursor, true
		}
		cursor = cursor.Add(time.Minute)
	}
	return time.Time{}, false
}

// ToHuman 把 cron 表达式转换为中文可读描述。
func ToHuman(expr string) string {
	schedule, err := Parse(expr)
	if err != nil {
		raw := strings.TrimSpace(expr)
		if raw == "" {
			return "未设置计划"
		}
		return "自定义计划 " + raw
	}
	parts := schedule.parts

	if step, ok := starStep(parts[0]); ok && allStar(parts[1:]) {
		return fmt.Sprintf("每 %d 分钟", step)
	}
	if minute, ok := singleValue(parts[0], 0, 59); ok && allStar(parts[1:]) {
		if minute == 0 {
			return "每小时"
		}
		return fmt.Sprintf("每小时第 %d 分", minute)
	}
	if minute, ok := singleValue(parts[0], 0, 59); ok {
		if step, hourStep := starStep(parts[1]); hourStep && allStar(parts[2:]) {
			if minute == 0 {
				return fmt.Sprintf("每 %d 小时", step)
			}
			return fmt.Sprintf("每 %d 小时第 %d 分", step, minute)
		}
		if hour, hourOK := singleValue(parts[1], 0, 23); hourOK {
			clock := fmt.Sprintf("%02d:%02d", hour, minute)
			dayStar := allStar(parts[2:4])
			if dayStar && allStar(parts[4:]) {
				return "每天 " + clock
			}
			if dayStar && !allStar(parts[4:]) {
				return "每" + humanWeekdays(schedule.fields.week) + " " + clock
			}
			if allStar(parts[4:]) && !allStar(parts[2:3]) {
				if !allStar(parts[3:4]) {
					if month, monthOK := singleValue(parts[3], 1, 12); monthOK {
						if day, dayOK := singleValue(parts[2], 1, 31); dayOK {
							return fmt.Sprintf("每年 %d 月 %d 日 %s", month, day, clock)
						}
					}
					return "自定义计划 " + schedule.raw
				}
				return "每月 " + humanDays(schedule.fields.day) + " 日 " + clock
			}
		}
	}
	return "自定义计划 " + schedule.raw
}

func parseField(text string, minimum, maximum int) (field, error) {
	value := strings.TrimSpace(text)
	if value == "" {
		return 0, fmt.Errorf("empty field")
	}
	var result field
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return 0, fmt.Errorf("empty list item")
		}
		base := part
		step := 1
		hasStep := false
		if strings.Count(part, "/") > 1 {
			return 0, fmt.Errorf("invalid step in %q", part)
		}
		if index := strings.Index(part, "/"); index >= 0 {
			base = strings.TrimSpace(part[:index])
			stepText := strings.TrimSpace(part[index+1:])
			parsedStep, err := strconv.Atoi(stepText)
			if err != nil || parsedStep <= 0 {
				return 0, fmt.Errorf("invalid step in %q", part)
			}
			step = parsedStep
			hasStep = true
		}

		start, end, err := fieldRange(base, minimum, maximum)
		if err != nil {
			return 0, err
		}
		if hasStep && base != "*" && base != "?" && !strings.Contains(base, "-") {
			end = maximum
		}
		for current := start; current <= end; current += step {
			result |= 1 << uint(current)
		}
	}
	return result, nil
}

func fieldRange(base string, minimum, maximum int) (int, int, error) {
	switch base {
	case "*", "?":
		return minimum, maximum, nil
	}
	if strings.Contains(base, "-") {
		segments := strings.Split(base, "-")
		if len(segments) != 2 {
			return 0, 0, fmt.Errorf("invalid range %q", base)
		}
		start, err := strconv.Atoi(strings.TrimSpace(segments[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range start %q", segments[0])
		}
		end, err := strconv.Atoi(strings.TrimSpace(segments[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range end %q", segments[1])
		}
		if start < minimum || end > maximum || start > end {
			return 0, 0, fmt.Errorf("range %q out of bounds", base)
		}
		return start, end, nil
	}
	value, err := strconv.Atoi(base)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid value %q", base)
	}
	if value < minimum || value > maximum {
		return 0, 0, fmt.Errorf("value %d out of bounds", value)
	}
	return value, value, nil
}

func (f field) has(value int) bool {
	if value < 0 || value >= 64 {
		return false
	}
	return f&(1<<uint(value)) != 0
}

func allStar(parts []string) bool {
	for _, part := range parts {
		if part != "*" && part != "?" {
			return false
		}
	}
	return true
}

func starStep(part string) (int, bool) {
	if !strings.HasPrefix(part, "*/") {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimPrefix(part, "*/"))
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func singleValue(part string, minimum, maximum int) (int, bool) {
	value, err := strconv.Atoi(part)
	if err != nil || value < minimum || value > maximum {
		return 0, false
	}
	return value, true
}

func humanWeekdays(week field) string {
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	days := make([]int, 0, 7)
	for day := 0; day <= 6; day++ {
		if week.has(day) {
			days = append(days, day)
		}
	}
	if len(days) == 0 {
		return "周日"
	}
	parts := make([]string, 0, len(days))
	for start := 0; start < len(days); {
		end := start
		for end+1 < len(days) && days[end+1] == days[end]+1 {
			end++
		}
		if end > start {
			parts = append(parts, names[days[start]]+"至"+names[days[end]])
		} else {
			parts = append(parts, names[days[start]])
		}
		start = end + 1
	}
	return strings.Join(parts, "、")
}

func humanDays(days field) string {
	values := make([]string, 0, 31)
	for day := 1; day <= 31; day++ {
		if days.has(day) {
			values = append(values, strconv.Itoa(day))
		}
	}
	if len(values) == 0 {
		return "1"
	}
	return strings.Join(values, "、")
}
