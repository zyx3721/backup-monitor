package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Cron 解析后的 5 段计划表达式。
type Cron struct {
	minute, hour, dom, month, dow fieldMatcher
	expr                          string
}

type fieldMatcher func(int) bool

type cronField struct {
	min, max int
}

var cronFields = []cronField{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}

// IsValidCron 校验 5 段 cron 表达式。
func IsValidCron(expr string) bool {
	_, err := ParseCron(expr)
	return err == nil
}

// ParseCron 解析 5 段 cron 表达式，支持 * ? 列表 区间 步进。
func ParseCron(expr string) (*Cron, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron 表达式需为 5 段（分 时 日 月 周）")
	}
	matchers := make([]fieldMatcher, 5)
	for i, part := range parts {
		matcher, err := parseCronField(part, cronFields[i].min, cronFields[i].max)
		if err != nil {
			return nil, fmt.Errorf("第 %d 段无效：%w", i+1, err)
		}
		matchers[i] = matcher
	}
	return &Cron{minute: matchers[0], hour: matchers[1], dom: matchers[2], month: matchers[3], dow: matchers[4], expr: strings.Join(parts, " ")}, nil
}

func parseCronField(field string, min, max int) (fieldMatcher, error) {
	if field == "" {
		return nil, fmt.Errorf("字段为空")
	}
	matchers := []fieldMatcher{}
	for _, part := range strings.Split(field, ",") {
		if part == "" {
			return nil, fmt.Errorf("存在空列表项")
		}
		step := 1
		base := part
		if strings.Contains(part, "/") {
			chunks := strings.Split(part, "/")
			if len(chunks) != 2 {
				return nil, fmt.Errorf("步进格式无效：%s", part)
			}
			parsed, err := strconv.Atoi(chunks[1])
			if err != nil || parsed <= 0 {
				return nil, fmt.Errorf("步进值无效：%s", part)
			}
			step = parsed
			base = chunks[0]
		}
		start, end := min, max
		switch {
		case base == "*" || base == "?":
		case strings.Contains(base, "-"):
			bounds := strings.Split(base, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("区间格式无效：%s", part)
			}
			start, _ = strconv.Atoi(bounds[0])
			end, _ = strconv.Atoi(bounds[1])
			if start < min || end > max || start > end {
				return nil, fmt.Errorf("区间越界：%s", part)
			}
		default:
			value, err := strconv.Atoi(base)
			if err != nil || value < min || value > max {
				return nil, fmt.Errorf("取值无效：%s", part)
			}
			start, end = value, value
		}
		s, e, st := start, end, step
		matchers = append(matchers, func(v int) bool {
			return v >= s && v <= e && (v-s)%st == 0
		})
	}
	return func(v int) bool {
		for _, matcher := range matchers {
			if matcher(v) {
				return true
			}
		}
		return false
	}, nil
}

// Matches 判断时间是否命中表达式（周字段 7 视为周日）。
func (c *Cron) Matches(t time.Time) bool {
	if c == nil {
		return false
	}
	dow := int(t.Weekday())
	if !c.dow(dow) && !(dow == 0 && c.dow(7)) {
		return false
	}
	return c.minute(t.Minute()) && c.hour(t.Hour()) && c.dom(t.Day()) && c.month(int(t.Month()))
}

// LastScheduled 返回不晚于 now 的最近一次计划时刻。
func LastScheduled(expr string, now time.Time) (time.Time, bool) {
	cron, err := ParseCron(expr)
	if err != nil {
		return time.Time{}, false
	}
	cursor := now.Truncate(time.Minute)
	for i := 0; i < 64*24*60; i++ {
		if cron.Matches(cursor) {
			return cursor, true
		}
		cursor = cursor.Add(-time.Minute)
	}
	return time.Time{}, false
}

// NextScheduled 返回晚于 after 的下一次计划时刻。
func NextScheduled(expr string, after time.Time) (time.Time, bool) {
	cron, err := ParseCron(expr)
	if err != nil {
		return time.Time{}, false
	}
	cursor := after.Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < 64*24*60; i++ {
		if cron.Matches(cursor) {
			return cursor, true
		}
		cursor = cursor.Add(time.Minute)
	}
	return time.Time{}, false
}

// CronFromHuman 把描述性计划（每天/每周/每小时）转换为 cron，无法识别返回空串。
func CronFromHuman(schedule string) string {
	text := strings.TrimSpace(schedule)
	if text == "" {
		return ""
	}
	if strings.HasPrefix(text, "每天") {
		if hour, minute, ok := parseClock(text); ok {
			return fmt.Sprintf("%d %d * * *", minute, hour)
		}
	}
	if strings.HasPrefix(text, "每小时") {
		return "0 * * * *"
	}
	weekdays := map[string]int{"日": 0, "天": 0, "一": 1, "二": 2, "三": 3, "四": 4, "五": 5, "六": 6}
	if strings.HasPrefix(text, "每周") && len(text) >= 3 {
		if dow, ok := weekdays[string([]rune(text)[2])]; ok {
			if hour, minute, ok := parseClock(text); ok {
				return fmt.Sprintf("%d %d * * %d", minute, hour, dow)
			}
		}
	}
	return ""
}

func parseClock(text string) (hour, minute int, ok bool) {
	idx := strings.Index(text, ":")
	if idx < 0 {
		return 0, 0, false
	}
	left := text[:idx]
	right := text[idx+1:]
	hourDigits := ""
	for i := len(left) - 1; i >= 0 && left[i] >= '0' && left[i] <= '9'; i-- {
		hourDigits = string(left[i]) + hourDigits
	}
	if len(right) < 2 || right[0] < '0' || right[0] > '9' || right[1] < '0' || right[1] > '9' {
		return 0, 0, false
	}
	hour, err := strconv.Atoi(hourDigits)
	if err != nil || hour > 23 {
		return 0, 0, false
	}
	minute, err = strconv.Atoi(right[:2])
	if err != nil || minute > 59 {
		return 0, 0, false
	}
	return hour, minute, true
}

// HumanScheduleFromCron 把常见 5 段 cron 转为中文计划描述，无法识别返回空串。
func HumanScheduleFromCron(expr string) string {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 || !IsValidCron(strings.Join(parts, " ")) {
		return ""
	}
	minute, okMinute := cronNumber(parts[0], 0, 59)
	if !okMinute {
		return ""
	}
	if cronStar(parts[1]) && allCronStar(parts[2:]) {
		if minute == 0 {
			return "每小时"
		}
		return fmt.Sprintf("每小时第 %d 分", minute)
	}
	hour, okHour := cronNumber(parts[1], 0, 23)
	if !okHour {
		return ""
	}
	clock := fmt.Sprintf("%02d:%02d", hour, minute)
	if allCronStar(parts[2:]) {
		return "每天 " + clock
	}
	if cronStar(parts[2]) && cronStar(parts[3]) {
		if dow, ok := cronNumber(parts[4], 0, 7); ok {
			return "每周" + cronWeekday(dow) + " " + clock
		}
	}
	if cronStar(parts[3]) && cronStar(parts[4]) {
		if dom, ok := cronNumber(parts[2], 1, 31); ok {
			return fmt.Sprintf("每月 %d 日 %s", dom, clock)
		}
	}
	return ""
}

func cronNumber(field string, min, max int) (int, bool) {
	value := strings.TrimSpace(field)
	if value == "" || strings.ContainsAny(value, "*/,-") {
		return 0, false
	}
	number, err := strconv.Atoi(value)
	if err != nil || number < min || number > max {
		return 0, false
	}
	return number, true
}

func cronStar(field string) bool {
	value := strings.TrimSpace(field)
	return value == "*" || value == "?"
}

func allCronStar(fields []string) bool {
	for _, field := range fields {
		if !cronStar(field) {
			return false
		}
	}
	return true
}

func cronWeekday(dow int) string {
	labels := []string{"日", "一", "二", "三", "四", "五", "六", "日"}
	if dow < 0 || dow >= len(labels) {
		return ""
	}
	return labels[dow]
}

func normalizeIncomingSchedule(schedule, cronExpr string) (string, bool) {
	text := strings.TrimSpace(schedule)
	if !disabledSchedule(text) {
		return text, false
	}
	if human := HumanScheduleFromCron(cronExpr); human != "" {
		return human, true
	}
	return "", true
}

func disabledSchedule(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "未启用", "已禁用", "disabled", "not scheduled", "not enabled", "off":
		return true
	}
	return false
}
