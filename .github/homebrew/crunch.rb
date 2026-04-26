cask "crunch" do
  version "VERSION_PLACEHOLDER"

  on_arm do
    url "https://github.com/snsnf/crunch/releases/download/vVERSION_PLACEHOLDER/Crunch-macos-arm64.zip"
    sha256 "SHA_ARM64_PLACEHOLDER"
  end
  on_intel do
    url "https://github.com/snsnf/crunch/releases/download/vVERSION_PLACEHOLDER/Crunch-macos-amd64.zip"
    sha256 "SHA_AMD64_PLACEHOLDER"
  end

  name "Crunch"
  desc "Fast media compressor — video, image, audio, and PDF"
  homepage "https://github.com/snsnf/crunch"

  app "Crunch.app"

  zap trash: [
    "~/.config/crunch",
  ]
end
