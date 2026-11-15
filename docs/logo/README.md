# Logos

Ein Ordner pro Marke, jeweils mit denselben Vorlagen (EPS als Quelle, SVG als Vorschau):

| Datei | Inhalt |
|---|---|
| `color1_icon_transparent_background` | Icon, 56 × 56 |
| `color1_logo_transparent_background` | Icon, Trennlinie, Name, Unterzeile, Höhe 56 |
| `base_logo_transparent_background` | dasselbe, Höhe 48 |

- `midnightsports-aadorf/`: Logo von Midnightsports Aadorf. Gehört Aadorf, nur für Aadorf verwenden
  (als hochgeladenes Projektlogo).
- `conventus/`: Arbeitsname der mandantenfähigen App, Name noch nicht fix. Erzeugt mit `build/logo/brand.py`.

## Neues Logo für einen anderen Namen

```sh
make brand NAME=conventus TAGLINE="events · teams · check-in" COLOR=#3346d3
```

- `NAME`: Name der App, wird in Versalien gesetzt. Ordnername ist der Name in Kleinbuchstaben
- `TAGLINE`: Unterzeile, kurz halten (sonst wird das Logo breiter als der Name)
- `COLOR`: eine Farbe für alles, Standard `--primary` aus `web/src/styles.css`

Erlaubte Zeichen: A–Z, Leerzeichen, `-`, `.`, `·`. Umlaute oder Ziffern fehlen, `brand.py` meldet das
fehlende Zeichen. Neue Zeichen ergänzen in `GLYPHS` (siehe unten).

Danach die SVG-Dateien im neuen Ordner ansehen. Soll die App das neue Logo verwenden:

```sh
make logos LOGO=conventus
```

Das überschreibt Favicon, `logo.svg`, `logo-white.svg`, App-Icons und Wallet-Bilder (siehe `CLAUDE.md`).
Ohne `LOGO` erzeugt `make logos` weiterhin die Aadorf-Logos.

## Gestaltung

Gleicher Aufbau wie das Aadorf-Logo, aber eigene Formen:

- **Icon**: abgerundetes Quadrat (Radius 12 von 56), ausgespart ein Kreisbogen (C, offen nach rechts) um einen
  Punkt: Menschen, die sich um etwas versammeln (lat. *conventus* = Versammlung). Der Hintergrund scheint durch
  (`evenodd`). Das Icon hängt nicht vom Namen ab, bei einem neuen Namen `icon()` anpassen, z.B. den Buchstaben
- **Trennlinie**: dünn, volle Höhe, rechts vom Icon
- **Name**: eigene geometrische Schablonenschrift, Versalhöhe 22 von 56, weit gesperrt
- **Unterzeile**: dieselbe Schrift, Versalhöhe 7, noch weiter gesperrt
- **Farbe**: eine einzige; die App erzeugt daraus die weisse Variante für Kopfzeile und Dark Mode

Die Schrift besteht aus Rechtecken, Schrägbalken (`diag`) und Kreisbögen (`ring`) bei Versalhöhe 20 und
Strichstärke 3. An Verbindungen bleibt eine Lücke von 1.8 (Schablone), z.B. zwischen Stamm und Querbalken
beim E. Ein neues Zeichen ist ein Eintrag in `GLYPHS` mit diesen Bausteinen; die Breite ergibt sich aus der
Form.
