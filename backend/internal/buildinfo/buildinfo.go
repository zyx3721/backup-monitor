// Package buildinfo 承载编译期通过 -ldflags -X 注入的版本构建信息
package buildinfo

// Version 语义化版本号，由 CI 在打 tag 构建时注入，默认 dev
var Version = "dev"

// Commit 构建 commit 短哈希，由 CI 注入，默认 unknown
var Commit = "unknown"

// BuildDate 构建时间（Asia/Shanghai），由 CI 注入，默认 unknown
var BuildDate = "unknown"
