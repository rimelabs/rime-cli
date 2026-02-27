class RimeCli < Formula
  desc "Command-line interface for Rime text-to-speech"
  homepage "https://github.com/rimelabs/rime-cli"
  version "0.2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-arm64.tar.gz"
      sha256 "b3d35f8c126dc1282c8a1d09b002db7b09ee9be044d1cc75348a6f898e26410a"
    else
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-amd64.tar.gz"
      sha256 "96c5ed64cccfe38ee7981c1e3a9c7f73e9c0ad27584fc309efa6289e085b4573"
    end
  end

  on_linux do
    url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-linux-amd64.tar.gz"
    sha256 "1c367a810974d9dea3c8cde9e36247cb672e3496bac02483e9e6ce1139d1c053"
  end

  def install
    bin.install "rime"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/rime --version")
  end
end
