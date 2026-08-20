class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.4.0"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.0/sigfmt_1.4.0_darwin_amd64.tar.gz"
      sha256 "f708392261c109c1e53739194b0cdc662e002b245097ebd228c8be3166a4ccbf"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.0/sigfmt_1.4.0_darwin_arm64.tar.gz"
      sha256 "9b2ac0762d854c2a9d98f1d6b0271865306404da9ce08ab5384a971df138f750"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.0/sigfmt_1.4.0_linux_amd64.tar.gz"
      sha256 "ed251f235023ea0a6dc2897c2652bc7d58fd33c256312cb0efc8d136cdcd99d4"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.0/sigfmt_1.4.0_linux_arm64.tar.gz"
      sha256 "e02e8ad6600f0960bff2de754fc2ec1d5af0b2aaed148e49f7163a2106068ad6"
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
