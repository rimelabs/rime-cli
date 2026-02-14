class RimeCli < Formula
  desc "Command-line interface for Rime text-to-speech"
  homepage "https://github.com/rimelabs/rime-cli"
  version "0.0.1-test"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rimelabs/rime-cli/releases/download/v0.0.1-test/rime-darwin-arm64.tar.gz"
      sha256 "d0f9c83827947a9fc8165664b06ab90b73c76b149283d55e7870912c422e58bc"
    else
      url "https://github.com/rimelabs/rime-cli/releases/download/v0.0.1-test/rime-darwin-amd64.tar.gz"
      sha256 "07fa92210ab9d713e2e9fcc9a85369152a92912d7d9dbf7af670bcb1581176d1"
    end
  end

  on_linux do
    url "https://github.com/rimelabs/rime-cli/releases/download/v0.0.1-test/rime-linux-amd64.tar.gz"
    sha256 "d4552785e281e52330e9353b093d995bb12cf76bef64a9dab54d44c0895d6d7f"
  end

  def install
    bin.install "rime"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/rime --version")
  end
end
