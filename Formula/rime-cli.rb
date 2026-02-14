class RimeCli < Formula
  desc "Command-line interface for Rime text-to-speech"
  homepage "https://github.com/rimelabs/rime-cli"
  version "0.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rimelabs/rime-cli/releases/download/v0.0.0/rime-darwin-arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    else
      url "https://github.com/rimelabs/rime-cli/releases/download/v0.0.0/rime-darwin-amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    url "https://github.com/rimelabs/rime-cli/releases/download/v0.0.0/rime-linux-amd64.tar.gz"
    sha256 "0000000000000000000000000000000000000000000000000000000000000000"
  end

  def install
    bin.install "rime"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/rime --version")
  end
end
