package ffmpeg

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BuildPNGPalettegenArgs generates a palette PNG from the source image (PNG pass 1).
func BuildPNGPalettegenArgs(input, paletteOut string, colors int) []string {
	return []string{
		"-y", "-i", input,
		"-vf", fmt.Sprintf("palettegen=max_colors=%d:stats_mode=diff", colors),
		paletteOut,
	}
}

// BuildPNGPaletteuseArgs applies a palette to produce the final quantized PNG (PNG pass 2).
func BuildPNGPaletteuseArgs(input, paletteIn, output string) []string {
	return []string{
		"-y", "-i", input, "-i", paletteIn,
		"-lavfi", "paletteuse=dither=sierra2_4a",
		"-compression_level", "9",
		output,
	}
}

// BuildImageArgs builds ffmpeg arguments for image compression.
// Quality is 1-100 (100 = best). Mapping to encoder-specific values is handled here.
func BuildImageArgs(input, output string, quality, maxWidth, maxHeight int, stripMeta bool) []string {
	args := []string{"-y", "-i", input}

	// Scale filter if max dimensions are set
	var filters []string
	if maxWidth > 0 || maxHeight > 0 {
		w := maxWidth
		h := maxHeight
		if w == 0 {
			w = -1
		}
		if h == 0 {
			h = -1
		}
		// scale only if image exceeds max, preserve aspect ratio
		filters = append(filters, fmt.Sprintf("scale='min(%d,iw)':min(%d,ih):force_original_aspect_ratio=decrease", w, h))
	}

	// Encoder-specific quality settings based on output format
	ext := strings.ToLower(filepath.Ext(output))
	switch ext {
	case ".jpg", ".jpeg":
		// ffmpeg MJPEG quality: -q:v 2 (best) to 31 (worst)
		q := 2 + (100-quality)*29/100
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
		args = append(args, "-q:v", fmt.Sprintf("%d", q))

	case ".png":
		// PNG is lossless — we can't reduce quality like JPEG.
		// Strategy: quantize to a reduced palette for real size savings.
		// Map quality to number of colors: 100=256, 75=192, 50=128, 25=64, 1=32
		colors := 32 + (quality * 224 / 100)
		if colors > 256 {
			colors = 256
		}

		// Build a proper palettegen/paletteuse filter pipeline
		scaleFilter := ""
		if len(filters) > 0 {
			scaleFilter = strings.Join(filters, ",") + ","
		}
		pngFilter := fmt.Sprintf("%ssplit[s0][s1];[s0]palettegen=max_colors=%d:stats_mode=diff[p];[s1][p]paletteuse=dither=sierra2_4a", scaleFilter, colors)
		args = append(args, "-vf", pngFilter, "-compression_level", "9")

	case ".webp":
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
		// WebP quality 0-100
		args = append(args, "-quality", fmt.Sprintf("%d", quality))
		if quality < 100 {
			args = append(args, "-preset", "photo")
		}

	case ".avif":
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
		// AVIF uses CRF: 0 (lossless) to 63 (worst). Map quality 100->0, 1->56
		crf := (100 - quality) * 56 / 100
		args = append(args, "-c:v", "libaom-av1", "-crf", fmt.Sprintf("%d", crf), "-still-picture", "1")

	case ".heic", ".heif":
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
		// HEIF via libx265
		crf := (100 - quality) * 45 / 100
		args = append(args, "-c:v", "libx265", "-crf", fmt.Sprintf("%d", crf), "-frames:v", "1")

	case ".tiff", ".tif":
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
		args = append(args, "-compression_algo", "deflate")

	case ".bmp":
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}

	case ".gif":
		// GIF: optimize palette, reduce colors based on quality
		colors := 256
		if quality < 75 {
			colors = 128
		}
		if quality < 50 {
			colors = 64
		}
		if quality < 25 {
			colors = 32
		}
		scaleFilter := ""
		if len(filters) > 0 {
			scaleFilter = strings.Join(filters, ",") + ","
		}
		args = append(args, "-vf",
			fmt.Sprintf("%ssplit[s0][s1];[s0]palettegen=max_colors=%d[p];[s1][p]paletteuse=dither=bayer", scaleFilter, colors))

	case ".jxl":
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
		// JPEG XL via libjxl
		dist := float64(100-quality) * 15.0 / 100.0
		args = append(args, "-c:v", "libjxl", "-distance", fmt.Sprintf("%.1f", dist))

	default:
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
		// For unknown formats, try generic quality
		q := 2 + (100-quality)*29/100
		args = append(args, "-q:v", fmt.Sprintf("%d", q))
	}

	if stripMeta {
		args = append(args, "-map_metadata", "-1")
	}

	args = append(args, output)
	return args
}
