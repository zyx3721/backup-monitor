package service

import "testing"

func TestParseCronAndSchedule(t *testing.T) {
	cases := []struct {
		expr    string
		now     string
		want    string
		wantErr bool
	}{
		{expr: "0 23 * * *", now: "2026-10-21T22:10:00Z", want: "2026-10-20T23:00:00Z"},
		{expr: "30 1 * * 1", now: "2026-10-21T02:00:00Z", want: "2026-10-19T01:30:00Z"},
		{expr: "*/15 * * * *", now: "2026-10-21T10:07:00Z", want: "2026-10-21T10:00:00Z"},
		{expr: "0 0 1 * *", now: "2026-10-21T10:00:00Z", want: "2026-10-01T00:00:00Z"},
		{expr: "bad", now: "2026-10-21T10:00:00Z", wantErr: true},
	}
	for _, item := range cases {
		now, err := parseTime(item.now)
		if err != nil {
			t.Fatalf("parse now: %v", err)
		}
		got, ok := LastScheduled(item.expr, now)
		if item.wantErr {
			if ok {
				t.Fatalf("%s expected invalid", item.expr)
			}
			continue
		}
		if !ok {
			t.Fatalf("%s: no schedule", item.expr)
		}
		if got.UTC().Format("2006-01-02T15:04:05Z") != item.want {
			t.Fatalf("%s: got %s want %s", item.expr, got.UTC().Format("2006-01-02T15:04:05Z"), item.want)
		}
	}
}

func TestCronValidation(t *testing.T) {
	valid := []string{"* * * * *", "0 0 * * 0", "0 23 * * 1-5", "*/5 * * * *", "1,15 2 * * *"}
	for _, expr := range valid {
		if !IsValidCron(expr) {
			t.Fatalf("expected valid: %s", expr)
		}
	}
	invalid := []string{"", "* * * *", "60 * * * *", "0 24 * * *", "*/0 * * * *", "5-1 * * * *", "a b c d e"}
	for _, expr := range invalid {
		if IsValidCron(expr) {
			t.Fatalf("expected invalid: %s", expr)
		}
	}
}

func TestCronFromHuman(t *testing.T) {
	cases := map[string]string{
		"每天 23:30":  "30 23 * * *",
		"每周一 02:00": "0 2 * * 1",
		"每小时":       "0 * * * *",
	}
	for input, want := range cases {
		if got := CronFromHuman(input); got != want {
			t.Fatalf("%s: got %q want %q", input, got, want)
		}
	}
}

func TestHumanScheduleFromCron(t *testing.T) {
	cases := map[string]string{
		"0 6 * * *":   "每天 06:00",
		"30 23 * * *": "每天 23:30",
		"0 2 * * 1":   "每周一 02:00",
		"10 0 1 * *":  "每月 1 日 00:10",
		"0 * * * *":   "每小时",
		"15 * * * *":  "每小时第 15 分",
		"0 0 * * 0":   "每周日 00:00",
	}
	for expr, want := range cases {
		if got := HumanScheduleFromCron(expr); got != want {
			t.Errorf("HumanScheduleFromCron(%q) = %q, want %q", expr, got, want)
		}
	}
	if got := HumanScheduleFromCron("bad cron"); got != "" {
		t.Errorf("invalid cron = %q, want empty", got)
	}
}

func TestNormalizeIncomingSchedule(t *testing.T) {
	if got, disabled := normalizeIncomingSchedule("未启用", "0 6 * * *"); !disabled || got != "每天 06:00" {
		t.Fatalf("disabled with cron = %q (disabled=%v)", got, disabled)
	}
	if got, disabled := normalizeIncomingSchedule("未启用", ""); !disabled || got != "" {
		t.Fatalf("disabled without cron = %q (disabled=%v)", got, disabled)
	}
	if got, disabled := normalizeIncomingSchedule("每天 06:00", "0 6 * * *"); disabled || got != "每天 06:00" {
		t.Fatalf("normal schedule = %q (disabled=%v)", got, disabled)
	}
}
