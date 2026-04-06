package compress

type PresetName string

const (
	PresetWhatsApp PresetName = "whatsapp"
	PresetGeneric  PresetName = "generic"
)

type Preset struct {
	Name           PresetName
	LandscapeW     int
	LandscapeH     int
	PortraitW      int
	PortraitH      int
	KeepResolution bool
	Profile        string
	Level          string
	FPSCap         int
	AudioBitrate   int
	X264Params     string
	Tag            string
	Brand          string
	DefaultTarget  int
	MaxTarget      int
}

var presets = map[PresetName]Preset{
	PresetWhatsApp: {
		Name:          PresetWhatsApp,
		LandscapeW:    1280,
		LandscapeH:    720,
		PortraitW:     540,
		PortraitH:     960,
		Profile:       "main",
		Level:         "3.1",
		FPSCap:        30,
		AudioBitrate:  64,
		X264Params:    "ref=3",
		Tag:           "avc1",
		Brand:         "mp42",
		DefaultTarget: 8,
	},
	PresetGeneric: {
		Name:           PresetGeneric,
		KeepResolution: true,
		Profile:        "main",
		FPSCap:         30,
		AudioBitrate:   128,
	},
}

func GetPreset(name PresetName) Preset {
	if p, ok := presets[name]; ok {
		return p
	}
	return presets[PresetGeneric]
}
