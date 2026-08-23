class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.5.0"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.0/sigfmt_1.5.0_darwin_amd64.tar.gz"
      sha256 "2f694176549a225229a7670093551862fadf689f57103e96a7046a47c248fa7c"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.0/sigfmt_1.5.0_darwin_arm64.tar.gz"
      sha256 "6aa24f44fb51899870f0d35b4d2af85973df4da8b90152e7a4bea674a55bd750"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.0/sigfmt_1.5.0_linux_amd64.tar.gz"
      sha256 "838e8ad215e4a7cb15f498452ac30b5206eef796cdc2f3fda99a199ab6e9dedb"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.0/sigfmt_1.5.0_linux_arm64.tar.gz"
      sha256 "0940ff5909f151d988d62d42014c826ba59a6cd8a44ed70ae79b2e50679ac914"
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
