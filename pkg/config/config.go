package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

type RecentFile struct {
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
}

func dir() (string, error) {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
		if base == "" {
			base, _ = os.UserHomeDir()
		}
	} else {
		base, _ = os.UserHomeDir()
		base = filepath.Join(base, ".config")
	}
	dir := filepath.Join(base, "crunch")
	return dir, os.MkdirAll(dir, 0700)
}

func FFmpegDir() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(d, "ffmpeg")
	return dir, os.MkdirAll(dir, 0700)
}

func recentPath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "recent.json"), nil
}

func LoadRecent() ([]RecentFile, error) {
	p, err := recentPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []RecentFile
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func AddRecent(path string) error {
	files, _ := LoadRecent()
	filtered := make([]RecentFile, 0, len(files))
	for _, f := range files {
		if f.Path != path {
			filtered = append(filtered, f)
		}
	}
	filtered = append(filtered, RecentFile{Path: path, Timestamp: time.Now()})
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	p, err := recentPath()
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}
