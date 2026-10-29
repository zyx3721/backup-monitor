//go:build windows

package service

import (
	"strings"
	"testing"
)

func TestServiceExecutable(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"quoted with args", `"C:\Program Files\Backup Monitor\backup-agent.exe" -service`, `C:\Program Files\Backup Monitor\backup-agent.exe`},
		{"unquoted with args", `C:\Backup\backup-agent.exe -service`, `C:\Backup\backup-agent.exe`},
		{"unquoted no args", `C:\Backup\backup-agent.exe`, `C:\Backup\backup-agent.exe`},
		{"empty", ``, ``},
	}
	for _, item := range cases {
		if got := serviceExecutable(item.input); got != item.want {
			t.Errorf("%s: serviceExecutable(%q) = %q, want %q", item.name, item.input, got, item.want)
		}
	}
}

func TestMatchesExecutable(t *testing.T) {
	if !matchesExecutable(`C:\Program Files\Backup\agent.exe`, `c:\program files\backup\AGENT.EXE`) {
		t.Error("expected case-insensitive match")
	}
	if matchesExecutable(`C:\Backup\agent.exe`, `C:\Backup\agent2.exe`) {
		t.Error("expected different paths not to match")
	}
	if matchesExecutable("", `C:\Backup\agent.exe`) {
		t.Error("expected empty path not to match")
	}
}

func TestRestartScript(t *testing.T) {
	script := restartScript("backup-agent2")
	for _, fragment := range []string{
		`sc.exe stop "backup-agent2"`,
		`sc.exe query "backup-agent2"`,
		`findstr /C:"STOPPED"`,
		`for /l %%i in (1,1,30)`,
		`sc.exe start "backup-agent2"`,
	} {
		if !strings.Contains(script, fragment) {
			t.Errorf("restart script missing %q", fragment)
		}
	}
	if strings.Contains(script, "{{name}}") {
		t.Error("restart script still contains placeholder")
	}
}
