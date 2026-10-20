"""Erzeugt alle Logos der App aus den EPS-Vorlagen in docs/logo (Aufruf: make logos).

Die EPS-Dateien stammen von Cairo und enthalten nur Pfade und Füllfarben. Sie werden deshalb
verlustfrei in SVG umgewandelt. Die PNG-Dateien (Homescreen, Wallet) erzeugt rsvg-convert.
"""

import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SRC = ROOT / "docs/logo"
PUBLIC = ROOT / "web/public"
WALLET = ROOT / "api/internal/wallet/assets"
WHITE = "#ffffff"


def mul(a, b):
    """Affine Matrizen [a b c d e f]: erst a, dann b."""
    return [a[0]*b[0] + a[1]*b[2], a[0]*b[1] + a[1]*b[3], a[2]*b[0] + a[3]*b[2],
            a[2]*b[1] + a[3]*b[3], a[4]*b[0] + a[5]*b[2] + b[4], a[4]*b[1] + a[5]*b[3] + b[5]]


def fmt(v):
    return ("%.3f" % v).rstrip("0").rstrip(".")


def parse_eps(path):
    """Liefert Breite, Höhe und die gefüllten Pfade [(farbe, d, evenodd)] in SVG-Koordinaten."""
    text = path.read_text()
    w, h = map(float, re.search(r"%%BoundingBox: 0 0 (\S+) (\S+)", text).groups())
    body = text.split("%%EndPageSetup", 1)[1].split("showpage", 1)[0]
    # PostScript zählt y von unten, SVG von oben
    ctm, stack, color, d, paths, args = [1, 0, 0, -1, 0, h], [], (0, 0, 0), [], [], []

    def p(x, y):
        return "%s %s" % (fmt(x*ctm[0] + y*ctm[2] + ctm[4]), fmt(x*ctm[1] + y*ctm[3] + ctm[5]))

    for tok in body.split():
        try:
            args.append(float(tok))
            continue
        except ValueError:
            pass
        if tok == "q":
            stack.append((ctm[:], color))
        elif tok == "Q":
            ctm, color = stack.pop()
        elif tok == "cm":
            ctm = mul(args[-6:], ctm)
        elif tok == "rg":
            color = tuple(args[-3:])
        elif tok == "g":
            color = (args[-1],) * 3
        elif tok == "m":
            d.append("M" + p(*args[-2:]))
        elif tok == "l":
            d.append("L" + p(*args[-2:]))
        elif tok == "c":
            a = args[-6:]
            d.append("C" + " ".join(p(a[i], a[i+1]) for i in (0, 2, 4)))
        elif tok == "h":
            d.append("Z")
        elif tok == "re":
            x, y, rw, rh = args[-4:]
            d.append("M%sL%sL%sL%sZ" % (p(x, y), p(x+rw, y), p(x+rw, y+rh), p(x, y+rh)))
        elif tok in ("f", "f*"):
            hexc = "#%02x%02x%02x" % tuple(round(c*255) for c in color)
            paths.append((hexc, "".join(d), tok == "f*"))
            d = []
        elif tok in ("n", "W", "W*", "rectclip"):
            # Cairo schneidet nur auf die ganze Seite zu
            d = []
        else:
            raise SystemExit("Unbekannter Operator %s in %s" % (tok, path.name))
        args = []
    return w, h, paths


def path_elements(paths, fill=None):
    return "\n".join('<path fill="%s"%s d="%s"/>' % (fill or c, ' fill-rule="evenodd"' if eo else "", d)
                     for c, d, eo in paths)


def svg(w, h, paths, fill=None):
    return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %g %g">\n%s\n</svg>\n' % (
        w, h, path_elements(paths, fill))


def tile(w, h, paths, padding=0.18):
    """Quadratisches App-Icon: Logo zentriert auf weissem Grund (iOS und Wallet brauchen einen Hintergrund)."""
    size = 100
    scale = size * (1 - 2*padding) / max(w, h)
    x, y = (size - w*scale) / 2, (size - h*scale) / 2
    return ('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d">\n<rect width="%d" height="%d" fill="%s"/>\n'
            '<g transform="translate(%s %s) scale(%s)">\n%s\n</g>\n</svg>\n') % (
        size, size, size, size, WHITE, fmt(x), fmt(y), fmt(scale), path_elements(paths))


def png(svg_text, out, width=None, height=None):
    cmd = ["rsvg-convert", "-o", str(out)]
    if width:
        cmd += ["-w", str(width)]
    if height:
        cmd += ["-h", str(height)]
    subprocess.run(cmd, input=svg_text.encode(), check=True)


def main():
    icon = parse_eps(SRC / "color1_icon_transparent_background.eps")
    logo = parse_eps(SRC / "color1_logo_transparent_background.eps")
    base = parse_eps(SRC / "base_logo_transparent_background.eps")

    # Alle Vorlagen auch als SVG archivieren
    for name, data in (("color1_icon", icon), ("color1_logo", logo), ("base_logo", base)):
        (SRC / (name + "_transparent_background.svg")).write_text(svg(*data))

    # Web: Favicon, Logo für helle Flächen und weisse Variante für Kopfzeile und Dark Mode
    (PUBLIC / "icon.svg").write_text(svg(*icon))
    (PUBLIC / "logo.svg").write_text(svg(*logo))
    (PUBLIC / "logo-white.svg").write_text(svg(*logo, fill=WHITE))
    icon_tile = tile(*icon)
    for size in (180, 192, 512):
        png(icon_tile, PUBLIC / ("icon-%d.png" % size), width=size)

    # Wallet: Icon mit Hintergrund, Logo weiss auf dem dunklen Pass (Apple: max. 160 x 50 Punkte)
    WALLET.mkdir(exist_ok=True)
    for size in (29, 58, 87, 660):
        png(icon_tile, WALLET / ("icon-%d.png" % size), width=size)
    logo_white = svg(*logo, fill=WHITE)
    png(logo_white, WALLET / "logo.png", width=160)
    png(logo_white, WALLET / "logo@2x.png", width=320)


if __name__ == "__main__":
    main()
