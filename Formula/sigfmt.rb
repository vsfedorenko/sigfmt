class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "1.5.3"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.3/sigfmt_1.5.3_darwin_amd64.tar.gz"
      sha256 "264086ea8e64b8dee87dc4eed7b9112515c750936ab6058bfb27821e5579f620"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.3/sigfmt_1.5.3_darwin_arm64.tar.gz"
      sha256 "384c633dd8897b4defca9913368fc2ef98701c7907e446f6bbafb9afb172e733"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.3/sigfmt_1.5.3_linux_amd64.tar.gz"
      sha256 "dc1e91f79f1288ce432f3a8e8c82dce841063f0c36bab449802960afe96da2c0"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v1.5.3/sigfmt_1.5.3_linux_arm64.tar.gz"
      sha256 "cc5aff5084bdc6a2630862d941a16cb160b57940ba2634ed6fb3a0408a0a084e"
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
