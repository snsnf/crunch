package compress

import (
	"fmt"
	"os"
	"path/filepath"

	"crunch/pkg/ghostscript"
)

type PDFOptions struct {
	InputPath  string
	OutputPath string
	Quality    ghostscript.PDFQuality // screen, ebook, printer, prepress
	ImageDPI   int                    // custom DPI override, 0 = use quality default
}

type PDFResult struct {
	InputPath    string
	OutputPath   string
	InputSizeKB  int64
	OutputSizeKB int64
}

func RunPDF(gsPath string, opts PDFOptions) (*PDFResult, error) {
	inputStat, err := os.Stat(opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read input: %w", err)
	}
	inputSizeKB := inputStat.Size() / 1024

	quality := opts.Quality
	if quality == "" {
		quality = ghostscript.QualityEbook
	}

	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = defaultOutput(opts.InputPath, "", ".pdf")
	}

	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot write to output directory: %w", err)
	}

	if err := ghostscript.Compress(gsPath, opts.InputPath, outputPath, quality, opts.ImageDPI); err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("PDF compression failed: %w", err)
	}

	outputStat, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read output: %w", err)
	}

	return &PDFResult{
		InputPath:    opts.InputPath,
		OutputPath:   outputPath,
		InputSizeKB:  inputSizeKB,
		OutputSizeKB: outputStat.Size() / 1024,
	}, nil
}
