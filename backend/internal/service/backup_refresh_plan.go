package service

import (
	"sort"
	"strings"

	"backup-monitor-backend/internal/domain"
)

const (
	minAgentReportIntervalSec = 10
	maxAgentReportIntervalSec = 86400
)

// HostRefreshPlan 按主机实际匹配到的上报间隔分组，供管理端按组局部刷新。
func (s *BackupService) HostRefreshPlan() (domain.HostRefreshPlan, error) {
	hosts, err := s.repo.ListHosts()
	if err != nil {
		return domain.HostRefreshPlan{}, err
	}
	profiles, err := s.repo.ListAgentProfiles()
	if err != nil {
		return domain.HostRefreshPlan{}, err
	}
	defaultInterval := clampAgentReportInterval(s.Settings().Agent.ReportIntervalSec)
	grouped := make(map[int][]string, 4)
	for _, host := range hosts {
		interval := defaultInterval
		if profile := matchProfileFrom(profiles, host.Hostname); profile != nil && profile.Config.ReportIntervalSec > 0 {
			interval = clampAgentReportInterval(profile.Config.ReportIntervalSec)
		}
		grouped[interval] = append(grouped[interval], host.ID)
	}
	intervals := make([]int, 0, len(grouped))
	for interval := range grouped {
		intervals = append(intervals, interval)
	}
	sort.Ints(intervals)
	groups := make([]domain.HostRefreshGroup, 0, len(intervals))
	for _, interval := range intervals {
		groups = append(groups, domain.HostRefreshGroup{IntervalSec: interval, HostIDs: grouped[interval]})
	}
	minInterval := defaultInterval
	if len(intervals) > 0 {
		minInterval = intervals[0]
	}
	return domain.HostRefreshPlan{DefaultIntervalSec: defaultInterval, MinIntervalSec: minInterval, Groups: groups}, nil
}

// FilterHostsByIDs 按逗号分隔的主机 ID 过滤列表，空参数返回原列表。
func FilterHostsByIDs(hosts []domain.Host, raw string) []domain.Host {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return hosts
	}
	wanted := make(map[string]bool, 8)
	for _, id := range strings.Split(raw, ",") {
		if id = strings.TrimSpace(id); id != "" {
			wanted[id] = true
		}
	}
	filtered := make([]domain.Host, 0, len(wanted))
	for _, host := range hosts {
		if wanted[host.ID] {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

func clampAgentReportInterval(seconds int) int {
	if seconds < minAgentReportIntervalSec {
		return minAgentReportIntervalSec
	}
	if seconds > maxAgentReportIntervalSec {
		return maxAgentReportIntervalSec
	}
	return seconds
}

// matchProfileFrom 在已加载的采集配置中按主机名选出最优项。
func matchProfileFrom(profiles []domain.AgentProfile, hostname string) *domain.AgentProfile {
	type scored struct {
		profile *domain.AgentProfile
		score   int
	}
	candidates := []scored{}
	for i := range profiles {
		profile := &profiles[i]
		if !profile.Enabled || !globMatch(profile.Hostname, hostname) {
			continue
		}
		candidates = append(candidates, scored{profile: profile, score: patternScore(profile.Hostname)})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.profile.Priority != b.profile.Priority {
			return a.profile.Priority > b.profile.Priority
		}
		if a.score != b.score {
			return a.score > b.score
		}
		return a.profile.Name < b.profile.Name
	})
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0].profile
}
