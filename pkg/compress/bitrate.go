package compress

import (
	"fmt"
	"math"
)

type BitrateResult struct {
	VideoBitrate int
	AudioBitrate int
}

func CalculateBitrate(targetMB int, durationSec float64, audioBitrate int) (*BitrateResult, error) {
	if durationSec <= 0 {
		return nil, fmt.Errorf("invalid duration: %.1f seconds", durationSec)
	}
	targetKB := targetMB * 1024
	videoBitrate := int(float64(targetKB*8)/durationSec) - audioBitrate

	if videoBitrate <= 0 {
		minMB := int(math.Ceil(float64(audioBitrate) * durationSec / 8 / 1024))
		return nil, fmt.Errorf("target %dMB is too small for a %.0fs video (minimum: %dMB)", targetMB, durationSec, minMB+1)
	}

	return &BitrateResult{
		VideoBitrate: videoBitrate,
		AudioBitrate: audioBitrate,
	}, nil
}
