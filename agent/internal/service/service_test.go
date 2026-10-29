package service

import "testing"

func TestResolveName(t *testing.T) {
	cases := []struct {
		name string
		flag string
		env  string
		want string
	}{
		{"default", "", "", Name},
		{"flag wins", "my-agent", "env-agent", "my-agent"},
		{"env fallback", "", "env-agent", "env-agent"},
		{"strip unit suffix", "my-agent.service", "", "my-agent"},
		{"blank flag uses env", "  ", "env-agent", "env-agent"},
	}
	for _, item := range cases {
		t.Setenv("BM_SERVICE_NAME", item.env)
		if got := ResolveName(item.flag); got != item.want {
			t.Errorf("%s: ResolveName(%q) = %q, want %q", item.name, item.flag, got, item.want)
		}
	}
}

func TestDisplayNameFor(t *testing.T) {
	if got := DisplayNameFor(Name); got != DisplayName {
		t.Errorf("DisplayNameFor(%q) = %q, want %q", Name, got, DisplayName)
	}
	if got, want := DisplayNameFor("backup-agent-2"), DisplayName+" (backup-agent-2)"; got != want {
		t.Errorf("DisplayNameFor(backup-agent-2) = %q, want %q", got, want)
	}
}

func TestResolveDisplayName(t *testing.T) {
	t.Setenv("BM_DISPLAY_NAME", "")
	if got, want := ResolveDisplayName("", "backup-agent-2"), DisplayName+" (backup-agent-2)"; got != want {
		t.Errorf("default = %q, want %q", got, want)
	}
	if got := ResolveDisplayName("自定义显示名", "backup-agent-2"); got != "自定义显示名" {
		t.Errorf("flag = %q, want 自定义显示名", got)
	}
	t.Setenv("BM_DISPLAY_NAME", "环境显示名")
	if got := ResolveDisplayName("", "backup-agent-2"); got != "环境显示名" {
		t.Errorf("env = %q, want 环境显示名", got)
	}
	if got := ResolveDisplayName("命令行优先", "backup-agent-2"); got != "命令行优先" {
		t.Errorf("flag over env = %q, want 命令行优先", got)
	}
}
