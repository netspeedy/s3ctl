// Release details for the pages, read once at build time.
//
// The Pages workflow resolves the latest release and the APT signing key before it builds the
// site, writes them to a JSON file and passes its path in SITE_METADATA_FILE. Local and pull
// request builds don't have that file, so they fall back to the newest local tag and leave the
// date, commit and fingerprint out rather than guess them.
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

function resolveRelease(): Release {
  const metadata = readMetadata();
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
