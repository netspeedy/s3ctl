import "./style.css";

const defaultSiteURL = new URL("./", window.location.href).href;

const fallbackHighlights = [
  "Provision buckets and scoped credentials from one CLI",
  "Ship Homebrew, APT, Debian packages, archives, and GHCR images",
  "Keep OVHcloud credential rotation and deletion guarded",
];

const defaultMetadata = {
  site_url: defaultSiteURL,
  github_repository: "netspeedy/s3ctl",
  github_url: "https://github.com/netspeedy/s3ctl",
  release_url: "https://github.com/netspeedy/s3ctl/releases",
  homebrew_url: "https://github.com/netspeedy/homebrew-s3ctl",
  container_url: "https://github.com/netspeedy/s3ctl/pkgs/container/s3ctl",
  container_image: "ghcr.io/netspeedy/s3ctl",
  install_script_url: `${defaultSiteURL}install.sh`,
  release_commit: "",
  latest_release: null,
  apt_repository: {
    available: false,
    url: `${defaultSiteURL}apt/`,
    suite: "stable",
    component: "main",
    key_url: `${defaultSiteURL}apt/s3ctl-archive-keyring.gpg`,
    fingerprint: "",
  },
};

function normalizeMetadata(metadata = {}) {
  const apt = metadata.apt_repository || {};
  const rawContainerURL = metadata.container_url || defaultMetadata.container_url;
  const rawContainerImage = metadata.container_image || (rawContainerURL.includes("ghcr.io/") ? rawContainerURL.replace(/^https?:\/\//, "") : "");
  const aptKeyUrl = apt.key_url || defaultMetadata.apt_repository.key_url;

  return {
    ...defaultMetadata,
    ...metadata,
    site_url: `${metadata.site_url || defaultMetadata.site_url}`.replace(/\/?$/, "/"),
    github_url: metadata.github_url || defaultMetadata.github_url,
    release_url: metadata.release_url || defaultMetadata.release_url,
    homebrew_url: metadata.homebrew_url || defaultMetadata.homebrew_url,
    container_url: rawContainerURL.includes("ghcr.io/") ? defaultMetadata.container_url : rawContainerURL,
    container_image: rawContainerImage || defaultMetadata.container_image,
    install_script_url: metadata.install_script_url || defaultMetadata.install_script_url,
    release_commit: metadata.release_commit || "",
    latest_release: metadata.latest_release || null,
    apt_repository: {
      ...defaultMetadata.apt_repository,
      ...apt,
      url: `${apt.url || defaultMetadata.apt_repository.url}`.replace(/\/?$/, "/"),
      key_url: aptKeyUrl,
      key_asc_url: apt.key_asc_url || aptKeyUrl.replace(/\.gpg$/, ".asc"),
      fingerprint: apt.fingerprint || "",
      available: Boolean(apt.available),
    },
  };
}

function formatDate(value) {
  if (!value) {
    return "Not published yet";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.valueOf())) {
    return value;
  }

  return `${new Intl.DateTimeFormat("en-GB", {
    dateStyle: "long",
    timeStyle: "short",
    timeZone: "UTC",
  }).format(parsed)} UTC`;
}

function shortCommit(value) {
  if (!value) {
    return "---";
  }

  return value.length > 10 ? value.slice(0, 7) : value;
}

function releaseCommit(metadata) {
  if (metadata.release_commit) {
    return metadata.release_commit;
  }

  const body = metadata.latest_release?.body || "";
  const match = body.match(/\/commit\/([0-9a-f]{7,40})/i);
  return match ? match[1] : "";
}

function stripMarkdown(value) {
  return value
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/\s+/g, " ")
    .trim();
}

function releaseHighlights(metadata) {
  const release = metadata.latest_release;
  const body = release?.body || "";
  const lines = body.split("\n");
  const highlights = [];
  let inIncludedChanges = false;

  for (const rawLine of lines) {
    const line = rawLine.trim();

    if (line === "## Included Changes") {
      inIncludedChanges = true;
      continue;
    }

    if (inIncludedChanges && line.startsWith("## ")) {
      break;
    }

    if (!inIncludedChanges || !line.startsWith("- ")) {
      continue;
    }

    const cleaned = stripMarkdown(line.slice(2)).replace(/\s+\([^)]*\)\s*$/, "");
    if (cleaned && !cleaned.startsWith("Automatically merged")) {
      highlights.push(cleaned);
    }

    if (highlights.length === 3) {
      break;
    }
  }

  if (highlights.length > 0) {
    return highlights;
  }

  if (release?.tag_name) {
    return [`Stable ${release.tag_name} release metadata is published`, ...fallbackHighlights.slice(1)];
  }

  return fallbackHighlights;
}

function chooseDirectDebAsset(assets = []) {
  return assets.find((asset) => asset.name.endsWith("_amd64.deb")) || assets.find((asset) => asset.name.endsWith(".deb")) || null;
}

function chooseChecksumAsset(assets = []) {
  return assets.find((asset) => asset.name === "SHA256SUMS") || assets.find((asset) => asset.name.includes("SHA256SUMS") && !asset.name.endsWith(".asc")) || null;
}

function chooseSignatureAsset(assets = []) {
  return assets.find((asset) => asset.name === "SHA256SUMS.asc") || assets.find((asset) => asset.name.endsWith("SHA256SUMS.asc")) || null;
}

function setText(id, value) {
  const element = document.getElementById(id);
  if (element) {
    element.textContent = value;
  }
}

function setHref(id, value) {
  const element = document.getElementById(id);
  if (element && value) {
    element.href = value;
  }
}

function renderCommands(metadata) {
  const release = metadata.latest_release;
  const assets = release?.assets || [];
  const directDebAsset = chooseDirectDebAsset(assets);
  const checksumAsset = chooseChecksumAsset(assets);
  const signatureAsset = chooseSignatureAsset(assets);
  const releaseTag = release?.tag_name || "latest";
  const containerImage = metadata.container_image.replace(/^https?:\/\//, "");
  const keyAscUrl = metadata.apt_repository.key_asc_url;

  setText("homebrew-command", "brew tap netspeedy/s3ctl\nbrew install s3ctl");
  setText(
    "install-script-command",
    release?.tag_name
      ? `curl -fsSL ${metadata.install_script_url} | bash -s -- --version ${release.tag_name}`
      : `curl -fsSL ${metadata.install_script_url} | bash`,
  );
  if (metadata.apt_repository.available) {
    setText(
      "apt-command",
      `sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL ${metadata.apt_repository.key_url} \\
  | sudo tee /etc/apt/keyrings/s3ctl-archive-keyring.gpg >/dev/null

sudo tee /etc/apt/sources.list.d/s3ctl.sources >/dev/null <<'EOF'
Types: deb
URIs: ${metadata.apt_repository.url}
Suites: ${metadata.apt_repository.suite}
Components: ${metadata.apt_repository.component}
Signed-By: /etc/apt/keyrings/s3ctl-archive-keyring.gpg
EOF

sudo apt update && sudo apt install s3ctl`,
    );
  } else {
    setText("apt-command", "APT repository metadata has not been published yet. Use Homebrew, the installer, or a direct .deb package.");
  }

  setText("apt-fingerprint-row", metadata.apt_repository.fingerprint ? `Archive fingerprint: ${metadata.apt_repository.fingerprint}` : "");
  setText("container-command", `docker run --rm ${containerImage}:${releaseTag}`);

  if (directDebAsset) {
    const releaseBase = `https://github.com/${metadata.github_repository}/releases/download/${release?.tag_name || "latest"}`;
    setText("deb-command", `curl -fsSLO ${directDebAsset.browser_download_url}\nsudo apt install ./${directDebAsset.name}`);

    if (signatureAsset) {
      setText(
        "deb-verify-command",
        `curl -fsSLO ${releaseBase}/SHA256SUMS
curl -fsSLO ${signatureAsset.browser_download_url}
curl -fsSL ${keyAscUrl} | gpg --import
gpg --verify ${signatureAsset.name} SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing`,
      );
      setText("checksum-note", "Checksums are GPG-signed with the release key shown above.");
    } else {
      setText(
        "deb-verify-command",
        `curl -fsSLO ${releaseBase}/SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing`,
      );
      setText(
        "checksum-note",
        checksumAsset ? `Verify direct downloads with ${checksumAsset.name} from the release page.` : "Checksums are linked from the matching GitHub release.",
      );
    }
  } else {
    setText("deb-command", "Linux release packages will appear here after the next published stable release.");
    setText("deb-verify-command", "Release verification metadata will appear here after publication.");
    setText("checksum-note", "Checksums are linked from the matching GitHub release.");
  }
}

function renderMetadata(rawMetadata) {
  const metadata = normalizeMetadata(rawMetadata);
  const release = metadata.latest_release;
  const commit = shortCommit(releaseCommit(metadata));

  setHref("site-home-link", metadata.site_url);
  setHref("nav-github-link", metadata.github_url);
  setHref("nav-releases-link", metadata.release_url);
  setHref("nav-homebrew-link", metadata.homebrew_url);
  setHref("nav-apt-link", metadata.apt_repository.url);
  setHref("nav-container-link", metadata.container_url);
  setHref("install-homebrew-link", metadata.homebrew_url);
  setHref("install-apt-link", metadata.apt_repository.url);
  setHref("install-container-link", metadata.container_url);
  setHref("install-release-link", metadata.release_url);
  setHref("footer-apt-link", metadata.apt_repository.url);
  setHref("footer-container-link", metadata.container_url);
  setHref("footer-release-link", release?.html_url || metadata.release_url);

  setText("release-version", release?.tag_name || "Awaiting release");
  setText("release-commit", commit);
  setText("release-date", formatDate(release?.published_at));
  setText("release-fingerprint", metadata.apt_repository.fingerprint || "Awaiting signed APT metadata");
  setText("footer-version", release?.tag_name?.replace(/^v/, "") || "---");
  setText("footer-commit", commit);

  const highlightsList = document.getElementById("release-highlights");
  if (highlightsList) {
    highlightsList.replaceChildren(
      ...releaseHighlights(metadata).map((highlight) => {
        const item = document.createElement("li");
        item.textContent = highlight;
        return item;
      }),
    );
  }

  renderCommands(metadata);
}

async function loadMetadata() {
  renderMetadata(defaultMetadata);

  for (const path of ["./website-metadata.json"]) {
    try {
      const response = await fetch(path, { cache: "no-store" });
      if (!response.ok) {
        continue;
      }

      renderMetadata(await response.json());
      return;
    } catch {
      // Try the next metadata source.
    }
  }
}

function wireTabs() {
  document.querySelectorAll(".install-tabs").forEach((container) => {
    container.addEventListener("click", (event) => {
      const button = event.target.closest(".tab");
      if (!button) {
        return;
      }

      const tabID = button.dataset.tab;
      container.querySelectorAll(".tab").forEach((tab) => {
        const active = tab === button;
        tab.classList.toggle("active", active);
        tab.setAttribute("aria-selected", active ? "true" : "false");
      });
      container.querySelectorAll(".tab-panel").forEach((panel) => {
        panel.classList.toggle("active", panel.id === `panel-${tabID}`);
      });
    });
  });
}

document.addEventListener("DOMContentLoaded", () => {
  wireTabs();
  void loadMetadata();
});
