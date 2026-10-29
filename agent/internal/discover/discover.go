// Package discover 发现并解析 Linux crontab 与 Windows 计划任务。
package discover

import (
	"path/filepath"
	"strings"
	"time"

	"backup-monitor-agent/internal/cron"
)

const defaultLabelPrefix = "# backup-monitor:"

// Filter 是任务发现的关键字与标签过滤条件。
type Filter struct {
	LabelPrefix     string
	IncludeKeywords []string
	IncludePaths    []string
	ExcludeKeywords []string
}

// Task 是发现到的备份任务及其最近运行信息。
type Task struct {
	Name          string
	Key           string
	Type          string
	Tool          string
	Schedule      string
	ScheduleCron  string
	Config        map[string]any
	Source        string
	User          string
	Command       string
	Script        string
	ProcKeyword   string
	LastRun       time.Time
	HasLastRun    bool
	LastResult    int
	HasLastResult bool
	State         string
	Disabled      bool
}

type labelInfo struct {
	Name       string
	Key        string
	Script     string
	StatusFile string
	Artifact   string
	Match      string
	Proc       string
}

// IsExcluded 判断命令是否命中排除关键字。
func IsExcluded(command string, filter Filter) bool {
	lower := strings.ToLower(command)
	for _, keyword := range filter.ExcludeKeywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// IsBackupTask 判断命令是否命中备份任务关键字或脚本路径。
func IsBackupTask(command string, filter Filter) bool {
	lower := strings.ToLower(command)
	for _, keyword := range filter.IncludeKeywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(lower, keyword) {
			return true
		}
	}
	for _, path := range filter.IncludePaths {
		path = strings.ToLower(strings.TrimSpace(path))
		if path != "" && strings.Contains(lower, path) {
			return true
		}
	}
	return false
}

// ResolveName 按标签名、稳定 key、脚本名、命令首段的顺序生成任务名。
func ResolveName(command, labelName, key string) string {
	if value := strings.TrimSpace(labelName); value != "" {
		return truncateName(value)
	}
	if value := strings.TrimSpace(key); value != "" &&
		!strings.HasPrefix(value, "cron:") &&
		!strings.HasPrefix(value, "schtasks:") &&
		!strings.HasPrefix(value, "veeam:") {
		return truncateName(value)
	}
	script := ScriptPath(command)
	if script != "" {
		base := filepath.Base(script)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		if base != "" && base != "." && base != "/" {
			return truncateName(base)
		}
	}
	tokens := commandTokens(command)
	for _, token := range tokens {
		if token == "" || strings.HasPrefix(token, "-") || strings.Contains(token, "=") {
			continue
		}
		base := filepath.Base(token)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		if base != "" && base != "." && base != "/" {
			return truncateName(base)
		}
	}
	return "backup-task"
}

// ScriptPath 从命令中提取脚本或可执行文件路径。
func ScriptPath(command string) string {
	tokens := commandTokens(command)
	first := ""
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if token == "" || isControlToken(token) {
			continue
		}
		if isEnvAssignment(token) {
			continue
		}
		if isInterpreter(token) {
			base := strings.ToLower(filepath.Base(token))
			for next := index + 1; next < len(tokens); next++ {
				candidate := tokens[next]
				if strings.HasPrefix(candidate, "-") || isEnvAssignment(candidate) {
					continue
				}
				if (base == "cmd" || base == "cmd.exe") && strings.HasPrefix(candidate, "/") {
					continue
				}
				return candidate
			}
			return token
		}
		if hasScriptExtension(token) {
			return token
		}
		if first == "" {
			first = token
		}
	}
	return first
}

func taskKey(user, expr string) string {
	normalized := normalizeSchedule(expr)
	if strings.TrimSpace(user) == "" {
		return "cron::" + normalized
	}
	return "cron:" + strings.TrimSpace(user) + ":" + normalized
}

func scheduledTaskKey(path, name string) string {
	full := strings.TrimSpace(path) + strings.TrimSpace(name)
	full = strings.ReplaceAll(full, "/", `\`)
	full = strings.TrimPrefix(full, `\`)
	return "schtasks:" + strings.ToLower(full)
}

func normalizeSchedule(expr string) string {
	if expanded, ok := cron.Expand(expr); ok {
		expr = expanded
	}
	return strings.Join(strings.Fields(expr), " ")
}

func parseLabel(value string) labelInfo {
	text := strings.TrimSpace(value)
	text = strings.Trim(text, `"'`)
	if text == "" {
		return labelInfo{}
	}
	if !strings.Contains(text, "=") {
		return labelInfo{Name: text, Key: text}
	}

	parts := splitLabelParts(text)
	info := labelInfo{}
	recognized := false
	for _, part := range parts {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		switch key {
		case "name", "task", "title":
			info.Name = val
			recognized = true
		case "key", "id":
			info.Key = val
			recognized = true
		case "script", "command":
			info.Script = val
			recognized = true
		case "status", "statusfile", "status_file":
			info.StatusFile = val
			recognized = true
		case "artifact", "artifacts", "path", "dir", "artifactdir", "artifact_dir", "backupdir", "backup_dir", "backup":
			info.Artifact = val
			recognized = true
		case "match", "pattern", "glob":
			info.Match = val
			recognized = true
		case "proc", "process", "processkeyword":
			info.Proc = val
			recognized = true
		}
	}
	if !recognized {
		return labelInfo{Name: text, Key: text}
	}
	if info.Name == "" {
		info.Name = info.Key
	}
	return info
}

func splitLabelParts(text string) []string {
	separators := ";"
	if !strings.Contains(text, ";") && strings.Count(text, "=") > 1 {
		separators = ";,"
	}
	runes := []rune(text)
	parts := make([]string, 0, 4)
	var current strings.Builder
	var spaces strings.Builder
	var quote rune
	var last rune
	flush := func() {
		if part := strings.TrimSpace(current.String()); part != "" {
			parts = append(parts, part)
		}
		current.Reset()
	}
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		if quote != 0 {
			current.WriteRune(char)
			if char == quote {
				quote = 0
				last = char
			}
			continue
		}
		if (char == '"' || char == '\'') && (last == '=' || last == 0) {
			quote = char
			current.WriteString(spaces.String())
			spaces.Reset()
			current.WriteRune(char)
			last = char
			continue
		}
		if strings.ContainsRune(separators, char) {
			flush()
			spaces.Reset()
			last = 0
			continue
		}
		if char == ' ' || char == '\t' {
			spaces.WriteRune(char)
			if startsLabelKey(runes[index+1:]) {
				flush()
				spaces.Reset()
			}
			continue
		}
		current.WriteString(spaces.String())
		spaces.Reset()
		current.WriteRune(char)
		last = char
	}
	flush()
	return parts
}

// startsLabelKey 判断剩余文本是否以 key= 开头，用于拆分空格分隔的标签键值对
func startsLabelKey(rest []rune) bool {
	index := 0
	for index < len(rest) && (rest[index] == ' ' || rest[index] == '\t') {
		index++
	}
	if index >= len(rest) || rest[index] >= '0' && rest[index] <= '9' {
		return false
	}
	start := index
	for index < len(rest) && isLabelKeyRune(rest[index]) {
		index++
	}
	if index == start {
		return false
	}
	for index < len(rest) && (rest[index] == ' ' || rest[index] == '\t') {
		index++
	}
	return index < len(rest) && rest[index] == '='
}

func isLabelKeyRune(char rune) bool {
	switch {
	case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z':
		return true
	case char >= '0' && char <= '9':
		return true
	}
	return char == '_' || char == '-'
}

func mergeLabel(base, override labelInfo) labelInfo {
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.Key != "" {
		base.Key = override.Key
	}
	if override.Script != "" {
		base.Script = override.Script
	}
	if override.StatusFile != "" {
		base.StatusFile = override.StatusFile
	}
	if override.Artifact != "" {
		base.Artifact = override.Artifact
	}
	if override.Proc != "" {
		base.Proc = override.Proc
	}
	return base
}

func labelFromComment(line, prefix string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	content := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = defaultLabelPrefix
	}
	normalizedPrefix := strings.TrimSpace(strings.TrimPrefix(prefix, "#"))
	if !strings.HasPrefix(strings.ToLower(content), strings.ToLower(normalizedPrefix)) {
		return "", false
	}
	return strings.TrimSpace(content[len(normalizedPrefix):]), true
}

func labelPrefix(filter Filter) string {
	if strings.TrimSpace(filter.LabelPrefix) == "" {
		return defaultLabelPrefix
	}
	return filter.LabelPrefix
}

func configForTask(source, user, command, script, file string, label labelInfo) map[string]any {
	config := map[string]any{
		"source":  source,
		"command": command,
	}
	if user != "" {
		config["user"] = user
	}
	if script != "" {
		config["script"] = script
	}
	if file != "" && strings.ContainsAny(file, `/\`) {
		config["file"] = file
	}
	if label.StatusFile != "" {
		config["statusFile"] = label.StatusFile
	}
	if label.Artifact != "" {
		config["artifactPath"] = label.Artifact
		config["backupDir"] = label.Artifact
	}
	if label.Match != "" {
		config["match"] = label.Match
	}
	return config
}

func commandTokens(command string) []string {
	var tokens []string
	var current strings.Builder
	var quote rune
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for _, char := range command {
		switch {
		case quote != 0:
			if char == quote {
				quote = 0
			} else {
				current.WriteRune(char)
			}
		case char == '\'' || char == '"':
			quote = char
		case char == ' ' || char == '\t' || char == '\r' || char == '\n':
			flush()
		case char == '&' || char == ';' || char == '|':
			flush()
		default:
			current.WriteRune(char)
		}
	}
	flush()
	return tokens
}

func isControlToken(token string) bool {
	switch token {
	case "cd", "&&", "||", ";", "|", "export", "set", "source", ".":
		return true
	default:
		return false
	}
}

func isEnvAssignment(token string) bool {
	index := strings.Index(token, "=")
	if index <= 0 {
		return false
	}
	name := token[:index]
	for position, char := range name {
		if position == 0 {
			if !(char == '_' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')) {
				return false
			}
			continue
		}
		if !(char == '_' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}

func isInterpreter(token string) bool {
	base := strings.ToLower(filepath.Base(token))
	switch base {
	case "sh", "bash", "dash", "zsh", "ksh", "csh", "tcsh", "python", "python2", "python3",
		"perl", "ruby", "php", "node", "pwsh", "powershell", "powershell.exe", "cmd", "cmd.exe", "wscript", "cscript":
		return true
	default:
		return false
	}
}

func hasScriptExtension(token string) bool {
	extension := strings.ToLower(filepath.Ext(token))
	switch extension {
	case ".sh", ".bash", ".py", ".pl", ".rb", ".php", ".ps1", ".bat", ".cmd", ".jar", ".exe", ".vbs", ".js":
		return true
	default:
		return false
	}
}

func truncateName(name string) string {
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) <= 120 {
		return name
	}
	return string(runes[:120])
}

func dedupeTasks(tasks []Task) []Task {
	seen := make(map[string]struct{}, len(tasks))
	result := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		identity := task.Key
		if identity == "" {
			identity = task.Name
		}
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		result = append(result, task)
	}
	return result
}
