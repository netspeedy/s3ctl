# Installation

Published artifacts cover the supported installation paths:

- GitHub release archives for `linux/amd64`, `linux/arm64`, `linux/arm/v7`, `darwin/amd64`, and `darwin/arm64`
- a signed APT repository
- Debian `.deb` packages for `amd64`, `arm64`, and `armhf`
- signed `SHA256SUMS` checksums (with a detached GPG signature) for every archive and Debian package
- a Homebrew tap for macOS and Linux
- a GitHub Pages release hub with install commands and release metadata
- a multi-arch GHCR image

Release candidates use tags like `v1.2.3-rc.1`. They are useful for testing a
version before it reaches the stable installer, stable APT channel, or
`:latest` container tag.

## Homebrew

```bash
brew tap netspeedy/s3ctl
brew install s3ctl
```

> On recent Homebrew, new third-party taps may require explicit trust. If
> installation is refused, run `brew trust netspeedy/s3ctl` once, then
> `brew install s3ctl`.

## Direct installer

Recommended for macOS:

```bash
curl -fsSL https://netspeedy.github.io/s3ctl/install.sh | bash
```

On macOS, install via this script unless you specifically need to handle the
archive yourself. The installer defaults to a user-owned bin directory, prefers
an existing home bin path already present in `PATH` such as `$HOME/.local/bin`,
`$HOME/bin`, or `$HOME/.bin`, and otherwise uses `$HOME/.local/bin` with a PATH
hint. It also clears the macOS download quarantine marker from the installed
binary.

If you download and extract a macOS archive manually, Finder may block the binary
because the release is not Apple-notarized yet. Prefer the installer, or clear
the quarantine marker yourself after verifying the checksum:

```bash
xattr -d com.apple.quarantine ./s3ctl-darwin-arm64
```

Pinned installer run:

```bash
curl -fsSL https://netspeedy.github.io/s3ctl/install.sh | bash -s -- --version v1.2.3
```

Custom install location:

```bash
curl -fsSL https://netspeedy.github.io/s3ctl/install.sh | bash -s -- --install-dir "$HOME/.local/bin"
```

## Signed APT repository

```bash
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL https://netspeedy.github.io/s3ctl/apt/s3ctl-archive-keyring.gpg \
  | sudo tee /etc/apt/keyrings/s3ctl-archive-keyring.gpg >/dev/null

sudo tee /etc/apt/sources.list.d/s3ctl.sources >/dev/null <<'EOF'
Types: deb
URIs: https://netspeedy.github.io/s3ctl/apt/
Suites: stable
Components: main
Signed-By: /etc/apt/keyrings/s3ctl-archive-keyring.gpg
EOF

sudo apt update && sudo apt install s3ctl
```

### Direct Debian package

Install a single package without wiring an APT source:

```bash
curl -fsSLO https://github.com/netspeedy/s3ctl/releases/latest/download/s3ctl_1.2.3_amd64.deb
sudo apt install ./s3ctl_1.2.3_amd64.deb
```

Verify the download against the signed checksums. The detached signature is
produced by the same key that signs the APT repository:

```bash
curl -fsSLO https://github.com/netspeedy/s3ctl/releases/latest/download/SHA256SUMS
curl -fsSLO https://github.com/netspeedy/s3ctl/releases/latest/download/SHA256SUMS.asc
curl -fsSL https://netspeedy.github.io/s3ctl/apt/s3ctl-archive-keyring.asc | gpg --import
gpg --verify SHA256SUMS.asc SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing
```

## Container

Package page: [github.com/netspeedy/s3ctl/pkgs/container/s3ctl](https://github.com/netspeedy/s3ctl/pkgs/container/s3ctl)

Use the published image:

```bash
docker run --rm ghcr.io/netspeedy/s3ctl:latest
```

Run against the bundled examples from the host:

```bash
docker run --rm \
  -v "$PWD/examples:/examples:ro" \
  ghcr.io/netspeedy/s3ctl:latest \
  --config /examples/s3ctl.json \
  --dry-run \
  --output json
```
