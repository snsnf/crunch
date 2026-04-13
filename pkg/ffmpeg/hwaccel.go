package ffmpeg

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// HWEncoder represents a detected hardware encoder.
type HWEncoder struct {
	Name    string // human-readable name (e.g. "NVIDIA NVENC")
	Codec   string // ffmpeg encoder name (e.g. "h264_nvenc")
	HWAccel string // -hwaccel value, empty if not needed
	Device  string // -hwaccel_device or -vaapi_device value
	// InitFilters is prepended to the filter chain (e.g. "format=nv12,hwupload" for VAAPI)
	InitFilters string
}

// encoderDefs lists GPU encoders to probe, in priority order per platform.
var encoderDefs = []struct {
	os      string // "windows", "linux", or "" for both
	encoder HWEncoder
}{
	// NVIDIA NVENC — works on both Windows and Linux
	{"", HWEncoder{
		Name:  "NVIDIA NVENC",
		Codec: "h264_nvenc",
	}},
	// Intel Quick Sync — works on both Windows and Linux
	{"", HWEncoder{
		Name:    "Intel Quick Sync",
		Codec:   "h264_qsv",
		HWAccel: "qsv",
	}},
	// AMD AMF — Windows only
	{"windows", HWEncoder{
		Name:  "AMD AMF",
		Codec: "h264_amf",
	}},
	// VAAPI — Linux only (AMD/Intel)
	{"linux", HWEncoder{
		Name:        "VAAPI",
		Codec:       "h264_vaapi",
		HWAccel:     "vaapi",
		Device:      "/dev/dri/renderD128",
		InitFilters: "format=nv12,hwupload",
	}},
}

// DetectGPUEncoder probes ffmpeg for available hardware encoders and returns
// the first working one, or nil if none are available.
func DetectGPUEncoder(ffmpegPath string) *HWEncoder {
	if runtime.GOOS == "darwin" {
		// macOS always uses VideoToolbox via the existing path
		return &HWEncoder{
			Name:  "Apple VideoToolbox",
			Codec: "h264_videotoolbox",
		}
	}

	// First get the list of available encoders from ffmpeg
	available := listEncoders(ffmpegPath)

	for _, def := range encoderDefs {
		if def.os != "" && def.os != runtime.GOOS {
			continue
		}
		if !available[def.encoder.Codec] {
			continue
		}
		// Try a quick encode to verify the encoder actually works
		if testEncoder(ffmpegPath, def.encoder) {
			found := def.encoder
			return &found
		}
	}
	return nil
}

// listEncoders returns a set of available encoder names from ffmpeg -encoders.
func listEncoders(ffmpegPath string) map[string]bool {
	cmd := exec.Command(ffmpegPath, "-encoders", "-hide_banner")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	result := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// Lines look like: " V..... h264_nvenc  NVIDIA NVENC H.264 encoder (codec h264)"
		if len(line) < 8 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			result[fields[1]] = true
		}
	}
	return result
}

// testEncoder runs a minimal encode to verify the hardware encoder actually works
// (driver installed, GPU present, etc).
func testEncoder(ffmpegPath string, encoder HWEncoder) bool {
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=black:s=64x64:d=0.1",
	}

	if encoder.HWAccel != "" {
		if encoder.Codec == "h264_vaapi" {
			// VAAPI needs device init before input for hwupload
			args = []string{
				"-hide_banner", "-loglevel", "error",
				"-init_hw_device", fmt.Sprintf("vaapi=va:%s", encoder.Device),
				"-filter_hw_device", "va",
				"-f", "lavfi", "-i", "color=black:s=64x64:d=0.1",
				"-vf", "format=nv12,hwupload",
			}
		}
	}

	args = append(args,
		"-c:v", encoder.Codec,
		"-frames:v", "1",
		"-f", "null", "-",
	)

	testCmd := exec.Command(ffmpegPath, args...)
	hideWindow(testCmd)
	return testCmd.Run() == nil
}

// BuildGPUArgs builds ffmpeg args for single-pass GPU hardware encoding on Windows/Linux.
func BuildGPUArgs(encoder *HWEncoder, input string, filters string, videoBitrate int, audioBitrate int, output string, extraParams map[string]string) []string {
	var args []string

	// Hardware acceleration and device setup
	if encoder.Codec == "h264_vaapi" {
		args = append(args,
			"-y",
			"-init_hw_device", fmt.Sprintf("vaapi=va:%s", encoder.Device),
			"-filter_hw_device", "va",
			"-i", input,
			"-map", "0:v:0",
		)
	} else {
		args = append(args, "-y")
		if encoder.HWAccel != "" {
			args = append(args, "-hwaccel", encoder.HWAccel)
			if encoder.Device != "" {
				args = append(args, "-hwaccel_device", encoder.Device)
			}
		}
		args = append(args, "-i", input, "-map", "0:v:0")
	}

	hasAudio := extraParams["has_audio"] == "true"
	if hasAudio {
		args = append(args, "-map", "0:a:0")
	}

	// Video filters — for VAAPI, append hwupload after software filters
	if encoder.InitFilters != "" {
		if filters != "" {
			filters = filters + "," + encoder.InitFilters
		} else {
			filters = encoder.InitFilters
		}
	}
	args = append(args, "-vf", filters)

	// Encoder
	args = append(args, "-c:v", encoder.Codec)

	// Profile
	profile := extraParams["profile"]
	if profile == "" {
		profile = "main"
	}
	// Map profile names to encoder-specific values
	switch encoder.Codec {
	case "h264_nvenc":
		args = append(args, "-profile:v", profile)
		args = append(args, "-preset", "p4", "-tune", "hq", "-rc", "vbr")
	case "h264_amf":
		args = append(args, "-profile:v", profile)
		args = append(args, "-quality", "balanced", "-rc", "vbr_peak")
	case "h264_qsv":
		args = append(args, "-profile:v", profile)
		args = append(args, "-preset", "medium")
	case "h264_vaapi":
		args = append(args, "-rc_mode", "VBR")
	}

	level := extraParams["level"]
	if level != "" && encoder.Codec != "h264_vaapi" {
		args = append(args, "-level:v", level)
	}

	// Bitrate control
	maxrate := videoBitrate * 3 / 2
	bufsize := videoBitrate * 2
	args = append(args,
		"-b:v", fmt.Sprintf("%dk", videoBitrate),
		"-maxrate", fmt.Sprintf("%dk", maxrate),
		"-bufsize", fmt.Sprintf("%dk", bufsize),
	)

	if encoder.Codec != "h264_vaapi" {
		args = append(args, "-pix_fmt", "yuv420p")
	}

	if tag := extraParams["tag"]; tag != "" {
		args = append(args, "-tag:v", tag)
	}

	// Audio
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
