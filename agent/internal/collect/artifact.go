package collect

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backup-monitor-agent/internal/api"
	"backup-monitor-agent/internal/discover"
)

type artifactInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
}

func (c *Collector) findArtifact(task discover.Task, profile api.ProfileConfig) *artifactInfo {
	if !profile.ArtifactCheck {
		return nil
	}
	path := configString(task.Config, "artifactPath", "backupDir", "artifact", "artifactDir", "dir", "path")
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.Mode().IsRegular() {
		return &artifactInfo{Path: path, Size: info.Size(), ModTime: info.ModTime()}
	}
	if !info.IsDir() {
		return nil
	}
	if match := configString(task.Config, "match", "pattern", "glob"); match != "" {
		if matched := newestMatchingArtifact(path, match, profile.ArtifactMaxDepth); matched != nil {
			return matched
		}
	}
	return newestArtifact(path, profile.ArtifactMaxDepth)
}

func newestArtifact(root string, maxDepth int) *artifactInfo {
	return newestMatchingArtifact(root, "", maxDepth)
}

func newestMatchingArtifact(root, match string, maxDepth int) *artifactInfo {
	var newest *artifactInfo
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		depth := strings.Count(relative, string(os.PathSeparator))
		if entry.IsDir() {
			if entry.Type()&fs.ModeSymlink != 0 {
				return fs.SkipDir
			}
			if maxDepth > 0 && depth >= maxDepth {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || isStatusArtifact(path) {
			return nil
		}
		if match != "" {
			matched, matchErr := filepath.Match(strings.ToLower(match), strings.ToLower(entry.Name()))
			if matchErr != nil || !matched {
				return nil
			}
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil
		}
		if newest == nil || info.ModTime().After(newest.ModTime) {
			newest = &artifactInfo{Path: path, Size: info.Size(), ModTime: info.ModTime()}
		}
		return nil
	})
	return newest
}

func isStatusArtifact(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if base == "veeam-status.txt" {
		return true
	}
	return strings.HasSuffix(base, ".status") || strings.HasSuffix(base, ".tmp") || strings.HasSuffix(base, ".part")
}
