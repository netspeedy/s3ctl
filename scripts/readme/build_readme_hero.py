#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = ["fonttools==4.67.0", "brotli==1.2.0"]
# ///
"""Build the animated README hero, assets/s3ctl-readme-hero.svg, and the icons made from its mark.

The banner follows the soakes/cv README hero: the mark, a two-line headline, a short body and a
meta row on the left, and a generated scene on the right. The mark is a pail seen slightly from
above with a shell prompt cut into it (see Mark). The scene is that pail drawn large in dots,
with the prompt lit and its cursor blinking, a scan running down it, and the steps s3ctl takes
(bucket, versioning, bucket policy, scoped key) lighting up beside it in turn. When the repo has
a website, the favicon, the Apple touch icon and favicon.ico are written from the same geometry.

Edit the copy, layout or timing here and run `make readme-hero` rather than editing the
generated SVG. The headline is positioned from measured advances (see ADVANCE); if you change
its words, measure them again with measure_headline.py. Inter and JetBrains Mono (SIL Open Font
License 1.1) are the Google Fonts builds, kept on assets.netspeedy.io and pinned by checksum in
FAMILIES. They're subset here to the characters the finished SVG renders and embedded, so the
hero renders the same on every platform, and because fontTools is pinned too, a rebuild gives
the same bytes. It runs under `uv run`, which installs that fontTools, and the first run needs
network access to download the fonts.
"""

from __future__ import annotations

import base64
import hashlib
import io
import math
import os
import random
import shutil
import struct
import subprocess
import tempfile
import urllib.request
import xml.etree.ElementTree as ET
from pathlib import Path

from fontTools import subset
from fontTools.ttLib import TTFont

ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "assets" / "s3ctl-readme-hero.svg"
# assets.netspeedy.io's Cloudflare zone turns away urllib's default user agent.
USER_AGENT = "netspeedy-readme-hero"

# The mark's cyan on its ink tile, amber for accents, and oakes.me.uk's inks so the netspeedy
# banners read as one family. DEEP_CYAN is the mark and link colour on light surfaces, where
# CYAN is too pale (it's about 5:1 on white).
BG = "#060B13"
TILE = "#0B1220"
DEEP_CYAN = "#0B74B8"
INK = "#F1F5FA"
INK_2 = "#B5C0CD"
INK_3 = "#808D9E"
CYAN = "#4BD8FF"
CYAN_2 = "#A8ECFF"
CYAN_3 = "#18A8FF"
AMBER = "#F5B841"
AMBER_2 = "#FDE1A7"
GREEN = "#3CD788"
RED = "#FF5D6C"

# Fonts: css class -> (source font, its sha256, weight range, embedded family name). The
# sources are the Google Fonts builds of Inter 4.001 and JetBrains Mono 2.211, kept on
# assets.netspeedy.io and cached after the first download. A new version goes up there
# under a new filename, so these files never change.
FONT_HOST = "https://assets.netspeedy.io/versioned/fonts"
FONT_CACHE = Path.home() / ".cache" / "netspeedy-fonts"
FAMILIES = {
    "sans": (
        "inter-opsz-wght.4.001.ttf",
        "29160a80ff49ddcab2c97711247e08b1fab27a484a329ce8b813d820dc559031",
        "400 800",
        "Hero Sans",
    ),
    "mono": (
        "jetbrains-mono-wght.2.211.ttf",
        "48715a42ec242c21e9f02692891e147d022299a52e48d5e413e1a942193ffeda",
        "400 700",
        "Hero Mono",
    ),
}
FONT_CSS = (
    ".sans{font-family:'Hero Sans',-apple-system,'Segoe UI','Helvetica Neue',Arial,sans-serif;"
    "font-optical-sizing:auto;}"
    ".mono{font-family:'Hero Mono','SF Mono',Menlo,Consolas,monospace;}"
)
TWINKLE_CSS = (
    ".tw{animation:tw 5.2s ease-in-out infinite;transform-box:fill-box;transform-origin:center}"
    ".tw1{animation-duration:6.4s}.tw2{animation-duration:4.6s}.tw3{animation-duration:7.3s}"
    "@keyframes tw{0%,100%{opacity:.85}50%{opacity:.15}}"
)

# Starfield: seeded, so rebuilding gives the same sky. No star sits behind the copy.
STAR_SEED = 11
STARS = 110
TWINKLING = 16
COPY_BOX = (36, 30, 520, 340)

# Headline advances in px, measured by measure_headline.py on Inter 800 at 44px (opsz 32,
# letter-spacing -1.3). The swash and the gradients are placed from these.
ADVANCE = {
    "No ": 63.17,
    "console": 154.93,
    "No console clicking.": 387.96,
    "No IAM drift.": 243.21,
}


# Fonts


def fetch(url: str) -> bytes:
    request = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    with urllib.request.urlopen(request, timeout=30) as response:
        return response.read()


def font_source(name: str, sha256: str) -> bytes:
    """A pinned source font from FONT_HOST, checked against its sha256 and cached."""
    cached = FONT_CACHE / name
    if cached.exists():
        data = cached.read_bytes()
        if hashlib.sha256(data).hexdigest() == sha256:
            return data
    data = fetch(f"{FONT_HOST}/{name}")
    if hashlib.sha256(data).hexdigest() != sha256:
        raise SystemExit(f"{FONT_HOST}/{name} doesn't match its pinned sha256")
    FONT_CACHE.mkdir(parents=True, exist_ok=True)
    cached.write_bytes(data)
    return data


def subset_font(source: bytes, text: str) -> bytes:
    """The variable font cut down to the given characters, as woff2.

    Every axis is kept: trimming them with fontTools' instancer moved the headline's
    pixels. The source's head timestamp is kept too, so the same text always gives the
    same bytes.
    """
    font = TTFont(io.BytesIO(source), recalcTimestamp=False)
    options = subset.Options()
    options.hinting = False
    subsetter = subset.Subsetter(options)
    subsetter.populate(text=text)
    subsetter.subset(font)
    font.flavor = "woff2"
    out = io.BytesIO()
    font.save(out)
    return out.getvalue()


def collect_chars(svg: str) -> dict[str, set[str]]:
    """Characters rendered by each font class, inherited through groups."""
    chars: dict[str, set[str]] = {name: set() for name in FAMILIES}

    def family_of(element: ET.Element, inherited: str | None) -> str | None:
        for cls in (element.get("class") or "").split():
            if cls in FAMILIES:
                return cls
        return inherited

    def walk(element: ET.Element, inherited: str | None, in_text: bool) -> None:
        tag = element.tag.split("}")[-1]
        family = family_of(element, inherited)
        if tag in ("style", "title", "desc", "script"):
            return
        is_text = in_text or tag in ("text", "tspan", "textPath")
        if is_text and tag in ("text", "tspan", "textPath") and element.text:
            if family is None:
                raise SystemExit(f"text without a font class: {element.text!r}")
            chars[family].update(element.text)
        for child in element:
            walk(child, family, is_text)
            if is_text and child.tail and child.tail.strip():
                chars[family or "sans"].update(child.tail)

    walk(ET.fromstring(svg), None, False)
    return chars


def font_faces(svg: str) -> str:
    """One @font-face per family, subset to exactly the characters the SVG renders."""
    chars = collect_chars(svg)
    faces = []
    for cls, (name, sha256, weights, local) in FAMILIES.items():
        used = chars[cls]
        if not used:
            continue
        text = "".join(sorted((used - set("\n\t\r")) | {" "}))
        data = base64.b64encode(subset_font(font_source(name, sha256), text)).decode()
        faces.append(
            f"@font-face{{font-family:'{local}';font-weight:{weights};"
            f"font-style:normal;src:url(data:font/woff2;base64,{data}) "
            "format('woff2');}"
        )
    return "".join(faces)


# Small helpers


def esc(text: str) -> str:
    return text.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def num(value: float) -> str:
    """One decimal place, without a trailing .0."""
    text = f"{value:.1f}"
    return text[:-2] if text.endswith(".0") else text


def mono_width(chars: int, size: float, spacing: float = 0.0) -> float:
    """JetBrains Mono advances exactly 0.6 em; letter-spacing applies per glyph."""
    return chars * (size * 0.6 + spacing)


def gradient(kind: str, attrs: str, stops: list[tuple[str, str, str]]) -> list[str]:
    lines = [f"    <{kind} {attrs}>"]
    for offset, colour, opacity in stops:
        alpha = f' stop-opacity="{opacity}"' if opacity else ""
        lines.append(f'      <stop offset="{offset}" stop-color="{colour}"{alpha} />')
    return [*lines, f"    </{kind}>"]


def dot_paths(
    points: list[tuple[float, float, int]],
    styles: dict[int, tuple[float, float]],
    colour: str = "#D3E4F4",
) -> str:
    """Dots grouped by band, each band one path of round-capped stubs."""
    paths = []
    for band, (opacity, width) in styles.items():
        pts = sorted(((x, y) for x, y, b in points if b == band), key=lambda p: (round(p[1]), p[0]))
        if not pts:
            continue
        moves = []
        px = py = 0.0
        for x, y in pts:
            # "h.01", not "h0": some renderers drop zero-length subpaths.
            moves.append(f"m{num(x - px)} {num(y - py)}h.01")
            px, py = x + 0.01, y
        paths.append(
            f'<path d="M0 0{"".join(moves)}" fill="none" stroke="{colour}" '
            f'stroke-opacity="{opacity}" stroke-width="{width}" stroke-linecap="round" />'
        )
    return "".join(paths)


def chip(
    x: float,
    y: float,
    label: str,
    colour: str,
    cls: str = "",
    size: float = 10.5,
    spacing: float = 1.2,
    fill_opacity: float = 0.9,
) -> str:
    """A mono label in a small outlined box; (x, y) is the box's top-left corner."""
    width = mono_width(len(label), size, spacing) - spacing + 16
    attr = f' class="{cls}"' if cls else ""
    return (
        f'<g{attr}><rect x="{num(x)}" y="{num(y)}" width="{num(width)}" height="20" rx="4" '
        f'fill="#0A0F18" fill-opacity="{fill_opacity}" stroke="{colour}" stroke-opacity="0.4" />'
        f'<text class="mono" x="{num(x + width / 2)}" y="{num(y + 13.8)}" text-anchor="middle" '
        f'font-size="{size}" letter-spacing="{spacing}" fill="{colour}">{esc(label)}</text></g>'
    )


# Shared card


def starfield(free: list[tuple[float, float, float, float]]) -> tuple[list[str], list[str]]:
    """Background gradient and nebula defs, plus the body: base, glows and seeded stars."""
    rng = random.Random(STAR_SEED)
    stars = []
    tries = 0
    while len(stars) < STARS and tries < STARS * 50:
        tries += 1
        x, y = rng.uniform(8, 1192), rng.uniform(8, 392)
        if any(x0 <= x <= x1 and y0 <= y <= y1 for x0, y0, x1, y1 in free):
            continue
        radius = rng.choice([0.55, 0.6, 0.7, 0.8, 0.9, 1.0, 1.15])
        stars.append((x, y, radius, rng.uniform(0.18, 0.6)))
    twinkling = [
        f'<circle class="tw tw{i % 4}" cx="{num(x)}" cy="{num(y)}" r="{num(r + 0.25)}" '
        f'fill="#DDF2FF" opacity="{o + 0.25:.2f}" '
        f'style="animation-delay:-{rng.uniform(0, 6):.1f}s" />'
        for i, (x, y, r, o) in enumerate(stars[:TWINKLING])
    ]
    plain = "".join(
        f'<circle cx="{num(x)}" cy="{num(y)}" r="{num(r)}" opacity="{o:.2f}" />'
        for x, y, r, o in stars[TWINKLING:]
    )
    defs = [
        '    <linearGradient id="spaceBg" x1="0" y1="0" x2="0.35" y2="1">',
        '      <stop offset="0" stop-color="#081020" />',
        f'      <stop offset="0.55" stop-color="{BG}" />',
        '      <stop offset="1" stop-color="#0B1424" />',
        "    </linearGradient>",
        *gradient(
            "radialGradient",
            'id="nebCyan" cx="0.5" cy="0.5" r="0.5"',
            [("0", CYAN_3, "0.34"), ("1", CYAN_3, "0")],
        ),
        *gradient(
            "radialGradient",
            'id="nebIndigo" cx="0.5" cy="0.5" r="0.5"',
            [("0", "#3B4BB8", "0.3"), ("1", "#3B4BB8", "0")],
        ),
    ]
    body = [
        '    <rect width="1200" height="400" fill="url(#spaceBg)" />',
        '    <ellipse cx="190" cy="-30" rx="520" ry="260" fill="url(#nebIndigo)" />',
        '    <ellipse cx="1000" cy="420" rx="560" ry="300" fill="url(#nebCyan)" opacity="0.5" />',
        f'    <g fill="#CFE6FF">{plain}</g>',
        f'    <g>{"".join(twinkling)}</g>',
    ]
    return defs, body


class Mark:
    """The s3ctl mark on a 64-unit grid.

    A pail seen slightly from above: an elliptical rim ring (each ellipse is `k` times as
    tall as it is wide), a gap, then a body that tapers to `taper` of the rim's width, with a
    shell prompt cut out of the body. Below 24px the mark uses SMALL, a heavier cut with a
    thicker rim, a wider gap and the chevron alone.
    """

    def __init__(
        self,
        top: float = 13.5,
        bottom: float = 52.5,
        rx: float = 21.0,
        taper: float = 0.8,
        k: float = 0.28,
        ring: float = 3.4,
        gap: float = 3.0,
        stroke: float = 3.6,
        cursor: bool = True,
        soften: float = 1.0,
    ) -> None:
        self.top, self.bottom, self.rx, self.rx_base = top, bottom, rx, rx * taper
        self.k, self.ring, self.gap, self.stroke = k, ring, gap, stroke
        self.cursor, self.soften = cursor, soften

    def rx_at(self, y: float) -> float:
        return self.rx + (self.rx_base - self.rx) * (y - self.top) / (self.bottom - self.top)

    def rim(self) -> str:
        """The rim ring: the outer ellipse with the mouth cut out (fill-rule evenodd)."""
        rx, ry = self.rx, self.rx * self.k
        ri, ryi = rx - self.ring, max(ry - self.ring * 0.75, 0.6)
        return ellipse(32, self.top, rx, ry, 0) + ellipse(32, self.top, ri, ryi, 1)

    def body(self) -> str:
        """The body's front face, from the gap under the rim down to the base."""
        y0, y1, k = self.top + self.gap, self.bottom, self.k
        r0, r1 = self.rx_at(y0), self.rx_base
        return (
            f"M{n2(32 - r0)} {n2(y0)}A{n2(r0)} {n2(r0 * k)} 0 0 0 {n2(32 + r0)} {n2(y0)}"
            f"L{n2(32 + r1)} {n2(y1)}A{n2(r1)} {n2(r1 * k)} 0 0 1 {n2(32 - r1)} {n2(y1)}Z"
        )

    def prompt(self) -> tuple[list[tuple[float, float]], list[tuple[float, float]]]:
        """The chevron's three points and the cursor's two, optically centred on the body."""
        mid = (self.top + self.gap + self.bottom) / 2 + 3.5
        chevron = [(23.0, mid - 6), (29.5, mid), (23.0, mid + 6)]
        cursor = [(33.0, mid + 6.5), (41.5, mid + 6.5)] if self.cursor else []
        return chevron, cursor

    def cut(self) -> str:
        """Mask contents that cut the prompt out of the body."""
        chevron, cursor = self.prompt()
        parts = [
            f'<path d="{polyline(chevron)}" fill="none" stroke="#000" stroke-width="{n2(self.stroke)}" '
            'stroke-linecap="round" stroke-linejoin="round" />'
        ]
        if cursor:
            parts.append(
                f'<path d="{polyline(cursor)}" fill="none" stroke="#000" '
                f'stroke-width="{n2(self.stroke)}" stroke-linecap="round" />'
            )
        return "".join(parts)

    def glyph(self, colour: str, mask_id: str) -> str:
        """The mark in one colour, referring to a mask made by mask()."""
        soft = (
            f' stroke="{colour}" stroke-width="{n2(self.soften)}" stroke-linejoin="round"'
            if self.soften
            else ""
        )
        return (
            f'<path d="{self.rim()}" fill="{colour}" fill-rule="evenodd" />'
            f'<path d="{self.body()}" fill="{colour}"{soft} mask="url(#{mask_id})" />'
        )

    def mask(self, mask_id: str) -> str:
        return (
            f'<mask id="{mask_id}" maskUnits="userSpaceOnUse" x="0" y="0" width="64" height="64">'
            f'<rect width="64" height="64" fill="#fff" />{self.cut()}</mask>'
        )


MARK = Mark()
SMALL = Mark(top=12.5, bottom=53.5, rx=22.5, ring=4.6, gap=3.8, stroke=6.4, cursor=False, soften=0)
TILE_SCALE = {MARK: 0.78, SMALL: 0.86}  # how much of the tile the glyph fills


def n2(value: float) -> str:
    """Two decimal places without trailing zeros, for the 64-unit mark."""
    return f"{value:.2f}".rstrip("0").rstrip(".")


def ellipse(cx: float, cy: float, rx: float, ry: float, sweep: int) -> str:
    return (
        f"M{n2(cx - rx)} {n2(cy)}A{n2(rx)} {n2(ry)} 0 1 {sweep} {n2(cx + rx)} {n2(cy)}"
        f"A{n2(rx)} {n2(ry)} 0 1 {sweep} {n2(cx - rx)} {n2(cy)}Z"
    )


def polyline(points: list[tuple[float, float]]) -> str:
    return "M" + "L".join(f"{n2(x)} {n2(y)}" for x, y in points)


def tile(mark: Mark, mask_id: str, radius: float = 14) -> str:
    """The app icon: the cyan glyph on the ink tile, 64 units square."""
    scale = TILE_SCALE[mark]
    offset = 32 - 32 * scale
    return (
        f'<rect width="64" height="64" rx="{n2(radius)}" fill="{TILE}" />'
        f'<g transform="translate({n2(offset)} {n2(offset)}) scale({n2(scale)})">'
        f"{mark.glyph(CYAN, mask_id)}</g>"
    )


def mark_symbols() -> list[str]:
    """The mark as a symbol, for the lockup."""
    return [
        f"    {MARK.mask('markCut')}",
        f'    <symbol id="mark" viewBox="0 0 64 64">{MARK.glyph(CYAN, "markCut")}</symbol>',
    ]


def lockup() -> list[str]:
    scale = TILE_SCALE[MARK]
    return [
        f'  <rect x="48.5" y="38.5" width="33" height="33" rx="7.5" fill="{TILE}" stroke="#22324A" '
        'stroke-width="1.2" />',
        f'  <use href="#mark" x="{num(48.5 + 16.5 * (1 - scale))}" y="{num(38.5 + 16.5 * (1 - scale))}" '
        f'width="{num(33 * scale)}" height="{num(33 * scale)}" />',
        f'  <text class="sans" x="94" y="52.5" font-size="15" font-weight="600" fill="{INK}">'
        "s3ctl</text>",
        f'  <text class="sans" x="94" y="69.5" font-size="12" fill="{INK_3}">'
        "netspeedy · S3 bucket provisioning CLI</text>",
    ]


def headline(prefix: str, word: str, suffix: str, second: str) -> tuple[list[str], list[str]]:
    """Two lines at 44px: line one with a highlighted word and an amber swash under it."""
    x = 46.0
    first = prefix + word + suffix
    start = x + ADVANCE[prefix]
    bare = word.rstrip(",.")
    end = start + ADVANCE[bare]
    defs = [
        *gradient(
            "linearGradient",
            f'id="titleWhite" gradientUnits="userSpaceOnUse" x1="{num(x)}" y1="0" '
            f'x2="{num(x + ADVANCE[first])}" y2="0"',
            [("0", "#FFFFFF", ""), ("1", "#D5ECF8", "")],
        ),
        *gradient(
            "linearGradient",
            f'id="hl" gradientUnits="userSpaceOnUse" x1="{num(start)}" y1="0" '
            f'x2="{num(end)}" y2="0"',
            [("0", CYAN_2, ""), ("0.5", "#F4F7FA", ""), ("1", AMBER, "")],
        ),
        *gradient(
            "linearGradient",
            f'id="titleCyan" gradientUnits="userSpaceOnUse" x1="{num(x)}" y1="0" '
            f'x2="{num(x + ADVANCE[second])}" y2="0"',
            [("0", CYAN_2, ""), ("0.55", CYAN, ""), ("1", CYAN_3, "")],
        ),
        *gradient(
            "linearGradient",
            'id="uline" x1="0" y1="0" x2="1" y2="0"',
            [("0", AMBER, "0.35"), ("0.5", AMBER, ""), ("1", AMBER, "0.5")],
        ),
    ]
    span = end - start
    tail = f'<tspan fill="url(#titleWhite)">{esc(suffix)}</tspan>' if suffix else ""
    els = [
        f'  <text class="sans" x="{num(x)}" y="160" font-size="44" font-weight="800" '
        'letter-spacing="-1.3" xml:space="preserve">'
        f'<tspan fill="url(#titleWhite)">{esc(prefix)}</tspan>'
        f'<tspan fill="url(#hl)">{esc(word)}</tspan>{tail}</text>',
        f'  <path d="M{num(start + 2)} 172.5q{num(span * 0.5)} -6.5 {num(span - 6)} -1.5" '
        'fill="none" stroke="url(#uline)" stroke-width="2.6" stroke-linecap="round" />',
        f'  <text class="sans" x="{num(x)}" y="212" font-size="44" font-weight="800" '
        f'letter-spacing="-1.3" fill="url(#titleCyan)">{esc(second)}</text>',
    ]
    return defs, els


def body(lines: list[list[tuple[str, bool]]]) -> list[str]:
    out = [f'  <text class="sans" font-size="15.5" fill="{INK_2}" xml:space="preserve">']
    for i, segments in enumerate(lines):
        spans = "".join(
            (
                f'<tspan class="mono" font-size="14" fill="{CYAN_2}">{esc(text)}</tspan>'
                if mono
                else esc(text)
            )
            for text, mono in segments
        )
        out.append(f'    <tspan x="48" y="{254 + 23 * i}">{spans}</tspan>')
    out.append("  </text>")
    return out


def meta_row(labels: list[str]) -> str:
    x = 48.0
    parts = []
    for i, label in enumerate(labels):
        if i:
            parts.append(f'<circle cx="{num(x + 9)}" cy="318.3" r="2.1" fill="{AMBER}" />')
            x += 19
        parts.append(f'<text x="{num(x)}" y="322">{esc(label)}</text>')
        x += mono_width(len(label), 11.5, 1.5)
    return (
        f'  <g class="mono" font-size="11.5" letter-spacing="1.5" fill="{INK_3}">'
        f'{"".join(parts)}</g>'
    )


# Scene: the pail


# The mark scaled up PAIL_SCALE times, with its rim centred on (PAIL_CX, PAIL_TOP).
PAIL_CX, PAIL_TOP, PAIL_SCALE = 944.0, 72.0, 7.0


def at(x: float, y: float) -> tuple[float, float]:
    """A point on the mark's 64-unit grid, placed in the scene."""
    return PAIL_CX + (x - 32) * PAIL_SCALE, PAIL_TOP + (y - MARK.top) * PAIL_SCALE


def pail_radius(t: float) -> float:
    """The pail's radius t px below the rim."""
    return MARK.rx_at(MARK.top + t / PAIL_SCALE) * PAIL_SCALE


def surface(t: float, theta: float) -> tuple[float, float]:
    r = pail_radius(t)
    return PAIL_CX + r * math.sin(theta), PAIL_TOP + t + r * MARK.k * math.cos(theta)


def front_arc(t: float, steps: int = 48) -> str:
    points = [surface(t, -math.pi / 2 + math.pi * q / steps) for q in range(steps + 1)]
    return "M" + "L".join(f"{num(x)} {num(y)}" for x, y in points)


def segment_distance(p: tuple[float, float], a: tuple[float, float], b: tuple[float, float]) -> float:
    (px, py), (ax, ay), (bx, by) = p, a, b
    dx, dy = bx - ax, by - ay
    t = max(0.0, min(1.0, ((px - ax) * dx + (py - ay) * dy) / (dx * dx + dy * dy)))
    return math.hypot(px - ax - t * dx, py - ay - t * dy)


def scene_pail() -> tuple[list[str], str, str, list[tuple[float, float, float, float]]]:
    """The mark as a dotted pail, its prompt lit, scanned top to bottom, with s3ctl's steps."""
    loop = 6.0
    s, k = PAIL_SCALE, MARK.k
    cx, top = PAIL_CX, PAIL_TOP
    rx0 = MARK.rx * s
    depth = (MARK.bottom - MARK.top) * s
    gap = MARK.gap * s
    r_top, r_base = pail_radius(gap), MARK.rx_base * s

    chevron, cursor = MARK.prompt()
    chevron = [at(x, y) for x, y in chevron]
    cursor = [at(x, y) for x, y in cursor]
    strokes = list(zip(chevron, chevron[1:])) + [(cursor[0], cursor[1])]
    clear = MARK.stroke * s / 2 + 5  # dots stay this far from the prompt's centre lines

    points = []
    t = gap + 7.0
    ring = 0
    while t <= depth:
        r = pail_radius(t)
        count = int(2 * math.pi * r / 9.5)
        for q in range(count):
            theta = 2 * math.pi * (q + (ring & 1) * 0.5) / count
            c = math.cos(theta)
            if c <= 0.03:
                continue
            x, y = surface(t, theta)
            if any(segment_distance((x, y), a, b) < clear for a, b in strokes):
                continue
            band = 0 if c < 0.3 else 1 if c < 0.55 else 2 if c < 0.8 else 3
            points.append((x, y, band))
        t += 9.0
        ring += 1
    field = dot_paths(points, {0: (0.22, 1.2), 1: (0.38, 1.4), 2: (0.55, 1.55), 3: (0.72, 1.65)})

    # The body's front face, as in the mark: the gap's front arc, down the right side, round
    # the front of the base and back up.
    y0, y1 = top + gap, top + depth
    body = (
        f"M{num(cx - r_top)} {num(y0)}A{num(r_top)} {num(r_top * k)} 0 0 0 {num(cx + r_top)} {num(y0)}"
        f"L{num(cx + r_base)} {num(y1)}A{num(r_base)} {num(r_base * k)} 0 0 1 {num(cx - r_base)} {num(y1)}Z"
    )
    ring_w = MARK.ring * s
    rim = (
        f'<path d="M{num(cx - rx0)} {num(top)}A{num(rx0)} {num(rx0 * k)} 0 1 0 {num(cx + rx0)} {num(top)}'
        f"A{num(rx0)} {num(rx0 * k)} 0 1 0 {num(cx - rx0)} {num(top)}Z"
        f"M{num(cx - rx0 + ring_w)} {num(top)}"
        f"A{num(rx0 - ring_w)} {num(rx0 * k - ring_w * 0.75)} 0 1 1 {num(cx + rx0 - ring_w)} {num(top)}"
        f"A{num(rx0 - ring_w)} {num(rx0 * k - ring_w * 0.75)} 0 1 1 {num(cx - rx0 + ring_w)} {num(top)}Z"
        '" fill="url(#rimGrad)" fill-rule="evenodd" />'
    )
    mouth = (
        f'<ellipse cx="{num(cx)}" cy="{num(top)}" rx="{num(rx0 - ring_w)}" '
        f'ry="{num(rx0 * k - ring_w * 0.75)}" fill="url(#mouth)" />'
    )

    scans = []
    step = 12.0
    count = int((depth - gap) / step)
    sweep = 2.6
    for q in range(count + 1):
        delay = q * sweep / count
        scans.append(
            f'<path class="sc" d="{front_arc(gap + q * step)}" fill="none" stroke="{CYAN_2}" '
            f'stroke-width="1.3" style="animation-delay:{delay:.2f}s" />'
        )

    # The prompt, lit: a soft glow under each stroke, the chevron in pale cyan and the cursor
    # in amber, blinking.
    width = MARK.stroke * s * 0.56
    chevron_d = "M" + "L".join(f"{num(x)} {num(y)}" for x, y in chevron)
    cursor_d = f"M{num(cursor[0][0])} {num(cursor[0][1])}H{num(cursor[1][0])}"
    prompt = (
        f'<path d="{chevron_d}" fill="none" stroke="{CYAN}" stroke-opacity="0.16" '
        f'stroke-width="{num(width * 2.4)}" stroke-linecap="round" stroke-linejoin="round" />'
        f'<path d="{chevron_d}" fill="none" stroke="{CYAN_2}" stroke-width="{num(width)}" '
        'stroke-linecap="round" stroke-linejoin="round" />'
        f'<g class="cur"><path d="{cursor_d}" fill="none" stroke="{AMBER}" stroke-opacity="0.2" '
        f'stroke-width="{num(width * 2.4)}" stroke-linecap="round" />'
        f'<path d="{cursor_d}" fill="none" stroke="{AMBER}" stroke-width="{num(width)}" '
        'stroke-linecap="round" /></g>'
    )

    steps = [
        ("BUCKET", 44.0, CYAN_2),
        ("VERSIONING", 106.0, CYAN_2),
        ("BUCKET POLICY", 168.0, CYAN_2),
        ("SCOPED KEY", 230.0, AMBER_2),
    ]
    labels = []
    for q, (label, depth_at, colour) in enumerate(steps):
        ax, ay = surface(depth_at, -1.05)
        chip_w = mono_width(len(label), 10.5, 1.2) - 1.2 + 16
        bx = 600.0
        by = ay - 10
        labels.append(
            f'<g class="st" style="animation-delay:{q * loop / len(steps) - loop:.2f}s">'
            f'<path d="M{num(bx + chip_w)} {num(ay)}H{num(ax - 4)}" fill="none" stroke="{colour}" '
            'stroke-opacity="0.5" stroke-width="1" />'
            f'<circle cx="{num(ax)}" cy="{num(ay)}" r="3" fill="{colour}" />'
            f'<circle class="ping" cx="{num(ax)}" cy="{num(ay)}" r="3" fill="none" '
            f'stroke="{colour}" stroke-width="1.1" />' + chip(bx, by, label, colour) + "</g>"
        )

    defs = [
        *gradient(
            "linearGradient",
            'id="pailBody" x1="0" y1="0" x2="1" y2="0"',
            [("0", "#081322", ""), ("0.5", "#0E2036", ""), ("1", "#081322", "")],
        ),
        *gradient(
            "radialGradient",
            'id="mouth" cx="0.5" cy="0.62" r="0.6"',
            [("0", "#02060C", ""), ("0.7", "#061020", ""), ("1", "#0B2033", "")],
        ),
        *gradient(
            "linearGradient",
            'id="rimGrad" x1="0" y1="0" x2="1" y2="1"',
            [("0", CYAN_2, ""), ("0.45", CYAN, ""), ("1", CYAN_3, "")],
        ),
        *gradient(
            "radialGradient",
            'id="aura" cx="0.5" cy="0.45" r="0.5"',
            [("0", CYAN_3, "0.32"), ("0.6", CYAN_3, "0.1"), ("1", CYAN_3, "0")],
        ),
        f'    <clipPath id="pailClip"><path d="{body}" /></clipPath>',
    ]
    css = (
        f".sc{{opacity:0;animation:sc {loop}s linear infinite}}"
        f"@keyframes sc{{0%{{opacity:0}}1.5%{{opacity:.85}}9%{{opacity:0}}100%{{opacity:0}}}}"
        ".cur{animation:cur 1.1s step-end infinite}"
        "@keyframes cur{0%{opacity:1}50%{opacity:0}}"
        f".st{{animation:st {loop}s ease-out infinite}}"
        f"@keyframes st{{0%{{opacity:1}}20%{{opacity:1}}30%{{opacity:.45}}100%{{opacity:.45}}}}"
        ".ping{transform-box:fill-box;transform-origin:center;opacity:0;"
        f"animation:ping {loop}s ease-out infinite;animation-delay:inherit}}"
        "@keyframes ping{0%{opacity:.9;transform:scale(1)}14%{opacity:0;transform:scale(4.5)}"
        "100%{opacity:0;transform:scale(4.5)}}"
    )
    scene = "".join(
        [
            f'<ellipse cx="{num(cx)}" cy="{num(top + 150)}" rx="330" ry="230" fill="url(#aura)" />',
            f'<path d="{body}" fill="url(#pailBody)" />',
            f'<path d="{body}" fill="none" stroke="{CYAN}" stroke-opacity="0.35" '
            'stroke-width="1.2" />',
            f'<g clip-path="url(#pailClip)">{field}{"".join(scans)}</g>',
            mouth,
            rim,
            prompt,
            "".join(labels),
        ]
    )
    bottom = top + depth + r_base * k
    return defs, css, scene, [(cx - rx0 - 20, top - rx0 * k - 30, cx + rx0 + 20, bottom + 20)]


# Copy

CARD = {
    "title": "s3ctl - no console clicking, no IAM drift",
    "desc": (
        "The s3ctl mark, a pail with a shell prompt cut into it, drawn as a large dotted pail with "
        "its prompt lit and the cursor blinking, scanned from top to bottom, with the steps s3ctl "
        "runs: bucket, versioning, bucket policy, scoped key."
    ),
    "headline": ("No ", "console", " clicking.", "No IAM drift."),
    "body": [
        [("Buckets, versioning, policies and scoped keys from", False)],
        [
            ("flags, JSON or a CSV batch. ", False),
            ("--dry-run", True),
            (" shows the plan first.", False),
        ],
    ],
    "meta": ["PLAN", "CREATE", "ROTATE", "DELETE"],
    "scene": scene_pail,
}


def build_svg(spec: dict = CARD) -> str:
    scene_defs, scene_css, scene, free = spec["scene"]()
    star_defs, star_body = starfield([COPY_BOX, *free])
    head_defs, head_els = headline(*spec["headline"])
    lines = [
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 400" width="960" height="320" '
        'role="img" aria-labelledby="title desc">',
        f'  <title id="title">{esc(spec["title"])}</title>',
        f'  <desc id="desc">{esc(spec["desc"])}</desc>',
        "",
        "  <defs>",
        "    <style>",
        "      /* Inter and JetBrains Mono, SIL Open Font License 1.1, subset to the glyphs used "
        "here. */",
        "      __FONT_FACES__",
        f"      {FONT_CSS}",
        f"      {TWINKLE_CSS}",
        f"      {scene_css}",
        "      @media (prefers-reduced-motion: reduce){",
        "        *{animation:none !important}",
        "      }",
        "    </style>",
        *star_defs,
        *mark_symbols(),
        *head_defs,
        *scene_defs,
        '    <clipPath id="frame"><rect x="1" y="1" width="1198" height="398" rx="20" />'
        "</clipPath>",
        "  </defs>",
        "",
        '  <g clip-path="url(#frame)">',
        *star_body,
        f"    {scene}",
        "  </g>",
        '  <rect x="0.75" y="0.75" width="1198.5" height="398.5" rx="20" fill="none" '
        'stroke="#1A2638" stroke-width="1.5" />',
        "",
        "  <!-- Left: brand and headline -->",
        *lockup(),
        "",
        *head_els,
        *body(spec["body"]),
        meta_row(spec["meta"]),
        "</svg>",
    ]
    return "\n".join(lines) + "\n"


# Social cards

# Both cards are the hero recomposed and frozen at its rest frame. The website's og:image is
# 1200x630, the size link previews expect, and is only written when the repo has a website.
# The GitHub social preview is 1280x640 and is uploaded by hand under Settings > General >
# Social preview, since GitHub has no API for it.
REPO = "s3ctl"
OG_CARD = ROOT / "website" / "public" / "og-image"
PREVIEW_CARD = ROOT / "assets" / f"{REPO}-social-preview"
SVG_NS = "http://www.w3.org/2000/svg"
CHROME_CANDIDATES = (
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "google-chrome",
    "chromium",
    "chromium-browser",
)


def build_social_svg(hero: str, width: int, height: int) -> str:
    """The hero's copy column on the left and its scene on the right, scaled to fill the card."""
    ET.register_namespace("", SVG_NS)
    root = ET.fromstring(hero)

    def tag(name: str) -> str:
        return f"{{{SVG_NS}}}{name}"

    children = list(root)
    defs = root.find(tag("defs"))
    style = defs.find(tag("style"))
    style.text = (style.text or "") + "  *{animation:none !important}\n    "
    frame = next(c for c in children if c.get("clip-path") == "url(#frame)")
    # Everything after the clipped group, bar the frame's outline, is the copy column.
    copy_els = [
        c
        for c in children[children.index(frame) + 1 :]
        if not (c.tag == tag("rect") and c.get("fill") == "none")
    ]
    background = list(frame)
    nebulae = [c for c in background[1:3] if c.tag == tag("ellipse")]
    scene_els = background[5:]  # after the base, the two nebulae and the two star groups

    wide = width >= 1280
    s_copy, s_scene = (1.2, 1.04) if wide else (1.12, 0.98)
    margin = 60 if wide else 56
    card = ET.Element(
        tag("svg"),
        {
            "viewBox": f"0 0 {width} {height}",
            "width": str(width),
            "height": str(height),
            "role": "img",
            "aria-labelledby": "title desc",
        },
    )
    card.extend([root.find(tag("title")), root.find(tag("desc")), defs])
    ET.SubElement(
        card, tag("rect"), {"width": str(width), "height": str(height), "fill": "url(#spaceBg)"}
    )
    glow = ET.SubElement(
        card, tag("g"), {"transform": f"scale({width / 1200:.4f} {height / 400:.4f})"}
    )
    glow.extend(nebulae)
    rng = random.Random(STAR_SEED)
    stars = ET.SubElement(card, tag("g"), {"fill": "#CFE6FF"})
    placed = 0
    while placed < 150:
        x, y = rng.uniform(6, width - 6), rng.uniform(6, height - 6)
        if 40 <= x <= 680 and 110 <= y <= 530:  # keep the copy column clear
            continue
        placed += 1
        radius = rng.choice([0.6, 0.7, 0.8, 0.9, 1.0, 1.2])
        opacity = f"{rng.uniform(0.2, 0.65):.2f}"
        ET.SubElement(
            stars, tag("circle"), {"cx": num(x), "cy": num(y), "r": num(radius), "opacity": opacity}
        )
    scene_x = width - 12 - 1200 * s_scene
    scene_y = (height - 400 * s_scene) / 2 + 6
    scene = ET.SubElement(
        card, tag("g"), {"transform": f"translate({num(scene_x)} {num(scene_y)}) scale({s_scene})"}
    )
    scene.extend(scene_els)
    copy_x = margin - 46 * s_copy
    copy_y = (height - 300 * s_copy) / 2 - 30 * s_copy - 8
    column = ET.SubElement(
        card, tag("g"), {"transform": f"translate({num(copy_x)} {num(copy_y)}) scale({s_copy})"}
    )
    column.extend(copy_els)
    footer = ET.SubElement(
        card,
        tag("text"),
        {
            "class": "mono",
            "x": str(margin),
            "y": str(height - 44),
            "font-size": "15",
            "letter-spacing": "1.5",
            "fill": "#808D9E",
        },
    )
    footer.text = f"github.com/netspeedy/{REPO}"
    return ET.tostring(card, encoding="unicode") + "\n"


def render_png(
    svg_path: Path, png_path: Path, width: int, height: int, transparent: bool = False
) -> None:
    """Screenshot an SVG in headless Chrome, which renders the embedded fonts exactly."""
    chrome = os.environ.get("CHROME") or next(
        (c for c in CHROME_CANDIDATES if Path(c).exists() or shutil.which(c)), None
    )
    if not chrome:
        raise SystemExit("the social cards and icons need Chrome or Chromium to render; set CHROME")
    background = ["--default-background-color=00000000"] if transparent else []
    with tempfile.TemporaryDirectory() as profile:
        subprocess.run(
            [
                chrome,
                "--headless=new",
                "--disable-gpu",
                "--hide-scrollbars",
                *background,
                f"--user-data-dir={profile}",
                f"--window-size={width},{height}",
                f"--screenshot={png_path}",
                svg_path.resolve().as_uri(),
            ],
            check=True,
            capture_output=True,
        )


def write_social_cards(hero: str) -> None:
    cards = [(PREVIEW_CARD, 1280, 640, False)]
    if (ROOT / "website").is_dir():
        cards.insert(0, (OG_CARD, 1200, 630, True))
    for base, width, height, keep_svg in cards:
        svg = build_social_svg(hero, width, height)
        svg = svg.replace("__FONT_FACES__", font_faces(svg))
        ET.fromstring(svg)
        base.parent.mkdir(parents=True, exist_ok=True)
        png = base.with_suffix(".png")
        with tempfile.TemporaryDirectory() as tmp:
            svg_path = base.with_suffix(".svg") if keep_svg else Path(tmp) / "card.svg"
            svg_path.write_text(svg, encoding="utf-8")
            render_png(svg_path, png, width, height)
        print(f"Wrote {png.relative_to(ROOT)} ({width}x{height})")


# Icons

# Written into the website's public folder, so they're only made when the repo has a website.
# The favicon is the 16px cut, which browsers scale up; the Apple touch icon fills its square
# because iOS rounds the corners itself; favicon.ico carries 16, 32 and 48 px PNGs for clients
# that ignore the SVG.
ICON_DIR = ROOT / "website" / "public"
SITE_MARK = ROOT / "website" / "src" / "assets" / "mark.svg"


def icon_svg(mark: Mark, size: int | None = None, square: bool = False) -> str:
    dims = f' width="{size}" height="{size}"' if size else ""
    if square:
        scale = 0.64
        offset = 32 - 32 * scale
        art = (
            f'<rect width="64" height="64" fill="{TILE}" />'
            f'<g transform="translate({n2(offset)} {n2(offset)}) scale({n2(scale)})">'
            f'{mark.glyph(CYAN, "cut")}</g>'
        )
    else:
        art = tile(mark, "cut")
    return (
        f'<svg xmlns="{SVG_NS}" viewBox="0 0 64 64"{dims}><title>s3ctl</title>'
        f'<defs>{mark.mask("cut")}</defs>{art}</svg>\n'
    )


def ico(images: list[tuple[int, bytes]]) -> bytes:
    """An ICO file holding PNG images, which every current client reads."""
    header = struct.pack("<HHH", 0, 1, len(images))
    entries, data = b"", b""
    offset = 6 + 16 * len(images)
    for size, png in images:
        entries += struct.pack("<BBBBHHII", size % 256, size % 256, 0, 0, 1, 32, len(png), offset)
        data += png
        offset += len(png)
    return header + entries + data


def write_icons() -> None:
    if not (ROOT / "website").is_dir():
        return
    # The site's header mark, in currentColor so the page's theme can colour it.
    SITE_MARK.parent.mkdir(parents=True, exist_ok=True)
    SITE_MARK.write_text(
        f'<svg xmlns="{SVG_NS}" viewBox="0 0 64 64" aria-hidden="true" focusable="false">'
        f'<defs>{MARK.mask("s3-mark-cut")}</defs>{MARK.glyph("currentColor", "s3-mark-cut")}</svg>\n',
        encoding="utf-8",
    )
    print(f"Wrote {SITE_MARK.relative_to(ROOT)}")
    ICON_DIR.mkdir(parents=True, exist_ok=True)
    favicon = ICON_DIR / "favicon.svg"
    favicon.write_text(icon_svg(SMALL), encoding="utf-8")
    print(f"Wrote {favicon.relative_to(ROOT)}")
    with tempfile.TemporaryDirectory() as tmp:
        work = Path(tmp)

        def png(mark: Mark, size: int, square: bool = False) -> bytes:
            src, out = work / f"icon-{size}.svg", work / f"icon-{size}.png"
            src.write_text(icon_svg(mark, size, square), encoding="utf-8")
            render_png(src, out, size, size, transparent=not square)
            return out.read_bytes()

        touch = ICON_DIR / "apple-touch-icon.png"
        touch.write_bytes(png(MARK, 180, square=True))
        print(f"Wrote {touch.relative_to(ROOT)} (180x180)")
        fallback = ICON_DIR / "favicon.ico"
        fallback.write_bytes(ico([(16, png(SMALL, 16)), (32, png(MARK, 32)), (48, png(MARK, 48))]))
        print(f"Wrote {fallback.relative_to(ROOT)} (16, 32 and 48 px)")


def main() -> None:
    hero = build_svg()
    svg = hero.replace("__FONT_FACES__", font_faces(hero))
    if "http://" in svg.replace("http://www.w3.org/2000/svg", "") or "https://" in svg:
        raise SystemExit("external reference found in the hero SVG")
    ET.fromstring(svg)  # well-formed, or this raises
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text(svg, encoding="utf-8")
    print(f"Wrote {OUTPUT.relative_to(ROOT)} ({len(svg.encode()) / 1024:.1f} KiB)")
    write_social_cards(hero)
    write_icons()


if __name__ == "__main__":
    main()
