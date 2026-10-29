package discover

import (
	"strings"

	"backup-monitor-agent/internal/cron"
)

type cronEntry struct {
	index        int
	raw          string
	scheduleRaw  string
	scheduleCron string
	human        string
	command      string
	user         string
	hasUser      bool
	label        labelInfo
	hasLabel     bool
}

func (e cronEntry) key() string {
	if e.label.Key != "" {
		return e.label.Key
	}
	expr := e.scheduleCron
	if expr == "" {
		expr = e.scheduleRaw
	}
	return taskKey(e.user, expr)
}

// ParseCrontab 解析 crontab 文本并按标签与关键字过滤。
func ParseCrontab(text, defaultUser, source string, hasUser bool, filter Filter) []Task {
	entries := parseCrontabEntries(text, defaultUser, hasUser, labelPrefix(filter))
	tasks := make([]Task, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, entry := range entries {
		key := entry.key()
		if _, exists := seen[key]; exists {
			continue
		}
		if !entry.hasLabel {
			if IsExcluded(entry.command, filter) {
				continue
			}
			if !IsBackupTask(entry.command, filter) {
				continue
			}
		}
		script := entry.label.Script
		if script == "" {
			script = ScriptPath(entry.command)
		}
		name := ResolveName(entry.command, entry.label.Name, key)
		config := configForTask("crontab", entry.user, entry.command, script, source, entry.label)
		tasks = append(tasks, Task{
			Name:         name,
			Key:          key,
			Type:         "shell",
			Schedule:     entry.human,
			ScheduleCron: entry.scheduleCron,
			Config:       config,
			Source:       source,
			User:         entry.user,
			Command:      entry.command,
			Script:       script,
			ProcKeyword:  entry.label.Proc,
		})
		seen[key] = struct{}{}
	}
	return tasks
}

func parseCrontabEntries(text, defaultUser string, hasUser bool, prefix string) []cronEntry {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	entries := make([]cronEntry, 0)
	pendingLabel := labelInfo{}
	hasPending := false

	for index, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if value, ok := labelFromComment(line, prefix); ok {
				pendingLabel = parseLabel(value)
				hasPending = true
			}
			continue
		}

		body, inlineValue, hasInline := stripInlineLabel(raw, prefix)
		label := labelInfo{}
		hasLabel := false
		if hasPending {
			label = pendingLabel
			hasLabel = true
		}
		if hasInline {
			label = mergeLabel(label, parseLabel(inlineValue))
			hasLabel = true
		}

		fields := strings.Fields(body)
		if len(fields) == 0 || isEnvAssignment(fields[0]) {
			continue
		}

		entry := cronEntry{
			index:    index + 1,
			raw:      raw,
			user:     defaultUser,
			hasUser:  hasUser,
			label:    label,
			hasLabel: hasLabel,
		}

		if strings.HasPrefix(fields[0], "@") {
			scheduleRaw := strings.ToLower(fields[0])
			if scheduleRaw == "@reboot" {
				entry.scheduleRaw = "@reboot"
				entry.human = "开机时"
			} else {
				expanded, ok := cron.Expand(scheduleRaw)
				if !ok {
					continue
				}
				normalized := normalizeSchedule(expanded)
				if !cron.Validate(normalized) {
					continue
				}
				entry.scheduleRaw = scheduleRaw
				entry.scheduleCron = normalized
				entry.human = cron.ToHuman(normalized)
			}
			commandStart := 1
			if hasUser {
				if len(fields) < 3 {
					continue
				}
				entry.user = fields[1]
				commandStart = 2
			}
			if len(fields) <= commandStart {
				continue
			}
			entry.command = strings.Join(fields[commandStart:], " ")
		} else {
			required := 6
			if hasUser {
				required = 7
			}
			if len(fields) < required {
				continue
			}
			expression := strings.Join(fields[:5], " ")
			if !cron.Validate(expression) {
				continue
			}
			entry.scheduleRaw = expression
			entry.scheduleCron = expression
			entry.human = cron.ToHuman(expression)
			commandStart := 5
			if hasUser {
				entry.user = fields[5]
				commandStart = 6
			}
			entry.command = strings.Join(fields[commandStart:], " ")
		}
		if strings.TrimSpace(entry.command) == "" {
			continue
		}
		entries = append(entries, entry)
		pendingLabel = labelInfo{}
		hasPending = false
	}
	return entries
}

func stripInlineLabel(line, prefix string) (string, string, bool) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = defaultLabelPrefix
	}
	lowerLine := strings.ToLower(line)
	lowerPrefix := strings.ToLower(prefix)
	index := strings.Index(lowerLine, lowerPrefix)
	if index <= 0 {
		return line, "", false
	}
	if index > 0 {
		previous := line[index-1]
		if previous != ' ' && previous != '\t' {
			return line, "", false
		}
	}
	body := strings.TrimRight(line[:index], " \t")
	value := strings.TrimSpace(line[index+len(prefix):])
	return body, value, true
}

type span struct {
	start int
	end   int
}

func fieldSpans(line string) []span {
	spans := make([]span, 0, 6)
	index := 0
	for index < len(line) {
		for index < len(line) && (line[index] == ' ' || line[index] == '\t') {
			index++
		}
		if index >= len(line) {
			break
		}
		start := index
		for index < len(line) && line[index] != ' ' && line[index] != '\t' {
			index++
		}
		spans = append(spans, span{start: start, end: index})
	}
	return spans
}

func rewriteCronLine(raw, newCron, prefix string) (string, bool) {
	body, inlineValue, hasInline := stripInlineLabel(raw, prefix)
	indent := body[:len(body)-len(strings.TrimLeft(body, " \t"))]
	trimmed := strings.TrimSpace(body)
	spans := fieldSpans(trimmed)
	if len(spans) < 1 {
		return "", false
	}

	var result string
	if strings.HasPrefix(trimmed[spans[0].start:spans[0].end], "@") {
		rest := strings.TrimSpace(trimmed[spans[0].end:])
		result = indent + newCron
		if rest != "" {
			result += " " + rest
		}
	} else {
		if len(spans) < 5 {
			return "", false
		}
		rest := trimmed[spans[4].end:]
		result = indent + newCron + rest
	}
	if hasInline {
		result = strings.TrimRight(result, " \t") + " " + strings.TrimSpace(prefix) + " " + inlineValue
	}
	return result, true
}
