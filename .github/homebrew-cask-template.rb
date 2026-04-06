cask "crunch" do
  version "VERSION"

  if Hardware::CPU.arm?
    url "https://github.com/snsnf/crunch/releases/download/v#{version}/Crunch-macos-arm64.zip"
    sha256 "SHA256_ARM64"
  else
    url "https://github.com/snsnf/crunch/releases/download/v#{version}/Crunch-macos-amd64.zip"
    sha256 "SHA256_AMD64"
  end

  name "Crunch"
  desc "Fast media compressor — video, image, audio, and PDF"
  homepage "https://github.com/snsnf/crunch"

  app "Crunch.app"

  zap trash: [
    "~/.config/crunch",
  ]
end
