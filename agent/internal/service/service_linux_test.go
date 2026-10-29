package service

import (
	"os/exec"
	"testing"
)

func TestUnitExecStartMatches(t *testing.T) {
	content := "[Service]\nExecStart=/opt/backup-agent2/backup-agent run --config /opt/backup-agent2/backup-agent.json\n"
	if !unitExecStartMatches(content, "/opt/backup-agent2/backup-agent", "/opt/backup-agent2/backup-agent") {
		t.Fatal("expected ExecStart match")
	}
	if unitExecStartMatches(content, "/opt/backup-agent/backup-agent-linux", "/opt/backup-agent/backup-agent-linux") {
		t.Fatal("unexpected match for another unit")
	}
}

func TestFirstField(t *testing.T) {
	cases := map[string]string{
		`/opt/backup-agent2/backup-agent run`:  "/opt/backup-agent2/backup-agent",
		`"/opt/backup agent/backup-agent" run`: "/opt/backup agent/backup-agent",
		`-/opt/backup-agent2/backup-agent run`: "/opt/backup-agent2/backup-agent",
	}
	for input, want := range cases {
		if got := firstField(input); got != want {
			t.Errorf("firstField(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestKilledBySignal(t *testing.T) {
	terminated, err := exec.Command("sh", "-c", "kill -TERM $$").CombinedOutput()
	if err == nil {
		t.Fatalf("expected signal error, output %q", string(terminated))
	}
	if !killedBySignal(err) {
		t.Fatalf("killedBySignal(%v) = false, want true", err)
	}

	exited, err := exec.Command("sh", "-c", "exit 1").CombinedOutput()
	if err == nil {
		t.Fatalf("expected exit error, output %q", string(exited))
	}
	if killedBySignal(err) {
		t.Fatalf("killedBySignal(%v) = true, want false", err)
	}
}
