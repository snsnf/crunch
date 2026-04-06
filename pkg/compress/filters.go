package compress

import (
	"fmt"
	"regexp"
	"strings"

	"crunch/pkg/probe"
)

var validResolution = regexp.MustCompile(`^\d{1,5}[x:]\d{1,5}$`)

func BuildFilterChain(info *probe.VideoInfo, preset Preset, resOverride string, fpsOverride int) string {
	var filters []string

	if !validResolution.MatchString(resOverride) {
		resOverride = ""
	}

	if resOverride != "" {
		filters = append(filters, fmt.Sprintf("scale=%s", strings.Replace(resOverride, "x", ":", 1)))
	} else if !preset.KeepResolution {
		if info.Width > info.Height {
			if info.Width > preset.LandscapeW || info.Height > preset.LandscapeH {
				filters = append(filters, fmt.Sprintf("scale=%d:%d", preset.LandscapeW, preset.LandscapeH))
			}
		} else {
			if info.Width > preset.PortraitW || info.Height > preset.PortraitH {
				filters = append(filters, fmt.Sprintf("scale=%d:%d", preset.PortraitW, preset.PortraitH))
			}
		}
	}

	fpsCap := preset.FPSCap
	if fpsOverride > 0 {
		fpsCap = fpsOverride
	}
	if fpsCap > 0 && info.FPS > float64(fpsCap) {
		filters = append(filters, fmt.Sprintf("fps=%d", fpsCap))
	}

	filters = append(filters, "format=yuv420p")

	return strings.Join(filters, ",")
}
