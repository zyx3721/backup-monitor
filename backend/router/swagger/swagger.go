// Package swagger 集中存放 OpenAPI 注解、文档模型与 Swagger UI 处理器。
package swagger

import (
	"net/http"

	_ "backup-monitor-backend/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Handler 返回挂载在 /swagger/* 的 Swagger UI 处理器
func Handler() http.Handler {
	return httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json"))
}
