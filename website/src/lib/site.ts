// Release details for the pages, read once at build time.
//
// The Pages workflow resolves the latest release and the APT signing key before it builds the
// site, writes them to a JSON file and passes its path in SITE_METADATA_FILE. Local and pull
// request builds don't have that file, so they fall back to the newest local tag and leave the
// date, commit and fingerprint out rather than guess them.
//
// The docs are versioned by release line (v0.8, v0.9 and so on). The latest line is built at the
// site root. When a newer line ships, the release workflow builds the old one again with
// SITE_ARCHIVE set to its line, which moves it under the root as <root>v0.8/, and the Pages
// workflow puts each archived line back beside the latest one and lists them in docs_versions.
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";

export const links = {
  org: "https://github.com/netspeedy",
  repo: "https://github.com/netspeedy/s3ctl",
  releases: "https://github.com/netspeedy/s3ctl/releases",
  issues: "https://github.com/netspeedy/s3ctl/issues",
  homebrew: "https://github.com/netspeedy/homebrew-s3ctl",
  container: "https://github.com/netspeedy/s3ctl/pkgs/container/s3ctl",
  examples: "https://github.com/netspeedy/s3ctl/tree/main/examples",
  siteUrl: "https://netspeedy.github.io/s3ctl/",
  installScript: "https://netspeedy.github.io/s3ctl/install.sh",
  aptUrl: "https://netspeedy.github.io/s3ctl/apt/",
  aptKey: "https://netspeedy.github.io/s3ctl/apt/s3ctl-archive-keyring.gpg",
  image: "ghcr.io/netspeedy/s3ctl",
};

export interface Release {
  /** The release tag, such as v0.8.38, or "dev" when no tag is known. */
  version: string;
  /** True when version is a real tag rather than the "dev" placeholder. */
  tagged: boolean;
  /** Publication date, only when the workflow supplied it. */
  published?: Date;
  /** Full commit SHA the tag points at, only when the workflow supplied it. */
  commit?: string;
  /** The APT signing key's fingerprint in groups of four, only when the repository is published. */
  aptFingerprint?: string;
  /** The release page, or the releases list when there's no tag. */
  url: string;
}

interface Metadata {
  release_commit?: string;
  latest_release?: { tag_name?: string; published_at?: string; html_url?: string } | null;
  apt_repository?: { available?: boolean; fingerprint?: string };
  /** The archived release lines published beside this build, each with its newest release. */
  docs_versions?: { version: string; release: string }[];
}

function readMetadata(): Metadata | undefined {
  const file = process.env.SITE_METADATA_FILE;
  if (!file) return undefined;
  if (!existsSync(file)) throw new Error(`SITE_METADATA_FILE points at ${file}, which doesn't exist`);
  return JSON.parse(readFileSync(file, "utf8")) as Metadata;
}

function latestTag(): string | undefined {
  try {
    const tag = execFileSync("git", ["describe", "--tags", "--abbrev=0", "--match", "v*"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
    return tag || undefined;
  } catch {
    return undefined;
  }
}

const metadata = readMetadata();

function resolveRelease(): Release {
  const latest = metadata?.latest_release ?? undefined;
  const tag = latest?.tag_name || latestTag();
  const published = latest?.published_at ? new Date(latest.published_at) : undefined;
  const apt = metadata?.apt_repository;
  return {
    version: tag ?? "dev",
    tagged: Boolean(tag),
    published: published && !Number.isNaN(published.valueOf()) ? published : undefined,
    commit: metadata?.release_commit || undefined,
    aptFingerprint: apt?.available && apt.fingerprint ? apt.fingerprint : undefined,
    url: latest?.html_url || (tag ? `${links.releases}/tag/${tag}` : links.releases),
  };
}

export const release = resolveRelease();

/** A date as "4 November 2026", in UTC so the build machine's zone doesn't matter. */
export function formatDate(date: Date): string {
  return new Intl.DateTimeFormat("en-GB", { dateStyle: "long", timeZone: "UTC" }).format(date);
}

/** The release line a version belongs to: v0.8.38 is in v0.8. */
export const lineOf = (version: string) => /^v\d+\.\d+/.exec(version)?.[0];

export interface DocsVersion {
  /** The release line, such as v0.8, or "dev" for an untagged build. */
  version: string;
  /** The line's newest release, such as v0.8.38. */
  release: string;
  /** Where the line's docs start, such as <root>v0.8/. */
  url: string;
  /** The newest release's notes. */
  notes: string;
  latest: boolean;
}

const archive = process.env.SITE_ARCHIVE || undefined;
const base = import.meta.env.BASE_URL.replace(/\/?$/, "/");

/** The site root every line lives under, also in an archived build. */
export const siteRoot = archive ? base.slice(0, base.length - archive.length - 1) : base;

/**
 * A link from the root, such as <root>quickstart/, as a link to that page in the line being
 * built. In the latest line it's unchanged. rehype-archive-links.mjs does the same for the pages.
 */
export const docsHref = (href: string) =>
  archive && href.startsWith(siteRoot) && !/^v\d+\.\d+\//.test(href.slice(siteRoot.length))
    ? base + href.slice(siteRoot.length)
    : href;

function resolveDocs() {
  const current = archive ?? lineOf(release.version) ?? "dev";
  const self: DocsVersion = {
    version: current,
    release: release.version,
    url: base,
    notes: release.url,
    latest: !archive,
  };
  // An archived build only knows itself. Its version menu fetches the full list from the root's
  // versions.json, and this link to the latest docs stands in until it does.
  if (archive) {
    const latest: DocsVersion = { version: "Latest", release: "", url: siteRoot, notes: links.releases, latest: true };
    return { current, archived: true, versions: [latest, self] };
  }
  const older = (metadata?.docs_versions ?? [])
    .filter(({ version }) => version !== current)
    .sort((a, b) => b.version.localeCompare(a.version, "en", { numeric: true }))
    .map(({ version, release: tag }) => ({
      version,
      release: tag,
      url: `${siteRoot}${version}/`,
      notes: `${links.releases}/tag/${tag}`,
      latest: false,
    }));
  return { current, archived: false, versions: [self, ...older] };
}

/** Which line these docs are, whether it's an archived one, and every line published. */
export const docs = resolveDocs();

// ------------------------------------------------------------------ this project's copy
//
// What the shared components say about this project. The components themselves are the same in
// every netspeedy site, so a project's name, description and footer links live here.

export const product = {
  name: "s3ctl",
  description: "A single-binary CLI for creating S3-compatible buckets and issuing bucket-scoped credentials.",
};

/** The line under "Browse the docs" on the home page. */
export const directorySubtitle = "From a first bucket to every flag.";

/** The site footer's link columns. */
export const footerColumns = [
  {
    title: "Docs",
    items: [
      { label: "Introduction", href: `${base}introduction/` },
      { label: "Quickstart", href: `${base}quickstart/` },
      { label: "Installation", href: `${base}installation/` },
      { label: "CLI reference", href: `${base}reference/cli/` },
    ],
  },
  {
    title: "Project",
    items: [
      { label: "GitHub", href: links.repo },
      { label: "Releases", href: links.releases },
      { label: "Issues", href: links.issues },
      { label: "Examples", href: links.examples },
    ],
  },
  {
    title: "Install",
    items: [
      { label: "Homebrew tap", href: links.homebrew },
      { label: "APT repository", href: `${base}installation/#signed-apt-repository` },
      { label: "Install script", href: `${base}installation/#installer-script` },
      { label: "GHCR image", href: links.container },
    ],
  },
];
