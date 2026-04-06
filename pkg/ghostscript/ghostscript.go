package ghostscript

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// PDFQuality maps to Ghostscript PDFSETTINGS.
type PDFQuality string

const (
	QualityScreen   PDFQuality = "/screen"   // 72 dpi — smallest, screen viewing
	QualityEbook    PDFQuality = "/ebook"    // 150 dpi — good balance
	QualityPrinter  PDFQuality = "/printer"  // 300 dpi — high quality print
	QualityPrepress PDFQuality = "/prepress" // 300 dpi — maximum quality
)

var QualityLabels = map[PDFQuality]string{
	QualityScreen:   "Screen (72 dpi)",
	QualityEbook:    "Ebook (150 dpi)",
	QualityPrinter:  "Printer (300 dpi)",
	QualityPrepress: "Prepress (300 dpi)",
}

var Qualities = []PDFQuality{QualityScreen, QualityEbook, QualityPrinter, QualityPrepress}

// Detect finds Ghostscript on the system. Returns the path or error.
func Detect() (string, error) {
	// Try common binary names
	names := []string{"gs"}
	if runtime.GOOS == "windows" {
		names = []string{"gswin64c", "gswin32c", "gs"}
	}

	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	// Try common install locations
	var dirs []string
	switch runtime.GOOS {
	case "darwin":
		dirs = []string{"/opt/homebrew/bin", "/usr/local/bin"}
	case "linux":
		dirs = []string{"/usr/bin", "/usr/local/bin"}
	case "windows":
		dirs = []string{
			`C:\Program Files\gs\bin`,
			`C:\Program Files (x86)\gs\bin`,
		}
	}

	for _, dir := range dirs {
		for _, name := range names {
			path := dir + "/" + name
			if runtime.GOOS == "windows" {
				path = dir + `\` + name + ".exe"
			}
			if err := exec.Command(path, "--version").Run(); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("ghostscript not found — install it: brew install ghostscript (macOS), apt install ghostscript (Linux), or download from ghostscript.com (Windows)")
}

// Compress runs Ghostscript to compress a PDF.
func Compress(gsPath, input, output string, quality PDFQuality, imageDPI int) error {
	// Reject paths that could trigger Ghostscript pipe execution.
	if strings.ContainsAny(input, "%|") {
		return fmt.Errorf("invalid input path: must not contain '%%' or '|'")
	}
	if strings.ContainsAny(output, "%|") {
		return fmt.Errorf("invalid output path: must not contain '%%' or '|'")
	}

	args := []string{
		"-dSAFER",
		"-sDEVICE=pdfwrite",
		"-dCompatibilityLevel=1.4",
		fmt.Sprintf("-dPDFSETTINGS=%s", quality),
		"-dNOPAUSE",
		"-dBATCH",
		"-dQUIET",
	}

	// Custom DPI override (0 = use quality preset default)
	if imageDPI > 0 {
		args = append(args,
			fmt.Sprintf("-dColorImageResolution=%d", imageDPI),
			fmt.Sprintf("-dGrayImageResolution=%d", imageDPI),
			fmt.Sprintf("-dMonoImageResolution=%d", imageDPI),
			"-dDownsampleColorImages=true",
			"-dDownsampleGrayImages=true",
			"-dDownsampleMonoImages=true",
		)
	}

	// Optimize for size
	args = append(args,
		"-dCompressFonts=true",
		"-dSubsetFonts=true",
		"-dDetectDuplicateImages=true",
		"-dColorImageDownsampleType=/Bicubic",
		"-dGrayImageDownsampleType=/Bicubic",
	)

	args = append(args,
		fmt.Sprintf("-sOutputFile=%s", output),
		input,
	)

	cmd := exec.Command(gsPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if outStr != "" {
			return fmt.Errorf("ghostscript error: %w\n%s", err, outStr)
		}
		return fmt.Errorf("ghostscript error: %w", err)
	}
	return nil
}
