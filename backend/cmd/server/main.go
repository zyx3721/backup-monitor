// backup-monitor 后端服务入口：解析命令行参数、加载配置并启动 HTTP 服务
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"backup-monitor-backend/config"
	"backup-monitor-backend/internal/buildinfo"
	"backup-monitor-backend/router"
)

// flagValues 汇总命令行参数的显式取值，零值表示未设置
type flagValues struct {
	envPath     string
	addr        string
	port        string
	dbPath      string
	jwtSecret   string
	sessionTTL  int
	corsOrigins string
}

// versionText 组装 -v/-version 输出的版本信息文本
func versionText() string {
	return fmt.Sprintf("backup-monitor %s\ncommit: %s\nbuild: %s\ngo: %s\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate, runtime.Version())
}

// setUsage 自定义 -h/--help 输出：-v 与 -version 合并一行，各参数描述统一换行缩进对齐
func setUsage() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s:\n", os.Args[0])
		flag.VisitAll(func(f *flag.Flag) {
			if f.Name == "version" {
				return
			}
			if f.Name == "v" {
				fmt.Fprintf(w, "  -v, -version\n    \t%s\n", f.Usage)
				return
			}
			name, usage := flag.UnquoteUsage(f)
			fmt.Fprintf(w, "  -%s %s\n    \t%s (default %q)\n", f.Name, name, usage, f.DefValue)
		})
	}
}

// overridesFromFlags 将命令行参数显式值映射为配置加载的覆盖键值
func overridesFromFlags(v flagValues) map[string]string {
	overrides := make(map[string]string)
	if v.envPath != "" {
		overrides["env"] = v.envPath
	}
	if v.addr != "" {
		overrides["addr"] = v.addr
	}
	if v.port != "" {
		overrides["port"] = v.port
	}
	if v.dbPath != "" {
		overrides["db"] = v.dbPath
	}
	if v.jwtSecret != "" {
		overrides["jwt_secret"] = v.jwtSecret
	}
	if v.sessionTTL > 0 {
		overrides["session_ttl"] = strconv.Itoa(v.sessionTTL)
	}
	if v.corsOrigins != "" {
		overrides["cors_origins"] = v.corsOrigins
	}
	return overrides
}

// @title Backup Monitor API
// @version 1.0.0
// @description Backup Monitor 通用前后端模板后端 API 文档。除登录、公开品牌与健康检查外，接口默认需要 Bearer Token。
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	showVersion := flag.Bool("v", false, "显示版本信息并退出")
	flag.BoolVar(showVersion, "version", false, "显示版本信息并退出")
	envPath := flag.String("env", "", ".env 配置文件路径，默认按 ./.env、backend/.env、可执行文件同目录顺序查找")
	addr := flag.String("addr", "", "后端监听地址，等价环境变量 SERVER_ADDR")
	port := flag.String("port", "", "后端监听端口，未设置 -addr 时生效，等价环境变量 PORT")
	dbPath := flag.String("db", "", "SQLite 数据库路径，等价环境变量 DB_PATH")
	jwtSecret := flag.String("jwt-secret", "", "JWT 签名密钥，等价环境变量 JWT_SECRET")
	sessionTTL := flag.Int("session-ttl", 0, "登录会话有效期（小时），等价环境变量 SESSION_TTL_HOURS")
	corsOrigins := flag.String("cors-origins", "", "允许的跨域来源，多个用逗号分隔，等价环境变量 CORS_ORIGINS")
	setUsage()
	flag.Parse()

	if *showVersion {
		fmt.Print(versionText())
		return
	}

	overrides := overridesFromFlags(flagValues{
		envPath:     *envPath,
		addr:        *addr,
		port:        *port,
		dbPath:      *dbPath,
		jwtSecret:   *jwtSecret,
		sessionTTL:  *sessionTTL,
		corsOrigins: *corsOrigins,
	})
	router.RunWithConfig(config.LoadWithOverrides(overrides))
}
