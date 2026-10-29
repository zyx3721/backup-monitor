package veeam

import "testing"

const sampleJobInfo = `Backup job
  ID: {8d2b779c-2717-4f7a-aaf2-333ec748ffc2}
  Name: website
  Type: Volume level
  Repository name: veeam
  Compression: Zstd3
  Max points: 30
  Schedule:
     Run job automatically: Enabled
     Daily at: 06:00
  Active full backup:
     Monthly: day 1
  Objects for backup:
     Include Disk: /dev/sda
     Include Disk: /dev/sdc
`

func TestParseJobInfo(t *testing.T) {
	info := parseJobInfo(sampleJobInfo)
	if info.Repository != "veeam" {
		t.Errorf("repository = %q", info.Repository)
	}
	if info.Compression != "Zstd3" {
		t.Errorf("compression = %q", info.Compression)
	}
	if info.Retention != "30" {
		t.Errorf("retention = %q", info.Retention)
	}
	if info.Schedule != "每天 06:00" {
		t.Errorf("schedule = %q", info.Schedule)
	}
	if info.ActiveFull != "每月 1 日" {
		t.Errorf("active full = %q", info.ActiveFull)
	}
	if len(info.Objects) != 2 || info.Objects[0] != "/dev/sda" || info.Objects[1] != "/dev/sdc" {
		t.Errorf("objects = %v", info.Objects)
	}
}

func TestParseJobInfoWeekly(t *testing.T) {
	output := "Backup job\n  Name: weekly\n  Repository: repo1\n  Schedule:\n     Weekly: Saturday at 03:30\n"
	info := parseJobInfo(output)
	if info.Schedule != "每周六 03:30" {
		t.Errorf("schedule = %q", info.Schedule)
	}
}

func TestParseJobInfoDisabledSchedule(t *testing.T) {
	output := "Backup job\n  Name: website\n  Schedule:\n     Run job automatically: Disabled\n  Active full backup: Monthly, day 1\n"
	info := parseJobInfo(output)
	if info.Schedule != "" {
		t.Errorf("schedule = %q, want empty", info.Schedule)
	}
	if info.ActiveFull != "每月 1 日" {
		t.Errorf("active full = %q", info.ActiveFull)
	}
}

func TestParseJobInfoFrequency(t *testing.T) {
	output := `Backup job
  Name: website
  Schedule:
     Frequency: Every day
     Time: 06:00
  Active full backup:
     Frequency: Every month
     Day: 1
`
	info := parseJobInfo(output)
	if info.Schedule != "每天 06:00" {
		t.Errorf("schedule = %q", info.Schedule)
	}
	if info.ActiveFull != "每月 1 日" {
		t.Errorf("active full = %q", info.ActiveFull)
	}
}

func TestParseJobInfoFullBackupScheduleHeading(t *testing.T) {
	output := "Backup job\n  Name: website\n  Full backup schedule:\n     Frequency: Every month\n     Day of month: 15\n"
	info := parseJobInfo(output)
	if info.ActiveFull != "每月 15 日" {
		t.Errorf("active full = %q", info.ActiveFull)
	}
}

func TestParseJobInfoRealOutput(t *testing.T) {
	output := `ID: {8d2b779c-2717-4f7a-aaf2-333ec748ffc2}
Name: website
Repository ID: {36764df4-85a0-416f-b80a-ee5204887d9e}
Repository name: veeam
Creation time: 2026-08-28 13:08:25
Options:
   Compression: Zstd3
   Max points: 30
   Block size: 1024 KB
   File system indexing is disabled
   Retries are allowed
   Snapshot required
Objects for backup:
   Include Disk: /dev/sda
   Include Disk: /dev/sdc
Schedule: 
   Every day
   At: 06:00
Active full: Every 1 day of every month.
Health check: 
   Every first Mon of every month
   disabled
Repository immutability: 
   disabled
`
	info := parseJobInfo(output)
	if info.Repository != "veeam" || info.Compression != "Zstd3" || info.Retention != "30" {
		t.Errorf("basic fields = %+v", info)
	}
	if info.Schedule != "每天 06:00" {
		t.Errorf("schedule = %q", info.Schedule)
	}
	if info.ActiveFull != "每月 1 日" {
		t.Errorf("active full = %q", info.ActiveFull)
	}
	if len(info.Objects) != 2 || info.Objects[0] != "/dev/sda" {
		t.Errorf("objects = %v", info.Objects)
	}
}
