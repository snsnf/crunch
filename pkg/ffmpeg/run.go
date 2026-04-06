package ffmpeg

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

type Progress struct {
	TimeSeconds float64
	Percent     float64
}

var timeRegex = regexp.MustCompile(`time=(\d+):(\d+):(\d+)\.(\d+)`)

func RunEncode(ffmpegPath string, args []string, duration float64, progressCh chan<- Progress) error {
	if progressCh != nil {
		defer close(progressCh)
	}

	cmd := exec.Command(ffmpegPath, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg failed to start: %w", err)
	}

	scanner := bufio.NewScanner(stderr)
	scanner.Split(scanFFmpegOutput)

	for scanner.Scan() {
		line := scanner.Text()
		if t := parseTime(line); t > 0 && progressCh != nil {
			pct := (t / duration) * 100
			if pct > 100 {
				pct = 100
			}
			progressCh <- Progress{TimeSeconds: t, Percent: pct}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg exited with error: %w", err)
	}
	return nil
}

func parseTime(line string) float64 {
	matches := timeRegex.FindStringSubmatch(line)
	if len(matches) < 5 {
		return 0
	}
	h, _ := strconv.ParseFloat(matches[1], 64)
	m, _ := strconv.ParseFloat(matches[2], 64)
	s, _ := strconv.ParseFloat(matches[3], 64)
	frac, _ := strconv.ParseFloat("0."+matches[4], 64)
	return h*3600 + m*60 + s + frac
}

func scanFFmpegOutput(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' || data[i] == '\n' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// BuildHWArgs builds ffmpeg args for single-pass VideoToolbox hardware encoding (macOS).
// Uses constrained VBR to approximate target file size with minimal CPU usage.
func BuildHWArgs(input string, filters string, videoBitrate int, audioBitrate int, output string, extraParams map[string]string) []string {
	args := []string{"-y", "-hwaccel", "videotoolbox", "-i", input, "-map", "0:v:0"}

	hasAudio := extraParams["has_audio"] == "true"
	if hasAudio {
		args = append(args, "-map", "0:a:0")
	}

	args = append(args,
		"-vf", filters,
		"-c:v", "h264_videotoolbox",
	)

	profile := extraParams["profile"]
	if profile == "" {
		profile = "main"
	}
	args = append(args, "-profile:v", profile)

	level := extraParams["level"]
	if level != "" {
		args = append(args, "-level:v", level)
	}

	// Constrained VBR: target bitrate with maxrate cap to prevent overshoot
	maxrate := videoBitrate * 3 / 2 // 1.5x target
	bufsize := videoBitrate * 2     // 2x target
	args = append(args,
		"-b:v", fmt.Sprintf("%dk", videoBitrate),
		"-maxrate", fmt.Sprintf("%dk", maxrate),
		"-bufsize", fmt.Sprintf("%dk", bufsize),
		"-pix_fmt", "yuv420p",
	)

	if tag := extraParams["tag"]; tag != "" {
		args = append(args, "-tag:v", tag)
	}

	if hasAudio {
		args = append(args,
			"-c:a", "aac",
			"-ac", "2",
			"-ar", "44100",
			"-b:a", fmt.Sprintf("%dk", audioBitrate),
		)
	}

	args = append(args, "-movflags", "+faststart", "-map_metadata", "-1")
	if brand := extraParams["brand"]; brand != "" {
		args = append(args, "-brand", brand)
	}
	args = append(args, output)

	return args
}

// BuildPassArgs builds ffmpeg args for two-pass libx264 software encoding.
// Used on Linux/Windows or as fallback when hardware encoding isn't available.
func BuildPassArgs(pass int, input string, filters string, videoBitrate int, audioBitrate int, output string, extraParams map[string]string) []string {
	args := []string{"-y", "-i", input, "-map", "0:v:0"}

	hasAudio := extraParams["has_audio"] == "true"
	if hasAudio {
		args = append(args, "-map", "0:a:0")
	}

	args = append(args,
		"-vf", filters,
		"-c:v", "libx264",
	)

	profile := extraParams["profile"]
	if profile == "" {
		profile = "main"
	}
	args = append(args, "-profile:v", profile)

	level := extraParams["level"]
	if level != "" {
		args = append(args, "-level:v", level)
	}

	encPreset := extraParams["encode_preset"]
	if encPreset == "" {
		encPreset = "fast"
	}
	threads := extraParams["threads"]
	if threads == "" {
		threads = "6"
	}
	args = append(args,
		"-preset", encPreset,
		"-threads", threads,
		"-b:v", fmt.Sprintf("%dk", videoBitrate),
		"-pix_fmt", "yuv420p",
	)

	if tag := extraParams["tag"]; tag != "" {
		args = append(args, "-tag:v", tag)
	}

	if x264 := extraParams["x264_params"]; x264 != "" {
		args = append(args, "-x264-params", x264)
	}

	args = append(args, "-pass", fmt.Sprintf("%d", pass))

	if passlogfile := extraParams["passlogfile"]; passlogfile != "" {
		args = append(args, "-passlogfile", passlogfile)
	}

	if pass == 1 {
		args = append(args, "-an", "-f", "null")
		args = append(args, os.DevNull)
	} else {
		if hasAudio {
			args = append(args,
				"-c:a", "aac",
				"-ac", "2",
				"-ar", "44100",
				"-b:a", fmt.Sprintf("%dk", audioBitrate),
			)
		}
		args = append(args, "-movflags", "+faststart", "-map_metadata", "-1")
		if brand := extraParams["brand"]; brand != "" {
			args = append(args, "-brand", brand)
		}
		args = append(args, output)
	}

	return args
}
