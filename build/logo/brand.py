"""Erzeugt die Logo-Vorlagen einer Marke (Icon, Logo, Basislogo) als EPS und SVG.

Aufruf: python3 build/logo/brand.py conventus "events · teams · check-in" "#3346d3"
Ergebnis: docs/logo/<name>/ mit denselben Dateien wie docs/logo/midnightsports-aadorf, danach
`make logos LOGO=<name>` für die App. Gestaltungsregeln: docs/logo/README.md.

Die Schrift ist eine eigene geometrische Schablonenschrift aus Rechtecken, Schrägbalken und Kreisbögen
(Strichstärke 3 bei Versalhöhe 20, Lücke 1.8 an Verbindungen). So braucht es keine Schriftdatei, und ein
neuer Name ist ein neuer Aufruf. Die EPS-Dateien haben das Format, das generate.py liest (nur Füllungen).
"""

import math
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
H = 20.0  # Versalhöhe
T = 3.0  # Strichstärke
G = 1.8  # Schablonenlücke


def rect(x, y, w, h):
    return [("M", x, y), ("L", x + w, y), ("L", x + w, y + h), ("L", x, y + h), ("Z",)]


def poly(*pts):
    return [("M",) + pts[0]] + [("L",) + p for p in pts[1:]] + [("Z",)]


def diag(xt, xb, y0=0.0, y1=H, w=3.4):
    """Schrägbalken mit waagrechten Enden: oben ab xt, unten ab xb, horizontal w breit."""
    return poly((xt, y0), (xt + w, y0), (xb + w, y1), (xb, y1))


def arc(cx, cy, r, a0, a1):
    """Kreisbogen als Bézierkurven, Winkel in Grad (0 = rechts, 90 = unten wie in SVG)."""
    n = max(1, math.ceil(abs(a1 - a0) / 90))
    step = math.radians(a1 - a0) / n
    k = 4 / 3 * math.tan(step / 4)
    out = []
    for i in range(n):
        s = math.radians(a0) + i * step
        e = s + step
        p0 = (cx + r * math.cos(s), cy + r * math.sin(s))
        p3 = (cx + r * math.cos(e), cy + r * math.sin(e))
        p1 = (p0[0] - k * r * math.sin(s), p0[1] + k * r * math.cos(s))
        p2 = (p3[0] + k * r * math.sin(e), p3[1] - k * r * math.cos(e))
        out.append(("C",) + p1 + p2 + p3)
    return out


def ring(cx, cy, ro, ri, a0, a1):
    """Kreisbogen mit Strichstärke ro - ri, gerade abgeschnittene Enden."""
    a, b = math.radians(a0), math.radians(a1)
    return ([("M", cx + ro * math.cos(a), cy + ro * math.sin(a))] + arc(cx, cy, ro, a0, a1)
            + [("L", cx + ri * math.cos(b), cy + ri * math.sin(b))] + arc(cx, cy, ri, a1, a0) + [("Z",)])


def circle(cx, cy, r):
    return [("M", cx + r, cy)] + arc(cx, cy, r, 0, 360) + [("Z",)]


def rounded(x, y, w, h, r):
    """Rechteck mit runden Ecken (Icon-Grundform)."""
    return ([("M", x + r, y), ("L", x + w - r, y)] + arc(x + w - r, y + r, r, 270, 360)
            + [("L", x + w, y + h - r)] + arc(x + w - r, y + h - r, r, 0, 90)
            + [("L", x + r, y + h)] + arc(x + r, y + h - r, r, 90, 180)
            + [("L", x, y + r)] + arc(x + r, y + r, r, 180, 270) + [("Z",)])


def stem(x=0.0, y=0.0, h=H):
    return rect(x, y, T, h)


S = T + G  # Beginn eines Elements neben einem Stamm
TOP, BOT = (9.0, 5.75), (9.0, 14.25)  # Bäuche von B, P, R, S
GLYPHS = {
    "A": [diag(4.8, 0), diag(9.8, 14.6), rect(5, 12, 8, T)],
    "B": [stem(), rect(S, 0, 4.2, T), rect(S, 8.5, 4.2, T), rect(S, 17, 4.2, T),
          ring(*TOP, 5.75, 2.75, 270, 450), ring(*BOT, 5.75, 2.75, 270, 450)],
    "C": [ring(10, 10, 10, 7, 45, 315)],
    "D": [stem(), rect(S, 0, 3.2, T), rect(S, 17, 3.2, T), ring(8, 10, 10, 7, 270, 450)],
    "E": [stem(), rect(S, 0, 9.2, T), rect(S, 8.5, 7.5, T), rect(S, 17, 9.2, T)],
    "F": [stem(), rect(S, 0, 9.2, T), rect(S, 8.5, 7.5, T)],
    "G": [ring(10, 10, 10, 7, 0, 315), rect(11, 10, 9, T)],
    "H": [stem(), stem(13), rect(S, 8.5, 13 - S - G, T)],
    "I": [stem()],
    "J": [stem(10, 0, 11.7), ring(6.5, 13.5, 6.5, 3.5, 0, 180)],
    "K": [stem(), diag(11, S, 0, 10), diag(S, 11, 10, H)],
    "L": [stem(), rect(S, 17, 8.2, T)],
    "M": [stem(), stem(19), diag(S, 9.3, 0, 14), diag(13.8, 9.3, 0, 14)],
    "N": [stem(), stem(13), diag(4.6, 8.0)],
    "O": [ring(10, 10, 10, 7, -80, 80), ring(10, 10, 10, 7, 100, 260)],
    "P": [stem(), rect(S, 0, 4.2, T), rect(S, 8.5, 4.2, T), ring(*TOP, 5.75, 2.75, 270, 450)],
    "Q": [ring(10, 10, 10, 7, -80, 80), ring(10, 10, 10, 7, 100, 260), diag(10.5, 14.6, 13, H)],
    "R": [stem(), rect(S, 0, 4.2, T), rect(S, 8.5, 4.2, T), ring(*TOP, 5.75, 2.75, 270, 450),
          diag(8.6, 11.6, 13.3, H)],
    # Zwei Bögen, die sich in der Mitte mit geradem Schnitt treffen
    "S": [ring(8, 5.75, 5.75, 2.75, 90, 330), ring(8, 14.25, 5.75, 2.75, -90, 150)],
    "T": [rect(0, 0, 16, T), stem(6.5, S, H - S)],
    "U": [stem(0, 0, 11 - G), stem(15, 0, 11 - G), ring(9, 11, 9, 6, 0, 180)],
    "V": [diag(0, 4.8), diag(14.6, 9.8)],
    "W": [diag(0, 3.8), diag(8.0, 3.8, 6, H), diag(8.0, 12.2, 6, H), diag(16.0, 12.2)],
    "X": [diag(0, 12.6), diag(12.6, 0)],
    "Y": [diag(0, 6.3, 0, 10), diag(12.6, 6.3, 0, 10), stem(6.5, 10 + G, H - 10 - G)],
    "Z": [rect(0, 0, 15, T), rect(0, 17, 15, T), diag(11.6, 0, S, H - S)],
    "-": [rect(0, 8.5, 6, T)],
    "·": [rect(0, 8.5, T, T)],
    ".": [rect(0, 17, T, T)],
}
SPACE = 8.0


def points(path):
    for seg in path:
        for i in range(1, len(seg), 2):
            yield seg[i], seg[i + 1]


def transform(path, dx, dy, s):
    return [(seg[0],) + tuple(v * s + (dx if i % 2 == 0 else dy) for i, v in enumerate(seg[1:])) for seg in path]


def text(s, x, y, size, tracking):
    """Pfade eines Texts, oben links bei (x, y), Versalhöhe size. Liefert (pfade, breite)."""
    scale, out, cx = size / H, [], 0.0
    for ch in s.upper():
        if ch == " ":
            cx += SPACE
            continue
        if ch not in GLYPHS:
            raise SystemExit("Zeichen %r fehlt in GLYPHS (build/logo/brand.py)" % ch)
        shapes = GLYPHS[ch]
        width = max(px for shape in shapes for px, _ in points(shape))
        out += [transform(shape, x + cx * scale, y, scale) for shape in shapes]
        cx += width + tracking
    return out, (cx - tracking) * scale


def icon(x, y, size):
    """Abgerundetes Quadrat mit ausgesparter Versammlung: Kreisbogen (C) um einen Punkt.
    Ein einziger Pfad mit evenodd, damit der Hintergrund durchscheint wie beim bisherigen Icon."""
    u = size / 56
    c = (x + 28 * u, y + 28 * u)
    return (rounded(x, y, size, size, 12 * u) + ring(*c, 18 * u, 11 * u, 40, 320)
            + circle(*c, 5 * u))


def layout(name, tagline, height):
    """Icon links, Trennlinie, Name in Versalien, darunter die Unterzeile. Wie das Aadorf-Logo:
    Name gross und weit gesperrt, Unterzeile klein und noch weiter gesperrt."""
    shapes = [(icon(0, 0, height), True)]
    u = height / 56
    shapes.append((rect(62 * u, 0, 1.2 * u, height), False))
    words, w1 = text(name, 74 * u, 4 * u, 22 * u, 9)
    tag, w2 = text(tagline, 75 * u, 40 * u, 7 * u, 10)
    shapes += [(p, False) for p in words + tag]
    return 74 * u + max(w1, w2) + 2 * u, shapes


def fmt(v):
    return ("%.3f" % v).rstrip("0").rstrip(".")


def svg_d(path):
    return "".join(seg[0] + " ".join(fmt(v) for v in seg[1:]) for seg in path)


def write_svg(path, w, h, shapes, color):
    body = "\n".join('<path fill="%s"%s d="%s"/>' % (color, ' fill-rule="evenodd"' if eo else "", svg_d(p))
                     for p, eo in shapes)
    path.write_text('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %s %s">\n%s\n</svg>\n' % (
        fmt(w), fmt(h), body))


def write_eps(path, w, h, shapes, color):
    """EPS wie von Cairo: y nach unten per cm, nur m l c h f f*."""
    r, g, b = (int(color[i:i + 2], 16) / 255 for i in (1, 3, 5))
    ops = {"M": "m", "L": "l", "C": "c", "Z": "h"}
    lines = []
    for p, eo in shapes:
        lines.append(" ".join(" ".join(fmt(v) for v in seg[1:]) + (" " if len(seg) > 1 else "") + ops[seg[0]]
                              for seg in p) + (" f*" if eo else " f"))
    bw, bh = math.ceil(w), math.ceil(h)
    path.write_text("\n".join([
        "%!PS-Adobe-3.0 EPSF-3.0", "%%Creator: build/logo/brand.py", "%%Pages: 1", "%%LanguageLevel: 2",
        "%%%%BoundingBox: 0 0 %d %d" % (bw, bh), "%%EndComments", "%%BeginProlog", "50 dict begin",
        "/q { gsave } bind def", "/Q { grestore } bind def", "/cm { 6 array astore concat } bind def",
        "/m { moveto } bind def", "/l { lineto } bind def", "/c { curveto } bind def",
        "/h { closepath } bind def", "/f { fill } bind def", "/f* { eofill } bind def",
        "/rg { setrgbcolor } bind def",
        "/rectclip { exch dup neg 3 1 roll 5 3 roll moveto 0 rlineto 0 exch rlineto 0 rlineto closepath clip newpath } bind def",
        "%%EndProlog", "%%Page: 1 1", "%%EndPageSetup",
        "q 0 0 %d %d rectclip" % (bw, bh), "1 0 0 -1 0 %d cm q" % bh, "%s %s %s rg" % (fmt(r), fmt(g), fmt(b)),
        *lines, "Q Q", "showpage", "%%Trailer", "end", "%%EOF", ""]))


def main():
    if len(sys.argv) != 4:
        raise SystemExit(__doc__)
    name, tagline, color = sys.argv[1:]
    out = ROOT / "docs/logo" / name.lower()
    out.mkdir(parents=True, exist_ok=True)
    w, logo = layout(name, tagline, 56)
    bw, base = layout(name, tagline, 48)
    files = {
        "color1_icon": (56, 56, [(icon(0, 0, 56), True)]),
        "color1_logo": (w, 56, logo),
        "base_logo": (bw, 48, base),
    }
    for stem_name, (fw, fh, shapes) in files.items():
        for ext, write in (("svg", write_svg), ("eps", write_eps)):
            write(out / ("%s_transparent_background.%s" % (stem_name, ext)), fw, fh, shapes, color)


if __name__ == "__main__":
    main()
