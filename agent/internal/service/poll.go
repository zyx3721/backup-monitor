package service

import (
	"fmt"
	"time"
)

// waitForStatus 轮询服务状态直至匹配或超时
func waitForStatus(query func() (uint32, error), desired uint32, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := query()
		if err != nil {
			return fmt.Errorf("query service status: %w", err)
		}
		if status == desired {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("service did not reach state %d within %s", desired, timeout)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
