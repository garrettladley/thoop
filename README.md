# thoop

<p>
    <a href="https://github.com/garrettladley/thoop/releases"><img src="https://img.shields.io/github/release/garrettladley/thoop" alt="Latest Release"></a>
    <a href="https://github.com/garrettladley/thoop/actions"><img src="https://github.com/garrettladley/thoop/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a>
</p>

tui for your whoop data

## Install

```bash
# Homebrew
brew install --cask garrettladley/tap/thoop

# Go
go install github.com/garrettladley/thoop/cmd/thoop@latest
```

### macOS Gatekeeper

If macOS blocks the binary, run:

```bash
xattr -d com.apple.quarantine $(which thoop)
```
