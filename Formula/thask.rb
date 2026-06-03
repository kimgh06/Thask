# Homebrew formula for Thask CLI
# Usage: brew install kimgh06/thask/thask
class Thask < Formula
  desc "Dependency visualization CLI for AI-assisted development"
  homepage "https://github.com/kimgh06/Thask"
  version "0.5.12"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-arm64.tar.gz"
      sha256 "94eb2f6b7e59986a0c484ff529a9e9a4fc245cef4472db2e1b545be230bca107"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-amd64.tar.gz"
      sha256 "ca14e09cf4a5d2317c715bfe2d2edf83819549536d852d22c18377dd404b2492"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-arm64.tar.gz"
      sha256 "a5ac70c00e2621c7d016dbf5a0a5ba0e7bb14a22e29923cd4abb5bdd754b6545"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-amd64.tar.gz"
      sha256 "347d8d25f25ca6f9b82f139224e709161e47b1e2d398c1aede2b20087fde59ed"
    end
  end

  def install
    bin.install "thask"
  end

  test do
    assert_match "thask", shell_output("#{bin}/thask version")
  end
end
