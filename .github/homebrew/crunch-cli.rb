class CrunchCli < Formula
  desc "Fast media compressor CLI — video, image, audio, and PDF"
  homepage "https://github.com/snsnf/crunch"
  version "VERSION_PLACEHOLDER"

  on_macos do
    on_arm do
      url "https://github.com/snsnf/crunch/releases/download/vVERSION_PLACEHOLDER/crunch-cli-macos-arm64.zip"
      sha256 "SHA_CLI_ARM64_PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/snsnf/crunch/releases/download/vVERSION_PLACEHOLDER/crunch-cli-macos-amd64.zip"
      sha256 "SHA_CLI_AMD64_PLACEHOLDER"
    end
  end

  on_linux do
    url "https://github.com/snsnf/crunch/releases/download/vVERSION_PLACEHOLDER/crunch-cli-linux-amd64.tar.gz"
    sha256 "SHA_CLI_LINUX_PLACEHOLDER"
  end

  def install
    bin.install "crunch"
  end

  test do
    assert_match "crunch", shell_output("#{bin}/crunch --help")
  end
end
