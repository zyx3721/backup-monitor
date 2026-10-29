package service_test

import (
	"testing"

	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/service"
)

func TestHostRefreshPlanGroupsByInterval(t *testing.T) {
	svc := newTestService(t)
	profile := domain.AgentProfile{Name: "数据库", Hostname: "db-*", Enabled: true, Priority: 10}
	profile.Config.ReportIntervalSec = 120
	if err := svc.Repo().CreateAgentProfile(profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	for _, host := range []struct{ name, node, ip string }{
		{"db-01", "node-db", "10.0.0.21"},
		{"app-01", "node-app", "10.0.0.22"},
	} {
		report := sampleReport()
		report.Host.Hostname = host.name
		report.Host.NodeID = host.node
		report.Host.IP = host.ip
		report.Tasks = nil
		report.Executions = nil
		if _, err := svc.Ingest(t.Context(), report); err != nil {
			t.Fatalf("ingest %s: %v", host.name, err)
		}
	}

	plan, err := svc.HostRefreshPlan()
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.DefaultIntervalSec != 60 || plan.MinIntervalSec != 60 {
		t.Fatalf("plan intervals = %+v", plan)
	}
	if len(plan.Groups) != 2 {
		t.Fatalf("groups = %+v, want 2", plan.Groups)
	}
	byInterval := map[int]int{}
	for _, group := range plan.Groups {
		byInterval[group.IntervalSec] = len(group.HostIDs)
	}
	if byInterval[60] != 1 || byInterval[120] != 1 {
		t.Fatalf("group sizes = %+v", byInterval)
	}
}

func TestHostRefreshPlanClampsInterval(t *testing.T) {
	svc := newTestService(t)
	profile := domain.AgentProfile{Name: "高频", Hostname: "*", Enabled: true}
	profile.Config.ReportIntervalSec = 10
	if err := svc.Repo().CreateAgentProfile(profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	report := sampleReport()
	report.Tasks = nil
	report.Executions = nil
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	plan, err := svc.HostRefreshPlan()
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.MinIntervalSec != 10 || len(plan.Groups) != 1 || plan.Groups[0].IntervalSec != 10 {
		t.Fatalf("plan = %+v, want interval 10", plan)
	}
}

func TestFilterHostsByIDs(t *testing.T) {
	hosts := []domain.Host{{ID: "a", Hostname: "host-a"}, {ID: "b", Hostname: "host-b"}, {ID: "c", Hostname: "host-c"}}
	if got := service.FilterHostsByIDs(hosts, ""); len(got) != 3 {
		t.Fatalf("empty filter = %d, want 3", len(got))
	}
	if got := service.FilterHostsByIDs(hosts, "c, a"); len(got) != 2 || got[0].ID != "a" || got[1].ID != "c" {
		t.Fatalf("filter = %+v", got)
	}
	if got := service.FilterHostsByIDs(hosts, "missing"); len(got) != 0 {
		t.Fatalf("missing filter = %+v", got)
	}
}
