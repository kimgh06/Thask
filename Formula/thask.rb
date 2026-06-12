# Homebrew formula for Thask CLI
# Usage: brew install kimgh06/thask/thask
class Thask < Formula
  desc "Dependency visualization CLI for AI-assisted development"
  homepage "https://github.com/kimgh06/Thask"
  version "0.5.13"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-arm64.tar.gz"
      sha256 "8b429815570aba8f9486754644e8c227049c94e100427b10cbae7c30b1d79e2e"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-amd64.tar.gz"
      sha256 "4280c8ce13dab28096b5aa9510b3ebeb3e1595c2894c5a6655b10fa6ce3d57ab"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-arm64.tar.gz"
      sha256 "0c9673df306aceac20d2588c7ff6352ea1e39de681e472adfbfd710b64c0b43e"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-amd64.tar.gz"
      sha256 "ddbb8dc14afb48d660b76fce389cd5e9144ffba0a97e3264249e4337c0b0d5d9"
    end
  end

  def install
    bin.install "thask"
  end

  test do
    assert_match "thask", shell_output("#{bin}/thask version")
  end
end
