package backup

import "testing"

func TestParsePlatform(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "linux", input: "linux", want: "linux"},
		{name: "windows with spaces", input: " Windows ", want: "windows"},
		{name: "empty", input: "", wantErr: true},
		{name: "unsupported", input: "darwin", wantErr: true},
		{name: "unknown suffix", input: "linux64", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePlatform(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parsePlatform(%q) expected error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePlatform(%q) error: %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("parsePlatform(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestAgentBinaryFileName(t *testing.T) {
	tests := map[string]string{
		"linux":   "backup-agent-linux",
		"windows": "backup-agent-windows.exe",
	}
	for platform, want := range tests {
		if got := agentBinaryFileName(platform); got != want {
			t.Fatalf("agentBinaryFileName(%q) = %q, want %q", platform, got, want)
		}
	}
}

func TestHostPlatform(t *testing.T) {
	tests := map[string]string{
		"linux":     "linux",
		"Windows":   "windows",
		"":          "",
		"darwin":    "",
		"windows10": "",
	}
	for input, want := range tests {
		if got := hostPlatform(input); got != want {
			t.Fatalf("hostPlatform(%q) = %q, want %q", input, got, want)
		}
	}
}
