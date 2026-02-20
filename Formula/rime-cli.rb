class RimeCli < Formula
  desc "Command-line interface for Rime text-to-speech"
  homepage "https://github.com/rimelabs/rime-cli"
  version "0.1.6"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-arm64.tar.gz"
      sha256 "04904dfab0709056408cb28b81bfd0725d6222f03d82ff8e0ebd7901ecdeef67"
    else
      url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-darwin-amd64.tar.gz"
      sha256 "8abd30024e7666ead1d8f7dcb9b129eba00390463a277cfdd72827e53d92ead1"
    end
  end

  on_linux do
    url "https://github.com/rimelabs/rime-cli/releases/download/v#{version}/rime-linux-amd64.tar.gz"
    sha256 "2b84d695032268ee87cd5d9f598db0d55a050cf5dfa76469ac929fdaaaba58ee"
  end

  def install
    bin.install "rime"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/rime --version")
  end
end
