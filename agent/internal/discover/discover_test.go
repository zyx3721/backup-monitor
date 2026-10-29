package discover

import (
	"strings"
	"testing"
)

func defaultFilter() Filter {
	return Filter{
		LabelPrefix:     "# backup-monitor:",
		IncludeKeywords: []string{"backup", "dump", "rsync", "tar"},
		IncludePaths:    []string{"/data/scripts", "/opt/backup"},
		ExcludeKeywords: []string{"logrotate", "certbot"},
	}
}

func TestParseCrontabProcKeyword(t *testing.T) {
	text := `# backup-monitor: name=数据库备份;key=db;proc=mysqldump
0 23 * * * /opt/backup/db.sh
`
	tasks := ParseCrontab(text, "alice", "crontab", false, defaultFilter())
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}
	if tasks[0].ProcKeyword != "mysqldump" {
		t.Errorf("proc keyword = %q", tasks[0].ProcKeyword)
	}
}

func TestParseLabelProcKeyword(t *testing.T) {
	info := parseLabel(`name=db;proc="/usr/bin/veeamconfig"`)
	if info.Proc != "/usr/bin/veeamconfig" {
		t.Errorf("proc = %q", info.Proc)
	}
	if merged := mergeLabel(labelInfo{}, parseLabel("process=rsync")); merged.Proc != "rsync" {
		t.Errorf("merged proc = %q", merged.Proc)
	}
}

func TestParseLabelSpaceSeparated(t *testing.T) {
	info := parseLabel(`name=rsync-mys-app key=rsync-mys-file status=/var/log/backup/rsync-mys-file.status`)
	if info.Name != "rsync-mys-app" {
		t.Errorf("name = %q", info.Name)
	}
	if info.Key != "rsync-mys-file" {
		t.Errorf("key = %q", info.Key)
	}
	if info.StatusFile != "/var/log/backup/rsync-mys-file.status" {
		t.Errorf("status file = %q", info.StatusFile)
	}
}

func TestParseLabelQuotedValueWithSpace(t *testing.T) {
	info := parseLabel(`name="每日 数据库备份" key=db-daily`)
	if info.Name != "每日 数据库备份" {
		t.Errorf("name = %q", info.Name)
	}
	if info.Key != "db-daily" {
		t.Errorf("key = %q", info.Key)
	}
}

func TestParseLabelUnquotedValueWithSpace(t *testing.T) {
	info := parseLabel(`name=每日 数据库备份`)
	if info.Name != "每日 数据库备份" {
		t.Errorf("name = %q", info.Name)
	}
}

func TestParseLabelKeepsApostrophe(t *testing.T) {
	info := parseLabel(`name=it's backup;key=db`)
	if info.Name != "it's backup" {
		t.Errorf("name = %q", info.Name)
	}
	if info.Key != "db" {
		t.Errorf("key = %q", info.Key)
	}
}

func TestParseCrontabSpaceSeparatedLabel(t *testing.T) {
	text := `# backup-monitor: name=rsync-mys-app key=rsync-mys-file status=/var/log/backup/rsync-mys-file.status
10 0 * * * sh /root/rsync-file.sh
`
	tasks := ParseCrontab(text, "root", "crontab", false, defaultFilter())
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}
	task := tasks[0]
	if task.Name != "rsync-mys-app" {
		t.Errorf("name = %q", task.Name)
	}
	if task.Key != "rsync-mys-file" {
		t.Errorf("key = %q", task.Key)
	}
	if task.Config["statusFile"] != "/var/log/backup/rsync-mys-file.status" {
		t.Errorf("statusFile = %#v", task.Config["statusFile"])
	}
}

func TestParseCrontabUserTable(t *testing.T) {
	text := `# backup-monitor: name=每日数据库备份;key=db-daily;status=/data/backup/db.status;artifact=/data/backup
0 23 * * * /opt/backup/db.sh >> /var/log/db.log 2>&1
*/10 * * * * /usr/bin/logrotate /etc/logrotate.conf
30 2 * * 1-5 /data/scripts/incremental.sh
`
	tasks := ParseCrontab(text, "alice", "crontab", false, defaultFilter())
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2: %#v", len(tasks), tasks)
	}

	first := tasks[0]
	if first.Name != "每日数据库备份" {
		t.Errorf("name = %q", first.Name)
	}
	if first.Key != "db-daily" {
		t.Errorf("key = %q", first.Key)
	}
	if first.Schedule != "每天 23:00" || first.ScheduleCron != "0 23 * * *" {
		t.Errorf("schedule = %q / %q", first.Schedule, first.ScheduleCron)
	}
	if first.User != "alice" {
		t.Errorf("user = %q", first.User)
	}
	if first.Script != "/opt/backup/db.sh" {
		t.Errorf("script = %q", first.Script)
	}
	if first.Config["statusFile"] != "/data/backup/db.status" {
		t.Errorf("statusFile = %#v", first.Config["statusFile"])
	}
	if first.Config["artifactPath"] != "/data/backup" {
		t.Errorf("artifactPath = %#v", first.Config["artifactPath"])
	}

	second := tasks[1]
	if second.Name != "incremental" {
		t.Errorf("name = %q", second.Name)
	}
	if second.Key != "cron:alice:30 2 * * 1-5" {
		t.Errorf("key = %q", second.Key)
	}
	if second.Schedule != "每周一至周五 02:30" {
		t.Errorf("schedule = %q", second.Schedule)
	}
}

func TestParseCrontabSystemTable(t *testing.T) {
	text := `SHELL=/bin/sh
# backup-monitor: sys-db
0 1 * * * root /usr/local/bin/backup-db.sh
15 3 * * * root /opt/backup/inline.sh # backup-monitor: inline-task
`
	tasks := ParseCrontab(text, "", "/etc/crontab", true, defaultFilter())
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2: %#v", len(tasks), tasks)
	}
	if tasks[0].User != "root" {
		t.Errorf("user = %q", tasks[0].User)
	}
	if tasks[0].Config["file"] != "/etc/crontab" {
		t.Errorf("file = %#v", tasks[0].Config["file"])
	}
	if tasks[1].Key != "inline-task" {
		t.Errorf("inline key = %q", tasks[1].Key)
	}
	if tasks[1].Name != "inline-task" {
		t.Errorf("inline name = %q", tasks[1].Name)
	}
}

func TestParseCrontabMacros(t *testing.T) {
	text := `@daily root /opt/backup/daily.sh
@reboot root /opt/backup/boot.sh
`
	tasks := ParseCrontab(text, "", "/etc/crontab", true, defaultFilter())
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(tasks))
	}
	if tasks[0].ScheduleCron != "0 0 * * *" || tasks[0].Schedule != "每天 00:00" {
		t.Errorf("daily = %q / %q", tasks[0].ScheduleCron, tasks[0].Schedule)
	}
	if tasks[1].ScheduleCron != "" || tasks[1].Schedule != "开机时" {
		t.Errorf("reboot = %q / %q", tasks[1].ScheduleCron, tasks[1].Schedule)
	}
}

func TestParseCrontabSkipsNoise(t *testing.T) {
	text := `MAILTO=root
0 0 * * * /usr/sbin/ntpdate pool.ntp.org
0 0 * * * /usr/bin/certbot renew
0 0 * * 8 /opt/backup/invalid.sh
`
	tasks := ParseCrontab(text, "root", "crontab", false, defaultFilter())
	if len(tasks) != 0 {
		t.Fatalf("got %d tasks, want 0: %#v", len(tasks), tasks)
	}
}

func TestScriptPath(t *testing.T) {
	cases := map[string]string{
		"/opt/backup/db.sh --full":                   "/opt/backup/db.sh",
		"/bin/bash -x /opt/backup/run.sh":            "/opt/backup/run.sh",
		"cd /data && ./backup.sh":                    "./backup.sh",
		"python3 /data/scripts/dump.py --all":        "/data/scripts/dump.py",
		`"C:\Program Files\Backup\agent.exe" /quiet`: `C:\Program Files\Backup\agent.exe`,
		"sudo /opt/backup/root.sh":                   "/opt/backup/root.sh",
		"/usr/bin/rsync -av /data /backup":           "/usr/bin/rsync",
		`cmd.exe /c C:\Backup\db.bat`:                `C:\Backup\db.bat`,
		`powershell.exe -File C:\Backup\db.ps1`:      `C:\Backup\db.ps1`,
	}
	for command, want := range cases {
		if got := ScriptPath(command); got != want {
			t.Errorf("ScriptPath(%q) = %q, want %q", command, got, want)
		}
	}
}

func TestResolveName(t *testing.T) {
	if got := ResolveName("/opt/backup/db.sh", "", ""); got != "db" {
		t.Errorf("script name = %q", got)
	}
	if got := ResolveName("/opt/backup/db.sh", "每日备份", ""); got != "每日备份" {
		t.Errorf("label name = %q", got)
	}
	if got := ResolveName("/opt/backup/db.sh", "", "db-daily"); got != "db-daily" {
		t.Errorf("key name = %q", got)
	}
}

func TestParseScheduledTasksJSON(t *testing.T) {
	data := []byte(`[
  {"TaskName":"DB Backup","TaskPath":"\\Backup\\","State":"Ready","LastRunTime":"2026-10-20T23:00:01","LastTaskResult":0,"Execute":"C:\\Backup\\db.bat","Arguments":"","Comment":"# backup-monitor: win-db","Schedule":"2026-10-21T23:00:00","TriggerType":"MSFT_TaskDailyTrigger","DaysOfWeek":"","DaysOfMonth":"","RepetitionInterval":"","Enabled":true},
  {"TaskName":"Weekly","TaskPath":"\\","State":"Ready","LastRunTime":"2026-10-19T02:00:00","LastTaskResult":267009,"Execute":"powershell.exe","Arguments":"-File C:\\Backup\\week.ps1","Comment":"backup weekly","Schedule":"2026-10-26T02:00:00","TriggerType":"MSFT_TaskWeeklyTrigger","DaysOfWeek":"Monday","DaysOfMonth":"","RepetitionInterval":"","Enabled":true},
  {"TaskName":"Noise","TaskPath":"\\","State":"Ready","Execute":"C:\\Windows\\notepad.exe","Comment":"logrotate","Enabled":true},
  {"TaskName":"Disabled","TaskPath":"\\","State":"Disabled","Execute":"C:\\Backup\\disabled.bat","Comment":"backup disabled","Enabled":false}
]`)
	tasks, err := ParseScheduledTasksJSON(data, defaultFilter())
	if err != nil {
		t.Fatalf("ParseScheduledTasksJSON returned error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2: %#v", len(tasks), tasks)
	}

	daily := tasks[0]
	if daily.Name != "win-db" {
		t.Errorf("name = %q", daily.Name)
	}
	if daily.Key != "win-db" {
		t.Errorf("key = %q", daily.Key)
	}
	if daily.ScheduleCron != "0 23 * * *" || daily.Schedule != "每天 23:00" {
		t.Errorf("daily schedule = %q / %q", daily.ScheduleCron, daily.Schedule)
	}
	if !daily.HasLastRun || daily.LastRun.IsZero() {
		t.Errorf("last run missing")
	}

	weekly := tasks[1]
	if weekly.ScheduleCron != "0 2 * * 1" || weekly.Schedule != "每周一 02:00" {
		t.Errorf("weekly schedule = %q / %q", weekly.ScheduleCron, weekly.Schedule)
	}
	if weekly.LastResult != 267009 {
		t.Errorf("last result = %d", weekly.LastResult)
	}
}

func TestParseSchtasksCSV(t *testing.T) {
	data := []byte(`"TaskName","Next Run Time","Status","Logon Mode","Last Run Time","Last Result","Author","Task To Run","Start In","Comment","Scheduled Task State","Idle Time","Power Management","Run As User","Schedule Type","Start Time","Start Date","End Date","Days","Months","Repeat: Every"
"\Backup\csv-db","2026/10/22 1:00:00","Ready","Interactive/Background","2026/10/21 1:00:00","0","SYSTEM","C:\Backup\csv.bat","","# backup-monitor: csv-db","Enabled","Disabled","Stop On Battery Mode","SYSTEM","Daily","01:00:00","2026/10/01","N/A","Every 1 days","","Disabled"
`)
	tasks, err := ParseSchtasksCSV(data, defaultFilter())
	if err != nil {
		t.Fatalf("ParseSchtasksCSV returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1: %#v", len(tasks), tasks)
	}
	task := tasks[0]
	if task.Key != "csv-db" {
		t.Errorf("key = %q", task.Key)
	}
	if task.Name != "csv-db" {
		t.Errorf("name = %q", task.Name)
	}
	if task.ScheduleCron != "0 1 * * *" || task.Schedule != "每天 01:00" {
		t.Errorf("schedule = %q / %q", task.ScheduleCron, task.Schedule)
	}
	if !task.HasLastRun {
		t.Errorf("last run missing")
	}
	if task.LastRun.Hour() != 1 {
		t.Errorf("last run = %s", task.LastRun)
	}
}

func TestParseSchtasksCSVWithHostName(t *testing.T) {
	header := `"HostName","TaskName","Next Run Time","Status","Logon Mode","Last Run Time","Last Result","Author","Task To Run","Start In","Comment","Scheduled Task State","Idle Time","Power Management","Run As User","Delete Task If Not Rescheduled","Stop Task If Runs X Hours and X Mins","Schedule","Schedule Type","Start Time","Start Date","End Date","Days","Months","Repeat: Every","Repeat: Until: Time","Repeat: Until: Duration","Repeat: Stop If Still Running"`
	row := `"SUNFILE","\veeam-backup","2026/10/30 0:00:00","Ready","Interactive only","2026/10/29 0:00:00","0","SUNFILE\Administrator","C:\veeam-backup.bat","N/A","# backup-monitor: key=veeam-105-e name=veeam-105-e status=c:\backup-monitor\veeam-status.txt","Enabled","Disabled","Stop On Battery Mode, No Start On Batteries","SUNFILE\Administrator","Enabled","72:00:00","Scheduling data is not available in this format.","Weekly","0:00:00","2025/7/1","N/A","MON, TUE, WED, THU, FRI, SAT","Every 1 week(s)","Disabled","Disabled","Disabled","Disabled"`
	data := []byte(header + "\n" + row + "\n")
	tasks, err := ParseSchtasksCSV(data, defaultFilter())
	if err != nil {
		t.Fatalf("ParseSchtasksCSV returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1: %#v", len(tasks), tasks)
	}
	task := tasks[0]
	if task.Key != "veeam-105-e" || task.Name != "veeam-105-e" {
		t.Errorf("key/name = %q/%q", task.Key, task.Name)
	}
	if !strings.Contains(task.Command, "veeam-backup.bat") {
		t.Errorf("command = %q", task.Command)
	}
	if task.ScheduleCron == "" || task.Schedule == "" {
		t.Errorf("schedule = %q / %q", task.ScheduleCron, task.Schedule)
	}
	if task.Schedule != "每周一至周六 00:00" {
		t.Errorf("schedule text = %q", task.Schedule)
	}
	if !task.HasLastRun || task.LastRun.Hour() != 0 {
		t.Errorf("last run = %v / %s", task.HasLastRun, task.LastRun)
	}
}

func TestTaskFromWindowsSkipsMicrosoftPath(t *testing.T) {
	enabled := true
	record := windowsTask{
		TaskName: "rundll32",
		TaskPath: `\Microsoft\Windows\AppID`,
		Execute:  `C:\backup.bat`,
		State:    "Ready",
		Enabled:  &enabled,
	}
	if _, ok := taskFromWindows(record, defaultFilter()); ok {
		t.Fatal("microsoft task should be skipped")
	}
	record.TaskPath = `\Backup`
	if _, ok := taskFromWindows(record, defaultFilter()); !ok {
		t.Fatal("regular backup task should be kept")
	}
}
