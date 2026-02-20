class RimeCli < Formula
  desc "Command-line interface for Rime text-to-speech"
  homepage "https://github.com/rimelabs/rime-cli"
  version "0.1.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-arm64.tar.gz"
      sha256 "cdccd9ab10206e081899ea861d57822c5c01bb4d772018fd79ca5878fab6d5d1"
    else
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-amd64.tar.gz"
      sha256 "2b726ed7b55d645cac89ac5f0841f40295638e60f8ae0141faa0633735c1b0cf"
    end
  end

  on_linux do
    url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-linux-amd64.tar.gz"
    sha256 "58a6027d107240fe2c9e6e005a292d420116b06f51930328c42268c0a88f29ed"
  end

  def install
    bin.install "rime"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/rime --version")
  end
end
