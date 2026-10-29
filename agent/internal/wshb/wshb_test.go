package wshb

import "testing"

func TestDrainWake(t *testing.T) {
	wake := make(chan struct{}, 1)
	wake <- struct{}{}
	client := &Client{options: Options{Wake: wake}}
	client.drainWake()
	select {
	case <-wake:
		t.Fatal("wake signal was not drained")
	default:
	}
}
