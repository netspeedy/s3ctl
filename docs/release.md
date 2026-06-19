# Release and Website Notes

Stable releases are published only after the project passes validation for
formatting, linting, vetting, tests, build output, packaging, website assets,
and CLI smoke checks.

Release candidates use tags such as `v1.2.3-rc.1` while a version is being
validated. Production installs should use the latest stable release unless you
are intentionally testing a candidate build.

Stable releases publish:

- Linux and macOS archives for `amd64`, `arm64`, and Linux `arm/v7`
- Debian packages for `amd64`, `arm64`, and `armhf`
- Homebrew formula updates for stable releases
- a signed `SHA256SUMS` checksum file with a detached GPG signature
- GHCR images for the stable version, `latest`, and semver convenience tags
- the GitHub Pages release hub with current installer and asset metadata
- signed APT repository metadata for the stable channel

## Website preview

Render the release hub locally with real browser screenshots:

```bash
make website-install
make website-check
make website-build
make website-capture
```

Desktop and mobile captures are written to `.github/assets/website/.captures/`. The website is
built with Vite and uses generated `website-metadata.json` when the Pages workflow
publishes the release hub.

## Dependency updates

Dependency updates are managed by Dependabot. Related AWS SDK for Go v2 module
updates are grouped into one pull request so shared `go.mod` and `go.sum`
changes do not create a queue of conflicting PRs.

The automation auto-merge workflow runs after `Build and Validate` succeeds and
on a weekly maintenance schedule. It handles Dependabot PRs and first-party
automation PRs, then merges only open, clean PRs with successful validation for
their current head revision.
