package statusfile

import (
	"strings"
	"testing"
	"time"
)

const sample = `# backup status
start=2026-10-20 15:00:01
end: 2026-10-20 15:08:35
size=1.2 GB
transferred=1.1 GB
exit=0
status=SUCCESS
`

func TestParse(t *testing.T) {
	info, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	wantStart := time.Date(2026, 10, 20, 15, 0, 1, 0, time.Local)
	if !info.HasStart || !info.Start.Equal(wantStart) {
		t.Fatalf("start = %s (has=%v), want %s", info.Start, info.HasStart, wantStart)
	}
	wantEnd := time.Date(2026, 10, 20, 15, 8, 35, 0, time.Local)
	if !info.HasEnd || !info.End.Equal(wantEnd) {
		t.Fatalf("end = %s (has=%v), want %s", info.End, info.HasEnd, wantEnd)
	}
	if !info.HasSize || info.SizeBytes != 1288490189 {
		t.Fatalf("size = %d (has=%v)", info.SizeBytes, info.HasSize)
	}
	if !info.HasTransferred || info.TransferredBytes != 1181116006 {
		t.Fatalf("transferred = %d (has=%v)", info.TransferredBytes, info.HasTransferred)
	}
	if !info.HasExit || info.ExitCode != 0 {
		t.Fatalf("exit = %d (has=%v)", info.ExitCode, info.HasExit)
	}
	if info.Status != "success" {
		t.Fatalf("status = %q", info.Status)
	}
}

func TestParseTolerant(t *testing.T) {
	input := "\nunknown line\nfoo=bar\nexit=0x1F\nstatus=failed\n"
	info, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !info.HasExit || info.ExitCode != 31 {
		t.Fatalf("exit = %d (has=%v)", info.ExitCode, info.HasExit)
	}
	if info.Status != "failed" {
		t.Fatalf("status = %q", info.Status)
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"1024":      1024,
		"2 KB":      2048,
		"1.5MB":     1572864,
		"1,5 GB":    1610612736,
		"1,234 KB":  1263616,
		"3 gb":      3221225472,
		"2TB":       2199023255552,
		"512 bytes": 512,
		"1.25 GiB":  1342177280,
	}
	for input, want := range cases {
		got, err := ParseSize(input)
		if err != nil {
			t.Fatalf("ParseSize(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Errorf("ParseSize(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseSizeInvalid(t *testing.T) {
	for _, input := range []string{"", "abc", "-1 MB", "1 PB"} {
		if _, err := ParseSize(input); err == nil {
			t.Fatalf("ParseSize(%q) expected error", input)
		}
	}
}

func TestFormatSize(t *testing.T) {
	cases := map[int64]string{
		0:             "0 B",
		999:           "999 B",
		2048:          "2.0 KB",
		1536:          "1.5 KB",
		1024 * 1024:   "1.0 MB",
		1288490189:    "1.2 GB",
		2199023255552: "2.0 TB",
	}
	for input, want := range cases {
		if got := FormatSize(input); got != want {
			t.Errorf("FormatSize(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestParseJSON(t *testing.T) {
	input := `{"start":"2026-10-26 00:10:01","end":"2026-10-26 00:10:08","status":"success","size":"1.2 GB"}`
	info, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !info.HasStart || !info.HasEnd || info.Status != "success" || !info.HasSize {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestParseSpacedKeys(t *testing.T) {
	input := "Start Time : 2026-10-26 00:10:01\nEnd Time: 2026-10-26 00:10:08\nStatus: SUCCESS\n"
	info, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !info.HasStart || !info.HasEnd || info.Status != "success" {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestParseFreeformLines(t *testing.T) {
	input := "2026-10-26 00:10:01 backup start\n2026-10-26 00:10:08 backup finished\n"
	info, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !info.HasStart || !info.HasEnd {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.Status != "success" {
		t.Fatalf("status = %q", info.Status)
	}
}
