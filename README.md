<p align="center">
  <img src="gui/build/appicon.png" width="160" height="160" alt="Crunch" />
</p>

<h1 align="center">Crunch</h1>

<p align="center">
  Compress videos, images, audio and PDFs without losing your mind.<br/>
  <strong>Desktop app with GUI + CLI. Offline. Fast. Free.</strong>
</p>

<p align="center">
  <a href="#install">Install</a> · <a href="#usage">Usage</a> · <a href="#the-whatsapp-preset">WhatsApp Preset</a> · <a href="#building-from-source">Build</a>
</p>

## The Problem

I needed to send a video on WhatsApp. Simple, right?

**Attempt 1:** Send directly. WhatsApp compresses it. Video looks like it was filmed through a potato.

**Attempt 2:** Use an online compressor. Upload 200MB, wait 10 minutes, download, send. WhatsApp compresses it AGAIN. Now it looks like a potato filmed through another potato.

**Attempt 3:** Try every compressor app out there. None of them understand what WhatsApp actually needs. The output either gets re-compressed on send or comes out looking worse than just sending the original.

**Attempt 4:** Ask AI for an ffmpeg command. Works once. Different video, different result. Every file needs a different command and you're back asking AI again.

**Attempt 5:** Build my own app. You're looking at it.

## What Crunch Does

Drop a file. Pick a size. Get a smaller file. That's it.

- **Video** - WhatsApp preset that actually produces WhatsApp-quality results. No double compression.
- **Image** - Quality-based compression with smart PNG palette optimization. Keeps your format, actually makes files smaller.
- **Audio** - Pick a quality level, get a smaller file. Revolutionary.
- **PDF** - Screen, Ebook, or Printer quality. Requires [Ghostscript](https://www.ghostscript.com/) installed separately.

## Install

### GUI App

**macOS (Homebrew):**
```bash
brew tap snsnf/crunch
brew install --cask crunch
```

**Windows:** Download [Crunch-Setup-windows-amd64.exe](https://github.com/snsnf/crunch/releases/latest) and run the installer.

**Linux / Manual download:** Grab the latest from the [Releases page](https://github.com/snsnf/crunch/releases/latest).

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | `Crunch-macos-arm64.zip` |
| macOS (Intel) | `Crunch-macos-amd64.zip` |
| Linux | `Crunch-linux-amd64.tar.gz` |

The GUI comes with ffmpeg bundled — no extra setup needed.

> **macOS note:** If you see "Apple could not verify", run:
> ```bash
> xattr -d com.apple.quarantine /path/to/Crunch.app
> ```

### CLI

**macOS / Linux (Homebrew):**
```bash
brew install snsnf/crunch/crunch-cli
```

**macOS / Linux (script):**
```bash
curl -sSL https://raw.githubusercontent.com/snsnf/crunch/main/install.sh | sh
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/snsnf/crunch/main/install.ps1 | iex
```

## Usage

### GUI

Open the app. Drop files. Click compress. Done.

The app figures out what type of file you dropped and shows the right settings:

| File Type | Settings |
|-----------|----------|
| Video | Preset (WhatsApp/Generic) + target size in MB |
| Image | Quality slider (1-100) |
| Audio | Quality (Low / Medium / High / Best) |
| PDF | Quality (Screen / Ebook / Printer) |

Mix different file types? No problem - it shows all relevant settings and compresses each file with its own type's settings.

### CLI

```bash
# Video - compress to 10MB for WhatsApp
crunch video.mp4 -t 10

# Video - generic preset, 30MB target
crunch video.mov -p generic -t 30

# Image - quality 60
crunch photo.png --quality 60

# Audio - 128kbps
crunch podcast.wav --bitrate 128

# PDF - ebook quality
crunch document.pdf --pdf-quality ebook

# Batch - compress everything
crunch *.mp4 -t 8
```

## Why Another Compressor?

| Feature | Crunch | Online Tools | Other Apps | Raw ffmpeg |
|---------|--------|-------------|------------|------------|
| Works offline | ✅ | ❌ | ✅ | ✅ |
| WhatsApp-optimized | ✅ | ❌ | ❌ | 🤷 if you know the flags |
| All file types | ✅ | ❌ | ❌ | ✅ |
| Hardware acceleration | ✅ | ❌ | ❌ | 🤷 |
| Pretty GUI | ✅ | ✅ | ✅ | ❌ |
| No PhD required | ✅ | ✅ | ✅ | ❌ |

## The WhatsApp Preset

This is why Crunch exists. The WhatsApp preset is specifically tuned so that:

1. The video is small enough to send as a media message
2. WhatsApp doesn't re-compress it into oblivion
3. The quality is actually watchable

## Tech Stack

- Go
- Wails (native desktop, not Electron)
- ffmpeg (bundled)
- Ghostscript for PDF (not bundled, app prompts to install if needed)
- Vanilla JS frontend

## The Mascot

The squished cat is called **Crunchy**. He was a normal cat until he went through the compressor. He's fine with it. Look at that face - he's happy :)

## Building from Source

```bash
# CLI
go build -o crunch .

# GUI (with bundled ffmpeg)
cd gui && bash build.sh

# GUI (without ffmpeg - uses system ffmpeg)
cd gui && wails build
```

Requires Go 1.26+, Node.js, and [Wails](https://wails.io).

## License

MIT
