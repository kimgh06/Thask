# Homebrew formula for Thask CLI
# Usage: brew install kimgh06/thask/thask
class Thask < Formula
  desc "Dependency visualization CLI for AI-assisted development"
  homepage "https://github.com/kimgh06/Thask"
  version "0.5.4"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-arm64.tar.gz"
      sha256 "b19c59c159d3a31e163646abd1c3f58af5a104451ff300406be2ef1577ac337d"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-darwin-amd64.tar.gz"
      sha256 "530583fbf4a35861fd7a580fc098da97e59644b3b7570aab6b468a75d5f23172"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-arm64.tar.gz"
      sha256 "6b42ea86347608e46dd33319974313fe2394aa62f75b480ced715ed3643d98b0"
    else
      url "https://github.com/kimgh06/Thask/releases/download/v#{version}/thask-linux-amd64.tar.gz"
      sha256 "eefa4200d441365c05f8c4d5a46871538ed8669b1d3cb4acc3251091618b435c"
    end
  end

  def install
    bin.install "thask"
  end

  test do
    assert_match "thask", shell_output("#{bin}/thask version")
  end
end
