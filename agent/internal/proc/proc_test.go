package proc

import "testing"

func TestKeyword(t *testing.T) {
	items := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"veeam", "veeam"},
		{"  mysqldump  ", "mysqldump"},
		{`"xtrabackup"`, "xtrabackup"},
		{"'rsync'", "rsync"},
	}
	for _, item := range items {
		if got := Keyword(item.in); got != item.want {
			t.Errorf("Keyword(%q) = %q, want %q", item.in, got, item.want)
		}
	}
}

func TestMatchCommands(t *testing.T) {
	commands := []string{
		"/bin/bash /opt/backup/mysql.sh",
		"/usr/bin/veeamconfig session list",
		"/usr/sbin/cron -f",
	}
	items := []struct {
		keyword string
		want    bool
	}{
		{"mysql.sh", true},
		{"veeam", true},
		{"MYSQL", true},
		{"", false},
		{"postgres", false},
		{"backup", true},
	}
	for _, item := range items {
		if got := matchCommands(commands, item.keyword); got != item.want {
			t.Errorf("matchCommands(%q) = %v, want %v", item.keyword, got, item.want)
		}
	}
}

func TestRunningWithEmptyKeyword(t *testing.T) {
	if Running("") {
		t.Fatal("Running with empty keyword must be false")
	}
	if Running("   ") {
		t.Fatal("Running with blank keyword must be false")
	}
}
