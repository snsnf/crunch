package compress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func RunImage(paths *ffmpeg.Paths, opts ImageOptions, onProgress func(float64)) (*ImageResult, error) {
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

	if onProgress != nil {
		onProgress(0)
	}

	ext := strings.ToLower(filepath.Ext(opts.InputPath))
	if ext == ".png" {
		// PNG palette quantization: split into two real ffmpeg passes so progress
		// reflects actual work (palette generation = 50%, image write = 100%).
		colors := 32 + (quality * 224 / 100)
		if colors > 256 {
			colors = 256
		}

		palettePath := outputPath + ".palette.png"
		defer os.Remove(palettePath)

		pass1 := ffmpeg.BuildPNGPalettegenArgs(opts.InputPath, palettePath, colors)
		if err := ffmpeg.RunSimple(paths.FFmpeg, pass1); err != nil {
			return nil, fmt.Errorf("PNG palettegen failed: %w", err)
		}
		if onProgress != nil {
			onProgress(50)
		}

		pass2 := ffmpeg.BuildPNGPaletteuseArgs(opts.InputPath, palettePath, outputPath)
		if err := ffmpeg.RunSimple(paths.FFmpeg, pass2); err != nil {
			os.Remove(outputPath)
			return nil, fmt.Errorf("PNG paletteuse failed: %w", err)
		}
	} else {
		args := ffmpeg.BuildImageArgs(opts.InputPath, outputPath, quality, opts.MaxWidth, opts.MaxHeight, opts.StripMeta)
		if err := ffmpeg.RunSimple(paths.FFmpeg, args); err != nil {
			os.Remove(outputPath)
			return nil, fmt.Errorf("image compression failed: %w", err)
		}
	}

	if onProgress != nil {
		onProgress(100)
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
