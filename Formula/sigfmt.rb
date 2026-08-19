class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.2.1"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.2.1/sigfmt_1.2.1_darwin_amd64.tar.gz"
      sha256 "28e8cde327b53e9b8a732680905ddb411c2c413b4fd713b75ad176d4209f7141"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.2.1/sigfmt_1.2.1_darwin_arm64.tar.gz"
      sha256 "e4faa7fb9d304110065e2e3104d306d5ee51d75e745927d531078eef81ac2412"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.2.1/sigfmt_1.2.1_linux_amd64.tar.gz"
      sha256 "99a287d714a344a41fc8dbc290c25aefb896389ba9a017bb6ed39fa41b49b07a"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.2.1/sigfmt_1.2.1_linux_arm64.tar.gz"
      sha256 "6fd1da978c05b2b4715dc51b77f4668599db92c635fe93fe2eefe68f647a2636"
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
