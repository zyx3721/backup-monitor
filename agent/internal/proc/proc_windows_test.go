//go:build windows

package proc

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCommandsWithWMICSuccess(t *testing.T) {
	var calls []string
	run := func(timeout time.Duration, name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return []byte("CommandLine\r\nC:\\Windows\\system32\\cmd.exe /c C:\\veeam-backup.bat\r\n"), nil
	}

	commands, available := commandsWith(run)
	if !available {
		t.Fatal("expected probe to be available")
	}
	if !matchCommands(commands, "veeam-backup.bat") {
		t.Fatalf("expected batch script to match, commands=%v", commands)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want only wmic", calls)
	}
	if want := "wmic process get CommandLine"; calls[0] != want {
		t.Fatalf("call = %q, want %q", calls[0], want)
	}
}

func TestCommandsWithPowerShellFallback(t *testing.T) {
	var calls []string
	run := func(timeout time.Duration, name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "wmic" {
			return nil, errors.New("wmic not available")
		}
		return []byte("C:\\VEEAM-BACKUP.BAT\r\n"), nil
	}

	commands, available := commandsWith(run)
	if !available {
		t.Fatal("expected probe to be available")
	}
	if !matchCommands(commands, "veeam-backup.bat") {
		t.Fatalf("expected fallback output to match, commands=%v", commands)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %v, want wmic then powershell", calls)
	}
	if !strings.HasPrefix(calls[1], "powershell -NoProfile -NonInteractive -Command "+powerShellQuery) {
		t.Fatalf("fallback call = %q", calls[1])
	}
}

func TestCommandsWithEmptyWMICOutputFallsBack(t *testing.T) {
	var calls []string
	run := func(timeout time.Duration, name string, args ...string) ([]byte, error) {
		calls = append(calls, name)
		if name == "wmic" {
			return []byte(" \r\n"), nil
		}
		return []byte("C:\\veeam-backup.bat\r\n"), nil
	}

	commands, available := commandsWith(run)
	if !available {
		t.Fatal("expected probe to be available")
	}
	if !matchCommands(commands, "veeam-backup.bat") {
		t.Fatalf("expected fallback output to match, commands=%v", commands)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %v, want wmic then powershell", calls)
	}
}

func TestCommandsWithUnavailableProbe(t *testing.T) {
	items := []struct {
		name string
		run  func(time.Duration, string, ...string) ([]byte, error)
	}{
		{
			name: "both commands fail",
			run: func(time.Duration, string, ...string) ([]byte, error) {
				return nil, errors.New("command failed")
			},
		},
		{
			name: "both commands return empty output",
			run: func(time.Duration, string, ...string) ([]byte, error) {
				return []byte("\r\n"), nil
			},
		},
	}

	for _, item := range items {
		commands, available := commandsWith(item.run)
		if available || len(commands) != 0 {
			t.Errorf("%s: commandsWith = (%v, %v), want (nil, false)", item.name, commands, available)
		}
	}
}

func TestCommandLines(t *testing.T) {
	output := []byte("\ufeffCommandLine\r\n\r\nC:\\Windows\\system32\\cmd.exe /c C:\\veeam-backup.bat\r\n")
	lines := commandLines(output)
	want := []string{"CommandLine", "C:\\Windows\\system32\\cmd.exe /c C:\\veeam-backup.bat"}
	if len(lines) != len(want) {
		t.Fatalf("commandLines = %v, want %v", lines, want)
	}
	for index := range want {
		if lines[index] != want[index] {
			t.Errorf("line %d = %q, want %q", index, lines[index], want[index])
		}
	}
}
