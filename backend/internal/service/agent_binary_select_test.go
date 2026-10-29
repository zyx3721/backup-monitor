package service

import (
	"strings"
	"testing"

	"backup-monitor-backend/internal/repository"
)

func binaryMeta(platform, arch, variant string) repository.AgentBinaryMeta {
	return repository.AgentBinaryMeta{Platform: platform, Arch: arch, Variant: variant, Version: "1.0.36"}
}

func TestSelectAgentBinary(t *testing.T) {
	linuxAMD64 := binaryMeta("linux", "amd64", VariantStandard)
	linuxARM64 := binaryMeta("linux", "arm64", VariantStandard)
	winStandard := binaryMeta("windows", "amd64", VariantStandard)
	winLegacy := binaryMeta("windows", "amd64", VariantLegacy)
	winARM64 := binaryMeta("windows", "arm64", VariantStandard)

	tests := []struct {
		name        string
		platform    string
		arch        string
		osBuild     int
		items       []repository.AgentBinaryMeta
		wantVariant string
		wantArch    string
		wantReason  string
		wantWarning string
	}{
		{
			name: "linux amd64", platform: "linux", arch: "amd64", items: []repository.AgentBinaryMeta{linuxAMD64},
			wantVariant: VariantStandard, wantArch: "amd64",
		},
		{
			name: "linux arm64", platform: "linux", arch: "arm64", items: []repository.AgentBinaryMeta{linuxARM64},
			wantVariant: VariantStandard, wantArch: "arm64",
		},
		{
			name: "windows modern prefers standard", platform: "windows", arch: "amd64", osBuild: 19045,
			items: []repository.AgentBinaryMeta{winStandard, winLegacy}, wantVariant: VariantStandard, wantArch: "amd64",
		},
		{
			name: "windows win7 uses legacy", platform: "windows", arch: "amd64", osBuild: 7601,
			items: []repository.AgentBinaryMeta{winStandard, winLegacy}, wantVariant: VariantLegacy, wantArch: "amd64",
		},
		{
			name: "windows modern never falls back to legacy", platform: "windows", arch: "amd64", osBuild: 19045,
			items:      []repository.AgentBinaryMeta{winLegacy},
			wantReason: "未托管 Windows amd64 标准安装包",
		},
		{
			name: "windows win7 does not use standard", platform: "windows", arch: "amd64", osBuild: 7601,
			items:      []repository.AgentBinaryMeta{winStandard},
			wantReason: "未托管 Windows amd64 legacy 安装包",
		},
		{
			name: "windows arm64", platform: "windows", arch: "arm64", osBuild: 19045,
			items: []repository.AgentBinaryMeta{winARM64}, wantVariant: VariantStandard, wantArch: "arm64",
		},
		{
			name: "unknown arch defaults to amd64", platform: "linux", arch: "", items: []repository.AgentBinaryMeta{linuxAMD64},
			wantVariant: VariantStandard, wantArch: "amd64", wantWarning: "主机未上报架构",
		},
		{
			name: "unknown build prefers standard", platform: "windows", arch: "amd64", osBuild: 0,
			items: []repository.AgentBinaryMeta{winStandard, winLegacy}, wantVariant: VariantStandard, wantArch: "amd64", wantWarning: "主机未上报系统版本",
		},
		{
			name: "unknown build falls back to legacy", platform: "windows", arch: "amd64", osBuild: 0,
			items: []repository.AgentBinaryMeta{winLegacy}, wantVariant: VariantLegacy, wantArch: "amd64", wantWarning: "主机未上报系统版本",
		},
		{
			name: "no binary for arch", platform: "windows", arch: "arm64", osBuild: 19045,
			items:      []repository.AgentBinaryMeta{winStandard},
			wantReason: "未托管 Windows/arm64 安装包",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			meta, warning, reason := selectAgentBinary(test.platform, test.arch, test.osBuild, test.items)
			if test.wantReason != "" {
				if meta != nil {
					t.Fatalf("meta = %#v, want nil", meta)
				}
				if !strings.Contains(reason, test.wantReason) {
					t.Fatalf("reason = %q, want contains %q", reason, test.wantReason)
				}
				return
			}
			if meta == nil {
				t.Fatalf("meta = nil, reason = %q", reason)
			}
			if meta.Variant != test.wantVariant || meta.Arch != test.wantArch {
				t.Fatalf("selected %s/%s, want %s/%s", meta.Arch, meta.Variant, test.wantArch, test.wantVariant)
			}
			if test.wantWarning != "" && !strings.Contains(warning, test.wantWarning) {
				t.Fatalf("warning = %q, want contains %q", warning, test.wantWarning)
			}
		})
	}
}

func TestResolveVariant(t *testing.T) {
	tests := []struct {
		name      string
		platform  string
		arch      string
		variant   string
		goVersion string
		want      string
		wantErr   bool
	}{
		{name: "legacy derived from go1.20", platform: "windows", arch: "amd64", goVersion: "go1.20.14", want: VariantLegacy},
		{name: "standard derived from go1.26", platform: "windows", arch: "amd64", goVersion: "go1.26.0", want: VariantStandard},
		{name: "linux go1.20 stays standard", platform: "linux", arch: "amd64", goVersion: "go1.20.14", want: VariantStandard},
		{name: "windows arm64 go1.20 stays standard", platform: "windows", arch: "arm64", goVersion: "go1.20.14", want: VariantStandard},
		{name: "explicit legacy", platform: "windows", arch: "amd64", variant: "legacy", want: VariantLegacy},
		{name: "explicit standard", platform: "windows", arch: "amd64", variant: "standard", goVersion: "go1.20.14", want: VariantStandard},
		{name: "legacy on linux rejected", platform: "linux", arch: "amd64", variant: "legacy", wantErr: true},
		{name: "legacy on windows arm64 rejected", platform: "windows", arch: "arm64", variant: "legacy", wantErr: true},
		{name: "unknown variant rejected", platform: "windows", arch: "amd64", variant: "beta", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveVariant(test.platform, test.arch, test.variant, test.goVersion)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveVariant: %v", err)
			}
			if got != test.want {
				t.Fatalf("variant = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGoVersionAtLeast(t *testing.T) {
	cases := []struct {
		version string
		minor   int
		want    bool
	}{
		{version: "go1.26.0", minor: 21, want: true},
		{version: "go1.21.0", minor: 21, want: true},
		{version: "go1.20.14", minor: 21, want: false},
		{version: "", minor: 21, want: false},
		{version: "bad", minor: 21, want: false},
	}
	for _, test := range cases {
		if got := goVersionAtLeast(test.version, 1, test.minor); got != test.want {
			t.Errorf("goVersionAtLeast(%q, 1, %d) = %v, want %v", test.version, test.minor, got, test.want)
		}
	}
}
