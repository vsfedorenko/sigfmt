class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.4.2"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.2/sigfmt_1.4.2_darwin_amd64.tar.gz"
      sha256 "1407ef2502e96f852df2b287a1c65cf0912b789e87a7f46144bdaeb69ea02b5a"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.2/sigfmt_1.4.2_darwin_arm64.tar.gz"
      sha256 "fce6ff30a1b3a0c3e1cd426b7ef198e29e9ae0cead07984dabfe7b7a632348dd"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.2/sigfmt_1.4.2_linux_amd64.tar.gz"
      sha256 "a26b72429f9366159d1a689950a0717e4a1cc8eaa15be406c3bd0d519b8601e5"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.4.2/sigfmt_1.4.2_linux_arm64.tar.gz"
      sha256 "14762afea59bb1f08e7934a867479007b8f28aade884d02e30b62711ae7a474a"
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
