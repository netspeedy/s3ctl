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
- when a release starts a new line, such as `v0.9.0`, an `s3ctl-docs-v0.8.tar.gz`
  archive of the previous line's docs, attached to that line's last release

## Website

The site at https://netspeedy.github.io/s3ctl/ is the s3ctl documentation, built
with Astro and Starlight from `website/`. The pages are Markdown and MDX in
`website/src/content/docs`. Run it locally with live reload:

```bash
make website-install
npm --prefix website run dev
```

Build it, validate the output, drive it in a browser, and take desktop and phone
screenshots in both themes:

```bash
make website-build
make website-check
make website-test
make website-capture
```

`website-test` runs `website/scripts/browser-test.mjs` in headless Chrome. It
builds the site twice into `website/.test/`, as an archived line and as the
latest with that line listed, unpacks the archive the way the Pages workflow
does, and checks the theme controls, the header, each tab's sidebar, the version
menu (including a long list of lines) and the archived notice. It reads the site
root, the tabs and the pages from the build, so it works on any project's site,
and exits non-zero if a check fails.

`website-check` validates `website/dist`, so it runs after the build. Captures
are written to `website/.captures/`; the first one downloads the Chrome build
Puppeteer pins, since installing the site's dependencies never does. Search only
works in a build, because Pagefind indexes the finished pages.

The build always starts with a fresh content cache (`astro build --force`, which
logs "data store cleared"). Astro notices changes to its config, but not edits
to the Markdown plugins in `website/src/lib`, so without it a plugin change
could leave pages rendered by the old code.

The Pages workflow resolves the latest release and the APT signing key before it
builds, and passes them to the site in `SITE_METADATA_FILE`, so the published
pages carry the real version, date and fingerprint. Local and pull request builds
fall back to the newest local tag and leave the rest out. The same JSON is still
published as `website-metadata.json`.

### Versioned docs

The docs are versioned by release line. The latest line is built at the site
root, and each older line is served under its own path, such as
`https://netspeedy.github.io/s3ctl/v0.8/`. The version menu in the header lists
them, and an older line's pages open with a notice that links to the same page
in the latest docs.

Only the latest line is kept in the repository. When the first release of a new
line is published, the Release Assets workflow finds the commit just before the
new line started (`scripts/release/docs-archive-commit.sh`), builds the site from
it with `SITE_ARCHIVE` set to the old line, and attaches the result to the old
line's last release as `s3ctl-docs-<line>.tar.gz`. The Pages workflow unpacks
every such archive into the site before `make website-check`, lists them in
`versions.json` at the root, and archived lines read that file when they load,
so their menus include lines released after them.

v0.8 is the first line that gets an archive. Earlier lines were replaced before
the docs were versioned, and their commits can't build under a line's own path.

To build an archived line by hand, for example to replace a lost archive, check
out the commit the script prints and build it with the line set. Without
`SITE_METADATA_FILE` the pages leave out the release date and APT fingerprint;
the docs-archive job in `.github/workflows/release-assets.yml` shows how it writes
that file.

```bash
git checkout "$(bash scripts/release/docs-archive-commit.sh v0.8.38 v0.9.0)"
SITE_ARCHIVE=v0.8 make website-build
SITE_ARCHIVE=v0.8 make website-check
tar -C website/dist -czf s3ctl-docs-v0.8.tar.gz .
gh release upload v0.8.38 s3ctl-docs-v0.8.tar.gz --clobber
```

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
