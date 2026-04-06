package compress

import (
	"fmt"
	"os"
	"path/filepath"

	"crunch/pkg/ffmpeg"
)

type ImageOptions struct {
	InputPath  string
	OutputPath string
	Quality    int    // 1-100 (100 = best quality, larger file)
	MaxWidth   int    // max width in pixels, 0 = keep original
	MaxHeight  int    // max height in pixels, 0 = keep original
	StripMeta  bool   // strip EXIF/metadata
	Format     string // output format override ("jpg","png","webp","avif"), empty = keep same
}

type ImageResult struct {
	InputPath    string
	OutputPath   string
	InputSizeKB  int64
	OutputSizeKB int64
}

func RunImage(paths *ffmpeg.Paths, opts ImageOptions) (*ImageResult, error) {
	inputStat, err := os.Stat(opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read input: %w", err)
	}
	inputSizeKB := inputStat.Size() / 1024

	quality := opts.Quality
	if quality <= 0 {
		quality = 75
	}
	if quality > 100 {
		quality = 100
	}

	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = defaultOutput(opts.InputPath, opts.Format, "")
	}

	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot write to output directory: %w", err)
	}

	args := ffmpeg.BuildImageArgs(opts.InputPath, outputPath, quality, opts.MaxWidth, opts.MaxHeight, opts.StripMeta)

	if err := ffmpeg.RunSimple(paths.FFmpeg, args); err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("image compression failed: %w", err)
	}

	outputStat, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read output: %w", err)
	}

	return &ImageResult{
		InputPath:    opts.InputPath,
		OutputPath:   outputPath,
		InputSizeKB:  inputSizeKB,
		OutputSizeKB: outputStat.Size() / 1024,
	}, nil
}
