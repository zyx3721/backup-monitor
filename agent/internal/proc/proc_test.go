package proc

import "testing"

func TestKeyword(t *testing.T) {
	items := []struct {
		in   []string
		want string
	}{
		{[]string{""}, ""},
		{[]string{"   "}, ""},
		{[]string{"veeam"}, "veeam"},
		{[]string{"  mysqldump  "}, "mysqldump"},
		{[]string{`"xtrabackup"`}, "xtrabackup"},
		{[]string{"'rsync'"}, "rsync"},
		{[]string{"", "  rsync  "}, "rsync"},
		{[]string{"", "", "rsync-ecs-gz"}, "rsync-ecs-gz"},
		{[]string{"  ", `"veeam"`, "task"}, "veeam"},
	}
	for _, item := range items {
		if got := Keyword(item.in...); got != item.want {
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
	if found, available := Running(""); found || available {
		t.Fatalf("Running with empty keyword = (%v, %v), want (false, false)", found, available)
	}
	if found, available := Running("   "); found || available {
		t.Fatalf("Running with blank keyword = (%v, %v), want (false, false)", found, available)
	}
}

func TestMatchSnapshot(t *testing.T) {
	commands := []string{"/bin/bash /opt/backup/mysql.sh"}
	items := []struct {
		name          string
		available     bool
		keyword       string
		wantFound     bool
		wantAvailable bool
	}{
		{"matched", true, "mysql.sh", true, true},
		{"not matched", true, "postgres", false, true},
		{"probe unavailable", false, "mysql.sh", false, false},
		{"empty keyword", true, "", false, false},
	}
	for _, item := range items {
		found, available := matchSnapshot(commands, item.available, item.keyword)
		if found != item.wantFound || available != item.wantAvailable {
			t.Errorf("%s: matchSnapshot = (%v, %v), want (%v, %v)", item.name, found, available, item.wantFound, item.wantAvailable)
		}
	}
}
