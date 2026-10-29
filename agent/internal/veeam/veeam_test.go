package veeam

import (
	"strings"
	"testing"
)

func TestParseJobs(t *testing.T) {
	output := strings.Join([]string{
		"Job ID                                  Job name             Type        Status",
		"{0f8fad5b-d9cb-469f-a165-70867728950e}  Daily Database       Endpoint    Success",
		"{1a9fad5b-d9cb-469f-a165-70867728950f}  Weekly Files         Endpoint    Warning",
	}, "\n")
	jobs := parseJobs(output)
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2: %#v", len(jobs), jobs)
	}
	if jobs[0].ID != "{0f8fad5b-d9cb-469f-a165-70867728950e}" {
		t.Errorf("id = %q", jobs[0].ID)
	}
	if jobs[0].Name != "Daily Database" {
		t.Errorf("name = %q", jobs[0].Name)
	}
	if jobs[0].Type != "Endpoint" {
		t.Errorf("type = %q", jobs[0].Type)
	}
	if jobs[0].Status != "Success" {
		t.Errorf("status = %q", jobs[0].Status)
	}
}

func TestParseSessions(t *testing.T) {
	output := strings.Join([]string{
		"Session ID                              Job name             State       Start time           End time             Size",
		"{aaaa}                                   Daily Database       Success     2026-10-20 23:00:01  2026-10-20 23:10:00  1.2 GB",
		"{bbbb}                                   Weekly Files         Running     2026-10-21 01:00:00                      2.4 GB",
	}, "\n")
	sessions := parseSessions(output)
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2: %#v", len(sessions), sessions)
	}
	first := sessions[0]
	if first.State != "success" {
		t.Errorf("state = %q", first.State)
	}
	if !first.HasStart || !first.HasEnd {
		t.Errorf("times missing: %#v", first)
	}
	if first.ProcessedSize != "1.2 GB" {
		t.Errorf("size = %q", first.ProcessedSize)
	}
	if sessions[1].State != "running" || sessions[1].HasEnd {
		t.Errorf("running session = %#v", sessions[1])
	}
}

func TestParseSessionInfo(t *testing.T) {
	output := strings.Join([]string{
		"Session ID: {aaaa}",
		"Job name: Daily Database",
		"Job ID: {bbbb}",
		"OIB ID: {cccc}",
		"State: Failed",
		"Start time: 2026-10-20 23:00:01",
		"End time: 2026-10-20 23:01:00",
		"Total size: 3.5 GB",
		"Transferred size: 3.1 GB",
		"Attempt: 1 / 4",
		"Last attempt: Yes",
	}, "\n")
	session := parseSessionInfo(output)
	if session.ID != "{aaaa}" || session.JobName != "Daily Database" {
		t.Errorf("identity = %#v", session)
	}
	if session.State != "failed" {
		t.Errorf("state = %q", session.State)
	}
	if !session.HasStart || !session.HasEnd {
		t.Errorf("times missing")
	}
	if session.ProcessedSize != "3.5 GB" || session.TransferredSize != "3.1 GB" {
		t.Errorf("sizes = %q / %q", session.ProcessedSize, session.TransferredSize)
	}
	if session.OIBID != "{cccc}" {
		t.Errorf("OIB ID = %q", session.OIBID)
	}
	if !session.HasAttempt || session.Attempt != 1 || !session.HasMaxAttempts || session.MaxAttempts != 4 {
		t.Errorf("attempt = %#v", session)
	}
	if !session.HasLastAttempt || !session.IsLastAttempt {
		t.Errorf("last attempt = %#v", session)
	}
}

func TestNormalizeState(t *testing.T) {
	cases := map[string]string{
		"Success":     "success",
		"Warning":     "success",
		"In Progress": "running",
		"Failed":      "failed",
		"Stopped":     "failed",
		"":            "",
	}
	for input, want := range cases {
		if got := NormalizeState(input); got != want {
			t.Errorf("NormalizeState(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseSessionInfoRealOutput(t *testing.T) {
	output := `Backup session
   ID: {369e80c8-ba71-4b07-afc3-86945ec35eb6}
   Job name: website
   Job ID: {8d2b779c-2717-4f7a-aaf2-333ec748ffc2}
   OIB ID: {d3930775-7a7f-4828-9521-dd677d43ff27}
   State: Success
   Creation time: 2026-10-26 06:00:14
   Start time: 2026-10-26 06:00:14
   End time: 2026-10-26 06:01:02
   Progress details:
      Processed: 99.8 GB
      Transferred: 445.4 MB
   Retries details:
      Attempt: #1 from 4
      Is last attempt: true`
	session := parseSessionInfo(output)
	if session.OIBID != "{d3930775-7a7f-4828-9521-dd677d43ff27}" {
		t.Errorf("OIB ID = %q", session.OIBID)
	}
	if !session.HasAttempt || session.Attempt != 1 || !session.HasMaxAttempts || session.MaxAttempts != 4 {
		t.Errorf("attempt = %#v", session)
	}
	if !session.HasLastAttempt || !session.IsLastAttempt {
		t.Errorf("last attempt = %#v", session)
	}
}
