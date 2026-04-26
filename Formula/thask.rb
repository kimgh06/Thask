# Homebrew formula for Thask CLI
# Usage: brew install kimgh06/thask/thask
class Thask < Formula
  desc "Dependency visualization CLI for AI-assisted development"
  homepage "https://github.com/kimgh06/Thask"
  version "0.5.6"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-arm64.tar.gz"
      sha256 "e0e275a38c71664030c63ec05d61af429182082a5ee90d2580bdacacf4b948a6"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-amd64.tar.gz"
      sha256 "2a0150671951e1cee7bcf99eefdfaea40a63fca20134303c5befa73b646ca76a"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-arm64.tar.gz"
      sha256 "148b63eb3e637b176642100eb9f9696b72bda35f01f26a0d6a8f9a78e6976fb4"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-amd64.tar.gz"
      sha256 "586efef566cf13a910fc95b973198e28d02249a198fce644b8b4f72f5ffc0ef7"
    end
  end

  def install
    bin.install "thask"
  end

  test do
    assert_match "thask", shell_output("#{bin}/thask version")
  end
end
