package compress

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"crunch/pkg/ffmpeg"
	"crunch/pkg/probe"
)

type Options struct {
	InputPath   string
	OutputPath  string
	PresetName  PresetName
	TargetMB    int
	Resolution  string
	FPSOverride int
	UseGPU      bool
}

type Result struct {
	InputInfo  *probe.VideoInfo
	OutputInfo *probe.VideoInfo
	OutputPath string
}

func Run(paths *ffmpeg.Paths, opts Options, onProgress func(pass int, percent float64)) (*Result, error) {
	inputInfo, err := probe.Run(paths.FFprobe, opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe input: %w", err)
	}

	preset := GetPreset(opts.PresetName)

	targetMB := opts.TargetMB
	if targetMB == 0 {
		targetMB = preset.DefaultTarget
	}
	if targetMB == 0 {
		return nil, fmt.Errorf("target size is required for preset %s", opts.PresetName)
	}
	if preset.MaxTarget > 0 && targetMB > preset.MaxTarget {
		targetMB = preset.MaxTarget - 1
	}

	audioBitrate := preset.AudioBitrate
	if !inputInfo.HasAudio {
		audioBitrate = 0
	}
	bitrate, err := CalculateBitrate(targetMB, inputInfo.Duration, audioBitrate)
	if err != nil {
		return nil, err
	}

	filters := BuildFilterChain(inputInfo, preset, opts.Resolution, opts.FPSOverride)

	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = defaultOutput(opts.InputPath, "", ".mp4")
	}

	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot write to output directory: %w", err)
	}

	extra := map[string]string{
		"profile":     preset.Profile,
		"level":       preset.Level,
		"tag":         preset.Tag,
		"x264_params": preset.X264Params,
		"brand":       preset.Brand,
	}
	if inputInfo.HasAudio {
		extra["has_audio"] = "true"
	}

	// Determine encoding path: hardware (GPU) or software (CPU)
	useHW := false
	var gpuEncoder *ffmpeg.HWEncoder

	if runtime.GOOS == "darwin" {
		// macOS always uses VideoToolbox
		useHW = true
	} else if opts.UseGPU {
		gpuEncoder = ffmpeg.DetectGPUEncoder(paths.FFmpeg)
		if gpuEncoder != nil {
			useHW = true
		}
	}

	if useHW {
		// Single-pass hardware encoding (VideoToolbox / NVENC / AMF / QSV / VAAPI)
		// Apply 98% bitrate safety margin since no two-pass
		hwBitrate := bitrate.VideoBitrate * 98 / 100

		var hwArgs []string
		if runtime.GOOS == "darwin" {
			hwArgs = ffmpeg.BuildHWArgs(opts.InputPath, filters, hwBitrate, bitrate.AudioBitrate, outputPath, extra)
		} else {
			hwArgs = ffmpeg.BuildGPUArgs(gpuEncoder, opts.InputPath, filters, hwBitrate, bitrate.AudioBitrate, outputPath, extra)
		}

		progressCh := make(chan ffmpeg.Progress, 100)
		go func() {
			for p := range progressCh {
				if onProgress != nil {
					onProgress(1, p.Percent)
				}
			}
		}()
		if err := ffmpeg.RunEncode(paths.FFmpeg, hwArgs, inputInfo.Duration, progressCh); err != nil {
			os.Remove(outputPath)
			encoderName := "VideoToolbox"
			if gpuEncoder != nil {
				encoderName = gpuEncoder.Name
			}
			return nil, fmt.Errorf("%s encode failed: %w", encoderName, err)
		}
	} else {
		// Two-pass libx264 software encoding
		// Use a temp directory for passlog files to avoid CWD pollution and race conditions
		passlogDir, err := os.MkdirTemp("", "crunch-pass-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir for passlog: %w", err)
		}
		defer os.RemoveAll(passlogDir)
		extra["passlogfile"] = filepath.Join(passlogDir, "pass")

		// Pass 1
		pass1Args := ffmpeg.BuildPassArgs(1, opts.InputPath, filters, bitrate.VideoBitrate, bitrate.AudioBitrate, outputPath, extra)
		pass1Progress := make(chan ffmpeg.Progress, 100)
		go func() {
			for p := range pass1Progress {
				if onProgress != nil {
					onProgress(1, p.Percent)
				}
			}
		}()
		if err := ffmpeg.RunEncode(paths.FFmpeg, pass1Args, inputInfo.Duration, pass1Progress); err != nil {
			return nil, fmt.Errorf("pass 1 failed: %w", err)
		}

		// Pass 2
		pass2Args := ffmpeg.BuildPassArgs(2, opts.InputPath, filters, bitrate.VideoBitrate, bitrate.AudioBitrate, outputPath, extra)
		pass2Progress := make(chan ffmpeg.Progress, 100)
		go func() {
			for p := range pass2Progress {
				if onProgress != nil {
					onProgress(2, p.Percent)
				}
			}
		}()
		if err := ffmpeg.RunEncode(paths.FFmpeg, pass2Args, inputInfo.Duration, pass2Progress); err != nil {
			os.Remove(outputPath)
			return nil, fmt.Errorf("pass 2 failed: %w", err)
		}
	}

	outputInfo, err := probe.Run(paths.FFprobe, outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe output: %w", err)
	}

	return &Result{
		InputInfo:  inputInfo,
		OutputInfo: outputInfo,
		OutputPath: outputPath,
	}, nil
}

// defaultOutput generates a "_crunched" output path, optionally changing the extension.
func defaultOutput(inputPath, formatOverride, fallbackExt string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	if formatOverride != "" {
		return base + "_crunched." + strings.TrimPrefix(formatOverride, ".")
	}
	if fallbackExt != "" {
		return base + "_crunched" + fallbackExt
	}
	return base + "_crunched" + ext
}
