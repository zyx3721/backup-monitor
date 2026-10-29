//go:build windows

package proc

import (
	"context"
	"encoding/csv"
	"os/exec"
	"strings"
	"time"
)

func commands() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "tasklist.exe", "/fo", "csv", "/nh").Output()
	if err != nil {
		return nil
	}
	reader := csv.NewReader(strings.NewReader(string(output)))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil
	}
	list := make([]string, 0, len(records))
	for _, record := range records {
		if len(record) == 0 {
			continue
		}
		if name := strings.TrimSpace(record[0]); name != "" {
			list = append(list, name)
		}
	}
	return list
}
