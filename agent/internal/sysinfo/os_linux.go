//go:build linux

package sysinfo

import (
	"os"
	"strings"
)

func osBuild() int {
	return 0
}

func osName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	if name := values["PRETTY_NAME"]; name != "" {
		return name
	}
	return values["NAME"]
}
