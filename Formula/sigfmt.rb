class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.3.0"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.3.0/sigfmt_1.3.0_darwin_amd64.tar.gz"
      sha256 "b441ad26788ad14157ee67a48074ff21d3bc401bd0c00d77a8a6326e1996b639"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.3.0/sigfmt_1.3.0_darwin_arm64.tar.gz"
      sha256 "be95f12ef2f7ef5fee19a2c7417bd0ec8b776c050c61c3c04bd4c39d7da34f07"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.3.0/sigfmt_1.3.0_linux_amd64.tar.gz"
      sha256 "62970e20eace0f5fc70e970b460e6df3b24fcba317a79c89c0eaf1c7a3952db3"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.3.0/sigfmt_1.3.0_linux_arm64.tar.gz"
      sha256 "3e08ed9fd277a4a195cc2ee79ed7228b388b5cfcf6fa774185b418f3c9bfb130"
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
