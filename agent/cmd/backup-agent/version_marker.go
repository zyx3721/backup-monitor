package main

// embeddedVersion 保留可被服务端扫描的版本字符串，发布新版本时与 config.Version 同步更新。
var embeddedVersion = "backup-agent-version:1.0.0"

func init() {
	if embeddedVersion == "" {
		panic("missing embedded version marker")
	}
}
