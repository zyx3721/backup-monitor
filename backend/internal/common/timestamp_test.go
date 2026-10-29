package common

import (
	"testing"
	"time"
)

func TestFormatTimestampUsesMillisecondUTC(t *testing.T) {
	value := time.Date(2026, 10, 22, 6, 39, 34, 691123456, time.UTC)
	if got := FormatTimestamp(value); got != "2026-10-22T06:39:34.691Z" {
		t.Fatalf("format = %q", got)
	}
}

func TestParseTimestampRoundTrip(t *testing.T) {
	value := "2026-10-22T06:39:34.691Z"
	parsed, err := ParseTimestamp(value)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := FormatTimestamp(parsed); got != value {
		t.Fatalf("round trip = %q, want %q", got, value)
	}
	if got := TimestampUnix(value); got != parsed.Unix() {
		t.Fatalf("unix = %d, want %d", got, parsed.Unix())
	}
}

func TestParseTimestampAcceptsRFC3339(t *testing.T) {
	for _, value := range []string{"2026-10-22T06:39:34Z", "2026-10-22T06:39:34.691123456Z"} {
		if _, err := ParseTimestamp(value); err != nil {
			t.Fatalf("parse %q: %v", value, err)
		}
	}
	if _, err := ParseTimestamp(""); err == nil {
		t.Fatal("empty timestamp should fail")
	}
	if got := TimestampUnix("not-a-time"); got != 0 {
		t.Fatalf("invalid unix = %d, want 0", got)
	}
}
