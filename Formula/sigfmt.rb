class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.5.2"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.2/sigfmt_1.5.2_darwin_amd64.tar.gz"
      sha256 "27993151f2ba026181bf782679b10ec2a7d2d2a7b3f7836f112386399cc06076"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.2/sigfmt_1.5.2_darwin_arm64.tar.gz"
      sha256 "7c138f39381064d786e3f19a788eb99388cf753999718946fbd859b720c71c0d"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.2/sigfmt_1.5.2_linux_amd64.tar.gz"
      sha256 "804ef4a275d7f11a974d4f5e1215c57301f562940edb4ec6a4668d716dec2ef6"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.2/sigfmt_1.5.2_linux_arm64.tar.gz"
      sha256 "733088dabda1b54b353cf34ab443afd3f86839ebfcfcac73a2cd250b7ca07cf9"
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
