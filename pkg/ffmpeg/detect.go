package ffmpeg

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"crunch/pkg/config"
)

var errNotFound = errors.New("ffmpeg not found")

type Paths struct {
	FFmpeg  string
	FFprobe string
}

func Detect() (*Paths, error) {
	// Check next to the running executable first (bundled in .app)
	if p, err := findNextToExe(); err == nil {
		return p, nil
	}
	if p, err := findInPath(); err == nil {
		return p, nil
	}
	if p, err := findInCommon(); err == nil {
		return p, nil
	}
	if p, err := findInLocal(); err == nil {
		return p, nil
	}
	return nil, errNotFound
}

func findNextToExe() (*Paths, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(exe)
	return checkDir(dir)
}

func findInPath() (*Paths, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, err
	}
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, err
	}
	return &Paths{FFmpeg: ffmpegPath, FFprobe: ffprobePath}, nil
}

func findInCommon() (*Paths, error) {
	var dirs []string
	switch runtime.GOOS {
	case "darwin":
		dirs = []string{"/opt/homebrew/bin", "/usr/local/bin"}
	case "linux":
		dirs = []string{"/usr/bin", "/usr/local/bin", "/snap/bin"}
	case "windows":
		dirs = []string{`C:\ffmpeg\bin`, `C:\Program Files\ffmpeg\bin`}
	}
	for _, dir := range dirs {
		if p, err := checkDir(dir); err == nil {
			return p, nil
		}
	}
	return nil, errNotFound
}

func findInLocal() (*Paths, error) {
	dir, err := config.FFmpegDir()
	if err != nil {
		return nil, err
	}
	return checkDir(dir)
}

func checkDir(dir string) (*Paths, error) {
	ffmpeg := filepath.Join(dir, ffmpegBinary())
	ffprobe := filepath.Join(dir, ffprobeBinary())
	if err := exec.Command(ffmpeg, "-version").Run(); err != nil {
		return nil, err
	}
	if err := exec.Command(ffprobe, "-version").Run(); err != nil {
		return nil, err
	}
	return &Paths{FFmpeg: ffmpeg, FFprobe: ffprobe}, nil
}

func ffmpegBinary() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

func ffprobeBinary() string {
	if runtime.GOOS == "windows" {
		return "ffprobe.exe"
	}
	return "ffprobe"
}
