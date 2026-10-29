package main

import (
	"io"
	"log"
	"testing"
	"time"

	"backup-monitor-agent/internal/api"
)

func newTestAgent() *agent {
	return &agent{logger: log.New(io.Discard, "", 0)}
}

func profileResponse(minutes int, enabled bool) *api.ConfigResponse {
	return &api.ConfigResponse{
		Profile: &api.Profile{Enabled: enabled, Config: api.ProfileConfig{AutoUpdateMinutes: minutes}},
	}
}

func TestSyncAutoUpdateIntervalRestartsCountdown(t *testing.T) {
	instance := newTestAgent()
	instance.lastUpdate = time.Now().Add(-time.Hour)

	changedAt := time.Now()
	instance.syncAutoUpdateInterval(profileResponse(5, true))
	if instance.autoUpdate != 5 {
		t.Fatalf("autoUpdate = %d, want 5", instance.autoUpdate)
	}
	if instance.lastUpdate.Before(changedAt) || time.Since(instance.lastUpdate) > time.Minute {
		t.Fatalf("间隔变化后应从当前时刻重新计时，lastUpdate = %s", instance.lastUpdate)
	}

	pinned := time.Now().Add(-2 * time.Hour)
	instance.lastUpdate = pinned
	instance.syncAutoUpdateInterval(profileResponse(5, true))
	if !instance.lastUpdate.Equal(pinned) {
		t.Fatalf("间隔未变化时不应重置计时，lastUpdate = %s", instance.lastUpdate)
	}

	instance.syncAutoUpdateInterval(profileResponse(0, true))
	if instance.autoUpdate != 0 {
		t.Fatalf("autoUpdate = %d, want 0", instance.autoUpdate)
	}
	if time.Since(instance.lastUpdate) > time.Minute {
		t.Fatalf("改为 0 时同样按修改时刻重新计时，lastUpdate = %s", instance.lastUpdate)
	}
}

func TestAutoUpdateMinutesOfIgnoresDisabledProfile(t *testing.T) {
	cases := []struct {
		name     string
		response *api.ConfigResponse
		want     int
	}{
		{name: "无配置", response: nil, want: 0},
		{name: "配置未启用", response: profileResponse(9, false), want: 0},
		{name: "间隔为负", response: profileResponse(-3, true), want: 0},
		{name: "已启用", response: profileResponse(9, true), want: 9},
	}
	for _, testCase := range cases {
		if got := autoUpdateMinutesOf(testCase.response); got != testCase.want {
			t.Errorf("%s: got %d, want %d", testCase.name, got, testCase.want)
		}
	}
}
