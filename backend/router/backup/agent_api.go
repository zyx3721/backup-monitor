package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/repository"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
)

func timeNow() string {
	return appcommon.NowTimestamp()
}

func (a *Router) handleIngest(w http.ResponseWriter, r *http.Request) {
	if !a.authorizeAgent(r) {
		common.WriteError(w, http.StatusUnauthorized, "Agent 令牌无效")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	var report domain.IngestReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		common.WriteError(w, http.StatusBadRequest, "上报数据格式无效")
		return
	}
	if strings.TrimSpace(report.Host.Hostname) == "" || strings.TrimSpace(report.Host.IP) == "" {
		common.WriteError(w, http.StatusBadRequest, "上报数据无效：缺少 host 信息")
		return
	}
	result, err := a.svc.Ingest(r.Context(), report)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "处理上报数据失败")
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "hostId": result.HostID, "newExecutions": result.NewExecutions})
}

func (a *Router) handleAgentConfig(w http.ResponseWriter, r *http.Request) {
	if !a.authorizeAgent(r) {
		common.WriteError(w, http.StatusUnauthorized, "Agent 令牌无效")
		return
	}
	hostname := strings.TrimSpace(r.URL.Query().Get("hostname"))
	if hostname == "" {
		common.WriteError(w, http.StatusBadRequest, "缺少 hostname 参数")
		return
	}
	profile, err := a.svc.MatchProfile(hostname)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "匹配采集配置失败")
		return
	}
	settings := a.svc.Settings()
	common.WriteJSON(w, http.StatusOK, domain.AgentConfigResponse{
		Hostname: hostname, ServerTime: timeNow(), Settings: settings.AgentFacing(), Profile: profile,
	})
}

func (a *Router) handleAgentDownload(w http.ResponseWriter, r *http.Request) {
	if !a.authorizeAgent(r) {
		common.WriteError(w, http.StatusUnauthorized, "Agent 令牌无效")
		return
	}
	a.serveAgentBinary(w, r)
}

// handlePublicAgentDownload 免令牌分发 Agent 二进制，便于目标主机首次部署时直接拉取安装包。
func (a *Router) handlePublicAgentDownload(w http.ResponseWriter, r *http.Request) {
	a.serveAgentBinary(w, r)
}

// binaryTarget 描述下载或查询的目标槽位。
type binaryTarget struct {
	Platform string
	Arch     string
	Variant  string
	OSBuild  int
}

func parseOSBuild(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	build, err := strconv.Atoi(trimmed)
	if err != nil || build < 0 {
		return 0, fmt.Errorf("osBuild 参数无效")
	}
	return build, nil
}

// parseBinaryTarget 解析下载/查询参数：arch+variant 精确指定，否则按 osBuild 选包。
func parseBinaryTarget(r *http.Request) (binaryTarget, error) {
	platform, err := parsePlatform(r.URL.Query().Get("platform"))
	if err != nil {
		return binaryTarget{}, err
	}
	osBuild, err := parseOSBuild(r.URL.Query().Get("osBuild"))
	if err != nil {
		return binaryTarget{}, err
	}
	return binaryTarget{
		Platform: platform,
		Arch:     strings.ToLower(strings.TrimSpace(r.URL.Query().Get("arch"))),
		Variant:  strings.ToLower(strings.TrimSpace(r.URL.Query().Get("variant"))),
		OSBuild:  osBuild,
	}, nil
}

// resolveAgentBinary 定位槽位：精确参数优先，否则按主机信息自动选包。
func (a *Router) resolveAgentBinary(target binaryTarget) (*repository.AgentBinaryMeta, string, error) {
	if target.Arch != "" && target.Variant != "" {
		meta, err := a.svc.AgentBinaryMeta(target.Platform, target.Arch, target.Variant)
		if err != nil {
			return nil, "", err
		}
		if meta == nil {
			return nil, fmt.Sprintf("尚未托管 %s/%s/%s 安装包", platformLabel(target.Platform), target.Arch, target.Variant), nil
		}
		return meta, "", nil
	}
	meta, _, reason, err := a.svc.SelectAgentBinary(target.Platform, target.Arch, target.OSBuild)
	return meta, reason, err
}

func (a *Router) serveAgentBinary(w http.ResponseWriter, r *http.Request) {
	target, err := parseBinaryTarget(r)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	meta, reason, err := a.resolveAgentBinary(target)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "读取 Agent 二进制失败")
		return
	}
	if meta == nil {
		if reason == "" {
			reason = "尚未托管匹配的 Agent 二进制"
		}
		common.WriteError(w, http.StatusNotFound, reason)
		return
	}
	path := a.svc.AgentBinaryPath(meta.Platform, meta.Arch, meta.Variant)
	file, err := os.Open(path)
	if err != nil {
		common.WriteError(w, http.StatusNotFound, "二进制文件不存在")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "读取二进制失败")
		return
	}
	name := agentBinaryFileName(meta.Platform)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.Header().Set("X-Agent-Binary-Version", meta.Version)
	w.Header().Set("X-Agent-Binary-Sha256", meta.Sha256)
	w.Header().Set("X-Agent-Binary-Variant", meta.Variant)
	w.Header().Set("X-Agent-Binary-Arch", meta.Arch)
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func agentBinaryFileName(platform string) string {
	if platform == "windows" {
		return "backup-agent-windows.exe"
	}
	return fmt.Sprintf("backup-agent-%s", platform)
}

func (a *Router) handleAgentBinaryLatest(w http.ResponseWriter, r *http.Request) {
	if !a.authorizeAgent(r) {
		common.WriteError(w, http.StatusUnauthorized, "Agent 令牌无效")
		return
	}
	target, err := parseBinaryTarget(r)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	meta, reason, err := a.resolveAgentBinary(target)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "读取 Agent 二进制失败")
		return
	}
	if meta == nil {
		if reason == "" {
			reason = "尚未托管匹配的 Agent 二进制"
		}
		common.WriteError(w, http.StatusNotFound, reason)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{
		"version": meta.Version, "sha256": meta.Sha256, "size": meta.Size,
		"arch": meta.Arch, "variant": meta.Variant, "goVersion": meta.GoVersion,
	})
}

func parsePlatform(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "linux", "windows":
		return trimmed, nil
	case "":
		return "", fmt.Errorf("缺少 platform 参数，仅支持 linux 或 windows")
	default:
		return "", fmt.Errorf("不支持的平台：%s，仅支持 linux 或 windows", strings.TrimSpace(value))
	}
}

// hostPlatform 返回主机平台；未知平台返回空字符串，不参与按平台推送
func hostPlatform(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	default:
		return ""
	}
}

// handleUploadAgentBinary 接收管理端上传的 Agent 二进制。
func (a *Router) handleUploadAgentBinary(w http.ResponseWriter, r *http.Request) {
	platform, err := parsePlatform(r.URL.Query().Get("platform"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	variant := strings.TrimSpace(r.URL.Query().Get("variant"))
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	r.Body = http.MaxBytesReader(w, r.Body, 128<<20)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "读取上传内容失败或超过 128MB")
		return
	}
	meta, err := a.svc.SaveAgentBinary(platform, variant, version, data)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := agentBinaryFileName(platform)
	detail := agentUploadAuditDetail(name, platform, meta.Arch, meta.Variant, meta.GoVersion, meta.Size, meta.Sha256)
	a.recordAudit(r, service.AuditModuleAgents, "上传 Agent 二进制", "Agent "+meta.Version, detail, service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, meta)
}

func platformLabel(platform string) string {
	if platform == "windows" {
		return "Windows"
	}
	return "Linux"
}

type agentTargetCount struct {
	Total  int `json:"total"`
	Online int `json:"online"`
}

type agentTargets struct {
	Linux   agentTargetCount `json:"linux"`
	Windows agentTargetCount `json:"windows"`
	Unknown int              `json:"unknown"`
}

func (a *Router) platformTargetCounts() (agentTargets, error) {
	hosts, err := a.svc.Repo().AllHostsForPush()
	if err != nil {
		return agentTargets{}, err
	}
	var targets agentTargets
	for _, host := range hosts {
		switch hostPlatform(host.Platform) {
		case "linux":
			targets.Linux.Total++
			if a.hub != nil && a.hub.IsConnected(host.ID) {
				targets.Linux.Online++
			}
		case "windows":
			targets.Windows.Total++
			if a.hub != nil && a.hub.IsConnected(host.ID) {
				targets.Windows.Online++
			}
		default:
			targets.Unknown++
		}
	}
	return targets, nil
}

// handleAgentBinaries 返回各平台托管的二进制元数据与推送目标统计。
func (a *Router) handleAgentBinaries(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListAgentBinaries()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	targets, err := a.platformTargetCounts()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "targets": targets})
}

type pushResult struct {
	Hostname       string `json:"hostname"`
	IP             string `json:"ip"`
	AgentVersion   string `json:"agentVersion"`
	AgentStartedAt string `json:"agentStartedAt,omitempty"`
	LastReportTime string `json:"lastReportTime"`
	Version        string `json:"version,omitempty"`
	Arch           string `json:"arch,omitempty"`
	Variant        string `json:"variant,omitempty"`
	OSBuild        int    `json:"osBuild,omitempty"`
	Delivered      bool   `json:"delivered"`
	OK             bool   `json:"ok"`
	Message        string `json:"message"`
	Skipped        bool   `json:"skipped"`
	Reason         string `json:"reason,omitempty"`
	Warning        string `json:"warning,omitempty"`
}

// handlePushUpdate 按主机架构与系统版本逐台选包并推送自更新。
func (a *Router) handlePushUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		All      bool   `json:"all"`
		IPs      string `json:"ips"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	platform, err := parsePlatform(body.Platform)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	binaries, err := a.svc.ListAgentBinaries()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	hasPlatformBinary := false
	for _, item := range binaries {
		if item.Platform == platform {
			hasPlatformBinary = true
			break
		}
	}
	if !hasPlatformBinary {
		common.WriteError(w, http.StatusBadRequest, "尚未上传该平台的 Agent 二进制，请先在「Agent 更新」中上传")
		return
	}
	var ranges []ipRange
	if strings.TrimSpace(body.IPs) != "" {
		ranges, err = parseIPFilters(body.IPs)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "IP 条件无效。支持单 IP、多 IP、CIDR 网段与 IP 范围")
			return
		}
	} else if !body.All {
		common.WriteError(w, http.StatusBadRequest, "请输入目标主机 IP，或选择全部主机")
		return
	}
	hosts, err := a.svc.Repo().AllHostsForPush()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type pushTarget struct {
		host    domain.Host
		meta    *repository.AgentBinaryMeta
		warning string
		reason  string
	}
	targets := make([]pushTarget, 0, len(hosts))
	for _, host := range hosts {
		if hostPlatform(host.Platform) != platform {
			continue
		}
		if len(ranges) > 0 && !matchIPFilters(host.IP, ranges) {
			continue
		}
		meta, warning, reason := service.SelectAgentBinaryFrom(platform, host.Arch, host.OSBuild, binaries)
		targets = append(targets, pushTarget{host: host, meta: meta, warning: warning, reason: reason})
	}
	if len(targets) == 0 {
		common.WriteError(w, http.StatusNotFound, fmt.Sprintf("没有匹配条件的主机：所选平台为 %s，平台未知的主机不参与按平台推送", platform))
		return
	}
	results := make([]pushResult, len(targets))
	var wg sync.WaitGroup
	for i, target := range targets {
		base := pushResult{Hostname: target.host.Hostname, IP: target.host.IP, AgentVersion: target.host.AgentVersion,
			AgentStartedAt: target.host.AgentStartedAt,
			LastReportTime: target.host.LastReportTime, Arch: target.host.Arch, OSBuild: target.host.OSBuild}
		if target.meta == nil {
			base.Skipped = true
			base.Message = target.reason
			base.Reason = target.reason
			results[i] = base
			continue
		}
		wg.Add(1)
		go func(index int, item pushTarget, result pushResult) {
			defer wg.Done()
			downloadURL := fmt.Sprintf("/api/agent/download?platform=%s&arch=%s&variant=%s",
				platform, item.meta.Arch, item.meta.Variant)
			delivered, ok, message := a.hub.SendUpdate(item.host.ID, item.meta.Version, item.meta.Sha256, downloadURL)
			result.Version = item.meta.Version
			result.Arch = item.meta.Arch
			result.Variant = item.meta.Variant
			result.Delivered = delivered
			result.OK = ok
			result.Message = message
			result.Warning = item.warning
			results[index] = result
		}(i, target, base)
	}
	wg.Wait()
	success := 0
	deliveredCount := 0
	skipped := 0
	successIPs := make([]string, 0, len(results))
	failedIPs := make([]string, 0, len(results))
	skippedIPs := make([]string, 0, len(results))
	versions := make([]string, 0, 2)
	seenVersion := map[string]bool{}
	for _, item := range results {
		if item.Delivered {
			deliveredCount++
		}
		if item.Version != "" && !seenVersion[item.Version] {
			seenVersion[item.Version] = true
			versions = append(versions, item.Version)
		}
		switch {
		case item.Skipped:
			skipped++
			skippedIPs = append(skippedIPs, firstNonEmpty(item.IP, item.Hostname))
		case item.Delivered && item.OK:
			success++
			successIPs = append(successIPs, firstNonEmpty(item.IP, item.Hostname))
		default:
			failedIPs = append(failedIPs, firstNonEmpty(item.IP, item.Hostname))
		}
	}
	detail := agentPushAuditDetail(platform, versions, successIPs, failedIPs, skippedIPs)
	auditTarget := "Agent 更新"
	if len(versions) == 1 {
		auditTarget = "Agent " + versions[0]
	}
	a.recordAudit(r, service.AuditModuleAgents, "推送 Agent 二进制", auditTarget, detail, service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{
		"platform": platform, "results": results,
		"summary": map[string]int{
			"total": len(results), "delivered": deliveredCount, "ok": success, "failed": len(failedIPs), "skipped": skipped,
		},
	})
}

type ipRange struct {
	start uint32
	end   uint32
}

func parseIPFilters(input string) ([]ipRange, error) {
	items := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	})
	if len(items) == 0 {
		return nil, fmt.Errorf("empty")
	}
	out := make([]ipRange, 0, len(items))
	for _, item := range items {
		switch {
		case strings.Contains(item, "/"):
			parts := strings.Split(item, "/")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid cidr")
			}
			base, err := ipToUint(parts[0])
			if err != nil {
				return nil, err
			}
			mask, err := strconv.Atoi(parts[1])
			if err != nil || mask < 0 || mask > 32 {
				return nil, fmt.Errorf("invalid mask")
			}
			size := uint32(1) << (32 - mask)
			network := base
			if mask > 0 {
				network = base &^ (size - 1)
			} else {
				network = 0
			}
			out = append(out, ipRange{start: network, end: network + size - 1})
		case strings.Contains(item, "-"):
			parts := strings.SplitN(item, "-", 2)
			start, err := ipToUint(parts[0])
			if err != nil {
				return nil, err
			}
			endText := parts[1]
			var end uint32
			if !strings.Contains(endText, ".") {
				prefix := parts[0][:strings.LastIndex(parts[0], ".")+1]
				end, err = ipToUint(prefix + endText)
			} else {
				end, err = ipToUint(endText)
			}
			if err != nil || end < start {
				return nil, fmt.Errorf("invalid range")
			}
			out = append(out, ipRange{start: start, end: end})
		default:
			value, err := ipToUint(item)
			if err != nil {
				return nil, err
			}
			out = append(out, ipRange{start: value, end: value})
		}
	}
	return out, nil
}

func ipToUint(value string) (uint32, error) {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil || ip.To4() == nil {
		return 0, fmt.Errorf("invalid ip")
	}
	bytes := ip.To4()
	return uint32(bytes[0])<<24 | uint32(bytes[1])<<16 | uint32(bytes[2])<<8 | uint32(bytes[3]), nil
}

func matchIPFilters(value string, ranges []ipRange) bool {
	ip, err := ipToUint(value)
	if err != nil {
		return false
	}
	for _, r := range ranges {
		if ip >= r.start && ip <= r.end {
			return true
		}
	}
	return false
}
