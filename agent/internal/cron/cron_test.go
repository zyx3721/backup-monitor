package cron

import (
	"testing"
	"time"
)

func TestParseValid(t *testing.T) {
	expressions := []string{
		"0 23 * * *",
		"*/15 * * * *",
		"0 2 * * 1-5",
		"30 3 1,15 * *",
		"0 0 1 1 *",
		"@daily",
		"0 0 * * 7",
	}
	for _, expression := range expressions {
		if _, err := Parse(expression); err != nil {
			t.Fatalf("Parse(%q) returned error: %v", expression, err)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	expressions := []string{
		"",
		"* * * *",
		"60 12 * * *",
		"*/0 * * * *",
		"0 24 * * *",
		"0 0 0 * *",
		"0 0 * 13 *",
		"0 0 * * 8",
		"0 5-1 * * *",
		"0 0 * * MON",
		"@reboot",
	}
	for _, expression := range expressions {
		if _, err := Parse(expression); err == nil {
			t.Fatalf("Parse(%q) expected error", expression)
		}
	}
}

func TestGetLastScheduledTime(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 10, 21, 10, 30, 0, 0, zone)
	got, ok := GetLastScheduledTime("0 23 * * *", now)
	if !ok {
		t.Fatal("expected a scheduled time")
	}
	want := time.Date(2026, 10, 20, 23, 0, 0, 0, zone)
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}

	exact := time.Date(2026, 10, 20, 23, 0, 0, 0, zone)
	got, ok = GetLastScheduledTime("0 23 * * *", exact)
	if !ok || !got.Equal(exact) {
		t.Fatalf("got %s ok=%v, want %s", got, ok, exact)
	}

	weekly, ok := GetLastScheduledTime("0 0 * * 1", now)
	if !ok {
		t.Fatal("expected weekly scheduled time")
	}
	wantWeekly := time.Date(2026, 10, 19, 0, 0, 0, 0, zone)
	if !weekly.Equal(wantWeekly) {
		t.Fatalf("got %s, want %s", weekly, wantWeekly)
	}

	macro, ok := GetLastScheduledTime("@daily", now)
	if !ok {
		t.Fatal("expected macro scheduled time")
	}
	wantMacro := time.Date(2026, 10, 21, 0, 0, 0, 0, zone)
	if !macro.Equal(wantMacro) {
		t.Fatalf("got %s, want %s", macro, wantMacro)
	}

	if _, ok := GetLastScheduledTime("0 0 29 2 *", now); ok {
		t.Fatal("expected no scheduled time within the search window")
	}
}

func TestGetNextScheduledTime(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 10, 21, 10, 30, 0, 0, zone)
	got, ok := GetNextScheduledTime("0 23 * * *", now)
	if !ok {
		t.Fatal("expected next scheduled time")
	}
	want := time.Date(2026, 10, 21, 23, 0, 0, 0, zone)
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestMatchesUsesAndSemantics(t *testing.T) {
	schedule, err := Parse("0 0 1 * 1")
	if err != nil {
		t.Fatal(err)
	}
	if !schedule.Matches(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected January 1 2024 to match")
	}
	if schedule.Matches(time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected January 8 2024 not to match")
	}
}

func TestToHuman(t *testing.T) {
	cases := map[string]string{
		"0 23 * * *":   "每天 23:00",
		"0 2 * * 1":    "每周一 02:00",
		"0 * * * *":    "每小时",
		"*/15 * * * *": "每 15 分钟",
		"30 3 1 * *":   "每月 1 日 03:30",
		"0 0 1 1 *":    "每年 1 月 1 日 00:00",
		"0 0 * * 0":    "每周日 00:00",
		"15 2 * * 1,3": "每周一、周三 02:15",
		"0 0 * * 1-5":  "每周一至周五 00:00",
		"0 23 * * 1-6": "每周一至周六 23:00",
		"0 6 1,15 * *": "每月 1、15 日 06:00",
		"@daily":       "每天 00:00",
		"5 * * * *":    "每小时第 5 分",
		"0 */2 * * *":  "每 2 小时",
		"7 0 1 2 *":    "每年 2 月 1 日 00:07",
	}
	for expression, want := range cases {
		if got := ToHuman(expression); got != want {
			t.Errorf("ToHuman(%q) = %q, want %q", expression, got, want)
		}
	}
}

func TestToHumanInvalid(t *testing.T) {
	if got := ToHuman(""); got != "未设置计划" {
		t.Fatalf("got %q", got)
	}
	if got := ToHuman("bad cron"); got != "自定义计划 bad cron" {
		t.Fatalf("got %q", got)
	}
}
