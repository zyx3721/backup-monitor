// Package sysinfo 采集主机名、首选内网地址与操作系统名称。
package sysinfo

import (
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
)

// Hostname 返回当前主机名。
func Hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown-host"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "unknown-host"
	}
	return name
}

// Platform 返回协议约定的平台名称。
func Platform() string {
	if runtime.GOOS == "windows" {
		return "windows"
	}
	return "linux"
}

// Arch 返回当前二进制的 CPU 架构（amd64 / arm64）。
func Arch() string {
	return runtime.GOARCH
}

// OSName 返回可读的操作系统名称。
func OSName() string {
	if name := strings.TrimSpace(osName()); name != "" {
		return name
	}
	return runtime.GOOS
}

// OSBuild 返回 Windows 构建号（如 7601、19045），非 Windows 平台返回 0。
func OSBuild() int {
	return osBuild()
}

// PrivateIP 返回首选的非回环、非链路本地地址，找不到时返回回环地址。
func PrivateIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	sort.Slice(interfaces, func(i, j int) bool { return interfaces[i].Index < interfaces[j].Index })

	best := ""
	bestScore := 0
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, addrErr := iface.Addrs()
		if addrErr != nil {
			continue
		}
		for _, address := range addresses {
			ip := addressIP(address)
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast() {
				continue
			}
			score := 0
			if ip.To4() != nil {
				if ip.IsPrivate() {
					score = 4
				} else if ip.IsGlobalUnicast() {
					score = 3
				}
			} else if ip.IsGlobalUnicast() {
				score = 2
			}
			if score > bestScore {
				best = ip.String()
				bestScore = score
			}
		}
	}
	if best == "" {
		return "127.0.0.1"
	}
	return best
}

func addressIP(address net.Addr) net.IP {
	switch value := address.(type) {
	case *net.IPNet:
		return value.IP
	case *net.IPAddr:
		return value.IP
	default:
		host, _, err := net.SplitHostPort(address.String())
		if err != nil {
			return nil
		}
		return net.ParseIP(host)
	}
}
