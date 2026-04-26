package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"crunch/pkg/compress"
	"crunch/pkg/config"
	"crunch/pkg/ffmpeg"
	"crunch/pkg/ghostscript"

	"github.com/spf13/cobra"
)

var (
	preset       string
	targetMB     int
	output       string
	resolution   string
	fps          int
	quiet        bool
	useGPU       bool
	imgQuality   int
	audioBitrate int
	pdfQuality   string
)

var rootCmd = &cobra.Command{
	Use:   "crunch [files...]",
	Short: "Fast media compressor — video, image, audio, and PDF",
	Long:  "Crunch compresses videos, images, audio files, and PDFs.\nRun without arguments for usage help. Use the GUI app for interactive mode.",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVarP(&preset, "preset", "p", "whatsapp", "Video preset: whatsapp, generic")
	rootCmd.Flags().IntVarP(&targetMB, "target", "t", 0, "Video target size in MB (default: 8 for whatsapp)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Output path (default: <input>_crunched.<ext>)")
	rootCmd.Flags().StringVarP(&resolution, "resolution", "r", "", "Video: override resolution (e.g. 1280x720)")
	rootCmd.Flags().IntVar(&fps, "fps", 0, "Video: override FPS cap")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress output")
	rootCmd.Flags().BoolVar(&useGPU, "gpu", false, "Use GPU hardware encoding (NVENC/AMF/QSV/VAAPI)")
	rootCmd.Flags().IntVar(&imgQuality, "quality", 75, "Image: quality 1-100 (default: 75)")
	rootCmd.Flags().IntVar(&audioBitrate, "bitrate", 0, "Audio: target bitrate in kbps (default: auto)")
	rootCmd.Flags().StringVar(&pdfQuality, "pdf-quality", "medium", "PDF: quality preset (low/medium/high)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func detectOrDownloadFFmpeg() (*ffmpeg.Paths, error) {
	paths, err := ffmpeg.Detect()
	if err == nil {
		return paths, nil
	}
	fmt.Println("ffmpeg not found on your system.")
	fmt.Print("Download it automatically? (~80MB) [Y/n] ")
	var answer string
	fmt.Scanln(&answer)
	if answer != "" && answer != "y" && answer != "Y" {
		return nil, fmt.Errorf("ffmpeg is required - install it manually and try again")
	}
	fmt.Println("Downloading ffmpeg...")
	progressCh := make(chan ffmpeg.DownloadProgress, 100)
	go func() {
		for p := range progressCh {
			if p.TotalBytes > 0 {
				pct := float64(p.BytesDownloaded) / float64(p.TotalBytes) * 100
				fmt.Printf("\rDownloading: %.0f%%", pct)
			}
		}
		fmt.Println()
	}()
	paths, err = ffmpeg.Download(progressCh)
	if err != nil {
		return nil, fmt.Errorf("failed to download ffmpeg: %w", err)
	}
	fmt.Println("ffmpeg installed successfully!")
	return paths, nil
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	if output != "" && len(args) > 1 {
		return fmt.Errorf("cannot use --output with multiple files")
	}

	paths, err := detectOrDownloadFFmpeg()
	if err != nil {
		return err
	}

	for i, inputPath := range args {
		if len(args) > 1 {
			fmt.Printf("\n[%d/%d] %s\n", i+1, len(args), filepath.Base(inputPath))
		}

		fileType := compress.DetectFileType(inputPath)

		switch fileType {
		case compress.FileTypeImage:
			err := processImage(paths, inputPath)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				continue
			}
		case compress.FileTypeAudio:
			err := processAudio(paths, inputPath)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				continue
			}
		case compress.FileTypePDF:
			err := processPDF(inputPath)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				continue
			}
		default:
			err := processVideo(paths, inputPath)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				continue
			}
		}

		config.AddRecent(inputPath)
	}

	return nil
}

func processVideo(paths *ffmpeg.Paths, inputPath string) error {
	presetName := compress.PresetName(preset)

	opts := compress.Options{
		InputPath:   inputPath,
		OutputPath:  output,
		PresetName:  presetName,
		TargetMB:    targetMB,
		Resolution:  resolution,
		FPSOverride: fps,
		UseGPU:      useGPU,
	}

	result, err := compress.Run(paths, opts, func(pass int, pct float64) {
		if !quiet {
			if runtime.GOOS == "darwin" || useGPU {
				fmt.Printf("\rEncoding (GPU): %.0f%%", pct)
			} else {
				fmt.Printf("\rPass %d/2: %.0f%%", pass, pct)
			}
		}
	})
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Println()
	}

	inputInfo := result.InputInfo
	outputInfo := result.OutputInfo
	fmt.Printf("Done! %dMB -> %dMB (%s %dx%d -> h264 %dx%d) -> %s\n",
		inputInfo.FileSizeMB, outputInfo.FileSizeMB,
		inputInfo.Codec, inputInfo.Width, inputInfo.Height,
		outputInfo.Width, outputInfo.Height,
		result.OutputPath,
	)
	return nil
}

func processImage(paths *ffmpeg.Paths, inputPath string) error {
	opts := compress.ImageOptions{
		InputPath: inputPath,
		OutputPath: output,
		Quality:   imgQuality,
		StripMeta: true,
	}

	if !quiet {
		fmt.Printf("Compressing image (quality=%d)...", imgQuality)
	}

	result, err := compress.RunImage(paths, opts, nil)
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Println()
	}

	fmt.Printf("Done! %s -> %s (%.1fx) -> %s\n",
		formatKB(result.InputSizeKB),
		formatKB(result.OutputSizeKB),
		safeDivide(result.InputSizeKB, result.OutputSizeKB),
		result.OutputPath,
	)
	return nil
}

func processAudio(paths *ffmpeg.Paths, inputPath string) error {
	opts := compress.AudioOptions{
		InputPath:  inputPath,
		OutputPath: output,
		Bitrate:    audioBitrate,
		StripMeta:  false,
	}

	result, err := compress.RunAudio(paths, opts, func(pct float64) {
		if !quiet {
			fmt.Printf("\rCompressing audio: %.0f%%", pct)
		}
	})
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Println()
	}

	fmt.Printf("Done! %s -> %s (%.1fx) -> %s\n",
		formatKB(result.InputSizeKB),
		formatKB(result.OutputSizeKB),
		safeDivide(result.InputSizeKB, result.OutputSizeKB),
		result.OutputPath,
	)
	if result.AlreadyOptimal {
		fmt.Println("File is already optimized at this quality level. Try a lower quality to reduce size.")
	}
	return nil
}

func processPDF(inputPath string) error {
	gsPath, err := ghostscript.Detect()
	if err != nil {
		return err
	}

	var quality ghostscript.PDFQuality
	switch pdfQuality {
	case "low":
		quality = ghostscript.QualityScreen
	case "medium":
		quality = ghostscript.QualityEbook
	case "high":
		quality = ghostscript.QualityPrinter
	default:
		quality = ghostscript.QualityEbook
	}

	opts := compress.PDFOptions{
		InputPath:  inputPath,
		OutputPath: output,
		Quality:    quality,
	}

	if !quiet {
		fmt.Printf("Compressing PDF (%s)...", ghostscript.QualityLabels[quality])
	}

	result, err := compress.RunPDF(gsPath, opts)
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Println()
	}

	fmt.Printf("Done! %s -> %s (%.1fx) -> %s\n",
		formatKB(result.InputSizeKB),
		formatKB(result.OutputSizeKB),
		safeDivide(result.InputSizeKB, result.OutputSizeKB),
		result.OutputPath,
	)
	return nil
}

func formatKB(kb int64) string {
	if kb >= 1024 {
		return fmt.Sprintf("%.1fMB", float64(kb)/1024.0)
	}
	return fmt.Sprintf("%dKB", kb)
}

func safeDivide(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}
