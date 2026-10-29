package update

import "testing"

func TestDownloadPath(t *testing.T) {
	tests := []struct {
		platform string
		arch     string
		osBuild  int
		want     string
	}{
		{platform: "windows", arch: "amd64", osBuild: 7601, want: "/api/agent/download?arch=amd64&osBuild=7601&platform=windows"},
		{platform: "linux", arch: "arm64", osBuild: 0, want: "/api/agent/download?arch=arm64&platform=linux"},
		{platform: "windows", arch: "", osBuild: 0, want: "/api/agent/download?platform=windows"},
	}
	for _, test := range tests {
		if got := DownloadPath(test.platform, test.arch, test.osBuild); got != test.want {
			t.Errorf("DownloadPath(%q, %q, %d) = %q, want %q",
				test.platform, test.arch, test.osBuild, got, test.want)
		}
	}
}
