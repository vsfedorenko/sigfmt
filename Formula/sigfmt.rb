class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.4.1"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.1/sigfmt_1.4.1_darwin_amd64.tar.gz"
      sha256 "59902d0fe4b001e43e0f0c30442dd6669e045458beaf80984cc0554eb1bc7646"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.1/sigfmt_1.4.1_darwin_arm64.tar.gz"
      sha256 "7b79776c9ebfc5f740596de4eca6e63af744e283f0050a9a3ac4dc16159535b1"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.1/sigfmt_1.4.1_linux_amd64.tar.gz"
      sha256 "e5b36278549bf76fc06d10385c6b5e694421d4eb78a3364b33d0e1e09ff3f3bc"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.1/sigfmt_1.4.1_linux_arm64.tar.gz"
      sha256 "73971a7f9ac7e68ba4a972a494846647b0d3790b049b7535e9e2ae5bcccc7a19"
    end
  end

  def install
    bin.install "sigfmt"
  end

  test do
    (testpath/"sample.go").write <<~EOS
      package main

      func Add(
      \ta int,
      \tb int,
      ) int {
      \treturn a + b
      }
    EOS
    # File-argument mode runs without a go.mod; exit 3 = diagnostics found
    # (verified against a published linux/amd64 release archive).
    assert_includes shell_output("#{bin}/sigfmt #{testpath}/sample.go", 3),
                    "formatted more compactly"
  end
end
