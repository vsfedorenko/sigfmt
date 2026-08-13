# Security Policy

## Supported Versions

sigfmt is actively developed. Security fixes are applied to the latest `main`
branch and included in the next release.

| Version | Supported          |
|---------|--------------------|
| latest  | :white_check_mark: |
| < 1.0   | :white_check_mark: (latest release) |

## Reporting a Vulnerability

We take security vulnerabilities seriously. Thank you for improving the safety
of this project.

**Please do NOT open a public GitHub issue for security vulnerabilities.**

### How to Report

1. Use GitHub's **private vulnerability reporting** feature:
   [Report a vulnerability](https://github.com/vsfedorenko/sigfmt/security/advisories/new)
2. Alternatively, email the maintainer directly.

Please include the following information in your report:

- A description of the vulnerability and its potential impact
- Steps to reproduce the issue
- Affected versions (if known)
- Any suggested fixes or mitigations

### Response Timeline

- **Acknowledgement**: within 48 hours
- **Initial assessment**: within 5 business days
- **Resolution**: dependent on severity; critical issues are prioritized

### Disclosure

Once a fix is available we will publish a GitHub Security Advisory and a new
release. We appreciate coordinated disclosure and are happy to credit
reporters.

## Scope

sigfmt is a `golangci-lint` plugin that operates on source code using the Go
AST. It does not:

- Execute arbitrary code
- Make network requests
- Require elevated privileges

A vulnerability in sigfmt itself would likely involve incorrect analysis
output or a crash (denial of service) when processing crafted source files.
