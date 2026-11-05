# Repository Guidelines

@/Users/soakes/.codex/RTK.md

## Project Structure & Ownership

- Keep the executable entrypoint in `cmd/s3ctl/`.
- Keep application logic under `internal/`; avoid adding new top-level Go
  packages at the repository root.
- Keep first-party automation, packaging, and release assets at the root,
  under `.github/`, `scripts/`, `examples/`, and `website/`.
- Treat `website/` as a first-party release surface, not a
  throwaway demo. It is the s3ctl documentation, built with Astro and
  Starlight; the pages are Markdown and MDX in `website/src/content/docs`.
- User-facing documentation lives in the site, not in `docs/`. The files in
  `docs/` other than `release.md` are pointers kept so old links still land
  somewhere; don't add guides there.
- When CLI behavior, flags, defaults, output or errors change, update the
  matching site pages in the same change. `TestCLIReferenceCoversEveryFlag` in
  `internal/cli` fails when a flag is added or removed without updating
  `website/src/content/docs/reference/cli.mdx`.
- The mark, the README banner, the social cards and the site's icons all come
  from `scripts/readme/build_readme_hero.py` via `make readme-hero`. Change the
  generator, not the generated files.
- Each sidebar group has its own colour and icon, and each header tab owns
  some of the groups and shows only those in its sidebar. Both are in
  `website/src/lib/sections.ts`, matched to the sidebar in
  `website/astro.config.mjs` by label, so adding or renaming a group means
  updating `sections` and `tabs` there.
- The site's components and styles are shared with the other netspeedy docs
  sites, so their classes, custom properties and elements use the neutral
  `ns-` prefix and they don't name s3ctl. What's particular to s3ctl (its
  name, description, links, footer columns and the home page's directory
  line) lives in `website/src/lib/site.ts`, and the pages written for s3ctl
  are `HomeTop.astro`, `InstallTabs.astro` and `src/pages/404.astro`. Keep it
  that way, so a change to a shared file can be copied to the other sites as
  it is.
- The docs are versioned by release line. Only the latest line lives in
  `website/src/content/docs`; never copy pages into per-version folders. When a
  new minor ships, the release workflow rebuilds the outgoing line from git with
  `SITE_ARCHIVE=vX.Y`, which serves it under `/s3ctl/vX.Y/`, and the Pages
  workflow publishes it beside the latest. Keep page links root-relative
  (`/s3ctl/...`): archived builds rewrite them into their own line.
- Keep the website out of `.github/`. Dependabot stopped being able to open npm
  pull requests for manifests under `.github/` in June 2026 (its create-PR call
  is rejected as "invalid or unauthorized changes"), which is why it lives at
  the root.

## Build, Lint, and Test Commands

- Install the pinned lint tool once per machine:
  ```bash
  make lint-install
  ```
- Format Go code:
  ```bash
  make fmt
  ```
- Check baseline formatting:
  ```bash
  make fmt-check
  ```
- Run the pinned lint suite:
  ```bash
  make lint
  ```
- Apply available lint fixes:
  ```bash
  make lint-fix
  ```
- Run static validation:
  ```bash
  make vet
  ```
- Run tests:
  ```bash
  make test
  ```
- Build the local binary:
  ```bash
  make build
  ```
- Lint shell scripts:
  ```bash
  shellcheck scripts/*.sh scripts/*/*.sh
  ```
- Build the docs site, then validate the built output (pages, search index,
  icons, share card and every internal link). The check reads `website/dist`,
  so it always runs after the build:
  ```bash
  make website-install
  make website-build
  make website-check
  ```
- Drive the site in headless Chrome: the theme controls, the header and its
  tabs, the version menu and an archived line. It builds its own latest and
  archived copies under `website/.test/`, so it needs no release metadata:
  ```bash
  make website-test
  ```
- Screenshot the built site in both themes at desktop and phone widths:
  ```bash
  make website-capture
  ```
- Serve the site with live reload while editing pages with `make website-dev`.
  When Astro detects an agent it starts the server in the background and returns,
  so stop it afterwards with `cd website && npx astro dev stop`.
  `website/README.md` summarises the folder, its targets and how it's published;
  keep it in step when any of those change.

## Go Style & Quality Rules

- Always use `gofmt` at minimum; prefer the pinned `golangci-lint` toolchain so
  `gofumpt`, `goimports`, `staticcheck`, `errcheck`, and `revive` are applied
  consistently.
- Favor clear, operator-facing errors that preserve the failing path,
  operation, or target resource.
- Keep exported APIs and package-level behavior documented when lint rules
  require it.
- Prefer small helpers and focused structs over large generic utility layers.
- Keep CLI help, examples, and configuration names polished and ready to copy
  into production workflows.
- Do not add new `S3CTL_*` configuration variables. Prefer CLI flags and JSON
  config for `s3ctl` settings, while leaving standard AWS/go-ovh credential
  discovery to the upstream SDK clients.

## Workflow & Security Rules

- Treat workflow changes as security-sensitive. Keep GitHub Actions permissions
  least-privilege and scoped to the jobs that need them.
- Pull requests should validate code and packaging without gaining access to
  signing or publish credentials.
- If workflow behavior changes, update the README and any operator-facing docs
  in the same change when user-visible behavior changes.
- Treat destructive storage operations as security-sensitive. Keep object
  removal guarded by explicit confirmation, allow unforced bucket deletes only
  after proving the bucket is empty, preserve dry-run behavior, and verify
  provider ownership assumptions before deleting credentials or users.

## Testing Expectations

- Minimum validation for Go code changes:
  ```bash
  make fmt-check
  make lint
  make vet
  make test
  make build
  ```
- If you change shell scripts, also run:
  ```bash
  bash -n scripts/*.sh scripts/*/*.sh
  shellcheck scripts/*.sh scripts/*/*.sh
  ```
- If you change the website, run `make website-build`, `make website-check` and
  `make website-test`, and look at `make website-capture` output for layout
  changes. If you change
  release automation, run the matching local validation that exercises those
  paths before considering the change complete.
- If you change release tagging, release notes, or publish workflows, also run:
  ```bash
  bash -n scripts/*.sh scripts/*/*.sh
  bash scripts/release/next-release.sh
  bash scripts/release/generate-release-notes.sh "$(git describe --tags --abbrev=0 2>/dev/null || true)" HEAD /tmp/s3ctl-release-notes.md
  ```

## Commit & PR Rules

- Do not create commits unless the user explicitly asks for a commit in the
  current turn.
- Prefer conventional commit subjects such as `feat:`, `fix:`, `docs:`, `ci:`,
  `deps:`, `chore:`, or `test:`.
- `feat:`, `fix:`, `perf:`, `revert:`, `deps:`, `container:`, `build:`,
  `packaging:` and `release:` commits cut releases
  (`scripts/release/next-release.sh` decides); `docs:`, `ci:`, `chore:` and
  `test:` don't. Every release-bearing push to `main` cuts a release candidate.
  Updates from Dependabot, Renovate and this repository's workflows are then
  promoted to stable on their own; a release-bearing change a person merged
  waits at the candidate until someone runs `Promote Release Candidate`
  (`scripts/release/automated-only.sh` decides). While that's waiting, every
  validated push to `main` tags the next candidate, docs-only pushes included.
  That's expected. `docs/release.md` describes the release flow.
- Keep changes focused by concern when practical: runtime logic, automation,
  docs, packaging, and website work should not be bundled casually.
- PR descriptions should explain the operational reason for the change, the
  validation performed, and any release or upgrade impact.
