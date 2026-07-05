# Homebrew formula for Thask CLI
# Usage: brew install kimgh06/thask/thask
class Thask < Formula
  desc "Dependency visualization CLI for AI-assisted development"
  homepage "https://github.com/kimgh06/Thask"
  version "0.6.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-arm64.tar.gz"
      sha256 "9ba8b2bd5fbeefda337def1e5c4cd6727f0033230ef52178fba6e3108923e6fa"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-amd64.tar.gz"
      sha256 "c9cc7ce0372026f7d03f64667b10fda390b3729c3c1aee0bd7cd82430fe9320f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-arm64.tar.gz"
      sha256 "94582e88032006a5c00618b6c7b85d02732c294fe87db16005c9c4740b614a7d"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-amd64.tar.gz"
      sha256 "f01088fd5b2571ea6fb19c9875700dee18723112a5a7b2e8458160d6c0f95904"
    end
  end

  def install
    bin.install "thask"
  end

  test do
    assert_match "thask", shell_output("#{bin}/thask version")
  end
end
