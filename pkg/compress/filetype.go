package compress

import (
	"path/filepath"
	"strings"
)

type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeVideo
	FileTypeImage
	FileTypeAudio
	FileTypePDF
)

var videoExts = map[string]bool{
	".mp4": true, ".mov": true, ".mkv": true, ".avi": true,
	".webm": true, ".flv": true, ".wmv": true, ".m4v": true,
	".ts": true, ".mts": true, ".3gp": true, ".ogv": true,
}

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".avif": true, ".heic": true, ".heif": true, ".tiff": true,
	".tif": true, ".bmp": true, ".gif": true, ".ico": true,
	".svg": true, ".jxl": true, ".qoi": true, ".raw": true,
	".cr2": true, ".nef": true, ".arw": true, ".dng": true,
}

var audioExts = map[string]bool{
	".mp3": true, ".wav": true, ".flac": true, ".aac": true,
	".ogg": true, ".opus": true, ".m4a": true, ".wma": true,
	".aiff": true, ".aif": true, ".alac": true, ".ape": true,
	".ac3": true, ".dts": true, ".pcm": true, ".amr": true,
	".wv": true, ".mka": true, ".ra": true, ".mid": true,
	".midi": true,
}

var pdfExts = map[string]bool{
	".pdf": true,
}

// AllSupportedExts returns all supported file extensions for the file picker.
func AllSupportedExts() map[string]bool {
	all := make(map[string]bool, len(videoExts)+len(imageExts)+len(audioExts)+len(pdfExts))
	for k := range videoExts {
		all[k] = true
	}
	for k := range imageExts {
		all[k] = true
	}
	for k := range audioExts {
		all[k] = true
	}
	for k := range pdfExts {
		all[k] = true
	}
	return all
}

func DetectFileType(path string) FileType {
	ext := strings.ToLower(filepath.Ext(path))
	if videoExts[ext] {
		return FileTypeVideo
	}
	if imageExts[ext] {
		return FileTypeImage
	}
	if audioExts[ext] {
		return FileTypeAudio
	}
	if pdfExts[ext] {
		return FileTypePDF
	}
	return FileTypeUnknown
}

// GroupByType splits a list of file paths by type.
func GroupByType(files []string) (video, image, audio, pdf []string) {
	for _, f := range files {
		switch DetectFileType(f) {
		case FileTypeVideo:
			video = append(video, f)
		case FileTypeImage:
			image = append(image, f)
		case FileTypeAudio:
			audio = append(audio, f)
		case FileTypePDF:
			pdf = append(pdf, f)
		}
	}
	return
}

// DominantType returns the file type of the majority of files.
func DominantType(files []string) FileType {
	v, i, a, p := GroupByType(files)
	counts := map[FileType]int{
		FileTypeVideo: len(v),
		FileTypeImage: len(i),
		FileTypeAudio: len(a),
		FileTypePDF:   len(p),
	}
	best := FileTypeVideo
	bestCount := 0
	for ft, c := range counts {
		if c > bestCount {
			best = ft
			bestCount = c
		}
	}
	return best
}
