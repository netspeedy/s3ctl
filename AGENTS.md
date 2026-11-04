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
- Each sidebar group has its own colour and icon. Their labels, colours and
  icons are in `website/src/lib/sections.ts`, but the sidebar draws its group
  icons from `website/src/styles/theme.css` by position. Adding, renaming or
  reordering a group in `website/astro.config.mjs` means updating both.
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
  shellcheck scripts/*.sh
  ```
- Build the docs site, then validate the built output (pages, search index,
  icons, share card and every internal link). The check reads `website/dist`,
  so it always runs after the build:
  ```bash
  make website-install
  make website-build
  make website-check
  ```
- Screenshot the built site in both themes at desktop and phone widths:
  ```bash
  make website-capture
  ```

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
  bash -n scripts/*.sh
  shellcheck scripts/*.sh
  ```
- If you change the website, run `make website-build` and `make website-check`,
  and look at `make website-capture` output for layout changes. If you change
  release automation, run the matching local validation that exercises those
  paths before considering the change complete.
- If you change release tagging, release notes, or publish workflows, also run:
  ```bash
  bash -n scripts/*.sh
  bash scripts/release/next-release.sh
  bash scripts/release/generate-release-notes.sh "$(git describe --tags --abbrev=0 2>/dev/null || true)" HEAD /tmp/s3ctl-release-notes.md
  ```

## Commit & PR Rules

- Do not create commits unless the user explicitly asks for a commit in the
  current turn.
- Prefer conventional commit subjects such as `feat:`, `fix:`, `docs:`, `ci:`,
  `deps:`, `chore:`, or `test:`.
- Keep changes focused by concern when practical: runtime logic, automation,
  docs, packaging, and website work should not be bundled casually.
- PR descriptions should explain the operational reason for the change, the
  validation performed, and any release or upgrade impact.
