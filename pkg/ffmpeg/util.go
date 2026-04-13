package ffmpeg

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// RunSimple runs ffmpeg with the given arguments and waits for it to finish.
// No progress tracking. Used for quick operations like image compression.
func RunSimple(ffmpegPath string, args []string) error {
	cmd := exec.Command(ffmpegPath, args...)
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w\n%s", err, string(out))
	}
	return nil
}

type probeFormatJSON struct {
	Duration string `json:"duration"`
}

type probeResultJSON struct {
	Format probeFormatJSON `json:"format"`
}

// ProbeDuration returns the duration of a media file in seconds.
func ProbeDuration(ffprobePath, filePath string) (float64, error) {
	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-show_format",
		"-of", "json",
		filePath,
	)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}
	var result probeResultJSON
	if err := json.Unmarshal(out, &result); err != nil {
		return 0, err
	}
	d, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	return d, nil
}
