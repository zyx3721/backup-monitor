// Package common 存放跨模块复用的轻量原生工具函数。
package common

import (
	"fmt"
	"strconv"
	"strings"
)

// AsString 将任意查询结果值归一为去空格字符串
func AsString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

// AsInt64 将任意查询结果值归一为 int64
func AsInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprintf("%v", v)), 10, 64)
		return n
	}
}
