#!/usr/bin/env python3
"""Regenerate Formula/sigfmt.rb from a published release.

Reads checksums.txt (either a local file or fetched from the GitHub release
assets) and rewrites the formula with the new version and per-platform
SHA256 sums. Exits non-zero if an expected platform archive is missing, so a
partial release never produces a broken formula.

Usage:
    ./scripts/update-brew-formula.py <version> [checksums.txt]
"""

from __future__ import annotations

import re
import sys
import urllib.request
from pathlib import Path

# Expected platform archives in checksums.txt.
REQUIRED_PLATFORMS = ("darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64")

FORMULA_PATH = Path(__file__).resolve().parent.parent / "Formula" / "sigfmt.rb"

FORMULA_TEMPLATE = '''class Sigfmt < Formula
  desc "Linter and formatter for Go function signatures"
  homepage "https://github.com/vsfedorenko/sigfmt"
  version "{version}"
  license "MIT"

  # Prebuilt release archives per platform (no bottles). The analyzer
  # shells out to the go toolchain at runtime even in single-file mode,
  # so go is declared as a test dependency for `brew test`.
  depends_on "go" => :test

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v{version}/sigfmt_{version}_darwin_amd64.tar.gz"
      sha256 "{darwin_amd64}"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v{version}/sigfmt_{version}_darwin_arm64.tar.gz"
      sha256 "{darwin_arm64}"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v{version}/sigfmt_{version}_linux_amd64.tar.gz"
      sha256 "{linux_amd64}"
    end
    if Hardware::CPU.arm?
      url "https://github.com/vsfedorenko/sigfmt/releases/download/v{version}/sigfmt_{version}_linux_arm64.tar.gz"
      sha256 "{linux_arm64}"
    end
  end

  def install
    bin.install "sigfmt"
  end

  test do
    (testpath/"sample.go").write <<~EOS
      package main

      func Add(
      \\ta int,
      \\tb int,
      ) int {
      \\treturn a + b
      }
    EOS
    # File-argument mode runs without a go.mod; exit 3 = diagnostics found
    # (verified against a published linux/amd64 release archive).
    assert_includes shell_output("#{bin}/sigfmt #{testpath}/sample.go", 3),
                    "formatted more compactly"
  end
end
'''


def render(version: str, sums: dict[str, str]) -> str:
    """Fill the formula template; '{{' escapes are not needed since we use
    plain marker replacement instead of str.format (the test block contains
    Ruby interpolation like #{bin} which format() would choke on)."""
    formula = FORMULA_TEMPLATE
    replacements = {"version": version, **{p: sums[p] for p in REQUIRED_PLATFORMS}}
    for key, value in replacements.items():
        formula = formula.replace("{" + key + "}", value)
    return formula


def parse_checksums(text: str) -> dict[str, str]:
    """Map platform key (e.g. darwin_amd64) to its sha256 digest."""
    sums: dict[str, str] = {}
    for line in text.splitlines():
        line = line.strip()
        if not line:
            continue
        digest, _, name = line.partition("  ")
        m = re.match(r"sigfmt_.+?_(darwin|linux)_(amd64|arm64)\.tar\.gz$", name)
        if m:
            sums[f"{m.group(1)}_{m.group(2)}"] = digest
    return sums


def main() -> int:
    if len(sys.argv) < 2:
        print(__doc__, file=sys.stderr)
        return 2
    version = sys.argv[1].lstrip("v")
    checksums_file = Path(sys.argv[2]) if len(sys.argv) > 2 else None

    if checksums_file is not None and checksums_file.exists():
        text = checksums_file.read_text()
    else:
        url = (
            "https://github.com/vsfedorenko/sigfmt/releases/download/"
            f"v{version}/checksums.txt"
        )
        with urllib.request.urlopen(url) as resp:
            text = resp.read().decode()

    sums = parse_checksums(text)
    missing = [p for p in REQUIRED_PLATFORMS if p not in sums]
    if missing:
        print(f"missing platform archives in checksums: {missing}", file=sys.stderr)
        return 1

    formula = render(version, sums)

    if FORMULA_PATH.exists() and FORMULA_PATH.read_text() == formula:
        print("formula already up to date")
        return 0
    FORMULA_PATH.write_text(formula)
    print(f"Formula/sigfmt.rb updated to v{version}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
