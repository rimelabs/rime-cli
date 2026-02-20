class RimeCli < Formula
  desc "Command-line interface for Rime text-to-speech"
  homepage "https://github.com/rimelabs/rime-cli"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-arm64.tar.gz"
      sha256 "c901b379e76f1854400406a6b95a0f46f35dd2fc55873d4876c4e2021dc3b731"
    else
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-amd64.tar.gz"
      sha256 "57392f55afafb80a73754e5b0c6505a2c48d3403c2c9b24db6eaa9f04e988313"
    end
  end

  on_linux do
    url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-linux-amd64.tar.gz"
    sha256 "990ba6cb3419cc45d42e6748e9d4559088eb13b268e0cb8e39ea6cf8ab88bab8"
  end

  def install
    bin.install "rime"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/rime --version")
  end
end
