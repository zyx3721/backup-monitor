package veeam

import (
	"strconv"
	"strings"
	"time"
)

type table struct {
	header []string
	rows   [][]string
}

func parseJobs(output string) []Job {
	parsed := buildTable(output)
	jobs := make([]Job, 0, len(parsed.rows))
	headers := lowerAll(parsed.header)
	idIndex := columnIndex(headers, "job id", "id")
	nameIndex := columnIndex(headers, "job name", "name")
	typeIndex := columnIndex(headers, "type", "backup type")
	statusIndex := columnIndex(headers, "status", "state")
	scheduleIndex := columnIndex(headers, "schedule", "next run")

	for _, row := range parsed.rows {
		job := Job{}
		if idIndex >= 0 {
			job.ID = cell(row, idIndex)
		}
		if nameIndex >= 0 {
			job.Name = cell(row, nameIndex)
		}
		if typeIndex >= 0 {
			job.Type = cell(row, typeIndex)
		}
		if statusIndex >= 0 {
			job.Status = cell(row, statusIndex)
		}
		if scheduleIndex >= 0 {
			job.Schedule = cell(row, scheduleIndex)
		}
		if job.ID == "" && job.Name == "" && len(row) > 0 {
			job = inferJob(row)
		}
		if job.ID == "" && job.Name == "" {
			continue
		}
		if isHeaderRow(job.Name) || isHeaderRow(job.ID) {
			continue
		}
		if job.Name == "" {
			job.Name = job.ID
		}
		if job.ID == "" {
			job.ID = job.Name
		}
		jobs = append(jobs, job)
	}
	return dedupeJobs(jobs)
}

func inferJob(row []string) Job {
	first := cell(row, 0)
	second := cell(row, 1)
	if looksLikeID(first) {
		return Job{ID: first, Name: second}
	}
	if looksLikeID(second) {
		return Job{Name: first, ID: second}
	}
	return Job{Name: first, ID: first}
}

func parseSessions(output string) []Session {
	parsed := buildTable(output)
	sessions := make([]Session, 0, len(parsed.rows))
	headers := lowerAll(parsed.header)
	idIndex := columnIndex(headers, "session id", "id")
	jobIDIndex := columnIndex(headers, "job id")
	jobNameIndex := columnIndex(headers, "job name", "job")
	oibIndex := columnIndex(headers, "oib id", "oib")
	stateIndex := columnIndex(headers, "state", "status", "result")
	startIndex := columnIndex(headers, "start time", "start")
	endIndex := columnIndex(headers, "end time", "end")
	sizeIndex := columnIndex(headers, "size", "processed", "total size")
	attemptIndex := columnIndex(headers, "attempt", "attempts")
	lastAttemptIndex := columnIndex(headers, "last attempt")

	for _, row := range parsed.rows {
		session := Session{}
		if idIndex >= 0 {
			session.ID = cell(row, idIndex)
		}
		if jobIDIndex >= 0 {
			session.JobID = cell(row, jobIDIndex)
		}
		if jobNameIndex >= 0 {
			session.JobName = cell(row, jobNameIndex)
		}
		if oibIndex >= 0 {
			session.OIBID = cell(row, oibIndex)
		}
		if stateIndex >= 0 {
			session.State = NormalizeState(cell(row, stateIndex))
			session.Result = cell(row, stateIndex)
		}
		if startIndex >= 0 {
			if parsed, ok := parseTime(cell(row, startIndex)); ok {
				session.Start, session.HasStart = parsed, true
			}
		}
		if endIndex >= 0 {
			if parsed, ok := parseTime(cell(row, endIndex)); ok {
				session.End, session.HasEnd = parsed, true
			}
		}
		if sizeIndex >= 0 {
			session.ProcessedSize = cell(row, sizeIndex)
		}
		if attemptIndex >= 0 {
			applyAttempt(&session, cell(row, attemptIndex))
		}
		if lastAttemptIndex >= 0 {
			if value, ok := parseBoolValue(cell(row, lastAttemptIndex)); ok {
				session.IsLastAttempt, session.HasLastAttempt = value, true
			}
		}
		if session.ID == "" && session.JobName == "" {
			continue
		}
		if session.Creation.IsZero() {
			if session.HasStart {
				session.Creation = session.Start
			} else if session.HasEnd {
				session.Creation = session.End
			}
		}
		sessions = append(sessions, session)
	}
	return dedupeSessions(sessions)
}

func parseSessionInfo(output string) Session {
	session := Session{}
	values := map[string]string{}
	for _, line := range cleanLines(output) {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		values[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	for key, value := range values {
		switch {
		case key == "session id" || key == "id":
			session.ID = value
		case key == "job id":
			session.JobID = value
		case key == "job name" || key == "job":
			session.JobName = value
		case key == "oib id" || key == "oib":
			session.OIBID = value
		case key == "state" || key == "status" || key == "result":
			session.Result = value
			session.State = NormalizeState(value)
		case key == "start time" || key == "start":
			if parsed, ok := parseTime(value); ok {
				session.Start, session.HasStart = parsed, true
			}
		case key == "end time" || key == "end":
			if parsed, ok := parseTime(value); ok {
				session.End, session.HasEnd = parsed, true
			}
		case key == "total size" || key == "processed size" || key == "processed":
			session.ProcessedSize = value
		case key == "transferred size" || key == "transferred":
			session.TransferredSize = value
		case key == "attempt" || key == "attempts":
			applyAttempt(&session, value)
		case key == "max attempts" || key == "maxattempts":
			if number, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
				session.MaxAttempts, session.HasMaxAttempts = number, true
			}
		case key == "last attempt" || key == "is last attempt":
			if parsed, ok := parseBoolValue(value); ok {
				session.IsLastAttempt, session.HasLastAttempt = parsed, true
			}
		case key == "creation time" || key == "creation":
			if parsed, ok := parseTime(value); ok {
				session.Creation = parsed
			}
		}
	}
	if session.Creation.IsZero() {
		if session.HasStart {
			session.Creation = session.Start
		} else if session.HasEnd {
			session.Creation = session.End
		}
	}
	return session
}

func applyAttempt(session *Session, value string) {
	attempt, maxAttempts, hasAttempt, hasMax := parseAttemptValue(value)
	if hasAttempt {
		session.Attempt, session.HasAttempt = attempt, true
	}
	if hasMax {
		session.MaxAttempts, session.HasMaxAttempts = maxAttempts, true
	}
}

func parseAttemptValue(value string) (int, int, bool, bool) {
	text := strings.ToLower(strings.TrimSpace(value))
	text = strings.NewReplacer(
		"attempts", "",
		"attempt", "",
		"from", "/",
		"of", "/",
		"：", "/",
		":", "/",
		",", "/",
	).Replace(text)
	text = strings.TrimPrefix(strings.TrimSpace(text), "#")
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '/' || r == ' ' || r == '\t'
	})
	if len(parts) == 0 {
		return 0, 0, false, false
	}
	attempt, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false, false
	}
	if len(parts) < 2 {
		return attempt, 0, true, false
	}
	maxAttempts, err := strconv.Atoi(parts[1])
	if err != nil {
		return attempt, 0, true, false
	}
	return attempt, maxAttempts, true, true
}

func parseBoolValue(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes", "true", "1", "是":
		return true, true
	case "no", "false", "0", "否":
		return false, true
	default:
		return false, false
	}
}

func buildTable(output string) table {
	lines := cleanLines(output)
	if len(lines) == 0 {
		return table{}
	}

	headerIndex := -1
	for index, line := range lines {
		if headerScore(line) >= 2 {
			headerIndex = index
			break
		}
	}
	if headerIndex < 0 {
		return tableFromLines(lines)
	}

	headerLine := lines[headerIndex]
	parsed := table{}
	if strings.Contains(headerLine, "\t") {
		parsed.header = splitTabs(headerLine)
		for _, line := range lines[headerIndex+1:] {
			if isSeparator(line) {
				continue
			}
			parsed.rows = append(parsed.rows, splitTabs(line))
		}
		return parsed
	}

	spans := headerSpans(headerLine)
	if len(spans) >= 2 {
		parsed.header = make([]string, len(spans))
		for index, span := range spans {
			parsed.header[index] = strings.TrimSpace(headerLine[span.start:span.end])
		}
		for _, line := range lines[headerIndex+1:] {
			if isSeparator(line) {
				continue
			}
			row := splitColumns(line)
			if len(row) != len(spans) {
				row = make([]string, len(spans))
				for index, span := range spans {
					end := len(line)
					if index+1 < len(spans) {
						end = spans[index+1].start
					}
					if span.start < len(line) {
						if end > len(line) {
							end = len(line)
						}
						row[index] = strings.TrimSpace(line[span.start:end])
					}
				}
			}
			parsed.rows = append(parsed.rows, row)
		}
		return parsed
	}
	return tableFromLines(lines[headerIndex:])
}

func tableFromLines(lines []string) table {
	parsed := table{}
	for _, line := range lines {
		if isSeparator(line) {
			continue
		}
		row := splitColumns(line)
		if len(row) == 0 {
			continue
		}
		if parsed.header == nil && looksLikeHeader(row) {
			parsed.header = row
			continue
		}
		parsed.rows = append(parsed.rows, row)
	}
	return parsed
}

func headerSpans(line string) []span {
	matches := spacePattern.FindAllStringIndex(strings.TrimRight(line, " \t"), -1)
	spans := make([]span, 0, len(matches)+1)
	start := 0
	for _, match := range matches {
		if match[0] > start {
			spans = append(spans, span{start: start, end: match[0]})
		}
		start = match[1]
	}
	if start < len(strings.TrimRight(line, " \t")) {
		spans = append(spans, span{start: start, end: len(strings.TrimRight(line, " \t"))})
	}
	return spans
}

type span struct {
	start int
	end   int
}

func cleanLines(output string) []string {
	output = ansiPattern.ReplaceAllString(output, "")
	raw := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func splitTabs(line string) []string {
	parts := strings.Split(line, "\t")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		result = append(result, strings.TrimSpace(part))
	}
	return result
}

func splitColumns(line string) []string {
	parts := spacePattern.Split(strings.TrimSpace(line), -1)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func headerScore(line string) int {
	lower := strings.ToLower(line)
	score := 0
	for _, keyword := range []string{"job id", "job name", "session id", "state", "status", "start", "end", "size", "type"} {
		if strings.Contains(lower, keyword) {
			score++
		}
	}
	return score
}

func looksLikeHeader(row []string) bool {
	score := 0
	for _, value := range row {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "job id", "job name", "session id", "state", "status", "start time", "end time", "size", "type":
			score++
		}
	}
	return score >= 2
}

func isHeaderRow(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "job id", "job name", "name", "id", "session id", "state", "status":
		return true
	default:
		return false
	}
}

func isSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	for _, char := range trimmed {
		if char != '-' && char != '=' && char != '+' && char != '|' && char != ' ' {
			return false
		}
	}
	return true
}

func lowerAll(values []string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = strings.ToLower(strings.TrimSpace(value))
	}
	return result
}

func columnIndex(headers []string, candidates ...string) int {
	for _, candidate := range candidates {
		for index, header := range headers {
			if header == candidate {
				return index
			}
		}
	}
	for _, candidate := range candidates {
		for index, header := range headers {
			if strings.Contains(header, candidate) {
				return index
			}
		}
	}
	return -1
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func looksLikeID(value string) bool {
	value = strings.TrimSpace(value)
	if uuidPattern.MatchString(value) {
		return true
	}
	if len(value) < 8 || strings.Contains(value, " ") {
		return false
	}
	digits := 0
	for _, char := range value {
		if char >= '0' && char <= '9' {
			digits++
		}
	}
	return digits >= 4
}

// NormalizeState 把 Veeam 状态文本归一化为协议状态。
func NormalizeState(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch {
	case lower == "":
		return ""
	case strings.Contains(lower, "success"), strings.Contains(lower, "succeeded"),
		strings.Contains(lower, "completed"), strings.Contains(lower, "finished"),
		strings.Contains(lower, "warning"), strings.Contains(lower, "warn"):
		return "success"
	case strings.Contains(lower, "running"), strings.Contains(lower, "progress"),
		strings.Contains(lower, "working"), strings.Contains(lower, "starting"):
		return "running"
	case strings.Contains(lower, "fail"), strings.Contains(lower, "error"),
		strings.Contains(lower, "stopped"), strings.Contains(lower, "stopping"):
		return "failed"
	default:
		return ""
	}
}

func parseTime(value string) (time.Time, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"02.01.2006 15:04:05",
		"01/02/2006 15:04:05",
		"1/2/2006 3:04:05 PM",
		"1/2/2006 3:04 PM",
		"2006/01/02 3:04:05 PM",
		"2006/01/02 3:04 PM",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed, true
		}
	}
	if index := strings.LastIndex(text, " "); index > 0 {
		if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", text[:index], time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func dedupeJobs(jobs []Job) []Job {
	seen := make(map[string]struct{}, len(jobs))
	result := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		key := strings.ToLower(job.ID + "|" + job.Name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, job)
	}
	return result
}

func dedupeSessions(sessions []Session) []Session {
	seen := make(map[string]struct{}, len(sessions))
	result := make([]Session, 0, len(sessions))
	for _, session := range sessions {
		key := strings.ToLower(session.ID)
		if key == "" {
			key = strings.ToLower(session.JobName + "|" + session.Start.String())
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, session)
	}
	return result
}
