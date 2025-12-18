# Homebrew formula for Poros
# To install: brew install KilimcininKorOglu/tap/poros

class Poros < Formula
  desc "Modern, cross-platform network path tracer"
  homepage "https://github.com/KilimcininKorOglu/poros"
  license "MIT"
  version "1.0.0"

  on_macos do
    on_intel do
      url "https://github.com/KilimcininKorOglu/poros/releases/download/v#{version}/poros-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64"
    end
    on_arm do
      url "https://github.com/KilimcininKorOglu/poros/releases/download/v#{version}/poros-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/KilimcininKorOglu/poros/releases/download/v#{version}/poros-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    end
    on_arm do
      url "https://github.com/KilimcininKorOglu/poros/releases/download/v#{version}/poros-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    end
  end

  def install
    bin.install "poros"
  end

  def caveats
    <<~EOS
      Poros requires root/sudo for ICMP probes:
        sudo poros google.com

      Or set capabilities on Linux:
        sudo setcap cap_net_raw+ep #{bin}/poros
    EOS
  end

  test do
    assert_match "Poros", shell_output("#{bin}/poros version")
  end
end
