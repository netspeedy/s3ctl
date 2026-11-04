#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = ["fonttools==4.67.0", "brotli==1.2.0"]
# ///
"""Build the animated README hero, assets/s3ctl-readme-hero.svg.

The banner follows the soakes/cv README hero: the bucket mark, a two-line headline, a short body
and a meta row on the left, and a generated scene on the right. The scene is the bucket from
assets/logo.svg drawn large in dots, with a scan running down it and the steps s3ctl takes
(bucket, versioning, bucket policy, scoped key) lighting up beside it in turn.

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

# The bucket mark's cyan and amber (assets/logo.svg), with oakes.me.uk's inks so the netspeedy
# banners read as one family.
BG = "#060B13"
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


def bucket_symbols() -> list[str]:
    """The bucket from assets/logo.svg: the full tile, and the outline on its own."""
    outline = (
        '<ellipse cx="32" cy="16" rx="14" ry="4.2" />'
        '<path d="M18 16 L21.5 47 L42.5 47 L46 16" />'
        '<path d="M19 25 L45 25" stroke-width="2.2" stroke-opacity="0.85" />'
        '<path d="M20 34 L44 34" stroke-width="2.2" stroke-opacity="0.85" />'
    )
    keyhole = (
        f'<g fill="{AMBER}"><circle cx="32" cy="39" r="2.8" />'
        '<path d="M30.7 41 L33.3 41 L34.2 46.6 L29.8 46.6 Z" /></g>'
    )
    return [
        *gradient(
            "linearGradient",
            'id="markStroke" x1="16" y1="14" x2="48" y2="50" gradientUnits="userSpaceOnUse"',
            [("0", CYAN, ""), ("1", CYAN_3, "")],
        ),
        '    <symbol id="bucket" viewBox="0 0 64 64">',
        '      <g fill="none" stroke="url(#markStroke)" stroke-width="3.4" stroke-linecap="round" '
        f'stroke-linejoin="round">{outline}</g>{keyhole}',
        "    </symbol>",
    ]


def lockup() -> list[str]:
    return [
        '  <rect x="48.5" y="38.5" width="33" height="33" rx="8" fill="#0E1726" stroke="#3B4A5F" '
        'stroke-width="1.2" />',
        '  <use href="#bucket" x="49" y="39" width="32" height="32" />',
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


# Scene: monolith


MONO_CX, MONO_TOP = 944.0, 62.0  # centre of the rim
MONO_RX0, MONO_RX1 = 172.0, 126.0  # radius at the rim and at the base, so the bucket tapers
MONO_DEPTH = 300.0  # rim to base
MONO_RY = 0.22  # each ring's height as a share of its width: the view from slightly above


def radius(t: float) -> float:
    return MONO_RX0 + (MONO_RX1 - MONO_RX0) * t / MONO_DEPTH


def surface(t: float, theta: float) -> tuple[float, float]:
    r = radius(t)
    return MONO_CX + r * math.sin(theta), MONO_TOP + t + r * MONO_RY * math.cos(theta)


def front_arc(t: float, steps: int = 48) -> str:
    points = [surface(t, -math.pi / 2 + math.pi * q / steps) for q in range(steps + 1)]
    return "M" + "L".join(f"{num(x)} {num(y)}" for x, y in points)


def scene_monolith() -> tuple[list[str], str, str, list[tuple[float, float, float, float]]]:
    """The bucket mark as a dotted object, scanned top to bottom, with s3ctl's steps beside it."""
    loop = 6.0
    cx, top = MONO_CX, MONO_TOP
    bottom = top + MONO_DEPTH
    rx0, rx1 = MONO_RX0, MONO_RX1

    points = []
    t = 9.0
    ring = 0
    while t <= MONO_DEPTH:
        r = radius(t)
        count = int(2 * math.pi * r / 9.5)
        for q in range(count):
            theta = 2 * math.pi * (q + (ring & 1) * 0.5) / count
            c = math.cos(theta)
            if c <= 0.03:
                continue
            x, y = surface(t, theta)
            band = 0 if c < 0.3 else 1 if c < 0.55 else 2 if c < 0.8 else 3
            points.append((x, y, band))
        t += 9.0
        ring += 1
    field = dot_paths(points, {0: (0.22, 1.2), 1: (0.38, 1.4), 2: (0.55, 1.55), 3: (0.72, 1.65)})

    # The back half of the rim, down the right side, round the front of the base and back up.
    rim = f"A{num(rx0)} {num(rx0 * MONO_RY)} 0 0 1 {num(cx + rx0)} {num(top)}"
    base = f"A{num(rx1)} {num(rx1 * MONO_RY)} 0 0 1 {num(cx - rx1)} {num(bottom)}"
    silhouette = f"M{num(cx - rx0)} {num(top)}{rim}L{num(cx + rx1)} {num(bottom)}{base}Z"
    # The two bands across the logo's bucket.
    bands = "".join(
        f'<path d="{front_arc(MONO_DEPTH * f)}" fill="none" stroke="url(#bandGrad)" '
        'stroke-width="2.4" stroke-linecap="round" />'
        for f in (0.29, 0.58)
    )
    scans = []
    step = 12.0
    count = int(MONO_DEPTH / step)
    sweep = 2.6
    for k in range(count + 1):
        delay = k * sweep / count
        scans.append(
            f'<path class="sc" d="{front_arc(k * step)}" fill="none" stroke="{CYAN_2}" '
            f'stroke-width="1.3" style="animation-delay:{delay:.2f}s" />'
        )

    # Keyhole on the front, where the logo puts it.
    kt = MONO_DEPTH * 0.74
    kx, ky = surface(kt, 0)
    keyhole = (
        f'<circle class="kh" cx="{num(kx)}" cy="{num(ky - 6)}" r="34" fill="url(#keyholeGlow)" />'
        f'<circle cx="{num(kx)}" cy="{num(ky - 12)}" r="10" fill="{AMBER}" />'
        f'<path d="M{num(kx - 5)} {num(ky - 5)}h10l3.5 22h-17z" fill="{AMBER}" />'
    )

    steps = [
        ("BUCKET", 30.0, CYAN_2),
        ("VERSIONING", 100.0, CYAN_2),
        ("BUCKET POLICY", 170.0, CYAN_2),
        ("SCOPED KEY", 240.0, AMBER_2),
    ]
    labels = []
    for k, (label, at, colour) in enumerate(steps):
        ax, ay = surface(at, -1.05)
        width = mono_width(len(label), 10.5, 1.2) - 1.2 + 16
        bx = 600.0
        by = ay - 10
        labels.append(
            f'<g class="st" style="animation-delay:{k * loop / len(steps) - loop:.2f}s">'
            f'<path d="M{num(bx + width)} {num(ay)}H{num(ax - 4)}" fill="none" stroke="{colour}" '
            'stroke-opacity="0.5" stroke-width="1" />'
            f'<circle cx="{num(ax)}" cy="{num(ay)}" r="3" fill="{colour}" />'
            f'<circle class="ping" cx="{num(ax)}" cy="{num(ay)}" r="3" fill="none" '
            f'stroke="{colour}" stroke-width="1.1" />' + chip(bx, by, label, colour) + "</g>"
        )

    defs = [
        *gradient(
            "linearGradient",
            'id="monoBody" x1="0" y1="0" x2="1" y2="0"',
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
            [("0", CYAN, ""), ("1", CYAN_3, "")],
        ),
        *gradient(
            "linearGradient",
            'id="bandGrad" x1="0" y1="0" x2="1" y2="0"',
            [("0", CYAN_3, "0.15"), ("0.5", CYAN, "0.95"), ("1", CYAN_3, "0.15")],
        ),
        *gradient(
            "radialGradient",
            'id="keyholeGlow" cx="0.5" cy="0.5" r="0.5"',
            [("0", AMBER, "0.7"), ("0.35", AMBER, "0.22"), ("1", AMBER, "0")],
        ),
        *gradient(
            "radialGradient",
            'id="aura" cx="0.5" cy="0.45" r="0.5"',
            [("0", CYAN_3, "0.32"), ("0.6", CYAN_3, "0.1"), ("1", CYAN_3, "0")],
        ),
        f'    <clipPath id="monoClip"><path d="{silhouette}" /></clipPath>',
    ]
    css = (
        f".sc{{opacity:0;animation:sc {loop}s linear infinite}}"
        f"@keyframes sc{{0%{{opacity:0}}1.5%{{opacity:.85}}9%{{opacity:0}}100%{{opacity:0}}}}"
        ".kh{animation:kh 3s ease-in-out infinite}"
        "@keyframes kh{0%,100%{opacity:.6}50%{opacity:1}}"
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
            f'<path d="{silhouette}" fill="url(#monoBody)" />',
            f'<path d="{silhouette}" fill="none" stroke="{CYAN}" stroke-opacity="0.35" '
            'stroke-width="1.2" />',
            f'<g clip-path="url(#monoClip)">{field}{bands}{"".join(scans)}</g>',
            f'<ellipse cx="{num(cx)}" cy="{num(top)}" rx="{num(rx0)}" ry="{num(rx0 * MONO_RY)}" '
            'fill="url(#mouth)" stroke="url(#rimGrad)" stroke-width="2.6" />',
            keyhole,
            "".join(labels),
        ]
    )
    return defs, css, scene, [(cx - rx0 - 20, top - 50, cx + rx0 + 20, bottom + 40)]


# Copy

CARD = {
    "title": "s3ctl - no console clicking, no IAM drift",
    "desc": (
        "The s3ctl bucket mark drawn as a large dotted bucket with an amber keyhole, scanned from "
        "top to bottom, with the steps s3ctl runs: bucket, versioning, bucket policy, scoped key."
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
    "scene": scene_monolith,
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
        *bucket_symbols(),
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


def render_png(svg_path: Path, png_path: Path, width: int, height: int) -> None:
    """Screenshot the card in headless Chrome, which renders the embedded fonts exactly."""
    chrome = os.environ.get("CHROME") or next(
        (c for c in CHROME_CANDIDATES if Path(c).exists() or shutil.which(c)), None
    )
    if not chrome:
        raise SystemExit("the social cards need Chrome or Chromium to render; set CHROME")
    with tempfile.TemporaryDirectory() as profile:
        subprocess.run(
            [
                chrome,
                "--headless=new",
                "--disable-gpu",
                "--hide-scrollbars",
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


if __name__ == "__main__":
    main()
