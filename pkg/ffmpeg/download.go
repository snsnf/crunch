package ffmpeg

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"crunch/pkg/config"
)

type DownloadProgress struct {
	BytesDownloaded int64
	TotalBytes      int64
}

func downloadURLs() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"https://evermeet.cx/ffmpeg/getrelease/zip",
			"https://evermeet.cx/ffprobe/getrelease/zip",
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			return []string{"https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz"}
		}
		return []string{"https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz"}
	case "windows":
		return []string{"https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"}
	}
	return nil
}

func downloadURL() string {
	switch runtime.GOOS {
	case "darwin":
		return "https://evermeet.cx/ffmpeg/getrelease/zip"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz"
		}
		return "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz"
	case "windows":
		return "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
	}
	return ""
}

func Download(progressCh chan<- DownloadProgress) (*Paths, error) {
	urls := downloadURLs()
	if len(urls) == 0 {
		return nil, fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	destDir, err := config.FFmpegDir()
	if err != nil {
		return nil, err
	}

	for _, url := range urls {
		if err := downloadAndExtract(url, destDir, progressCh); err != nil {
			return nil, err
		}
		// Only use progressCh for the first download
		progressCh = nil
	}

	return findInLocal()
}

func downloadAndExtract(url, destDir string, progressCh chan<- DownloadProgress) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "crunch-ffmpeg-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	hasher := sha256.New()
	var reader io.Reader = resp.Body
	if progressCh != nil {
		reader = &progressReader{
			reader:     resp.Body,
			total:      resp.ContentLength,
			progressCh: progressCh,
		}
	}
	reader = io.TeeReader(reader, hasher)

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return fmt.Errorf("download interrupted: %w", err)
	}
	tmpFile.Close()

	if progressCh != nil {
		close(progressCh)
	}

	// Verify checksum if known
	gotHash := hex.EncodeToString(hasher.Sum(nil))
	if expectedHash, ok := knownHashes[url]; ok {
		if gotHash != expectedHash {
			return fmt.Errorf("checksum mismatch for %s:\n  expected: %s\n  got:      %s", url, expectedHash, gotHash)
		}
	}

	if strings.HasSuffix(url, ".zip") {
		return extractZip(tmpFile.Name(), destDir)
	} else if strings.HasSuffix(url, ".tar.xz") {
		return extractTarXz(tmpFile.Name(), destDir)
	}
	return extractTarGz(tmpFile.Name(), destDir)
}

// knownHashes maps download URLs to their expected SHA-256 checksums.
// Update these when upgrading ffmpeg versions.
// To get a hash: curl -L <url> | shasum -a 256
var knownHashes = map[string]string{
	// Populated as ffmpeg versions are pinned.
	// Example:
	// "https://evermeet.cx/ffmpeg/getrelease/zip": "abc123...",
}

type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	progressCh chan<- DownloadProgress
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.downloaded += int64(n)
	if pr.progressCh != nil {
		pr.progressCh <- DownloadProgress{
			BytesDownloaded: pr.downloaded,
			TotalBytes:      pr.total,
		}
	}
	return n, err
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if !isFFmpegBinary(name) {
			continue
		}
		outPath := filepath.Join(dest, name)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Base(hdr.Name)
		if !isFFmpegBinary(name) {
			continue
		}
		outPath := filepath.Join(dest, name)
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		_, err = io.Copy(outFile, tr)
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarXz(src, dest string) error {
	cmd := exec.Command("tar", "xf", src, "-C", dest, "--strip-components=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback: extract specific binaries
		cmd2 := exec.Command("tar", "xf", src, "-C", dest)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return fmt.Errorf("tar extraction failed: %w\n%s\n%s", err2, out, out2)
		}
	}
	// Move ffmpeg/ffprobe binaries to dest root if nested
	filepath.Walk(dest, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if isFFmpegBinary(info.Name()) && filepath.Dir(path) != dest {
			os.Rename(path, filepath.Join(dest, info.Name()))
		}
		return nil
	})
	return nil
}

func isFFmpegBinary(name string) bool {
	name = strings.ToLower(name)
	return name == "ffmpeg" || name == "ffprobe" ||
		name == "ffmpeg.exe" || name == "ffprobe.exe"
}
