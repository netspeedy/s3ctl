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

## Website

The site at https://netspeedy.github.io/s3ctl/ is the s3ctl documentation, built
with Astro and Starlight from `website/`. The pages are Markdown and MDX in
`website/src/content/docs`. Run it locally with live reload:

```bash
make website-install
npm --prefix website run dev
```

Build it, validate the output, and take desktop and phone screenshots in both
themes:

```bash
make website-build
make website-check
make website-capture
```

`website-check` validates `website/dist`, so it runs after the build. Captures
are written to `website/.captures/`. Search only works in a build, because
Pagefind indexes the finished pages.

The Pages workflow resolves the latest release and the APT signing key before it
builds, and passes them to the site in `SITE_METADATA_FILE`, so the published
pages carry the real version, date and fingerprint. Local and pull request builds
fall back to the newest local tag and leave the rest out. The same JSON is still
published as `website-metadata.json`.

`TestCLIReferenceCoversEveryFlag` in `internal/cli` fails when a flag is added
to or removed from the CLI without updating
`website/src/content/docs/reference/cli.mdx`.

The favicon, Apple touch icon, `favicon.ico` and share card in `website/public`
come from `make readme-hero`, along with the README banner.

## Dependency updates

Dependency updates are managed by Dependabot. Related AWS SDK for Go v2 module
updates are grouped into one pull request so shared `go.mod` and `go.sum`
changes do not create a queue of conflicting PRs.

The automation auto-merge workflow runs after `Build and Validate` succeeds and
daily at 05:00 UTC. It handles Dependabot PRs and first-party automation PRs,
then merges only open, clean PRs with successful validation for their current
head revision.

The toolchain refresh PR is opened with `GITHUB_TOKEN`, so its `pull_request`
workflows don't start. The refresh dispatches `Build and Validate` and a
validation-only `Container Image` build on its branch instead, and the daily
run merges it once both have passed on the PR head. The daily run asks for a
green `Container Image` run for any PR it merges, because `Build and Validate`
doesn't build the image.
