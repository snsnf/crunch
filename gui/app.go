package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"crunch/pkg/compress"
	"crunch/pkg/ffmpeg"
	"crunch/pkg/ghostscript"
	"crunch/pkg/probe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	ffmpegPaths      *ffmpeg.Paths
	ghostscriptPath  string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	paths, err := ffmpeg.Detect()
	if err != nil {
		// Notify frontend that ffmpeg needs downloading
		runtime.EventsEmit(ctx, "ffmpeg:missing", true)
	} else {
		a.ffmpegPaths = paths
	}

	gsPath, _ := ghostscript.Detect()
	a.ghostscriptPath = gsPath

	// Handle file drag & drop
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		var files []FileInfo
		for _, p := range paths {
			files = append(files, a.getFileInfo(p))
		}
		runtime.EventsEmit(ctx, "files:dropped", files)
	})
}

// DownloadFFmpeg downloads ffmpeg/ffprobe and returns true on success.
func (a *App) DownloadFFmpeg() bool {
	progressCh := make(chan ffmpeg.DownloadProgress, 100)
	go func() {
		for p := range progressCh {
			if p.TotalBytes > 0 {
				pct := float64(p.BytesDownloaded) / float64(p.TotalBytes) * 100
				runtime.EventsEmit(a.ctx, "ffmpeg:progress", pct)
			}
		}
	}()

	paths, err := ffmpeg.Download(progressCh)
	if err != nil {
		runtime.EventsEmit(a.ctx, "ffmpeg:error", err.Error())
		return false
	}

	a.ffmpegPaths = paths
	runtime.EventsEmit(a.ctx, "ffmpeg:ready", true)
	return true
}

// HasFFmpeg returns whether ffmpeg is available.
func (a *App) HasFFmpeg() bool {
	return a.ffmpegPaths != nil
}

// HasGhostscript returns whether ghostscript is available.
func (a *App) HasGhostscript() bool {
	return a.ghostscriptPath != ""
}

// FileInfo holds metadata about a media file for the frontend.
type FileInfo struct {
	Path       string  `json:"path"`
	Name       string  `json:"name"`
	SizeMB     int64   `json:"sizeMB"`
	SizeKB     int64   `json:"sizeKB"`
	Duration   float64 `json:"duration"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Codec      string  `json:"codec"`
	FPS        float64 `json:"fps"`
	IsPortrait bool    `json:"isPortrait"`
	FileType   string  `json:"fileType"`
}

// SelectFiles opens a native file dialog and returns selected file info.
func (a *App) SelectFiles() ([]FileInfo, error) {
	selection, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select files to compress",
		Filters: []runtime.FileFilter{
			{DisplayName: "Media Files", Pattern: "*.mp4;*.mov;*.avi;*.mkv;*.webm;*.m4v;*.wmv;*.flv;*.jpg;*.jpeg;*.png;*.webp;*.gif;*.bmp;*.tiff;*.mp3;*.wav;*.aac;*.ogg;*.flac;*.m4a;*.wma;*.pdf"},
			{DisplayName: "Videos", Pattern: "*.mp4;*.mov;*.avi;*.mkv;*.webm;*.m4v;*.wmv;*.flv"},
			{DisplayName: "Images", Pattern: "*.jpg;*.jpeg;*.png;*.webp;*.gif;*.bmp;*.tiff"},
			{DisplayName: "Audio", Pattern: "*.mp3;*.wav;*.aac;*.ogg;*.flac;*.m4a;*.wma"},
			{DisplayName: "PDF", Pattern: "*.pdf"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, path := range selection {
		info := a.getFileInfo(path)
		files = append(files, info)
	}
	return files, nil
}

func (a *App) getFileInfo(path string) FileInfo {
	ft := compress.DetectFileType(path)
	info := FileInfo{
		Path:     path,
		Name:     filepath.Base(path),
		FileType: fileTypeString(ft),
	}

	stat, err := os.Stat(path)
	if err == nil {
		info.SizeKB = stat.Size() / 1024
		info.SizeMB = stat.Size() / (1024 * 1024)
	}

	if ft == compress.FileTypeVideo && a.ffmpegPaths != nil {
		if videoInfo, err := probe.Run(a.ffmpegPaths.FFprobe, path); err == nil {
			info.SizeMB = videoInfo.FileSizeMB
			info.Duration = videoInfo.Duration
			info.Width = videoInfo.Width
			info.Height = videoInfo.Height
			info.Codec = videoInfo.Codec
			info.FPS = videoInfo.FPS
			info.IsPortrait = videoInfo.Height > videoInfo.Width
		}
	}

	return info
}

func fileTypeString(ft compress.FileType) string {
	switch ft {
	case compress.FileTypeVideo:
		return "video"
	case compress.FileTypeImage:
		return "image"
	case compress.FileTypeAudio:
		return "audio"
	case compress.FileTypePDF:
		return "pdf"
	default:
		return "unknown"
	}
}

// CompressOptions holds settings from the frontend.
type CompressOptions struct {
	Files     []string `json:"files"`
	OutputDir string   `json:"outputDir"`
	// Video settings
	Preset   string `json:"preset"`
	TargetMB int    `json:"targetMB"`
	// Image settings
	ImageQuality int `json:"imageQuality"`
	// Audio settings
	AudioBitrate int `json:"audioBitrate"`
	// PDF settings
	PDFQuality string `json:"pdfQuality"`
}

// CompressResult holds the result of a compression.
type CompressResult struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	InputSize  string `json:"inputSize"`
	OutputSize string `json:"outputSize"`
	InputRes   string `json:"inputRes"`
	OutputRes  string `json:"outputRes"`
	FileType   string `json:"fileType"`
	Error      string `json:"error,omitempty"`
	Note       string `json:"note,omitempty"`
}

// Compress runs compression on the given files and emits progress events.
func (a *App) Compress(opts CompressOptions) []CompressResult {
	var results []CompressResult

	for i, filePath := range opts.Files {
		runtime.EventsEmit(a.ctx, "compress:file", map[string]interface{}{
			"index": i,
			"total": len(opts.Files),
			"name":  filepath.Base(filePath),
		})

		result := a.compressOne(filePath, opts, i, len(opts.Files))
		results = append(results, result)
	}

	runtime.EventsEmit(a.ctx, "compress:done", results)
	return results
}

// outputPathInDir returns an output path inside the chosen directory, or empty for default.
func outputPathInDir(inputPath, dir, ext string) string {
	if dir == "" {
		return ""
	}
	// Validate output directory is an absolute path that exists
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return ""
	}
	dir = absDir

	base := filepath.Base(inputPath)
	baseNoExt := base[:len(base)-len(filepath.Ext(base))]
	if ext == "" {
		ext = filepath.Ext(base)
	}
	return filepath.Join(dir, baseNoExt+"_crunched"+ext)
}

func (a *App) compressOne(filePath string, opts CompressOptions, index, total int) CompressResult {
	ft := compress.DetectFileType(filePath)

	switch ft {
	case compress.FileTypeVideo:
		return a.compressVideo(filePath, opts, index, total)
	case compress.FileTypeImage:
		return a.compressImage(filePath, opts, index, total)
	case compress.FileTypeAudio:
		return a.compressAudio(filePath, opts, index, total)
	case compress.FileTypePDF:
		return a.compressPDF(filePath, opts, index, total)
	default:
		return CompressResult{InputPath: filePath, Error: "unsupported file type"}
	}
}

func (a *App) compressVideo(filePath string, opts CompressOptions, index, total int) CompressResult {
	if a.ffmpegPaths == nil {
		return CompressResult{InputPath: filePath, Error: "ffmpeg not available"}
	}

	copts := compress.Options{
		InputPath:  filePath,
		OutputPath: outputPathInDir(filePath, opts.OutputDir, ".mp4"),
		PresetName: compress.PresetName(opts.Preset),
		TargetMB:   opts.TargetMB,
	}

	result, err := compress.Run(a.ffmpegPaths, copts, func(pass int, pct float64) {
		runtime.EventsEmit(a.ctx, "compress:progress", map[string]interface{}{
			"index":   index,
			"total":   total,
			"pass":    pass,
			"percent": pct,
		})
	})
	if err != nil {
		return CompressResult{InputPath: filePath, FileType: "video", Error: err.Error()}
	}

	return CompressResult{
		InputPath:  filePath,
		OutputPath: result.OutputPath,
		InputSize:  fmt.Sprintf("%dMB", result.InputInfo.FileSizeMB),
		OutputSize: fmt.Sprintf("%dMB", result.OutputInfo.FileSizeMB),
		InputRes:   fmt.Sprintf("%dx%d", result.InputInfo.Width, result.InputInfo.Height),
		OutputRes:  fmt.Sprintf("%dx%d", result.OutputInfo.Width, result.OutputInfo.Height),
		FileType:   "video",
	}
}

func (a *App) compressImage(filePath string, opts CompressOptions, index, total int) CompressResult {
	if a.ffmpegPaths == nil {
		return CompressResult{InputPath: filePath, Error: "ffmpeg not available"}
	}

	quality := opts.ImageQuality
	if quality <= 0 {
		quality = 75
	}

	result, err := compress.RunImage(a.ffmpegPaths, compress.ImageOptions{
		InputPath:  filePath,
		OutputPath: outputPathInDir(filePath, opts.OutputDir, ""),
		Quality:    quality,
		StripMeta:  true,
	}, func(pct float64) {
		runtime.EventsEmit(a.ctx, "compress:progress", map[string]interface{}{
			"index": index, "total": total, "pass": 1, "percent": pct,
		})
	})
	if err != nil {
		return CompressResult{InputPath: filePath, FileType: "image", Error: err.Error()}
	}

	return CompressResult{
		InputPath:  filePath,
		OutputPath: result.OutputPath,
		InputSize:  formatKB(result.InputSizeKB),
		OutputSize: formatKB(result.OutputSizeKB),
		FileType:   "image",
	}
}

func (a *App) compressAudio(filePath string, opts CompressOptions, index, total int) CompressResult {
	if a.ffmpegPaths == nil {
		return CompressResult{InputPath: filePath, Error: "ffmpeg not available"}
	}

	result, err := compress.RunAudio(a.ffmpegPaths, compress.AudioOptions{
		InputPath:  filePath,
		OutputPath: outputPathInDir(filePath, opts.OutputDir, ""),
		Bitrate:    opts.AudioBitrate,
	}, func(pct float64) {
		runtime.EventsEmit(a.ctx, "compress:progress", map[string]interface{}{
			"index":   index,
			"total":   total,
			"pass":    1,
			"percent": pct,
		})
	})
	if err != nil {
		return CompressResult{InputPath: filePath, FileType: "audio", Error: err.Error()}
	}

	res := CompressResult{
		InputPath:  filePath,
		OutputPath: result.OutputPath,
		InputSize:  formatKB(result.InputSizeKB),
		OutputSize: formatKB(result.OutputSizeKB),
		FileType:   "audio",
	}
	if result.AlreadyOptimal {
		res.Note = "Already optimized at this quality. Try a lower quality to reduce size."
	}
	return res
}

func (a *App) compressPDF(filePath string, opts CompressOptions, index, total int) CompressResult {
	if a.ghostscriptPath == "" {
		return CompressResult{InputPath: filePath, FileType: "pdf", Error: "ghostscript not found — install from ghostscript.com"}
	}

	runtime.EventsEmit(a.ctx, "compress:progress", map[string]interface{}{
		"index": index, "total": total, "pass": 1, "percent": 0,
	})

	quality := ghostscript.QualityEbook
	switch opts.PDFQuality {
	case "screen":
		quality = ghostscript.QualityScreen
	case "ebook":
		quality = ghostscript.QualityEbook
	case "printer":
		quality = ghostscript.QualityPrinter
	case "prepress":
		quality = ghostscript.QualityPrepress
	}

	result, err := compress.RunPDF(a.ghostscriptPath, compress.PDFOptions{
		InputPath:  filePath,
		OutputPath: outputPathInDir(filePath, opts.OutputDir, ".pdf"),
		Quality:    quality,
	})
	if err != nil {
		return CompressResult{InputPath: filePath, FileType: "pdf", Error: err.Error()}
	}

	runtime.EventsEmit(a.ctx, "compress:progress", map[string]interface{}{
		"index": index, "total": total, "pass": 1, "percent": 100,
	})

	return CompressResult{
		InputPath:  filePath,
		OutputPath: result.OutputPath,
		InputSize:  formatKB(result.InputSizeKB),
		OutputSize: formatKB(result.OutputSizeKB),
		FileType:   "pdf",
	}
}

func formatKB(kb int64) string {
	if kb >= 1024 {
		return fmt.Sprintf("%.1fMB", float64(kb)/1024.0)
	}
	return fmt.Sprintf("%dKB", kb)
}

// GetVersion returns the app version string.
func (a *App) GetVersion() string {
	return "1.0.0"
}

// SelectOutputDir opens a directory picker dialog.
func (a *App) SelectOutputDir() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select output directory",
	})
}

// RevealFile opens and selects the file in Finder/Explorer.
func (a *App) RevealFile(path string) {
	// Validate path exists and is a regular file
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}

	switch goruntime.GOOS {
	case "darwin":
		exec.Command("open", "-R", "--", path).Start()
	case "windows":
		exec.Command("explorer", "/select,", path).Start()
	default:
		exec.Command("xdg-open", "--", filepath.Dir(path)).Start()
	}
}
