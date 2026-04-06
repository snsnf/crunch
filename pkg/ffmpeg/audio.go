package ffmpeg

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BuildAudioArgs builds ffmpeg arguments for audio compression.
func BuildAudioArgs(input, output string, bitrate, sampleRate, channels int, stripMeta bool) []string {
	args := []string{"-y", "-i", input}

	ext := strings.ToLower(filepath.Ext(output))

	// Pick encoder based on output format
	switch ext {
	case ".mp3":
		args = append(args, "-c:a", "libmp3lame")
		args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))
		// VBR quality for better results at target bitrate
		args = append(args, "-q:a", vbrQuality(bitrate))

	case ".aac", ".m4a":
		args = append(args, "-c:a", "aac")
		args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))

	case ".opus":
		args = append(args, "-c:a", "libopus")
		args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))
		args = append(args, "-vbr", "on")

	case ".ogg":
		args = append(args, "-c:a", "libvorbis")
		args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))

	case ".flac":
		// FLAC is lossless — compression_level controls speed/size tradeoff
		args = append(args, "-c:a", "flac")
		args = append(args, "-compression_level", "8")

	case ".wav":
		// WAV is uncompressed PCM — just copy or re-encode to reduce bit depth
		args = append(args, "-c:a", "pcm_s16le")

	case ".wma":
		args = append(args, "-c:a", "wmav2")
		args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))

	case ".aiff", ".aif":
		args = append(args, "-c:a", "pcm_s16be")

	default:
		// Generic: use AAC encoder as safe default
		args = append(args, "-c:a", "aac")
		args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))
	}

	if sampleRate > 0 {
		args = append(args, "-ar", fmt.Sprintf("%d", sampleRate))
	}

	if channels > 0 {
		args = append(args, "-ac", fmt.Sprintf("%d", channels))
	}

	// Strip video stream (album art) to save space
	args = append(args, "-vn")

	if stripMeta {
		args = append(args, "-map_metadata", "-1")
	}

	args = append(args, output)
	return args
}

// vbrQuality maps bitrate to LAME VBR quality (0=best, 9=worst).
func vbrQuality(bitrate int) string {
	switch {
	case bitrate >= 256:
		return "0"
	case bitrate >= 192:
		return "2"
	case bitrate >= 160:
		return "3"
	case bitrate >= 128:
		return "4"
	case bitrate >= 96:
		return "6"
	case bitrate >= 64:
		return "8"
	default:
		return "9"
	}
}
