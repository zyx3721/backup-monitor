package collect

import (
	"testing"

	"backup-monitor-agent/internal/api"
)

func TestNormalizeProfileFallsBackOnEmptyLists(t *testing.T) {
	profile := normalizeProfile(api.ProfileConfig{
		IncludeKeywords: []string{},
		IncludePaths:    []string{},
		ExcludeKeywords: []string{},
	})
	if len(profile.IncludeKeywords) == 0 || len(profile.IncludePaths) == 0 {
		t.Fatalf("empty include lists should fall back to defaults: %#v", profile)
	}
	if len(profile.ExcludeKeywords) != 0 {
		t.Fatalf("explicit empty exclusion should stay empty: %#v", profile.ExcludeKeywords)
	}
}
