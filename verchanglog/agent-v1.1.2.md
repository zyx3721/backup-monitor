# backup-monitor Agent agent-v1.1.2 版本更新日志

**发布日期**: 2026-11-09  
**提交数量**: 4 次提交  
**版本类型**: 修复版本

---

## 📋 版本概述

本版本修复 Windows 批处理备份任务在实际成功后被短暂误判为「执行中断（缺少结束时间）」并产生“备份失败”告警的问题，Windows 进程探测口径与原项目 Agent 1.2.32 对齐。

`agent-v1.1.1` 已经恢复 Linux 上的进程关键字三级回退与探测可用性语义，但 Windows 仍使用 `tasklist`，只能看到 `cmd.exe` 等进程映像名。批处理任务的实际命令行是 `cmd.exe /c C:\veeam-backup.bat`，脚本名 `veeam-backup.bat` 无法命中，导致状态文件只有 `start`、没有 `end` 时立即判为失败，随后成功上报又触发告警恢复。

原项目 Windows Agent 使用完整命令行探测：优先 `wmic process get CommandLine`，失败回退 PowerShell `Get-WmiObject Win32_Process`。本版本按该逻辑实现，界面展示的状态值、消息文案、会话 ID 与接口字段保持不变。

---

## ✨ 新增特性

本版本未新增采集来源、协议字段或配置项。

Windows 进程探测行为调整：

- 主用 `wmic process get CommandLine` 读取本机进程完整命令行；
- `wmic` 失败或输出为空时，回退 `powershell -NoProfile -NonInteractive -Command "Get-WmiObject Win32_Process | ForEach-Object { $_.CommandLine }"`；
- 两条命令都失败或输出为空时返回 `available=false`，进入原有 15 分钟兜底窗口；
- 命令成功时按行解析输出，继续使用原有大小写不敏感子串匹配。

---

## 🐛 Bug 修复

- 修复 Windows 批处理任务因 `tasklist` 只返回进程映像名而被误判为「执行中断（缺少结束时间）」的问题
- 修复 `veeam-31-d`、`veeam-105-e` 等批处理任务在备份实际成功后短暂产生“备份失败”告警、随后又自动恢复的问题
- 修复 `wmic` 不可用或输出为空时没有回退 PowerShell 的问题

---

## 🔧 重构优化

- `agent/internal/proc/proc_windows.go` 移除 `tasklist`，改为 `wmic` + PowerShell 两级完整命令行探测
- 新增 `agent/internal/proc/proc_windows_test.go`，覆盖 wmic 命中、PowerShell 回退、空输出回退、两条命令失败、CRLF/BOM 解析与大小写混合匹配
- 同步 `agent/README.md` 与 `agent/README.en.md` 的 Windows 进程探测说明
- 状态值、消息文案、`session_id` 规则、API 字段与前端展示均未修改

---

## 🔄 升级说明

**建议所有 Windows 主机升级**，尤其是通过计划任务运行 `.bat` / `.cmd` 批处理备份的主机。

- 本版本只修改 Agent 本地进程探测，不涉及上报协议字段，兼容现有服务端；服务端无需同步升级。
- 升级方式：管理端「Agent 管理 → Agent 更新」上传对应平台二进制后推送；或停止 Agent 服务后手工覆盖二进制。
- Windows 7 / Server 2008 R2 使用 `*_windows_amd64_legacy.zip`，只能停服务后手工覆盖，不要上传管理端。
- 已经产生的历史误报告警不会被自动改写；升级后新的采集轮次不再产生同类误报。

---

## 👥 贡献者

- Jerion

---

**完整提交记录**: 4 次提交  
**开发周期**: 2026-11-09 至 2026-11-09 (1 天)
