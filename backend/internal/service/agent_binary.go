package service

import (
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/repository"
)

const (
	// VariantStandard 标准包，由 Go 1.21 及以上编译，面向受支持系统。
	VariantStandard = "standard"
	// VariantLegacy 兼容包，由 Go 1.20 及以下编译，面向 Windows 7 / Server 2008 R2。
	VariantLegacy = "legacy"

	archAMD64 = "amd64"
	archARM64 = "arm64"

	// windowsBuildLegacy Windows 7 / Server 2008 R2 起始构建号。
	windowsBuildLegacy = 7600
	// windowsBuildModern Windows 10 / Server 2016 起始构建号。
	windowsBuildModern = 10240
)

var embeddedVersionPattern = regexp.MustCompile(`backup-agent-version:(\d+\.\d+\.\d+)`)

// AgentBinaryPath 返回指定槽位的托管路径。
func (s *BackupService) AgentBinaryPath(platform, arch, variant string) string {
	name := "backup-agent"
	if platform == "windows" {
		name = "backup-agent.exe"
	}
	return filepath.Join(s.dataDir, "agents", platform, arch, variant, name)
}

// AgentBinaryMeta 读取指定槽位的托管元数据。
func (s *BackupService) AgentBinaryMeta(platform, arch, variant string) (*repository.AgentBinaryMeta, error) {
	meta, err := s.repo.GetAgentBinaryMeta(platform, normalizeArch(arch), normalizeVariant(variant))
	if err != nil || meta == nil {
		return nil, err
	}
	if _, err := os.Stat(s.AgentBinaryPath(meta.Platform, meta.Arch, meta.Variant)); err != nil {
		return nil, nil
	}
	return meta, nil
}

// ListAgentBinaries 返回全部已落盘槽位的托管元数据。
func (s *BackupService) ListAgentBinaries() ([]repository.AgentBinaryMeta, error) {
	items, err := s.repo.ListAgentBinaryMeta()
	if err != nil {
		return nil, err
	}
	available := make([]repository.AgentBinaryMeta, 0, len(items))
	for _, item := range items {
		if _, err := os.Stat(s.AgentBinaryPath(item.Platform, item.Arch, item.Variant)); err == nil {
			available = append(available, item)
		}
	}
	return available, nil
}

// SelectAgentBinary 按主机平台、架构与系统版本选择安装包。
func (s *BackupService) SelectAgentBinary(platform, arch string, osBuild int) (*repository.AgentBinaryMeta, string, string, error) {
	items, err := s.ListAgentBinaries()
	if err != nil {
		return nil, "", "", err
	}
	meta, warning, reason := selectAgentBinary(platform, arch, osBuild, items)
	return meta, warning, reason, nil
}

// SelectAgentBinaryFrom 使用已加载的槽位列表选包，供批量推送复用。
func SelectAgentBinaryFrom(platform, arch string, osBuild int, items []repository.AgentBinaryMeta) (*repository.AgentBinaryMeta, string, string) {
	return selectAgentBinary(platform, arch, osBuild, items)
}

// selectAgentBinary 从候选槽位中选包，返回选中的包、未校验提示与跳过原因。
func selectAgentBinary(platform, arch string, osBuild int, items []repository.AgentBinaryMeta) (*repository.AgentBinaryMeta, string, string) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform != "linux" && platform != "windows" {
		return nil, "", "不支持的平台：" + platform
	}
	targetArch := normalizeArch(arch)
	warnings := make([]string, 0, 2)
	if targetArch == "" {
		targetArch = archAMD64
		warnings = append(warnings, "主机未上报架构，按 amd64 选择")
	}
	candidates := make([]repository.AgentBinaryMeta, 0, 2)
	for _, item := range items {
		if item.Platform == platform && item.Arch == targetArch {
			candidates = append(candidates, item)
		}
	}
	if len(candidates) == 0 {
		return nil, "", fmt.Sprintf("未托管 %s/%s 安装包", binaryPlatformLabel(platform), targetArch)
	}
	find := func(variant string) *repository.AgentBinaryMeta {
		for i := range candidates {
			if candidates[i].Variant == variant {
				return &candidates[i]
			}
		}
		return nil
	}
	standard := find(VariantStandard)
	legacy := find(VariantLegacy)

	if platform != "windows" {
		if standard == nil {
			return nil, "", fmt.Sprintf("未托管 %s/%s 标准安装包", binaryPlatformLabel(platform), targetArch)
		}
		return standard, strings.Join(warnings, "；"), ""
	}

	switch {
	case osBuild >= windowsBuildModern:
		if standard == nil {
			return nil, "", fmt.Sprintf("未托管 Windows %s 标准安装包（主机 Build %d，需 Windows 10 / Server 2016 及以上）", targetArch, osBuild)
		}
		return standard, strings.Join(warnings, "；"), ""
	case osBuild > 0:
		if legacy == nil {
			return nil, "", fmt.Sprintf("未托管 Windows %s legacy 安装包（主机 Build %d，需 Windows 7 / Server 2008 R2 兼容包）", targetArch, osBuild)
		}
		return legacy, strings.Join(warnings, "；"), ""
	default:
		warnings = append(warnings, "主机未上报系统版本，按标准包选择")
		if standard != nil {
			return standard, strings.Join(warnings, "；"), ""
		}
		if legacy != nil {
			return legacy, strings.Join(warnings, "；"), ""
		}
		return nil, "", fmt.Sprintf("未托管 Windows %s 安装包", targetArch)
	}
}

// normalizeArch 归一化 CPU 架构，无法识别时返回空串。
func normalizeArch(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == archAMD64 || normalized == archARM64 {
		return normalized
	}
	return ""
}

// normalizeVariant 归一化通道名称，未知时原样返回由调用方校验。
func normalizeVariant(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// resolveVariant 校验或推导上传包的通道。
func resolveVariant(platform, arch, variant, goVersion string) (string, error) {
	switch normalizeVariant(variant) {
	case VariantStandard:
		return VariantStandard, nil
	case VariantLegacy:
		if platform != "windows" || arch != archAMD64 {
			return "", fmt.Errorf("legacy 仅支持 Windows amd64 安装包")
		}
		return VariantLegacy, nil
	case "":
		if platform == "windows" && arch == archAMD64 && goVersion != "" && !goVersionAtLeast(goVersion, 1, 21) {
			return VariantLegacy, nil
		}
		return VariantStandard, nil
	default:
		return "", fmt.Errorf("不支持的通道：%s，仅支持 standard 或 legacy", strings.TrimSpace(variant))
	}
}

// goVersionAtLeast 判断 Go 版本是否不低于指定 major.minor，解析失败返回 false。
func goVersionAtLeast(goVersion string, major, minor int) bool {
	trimmed := strings.TrimPrefix(strings.TrimSpace(goVersion), "go")
	parts := strings.SplitN(trimmed, ".", 3)
	if len(parts) < 2 {
		return false
	}
	gotMajor, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	gotMinor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	if gotMajor != major {
		return gotMajor > major
	}
	return gotMinor >= minor
}

func binaryPlatformLabel(platform string) string {
	if platform == "windows" {
		return "Windows"
	}
	return "Linux"
}

// validateBinaryFormat 校验上传文件类型与平台一致，并解析 CPU 架构
func validateBinaryFormat(platform string, data []byte) (string, error) {
	switch platform {
	case "linux":
		if len(data) < 20 || !bytes.Equal(data[:4], []byte{0x7f, 'E', 'L', 'F'}) {
			return "", fmt.Errorf("文件类型与平台不匹配：linux 仅接受 ELF 二进制")
		}
		switch binary.LittleEndian.Uint16(data[18:20]) {
		case 0x3e:
			return archAMD64, nil
		case 0xb7:
			return archARM64, nil
		}
		return "", fmt.Errorf("不支持的 ELF 架构：仅支持 amd64 与 arm64")
	case "windows":
		if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
			return "", fmt.Errorf("文件类型与平台不匹配：windows 仅接受 PE（.exe）二进制")
		}
		offset := uint64(binary.LittleEndian.Uint32(data[0x3c:0x40]))
		if offset+6 > uint64(len(data)) || !bytes.Equal(data[offset:offset+4], []byte{'P', 'E', 0, 0}) {
			return "", fmt.Errorf("文件类型与平台不匹配：windows 仅接受 PE（.exe）二进制")
		}
		switch binary.LittleEndian.Uint16(data[offset+4 : offset+6]) {
		case 0x8664:
			return archAMD64, nil
		case 0xaa64:
			return archARM64, nil
		}
		return "", fmt.Errorf("不支持的 PE 架构：仅支持 amd64 与 arm64")
	}
	return "", fmt.Errorf("不支持的平台：%s", platform)
}

// readAgentGoVersion 从二进制内嵌构建信息读取 Go 版本，无法解析时返回空串
func readAgentGoVersion(data []byte) string {
	info, err := buildinfo.Read(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(info.GoVersion)
}

// SaveAgentBinary 保存上传的二进制到对应槽位：优先使用内嵌版本标记，其次使用显式版本。
func (s *BackupService) SaveAgentBinary(platform, variant, version string, data []byte) (repository.AgentBinaryMeta, error) {
	if platform != "linux" && platform != "windows" {
		return repository.AgentBinaryMeta{}, fmt.Errorf("不支持的平台：%s", platform)
	}
	if len(data) == 0 {
		return repository.AgentBinaryMeta{}, fmt.Errorf("上传内容为空")
	}
	if len(data) > 128<<20 {
		return repository.AgentBinaryMeta{}, fmt.Errorf("文件超过 128MB 限制")
	}
	arch, err := validateBinaryFormat(platform, data)
	if err != nil {
		return repository.AgentBinaryMeta{}, err
	}
	goVersion := readAgentGoVersion(data)
	variant, err = resolveVariant(platform, arch, variant, goVersion)
	if err != nil {
		return repository.AgentBinaryMeta{}, err
	}
	if match := embeddedVersionPattern.FindSubmatch(data); match != nil {
		version = string(match[1])
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return repository.AgentBinaryMeta{}, fmt.Errorf("无法识别版本号：请手动填写 version 参数")
	}
	digest := sha256.Sum256(data)
	meta := repository.AgentBinaryMeta{
		Platform: platform, Arch: arch, Variant: variant, GoVersion: goVersion, Version: version,
		Sha256: hex.EncodeToString(digest[:]), Size: int64(len(data)), UploadedAt: appcommon.NowTimestamp(),
	}
	target := s.AgentBinaryPath(platform, arch, variant)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return meta, err
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return meta, err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return meta, err
	}
	if err := s.repo.SaveAgentBinaryMeta(meta); err != nil {
		return meta, err
	}
	return meta, nil
}
