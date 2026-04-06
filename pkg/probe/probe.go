package probe

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type VideoInfo struct {
	Path       string
	Width      int
	Height     int
	Duration   float64
	Codec      string
	FPS        float64
	FileSizeMB int64
	HasAudio   bool
}

type probeFormat struct {
	Duration string `json:"duration"`
	Size     string `json:"size"`
}

type probeStream struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	RFrameRate string `json:"r_frame_rate"`
}

type probeResult struct {
	Format  probeFormat   `json:"format"`
	Streams []probeStream `json:"streams"`
}

func Run(ffprobePath, filePath string) (*VideoInfo, error) {
	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-show_format",
		"-show_streams",
		"-of", "json",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed for %s: %w", filePath, err)
	}

	var result probeResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &VideoInfo{Path: filePath}

	if d, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
		info.Duration = d
	}
	if size, err := strconv.ParseInt(result.Format.Size, 10, 64); err == nil {
		info.FileSizeMB = size / (1024 * 1024)
	}

	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			info.Width = stream.Width
			info.Height = stream.Height
			info.Codec = stream.CodecName
			if parts := strings.Split(stream.RFrameRate, "/"); len(parts) == 2 {
				num, _ := strconv.ParseFloat(parts[0], 64)
				den, _ := strconv.ParseFloat(parts[1], 64)
				if den > 0 {
					info.FPS = num / den
				}
			}
	case "audio":
			info.HasAudio = true
		}
	}

	if info.Width == 0 || info.Height == 0 {
		return nil, fmt.Errorf("no video stream found in %s", filePath)
	}

	return info, nil
}
