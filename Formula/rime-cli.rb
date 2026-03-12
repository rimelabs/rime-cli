class RimeCli < Formula
  desc "Command-line interface for Rime text-to-speech"
  homepage "https://github.com/rimelabs/rime-cli"
  version "0.4.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-arm64.tar.gz"
      sha256 "aa73fcf8ff457d3b65e4462f013a0dbc26cc04ad7c01c78c6bdbe00b440d582c"
    else
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-amd64.tar.gz"
      sha256 "133c0cbf80dd18fcd8da31a8c7cf7185e6840b542ba7dc5492140a575565c1ef"
    end
  end

  on_linux do
    url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-linux-amd64.tar.gz"
    sha256 "96d8d44c0a7a21c62ca098abf8f9fa1b30ef05d3fbbce02a6754593e0c6ec01d"
  end

  def install
    bin.install "rime"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/rime --version")
  end
end
