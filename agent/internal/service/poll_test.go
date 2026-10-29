package service

import (
	"errors"
	"testing"
	"time"
)

func TestWaitForStatus(t *testing.T) {
	calls := 0
	query := func() (uint32, error) {
		calls++
		if calls < 3 {
			return 2, nil
		}
		return 4, nil
	}
	if err := waitForStatus(query, 4, time.Second); err != nil {
		t.Fatalf("waitForStatus = %v, want nil", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}

	if err := waitForStatus(func() (uint32, error) { return 1, nil }, 4, 60*time.Millisecond); err == nil {
		t.Fatal("waitForStatus = nil, want timeout error")
	}

	sentinel := errors.New("query failed")
	if err := waitForStatus(func() (uint32, error) { return 0, sentinel }, 4, time.Second); !errors.Is(err, sentinel) {
		t.Fatalf("waitForStatus = %v, want sentinel", err)
	}
}
