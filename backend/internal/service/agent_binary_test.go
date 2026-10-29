package service_test

import (
	"encoding/binary"
	"testing"
)

const (
	elfAMD64 = 0x3e
	elfARM64 = 0xb7
	peAMD64  = 0x8664
	peARM64  = 0xaa64
)

func elfAgentBinary(version string, machine uint16) []byte {
	data := make([]byte, 20)
	copy(data, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(data[18:], machine)
	return append(data, []byte("backup-agent-version:"+version)...)
}

func peAgentBinary(version string, machine uint16) []byte {
	data := make([]byte, 0x40)
	copy(data, []byte{'M', 'Z'})
	binary.LittleEndian.PutUint32(data[0x3c:], 0x40)
	data = append(data, []byte{'P', 'E', 0, 0}...)
	machineBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(machineBytes, machine)
	data = append(data, machineBytes...)
	return append(data, []byte("backup-agent-version:"+version)...)
}

func TestSaveAgentBinaryDetectsArch(t *testing.T) {
	svc := newTestService(t)
	tests := []struct {
		name     string
		platform string
		data     []byte
		wantArch string
		wantErr  bool
	}{
		{name: "linux elf amd64", platform: "linux", data: elfAgentBinary("1.2.34", elfAMD64), wantArch: "amd64"},
		{name: "linux elf arm64", platform: "linux", data: elfAgentBinary("1.2.34", elfARM64), wantArch: "arm64"},
		{name: "windows pe amd64", platform: "windows", data: peAgentBinary("1.2.34", peAMD64), wantArch: "amd64"},
		{name: "windows pe arm64", platform: "windows", data: peAgentBinary("1.2.34", peARM64), wantArch: "arm64"},
		{name: "linux unsupported arch", platform: "linux", data: elfAgentBinary("1.2.34", 0x03), wantErr: true},
		{name: "windows with elf", platform: "windows", data: elfAgentBinary("1.2.34", elfAMD64), wantErr: true},
		{name: "linux with pe", platform: "linux", data: peAgentBinary("1.2.34", peAMD64), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			meta, err := svc.SaveAgentBinary(test.platform, "", "", test.data)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected format error for %s", test.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("save binary: %v", err)
			}
			if meta.Arch != test.wantArch {
				t.Fatalf("arch = %q, want %q", meta.Arch, test.wantArch)
			}
			if meta.Version != "1.2.34" {
				t.Fatalf("version = %q, want 1.2.34", meta.Version)
			}
			if meta.Variant != "standard" {
				t.Fatalf("variant = %q, want standard", meta.Variant)
			}
		})
	}
}
