package compress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"crunch/pkg/ffmpeg"
)

type AudioOptions struct {
	InputPath  string
	OutputPath string
	Bitrate    int    // target bitrate in kbps (e.g. 128, 192, 320), 0 = auto
	SampleRate int    // target sample rate (e.g. 44100, 48000), 0 = keep original
	Channels   int    // 1 = mono, 2 = stereo, 0 = keep original
	Format     string // output format override ("mp3","aac","opus","flac"), empty = keep same
	StripMeta  bool   // strip metadata/tags
}

type AudioResult struct {
	InputPath    string
	OutputPath   string
	InputSizeKB  int64
	OutputSizeKB int64
}

func RunAudio(paths *ffmpeg.Paths, opts AudioOptions, onProgress func(percent float64)) (*AudioResult, error) {
	inputStat, err := os.Stat(opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read input: %w", err)
	}
	inputSizeKB := inputStat.Size() / 1024

	duration, _ := ffmpeg.ProbeDuration(paths.FFprobe, opts.InputPath)

	bitrate := opts.Bitrate
	if bitrate <= 0 {
		bitrate = defaultBitrate(opts.InputPath, opts.Format)
	}

	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = defaultOutput(opts.InputPath, opts.Format, "")
	}

	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot write to output directory: %w", err)
	}

	args := ffmpeg.BuildAudioArgs(opts.InputPath, outputPath, bitrate, opts.SampleRate, opts.Channels, opts.StripMeta)

	if duration > 0 && onProgress != nil {
		progressCh := make(chan ffmpeg.Progress, 100)
		go func() {
			for p := range progressCh {
				onProgress(p.Percent)
			}
		}()
		if err := ffmpeg.RunEncode(paths.FFmpeg, args, duration, progressCh); err != nil {
			os.Remove(outputPath)
			return nil, fmt.Errorf("audio compression failed: %w", err)
		}
	} else {
		if err := ffmpeg.RunSimple(paths.FFmpeg, args); err != nil {
			os.Remove(outputPath)
			return nil, fmt.Errorf("audio compression failed: %w", err)
		}
	}

	outputStat, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read output: %w", err)
	}

	return &AudioResult{
		InputPath:    opts.InputPath,
		OutputPath:   outputPath,
		InputSizeKB:  inputSizeKB,
		OutputSizeKB: outputStat.Size() / 1024,
	}, nil
}

// defaultBitrate picks a reasonable target bitrate based on format.
func defaultBitrate(inputPath, formatOverride string) int {
	ext := formatOverride
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(inputPath))
	}
	ext = strings.TrimPrefix(ext, ".")

	switch ext {
	case "opus":
		return 96
	case "aac", "m4a":
		return 128
	case "mp3":
		return 128
	case "ogg":
		return 112
	case "flac", "wav", "aiff", "aif", "alac":
		return 192
	default:
		return 128
	}
}

