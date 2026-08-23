class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.5.1"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.1/sigfmt_1.5.1_darwin_amd64.tar.gz"
      sha256 "8a8615e716a834b3012d848f7af4863cd15c043497892a8345bbea2dfea6a625"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.1/sigfmt_1.5.1_darwin_arm64.tar.gz"
      sha256 "c47a892dc7f4a1355a785e7aca2dd494283bdeb72e49d957d9c5f74f53dff929"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.1/sigfmt_1.5.1_linux_amd64.tar.gz"
      sha256 "fea7d9f8789ab4aed5b20d7cc3e0b8396de49a7761269cac01997626dffb8fd6"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.1/sigfmt_1.5.1_linux_arm64.tar.gz"
      sha256 "9a02469f556ac435b2ff4149a6503adc8a69f503262a7c555444d631c508e668"
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
