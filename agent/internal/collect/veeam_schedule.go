package collect

import (
	"sort"
	"strconv"
	"time"

	"backup-monitor-agent/internal/cron"
	"backup-monitor-agent/internal/veeam"
)

const veeamScheduleToleranceMinutes = 5

// deriveVeeamSchedule 依据连续多日的会话开始时间推断每日执行计划，供未暴露计划的 Veeam 作业兜底
func deriveVeeamSchedule(sessions []veeam.Session) (string, string) {
	minutes := make([]int, 0, len(sessions))
	days := make(map[string]time.Time, len(sessions))
	for _, session := range sessions {
		start := session.Start
		if start.IsZero() {
			start = session.Creation
		}
		if start.IsZero() {
			continue
		}
		minutes = append(minutes, start.Hour()*60+start.Minute())
		day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		days[day.Format("2006-01-02")] = day
	}
	if len(minutes) < 2 || len(days) < 2 {
		return "", ""
	}
	reference := mostCommonMinute(minutes)
	for _, value := range minutes {
		if circularMinuteDistance(value, reference) > veeamScheduleToleranceMinutes {
			return "", ""
		}
	}
	if !consecutiveDays(days) {
		return "", ""
	}
	expr := strconv.Itoa(reference%60) + " " + strconv.Itoa(reference/60) + " * * *"
	return expr, cron.ToHuman(expr)
}

func consecutiveDays(days map[string]time.Time) bool {
	ordered := make([]time.Time, 0, len(days))
	for _, day := range days {
		ordered = append(ordered, day)
	}
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].Before(ordered[right]) })
	for index := 1; index < len(ordered); index++ {
		if ordered[index].Sub(ordered[index-1]) != 24*time.Hour {
			return false
		}
	}
	return true
}

func mostCommonMinute(minutes []int) int {
	counts := make(map[int]int, len(minutes))
	best, bestCount := minutes[0], 0
	for _, value := range minutes {
		counts[value]++
		if counts[value] > bestCount || (counts[value] == bestCount && value < best) {
			best, bestCount = value, counts[value]
		}
	}
	return best
}

func circularMinuteDistance(left, right int) int {
	diff := left - right
	if diff < 0 {
		diff = -diff
	}
	if diff > 12*60 {
		diff = 24*60 - diff
	}
	return diff
}
