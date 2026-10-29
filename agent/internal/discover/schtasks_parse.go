package discover

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"backup-monitor-agent/internal/cron"
)

type windowsTask struct {
	TaskName           string `json:"TaskName"`
	TaskPath           string `json:"TaskPath"`
	State              string `json:"State"`
	LastRunTime        string `json:"LastRunTime"`
	LastTaskResult     int64  `json:"LastTaskResult"`
	Execute            string `json:"Execute"`
	Arguments          string `json:"Arguments"`
	Comment            string `json:"Comment"`
	Schedule           string `json:"Schedule"`
	TriggerType        string `json:"TriggerType"`
	DaysOfWeek         string `json:"DaysOfWeek"`
	DaysOfMonth        string `json:"DaysOfMonth"`
	RepetitionInterval string `json:"RepetitionInterval"`
	Enabled            *bool  `json:"Enabled"`
}

// ParseScheduledTasksJSON 解析 PowerShell Get-ScheduledTask 的 JSON 输出。
func ParseScheduledTasksJSON(data []byte, filter Filter) ([]Task, error) {
	text := strings.TrimSpace(strings.TrimPrefix(string(data), "\ufeff"))
	if text == "" || text == "null" {
		return nil, nil
	}
	var records []windowsTask
	if strings.HasPrefix(text, "[") {
		if err := json.Unmarshal([]byte(text), &records); err != nil {
			return nil, fmt.Errorf("decode scheduled task JSON: %w", err)
		}
	} else {
		var record windowsTask
		if err := json.Unmarshal([]byte(text), &record); err != nil {
			return nil, fmt.Errorf("decode scheduled task JSON: %w", err)
		}
		records = []windowsTask{record}
	}

	tasks := make([]Task, 0, len(records))
	for _, record := range records {
		task, ok := taskFromWindows(record, filter)
		if ok {
			tasks = append(tasks, task)
		}
	}
	return dedupeTasks(tasks), nil
}

// ParseSchtasksCSV 解析 schtasks /query /fo CSV /v 的输出。
func ParseSchtasksCSV(data []byte, filter Filter) ([]Task, error) {
	text := decodeCSVText(data)
	reader := csv.NewReader(strings.NewReader(text))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("decode schtasks CSV: %w", err)
	}

	headerIndex := -1
	columns := map[string]int{}
	for index, row := range records {
		for _, value := range row {
			normalized := strings.ToLower(strings.TrimSpace(value))
			if normalized == "taskname" || normalized == "任务名" {
				headerIndex = index
				break
			}
		}
		if headerIndex >= 0 {
			for column, name := range records[headerIndex] {
				columns[strings.ToLower(strings.TrimSpace(name))] = column
			}
			break
		}
	}

	column := func(row []string, names []string, fallback int) string {
		for _, name := range names {
			if index, ok := columns[strings.ToLower(name)]; ok && index < len(row) {
				return strings.TrimSpace(row[index])
			}
		}
		if fallback >= 0 && fallback < len(row) {
			return strings.TrimSpace(row[fallback])
		}
		return ""
	}

	start := headerIndex + 1
	if headerIndex < 0 {
		start = 0
	}
	tasks := make([]Task, 0, len(records))
	for _, row := range records[start:] {
		if len(row) < 16 {
			continue
		}
		fullName := column(row, []string{"TaskName", "任务名"}, 0)
		if fullName == "" || isTaskNameHeader(fullName) {
			continue
		}
		path, name := splitFullTaskName(fullName)
		lastRun, _ := parseWindowsTime(column(row, []string{"Last Run Time"}, 4))
		lastResult, _ := parseWindowsResult(column(row, []string{"Last Result"}, 5))
		command := column(row, []string{"Task To Run"}, 7)
		if command == "" {
			continue
		}
		execute, arguments := splitCommandLine(command)
		state := column(row, []string{"Scheduled Task State"}, 10)
		enabled := !strings.Contains(strings.ToLower(state), "disabled") && !strings.Contains(state, "禁用")
		scheduleTypeText := column(row, []string{"Schedule Type"}, 14)
		scheduleType := strings.ToLower(scheduleTypeText)
		days := column(row, []string{"Days"}, 18)
		daysOfWeek, daysOfMonth := "", ""
		switch {
		case strings.Contains(scheduleType, "weekly"):
			daysOfWeek = days
		case strings.Contains(scheduleType, "monthly"):
			daysOfMonth = days
		}

		record := windowsTask{
			TaskName:       name,
			TaskPath:       path,
			State:          state,
			LastRunTime:    formatWindowsTime(lastRun),
			LastTaskResult: int64(lastResult),
			Execute:        execute,
			Arguments:      arguments,
			Comment:        column(row, []string{"Comment"}, 9),
			TriggerType:    triggerFromScheduleType(scheduleTypeText),
			Schedule:       buildCsvSchedule(column(row, []string{"Start Date"}, 16), column(row, []string{"Start Time"}, 15)),
			DaysOfWeek:     daysOfWeek,
			DaysOfMonth:    daysOfMonth,
			Enabled:        &enabled,
		}
		task, ok := taskFromWindows(record, filter)
		if ok {
			tasks = append(tasks, task)
		}
	}
	return dedupeTasks(tasks), nil
}

func taskFromWindows(record windowsTask, filter Filter) (Task, bool) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(record.TaskPath)), `\microsoft\`) {
		return Task{}, false
	}
	command := strings.TrimSpace(record.Execute)
	if command == "" {
		return Task{}, false
	}
	if strings.TrimSpace(record.Arguments) != "" {
		command += " " + strings.TrimSpace(record.Arguments)
	}

	label, hasLabel := labelFromText(record.Comment, labelPrefix(filter))
	if !hasLabel {
		if IsExcluded(command, filter) || !IsBackupTask(command, filter) {
			return Task{}, false
		}
	}
	if record.Enabled != nil && !*record.Enabled {
		return Task{}, false
	}
	if strings.Contains(strings.ToLower(record.State), "disabled") || strings.Contains(record.State, "禁用") {
		return Task{}, false
	}

	lastRun, hasLastRun := parseWindowsTime(record.LastRunTime)
	scheduleCron, schedule := scheduleFromWindows(record)
	script := label.Script
	if script == "" {
		script = ScriptPath(command)
	}
	key := label.Key
	if key == "" {
		key = scheduledTaskKey(record.TaskPath, record.TaskName)
	}
	name := ResolveName(command, label.Name, key)

	config := configForTask("schtasks", "", command, script, "", label)
	config["taskPath"] = record.TaskPath
	config["taskName"] = record.TaskName
	if state := strings.TrimSpace(record.State); state != "" {
		config["state"] = state
	}
	if record.LastTaskResult != 0 {
		config["lastResult"] = record.LastTaskResult
	}

	return Task{
		Name:          name,
		Key:           key,
		Type:          "shell",
		Schedule:      schedule,
		ScheduleCron:  scheduleCron,
		Config:        config,
		Source:        "schtasks",
		Command:       command,
		Script:        script,
		ProcKeyword:   label.Proc,
		LastRun:       lastRun,
		HasLastRun:    hasLastRun,
		LastResult:    int(record.LastTaskResult),
		HasLastResult: hasLastRun,
		State:         record.State,
	}, true
}

func scheduleFromWindows(record windowsTask) (string, string) {
	value := strings.TrimSpace(record.Schedule)
	if value != "" && cron.Validate(value) {
		normalized := normalizeSchedule(value)
		return normalized, cron.ToHuman(normalized)
	}
	if strings.Contains(value, "每天") || strings.Contains(value, "每周") || strings.Contains(value, "每小时") {
		return "", value
	}

	start, hasStart := parseWindowsTime(value)
	trigger := strings.ToLower(record.TriggerType)
	switch {
	case strings.Contains(trigger, "boot"):
		return "", "开机时"
	case strings.Contains(trigger, "logon"):
		return "", "登录时"
	}
	if !hasStart {
		if strings.Contains(trigger, "boot") {
			return "", "开机时"
		}
		if strings.Contains(trigger, "logon") {
			return "", "登录时"
		}
		return "", ""
	}

	clock := fmt.Sprintf("%02d:%02d", start.Hour(), start.Minute())
	switch {
	case strings.Contains(trigger, "weekly"):
		days := parseWeekdaySet(record.DaysOfWeek, int(start.Weekday()))
		expression := fmt.Sprintf("%d %d * * %s", start.Minute(), start.Hour(), joinInts(days))
		return expression, "每" + weekdayNames(days) + " " + clock
	case strings.Contains(trigger, "monthly"):
		days := parseDaySet(record.DaysOfMonth, start.Day())
		expression := fmt.Sprintf("%d %d %s * *", start.Minute(), start.Hour(), joinInts(days))
		return expression, "每月 " + joinInts(days) + " 日 " + clock
	case strings.Contains(trigger, "daily"):
		expression := fmt.Sprintf("%d %d * * *", start.Minute(), start.Hour())
		return expression, "每天 " + clock
	case strings.Contains(trigger, "time"), strings.Contains(trigger, "once"):
		if expression, human, ok := repetitionSchedule(record); ok {
			return expression, human
		}
		return "", "一次性计划 " + clock
	default:
		if expression, human, ok := repetitionSchedule(record); ok {
			return expression, human
		}
		expression := fmt.Sprintf("%d %d * * *", start.Minute(), start.Hour())
		return expression, "每天 " + clock
	}
}

func repetitionSchedule(record windowsTask) (string, string, bool) {
	minutes, ok := parseISODurationMinutes(record.RepetitionInterval)
	if !ok {
		return "", "", false
	}
	if minutes < 60 && 60%minutes == 0 {
		return fmt.Sprintf("*/%d * * * *", minutes), fmt.Sprintf("每 %d 分钟", minutes), true
	}
	hours := minutes / 60
	if minutes%60 == 0 && hours > 0 && 24%hours == 0 {
		return fmt.Sprintf("0 */%d * * *", hours), fmt.Sprintf("每 %d 小时", hours), true
	}
	return "", "", false
}

func parseWeekdaySet(value string, fallback int) []int {
	days := parseWeekdayTokens(value)
	if len(days) == 0 {
		days = []int{fallback}
	}
	return uniqueInts(days)
}

func parseWeekdayTokens(value string) []int {
	text := strings.TrimSpace(value)
	if text == "" {
		return nil
	}
	if parsed, err := strconv.Atoi(text); err == nil {
		return weekdayBits(parsed)
	}
	names := map[string]int{
		"sunday": 0, "sun": 0, "周日": 0, "星期日": 0, "日": 0,
		"monday": 1, "mon": 1, "星期一": 1, "周一": 1, "一": 1,
		"tuesday": 2, "tue": 2, "星期二": 2, "周二": 2, "二": 2,
		"wednesday": 3, "wed": 3, "星期三": 3, "周三": 3, "三": 3,
		"thursday": 4, "thu": 4, "星期四": 4, "周四": 4, "四": 4,
		"friday": 5, "fri": 5, "星期五": 5, "周五": 5, "五": 5,
		"saturday": 6, "sat": 6, "星期六": 6, "周六": 6, "六": 6,
	}
	result := make([]int, 0, 7)
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == '|' || r == ';' || r == ' '
	}) {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			continue
		}
		if day, ok := names[token]; ok {
			result = append(result, day)
			continue
		}
		if number, err := strconv.Atoi(token); err == nil {
			result = append(result, weekdayBits(number)...)
		}
	}
	return result
}

func weekdayBits(mask int) []int {
	result := make([]int, 0, 7)
	for day := 0; day <= 6; day++ {
		if mask&(1<<day) != 0 {
			result = append(result, day)
		}
	}
	if len(result) == 0 && mask >= 0 && mask <= 6 {
		result = append(result, mask)
	}
	return result
}

func weekdayNames(days []int) string {
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	values := uniqueInts(days)
	if len(values) == 0 {
		return ""
	}
	if len(values) >= 3 && consecutiveWeekdays(values) {
		return names[values[0]] + "至" + names[values[len(values)-1]]
	}
	labels := make([]string, 0, len(values))
	for _, day := range values {
		if day >= 0 && day <= 6 {
			labels = append(labels, names[day])
		}
	}
	return strings.Join(labels, "、")
}

func consecutiveWeekdays(days []int) bool {
	for index := 1; index < len(days); index++ {
		if days[index] != days[index-1]+1 {
			return false
		}
	}
	return true
}

func parseDaySet(value string, fallback int) []int {
	days := make([]int, 0, 31)
	for _, token := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '|' || r == ';' || r == ' '
	}) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		number, err := strconv.Atoi(token)
		if err == nil && number >= 1 && number <= 31 {
			days = append(days, number)
		}
	}
	if len(days) == 0 {
		days = append(days, fallback)
	}
	return uniqueInts(days)
}

func parseISODurationMinutes(value string) (int, bool) {
	text := strings.ToUpper(strings.TrimSpace(value))
	if text == "" || !strings.HasPrefix(text, "P") {
		return 0, false
	}
	text = strings.TrimPrefix(text, "P")
	days := 0
	hours := 0
	minutes := 0
	seconds := 0
	inTime := false
	number := ""
	for _, char := range text {
		switch {
		case char == 'T':
			inTime = true
		case char >= '0' && char <= '9':
			number += string(char)
		case char == 'D':
			days, _ = strconv.Atoi(number)
			number = ""
		case char == 'H' && inTime:
			hours, _ = strconv.Atoi(number)
			number = ""
		case char == 'M' && inTime:
			minutes, _ = strconv.Atoi(number)
			number = ""
		case char == 'S' && inTime:
			seconds, _ = strconv.Atoi(number)
			number = ""
		}
	}
	total := days*24*60 + hours*60 + minutes
	if seconds >= 60 {
		total += seconds / 60
	}
	return total, total > 0
}

func triggerFromScheduleType(value string) string {
	text := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(text, "daily"), strings.Contains(text, "每天"):
		return "Daily"
	case strings.Contains(text, "weekly"), strings.Contains(text, "每周"):
		return "Weekly"
	case strings.Contains(text, "monthly"), strings.Contains(text, "每月"):
		return "Monthly"
	case strings.Contains(text, "startup"), strings.Contains(text, "启动"):
		return "Boot"
	case strings.Contains(text, "logon"), strings.Contains(text, "登录"):
		return "Logon"
	case strings.Contains(text, "once"), strings.Contains(text, "一次"):
		return "Time"
	default:
		return ""
	}
}

func parseWindowsTime(value string) (time.Time, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Time{}, false
	}
	lower := strings.ToLower(text)
	if lower == "n/a" || strings.Contains(text, "从不") || strings.Contains(text, "从未") || strings.Contains(text, "尚未") {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006/1/2 15:04:05",
		"2006/1/2 15:04",
		"2006/01/02 15:04",
		"01/02/2006 15:04:05",
		"01/02/2006 15:04:05",
		"1/2/2006 3:04:05 PM",
		"1/2/2006 3:04 PM",
		"2006/01/02 3:04:05 PM",
		"2006/01/02 3:04 PM",
		"15:04:05",
		"15:04",
		"3:04:05 PM",
		"3:04 PM",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func formatWindowsTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func buildCsvSchedule(date, clock string) string {
	date = strings.TrimSpace(date)
	clock = strings.TrimSpace(clock)
	if date == "" || strings.EqualFold(date, "N/A") || strings.Contains(date, "从不") {
		return clock
	}
	return date + " " + clock
}

func parseWindowsResult(value string) (int, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(text, 0, 64)
	if err != nil {
		parsed, err = strconv.ParseInt(text, 10, 64)
	}
	if err != nil {
		return 0, err
	}
	return int(parsed), nil
}

func labelFromText(text, prefix string) (labelInfo, bool) {
	lower := strings.ToLower(text)
	normalized := strings.ToLower(strings.TrimSpace(prefix))
	if normalized == "" {
		normalized = strings.ToLower(defaultLabelPrefix)
	}
	index := strings.Index(lower, normalized)
	if index < 0 {
		return labelInfo{}, false
	}
	value := strings.TrimSpace(text[index+len(normalized):])
	return parseLabel(value), true
}

func splitFullTaskName(full string) (string, string) {
	index := strings.LastIndex(full, `\`)
	if index < 0 {
		return `\`, full
	}
	return full[:index+1], full[index+1:]
}

func splitCommandLine(command string) (string, string) {
	tokens := commandTokens(command)
	if len(tokens) == 0 {
		return "", ""
	}
	execute := tokens[0]
	if len(tokens) == 1 {
		return execute, ""
	}
	return execute, strings.Join(tokens[1:], " ")
}

func joinInts(values []int) string {
	values = uniqueInts(values)
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

func uniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func isTaskNameHeader(value string) bool {
	text := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))
	switch text {
	case "taskname", "task name", "任务名", "任务名称":
		return true
	default:
		return false
	}
}

func decodeCSVText(data []byte) string {
	if len(data) >= 2 && data[0] == 0xff && data[1] == 0xfe {
		units := make([]uint16, 0, (len(data)-2)/2)
		for index := 2; index+1 < len(data); index += 2 {
			units = append(units, uint16(data[index])|uint16(data[index+1])<<8)
		}
		return string(utf16.Decode(units))
	}
	return strings.TrimPrefix(string(data), "\ufeff")
}
