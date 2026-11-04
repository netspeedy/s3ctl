// The four sidebar groups each have a colour. The page's group colours its breadcrumb, step
// numbers, list markers, table of contents and pager (Head.astro sets --s3-page-* for that), and
// cards pointing into a group take its colour.
//
// The sidebar's own group icons and colours are in theme.css, by position, so they follow the
// group order in astro.config.mjs. Keep the two in step if the groups change.

export type Hue = "c1" | "c2" | "c3" | "c4";

export interface Section {
  label: string;
  hue: Hue;
  /** One line under the group's name in the home page's directory. */
  blurb: string;
  /** The inside of a 24x24 stroke icon. theme.css carries the same paths for the sidebar. */
  icon: string;
}

export const sections: Section[] = [
  {
    label: "Getting started",
    hue: "c1",
    blurb: "Install it and plan a first bucket.",
    icon: '<path d="M5 7l5 5-5 5"/><path d="M13 17h6"/>',
  },
  {
    label: "Guides",
    hue: "c2",
    blurb: "Everyday work with the S3 provider.",
    icon: '<path d="M3 6c3-1.3 6-1.3 9 .8 3-2.1 6-2.1 9-.8v13c-3-1.3-6-1.3-9 .8-3-2.1-6-2.1-9-.8z"/><path d="M12 6.8v13"/>',
  },
  {
    label: "OVHcloud",
    hue: "c3",
    blurb: "Public Cloud Object Storage.",
    icon: '<path d="M7.5 18.5h9.5a4 4 0 0 0 .6-7.95 6 6 0 0 0-11.5 1A3.5 3.5 0 0 0 7.5 18.5z"/>',
  },
  {
    label: "Reference",
    hue: "c4",
    blurb: "Look something up.",
    icon:
      '<path d="M9 4H7.5A2.5 2.5 0 0 0 5 6.5V10l-2 2 2 2v3.5A2.5 2.5 0 0 0 7.5 20H9"/>' +
      '<path d="M15 4h1.5A2.5 2.5 0 0 1 19 6.5V10l2 2-2 2v3.5a2.5 2.5 0 0 1-2.5 2.5H15"/>',
  },
];

interface Link {
  type: "link";
  label: string;
  href: string;
  isCurrent: boolean;
}
interface Group {
  type: "group";
  label: string;
  entries: Entry[];
}
type Entry = Link | Group;

/** The top-level group holding a link that passes `match`. */
function groupOf(entries: readonly Entry[], match: (link: Link) => boolean): Section | undefined {
  for (const entry of entries) {
    if (entry.type !== "group") continue;
    const holds = (list: readonly Entry[]): boolean =>
      list.some((child) => (child.type === "link" ? match(child) : holds(child.entries)));
    if (holds(entry.entries)) return sections.find((section) => section.label === entry.label);
  }
  return undefined;
}

/** Each sidebar group with its section and the links in it, in sidebar order. */
export function directory(sidebar: readonly Entry[]) {
  const links = (list: readonly Entry[]): Link[] =>
    list.flatMap((entry) => (entry.type === "link" ? [entry] : links(entry.entries)));
  return sidebar.flatMap((entry) => {
    if (entry.type !== "group") return [];
    const section = sections.find((candidate) => candidate.label === entry.label);
    return section ? [{ section, links: links(entry.entries) }] : [];
  });
}

/** The section of the page being rendered, or undefined for pages outside the sidebar. */
export const currentSection = (sidebar: readonly Entry[]) => groupOf(sidebar, (link) => link.isCurrent);

/** The section a link into the site lands in. */
export const sectionOfHref = (sidebar: readonly Entry[], href: string) =>
  groupOf(sidebar, (link) => link.href === href);
