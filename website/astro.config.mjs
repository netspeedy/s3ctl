import starlight from "@astrojs/starlight";
import { defineConfig, passthroughImageService } from "astro/config";

import { codeTheme } from "./src/lib/code-theme.mjs";
import rehypeCodeWords from "./src/lib/rehype-code-words.mjs";

const site = "https://netspeedy.github.io";
const base = "/s3ctl";
const repo = "https://github.com/netspeedy/s3ctl";

// The share card is written by scripts/readme/build_readme_hero.py. Bump ?v= whenever it
// changes so link previews fetch the new one.
const card = `${site}${base}/og-image.png?v=3`;
const cardAlt =
  "The s3ctl card: No console clicking. No IAM drift. Beside it, the s3ctl pail drawn in dots " +
  "with its shell prompt lit and the four provisioning steps.";

export default defineConfig({
  site,
  base,
  trailingSlash: "always",
  markdown: { rehypePlugins: [rehypeCodeWords] },
  // No page uses Astro's image pipeline, so skip sharp.
  image: { service: passthroughImageService() },
  integrations: [
    starlight({
      title: "s3ctl",
      description:
        "A single-binary CLI for creating S3-compatible buckets and issuing bucket-scoped credentials.",
      favicon: "/favicon.svg",
      editLink: { baseUrl: `${repo}/edit/main/website/` },
      // Pages build from a shallow clone, where every file would show the same commit date.
      lastUpdated: false,
      credits: false,
      customCss: [
        "@fontsource-variable/inter",
        "@fontsource-variable/jetbrains-mono",
        "./src/styles/theme.css",
      ],
      components: {
        Footer: "./src/components/Footer.astro",
        Head: "./src/components/Head.astro",
        Header: "./src/components/Header.astro",
        PageSidebar: "./src/components/PageSidebar.astro",
        PageTitle: "./src/components/PageTitle.astro",
        SiteTitle: "./src/components/SiteTitle.astro",
        ThemeProvider: "./src/components/ThemeProvider.astro",
        ThemeSelect: "./src/components/ThemeSelect.astro",
      },
      head: [
        { tag: "link", attrs: { rel: "icon", href: `${base}/favicon.ico`, sizes: "32x32" } },
        { tag: "link", attrs: { rel: "apple-touch-icon", href: `${base}/apple-touch-icon.png` } },
        {
          tag: "meta",
          attrs: { name: "theme-color", content: "#0A0D12", media: "(prefers-color-scheme: dark)" },
        },
        {
          tag: "meta",
          attrs: { name: "theme-color", content: "#FFFFFF", media: "(prefers-color-scheme: light)" },
        },
        { tag: "meta", attrs: { property: "og:image", content: card } },
        { tag: "meta", attrs: { property: "og:image:secure_url", content: card } },
        { tag: "meta", attrs: { property: "og:image:type", content: "image/png" } },
        { tag: "meta", attrs: { property: "og:image:width", content: "1200" } },
        { tag: "meta", attrs: { property: "og:image:height", content: "630" } },
        { tag: "meta", attrs: { property: "og:image:alt", content: cardAlt } },
        { tag: "meta", attrs: { name: "twitter:image", content: card } },
        { tag: "meta", attrs: { name: "twitter:image:alt", content: cardAlt } },
      ],
      tableOfContents: { minHeadingLevel: 2, maxHeadingLevel: 3 },
      expressiveCode: {
        // Code is always on a dark panel, in both site themes, so there's one theme and Starlight
        // neither swaps it nor recolours its frames from the page.
        themes: [codeTheme],
        useStarlightDarkModeSwitch: false,
        useStarlightUiThemeColors: false,
        // Shell commands read better without an empty terminal title bar. Blocks with a title,
        // like "Output" or a file name, keep their tab.
        defaultProps: { overridesByLang: { "bash,sh,shell": { frame: "none" } } },
        styleOverrides: {
          borderRadius: "12px",
          borderColor: "#1c2330",
          codeBackground: "#0b0f16",
          codeForeground: "#dce4ee",
          codeFontFamily: "var(--__sl-font-mono)",
          codeFontSize: "0.8125rem",
          codeLineHeight: "1.75",
          uiFontFamily: "var(--__sl-font)",
          uiFontSize: "0.8125rem",
          focusBorder: "#4bd8ff",
          scrollbarThumbColor: "#2a323d",
          scrollbarThumbHoverColor: "#3a4452",
          frames: {
            shadowColor: "transparent",
            editorTabBarBackground: "#0f141d",
            editorActiveTabBackground: "#0f141d",
            editorActiveTabForeground: "#e8ecf1",
            editorActiveTabIndicatorTopColor: "transparent",
            editorActiveTabIndicatorBottomColor: "#4bd8ff",
            editorTabBarBorderBottomColor: "#1c2330",
            editorBackground: "#0b0f16",
            terminalBackground: "#0b0f16",
            terminalTitlebarBackground: "#0f141d",
            terminalTitlebarForeground: "#8f9bab",
            terminalTitlebarBorderBottomColor: "#1c2330",
            terminalTitlebarDotsOpacity: "0",
            inlineButtonBackground: "#141b26",
            inlineButtonForeground: "#c3ccd8",
            inlineButtonBorder: "#273042",
            tooltipSuccessBackground: "#1c7c4d",
          },
        },
      },
      sidebar: [
        {
          label: "Getting started",
          items: [
            "introduction",
            "quickstart",
            "installation",
          ],
        },
        {
          label: "Guides",
          items: [
            "guides/configuration",
            "guides/batch-provisioning",
            "guides/scoped-credentials",
            "guides/bucket-policies",
            "guides/deleting-buckets",
          ],
        },
        {
          label: "OVHcloud",
          items: [
            { label: "Overview", slug: "ovhcloud" },
            "ovhcloud/oauth2-and-iam",
            "ovhcloud/credential-rotation",
            "ovhcloud/policy-repair",
            "ovhcloud/bucket-deletion",
          ],
        },
        {
          label: "Reference",
          items: [
            "reference/cli",
            "reference/json-output",
            "reference/policy-templates",
            "reference/iam-permissions",
          ],
        },
      ],
    }),
  ],
});
