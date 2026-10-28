# Anforderungen und Stand

Anforderungen des Auftraggebers und ihr Stand. Grundlage: `docs/Anpassungen.docx` (nicht im Repository) und
Absprachen. Bei neuen Anforderungen nachführen.

Stand: **ja** umgesetzt, **teilweise**, **offen**, **nein** bewusst nicht umgesetzt.

## Profil und Anmeldung

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Geschlecht und Beziehung zum Notfallkontakt als Auswahl | ja | W/M/D, Vater/Mutter/Andere |
| Status als Auswahl | ja | Pro Projekt beim Anmelden: Schüler-/in aus der Region, Gast extern. Mitarbeiter vergeben Projekt-Admins |
| PLZ und Ort als Auswahl | nein | Freie Eingabe |
| Feld für die Klasse | nein | Datenminimierung |
| Strasse und Nummer entfernen | teilweise | Freiwillig |
| Login mit Apple | ja | Braucht Apple-Developer-Konto (ca. CHF 100/Jahr) |

## Rollen, Team, Auszahlung

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Rollen Admin, Projektleiter, Mitarbeiter, Teilnehmer | ja | Pro Projekt |
| Events erfassen nur durch Projektleiter und Admins | ja | Mitarbeiter checken ein und erfassen Gäste |
| Funktionen mit Lohn pro Einsatz (JC, JSC, SC, PL über Zeiterfassung) | ja | Pro Projekt, Administration |
| Team am Event, nur PL | ja | Verantwortlicher PL automatisch dabei |
| Auszahlung mit Unterschrift, druckbar | ja | Spesenblatt nach Papiervorlage |
| Spalten „Hintergrund Stunden“, „Reise Tageskurs“ | offen | Keine Daten in der App |

## Events und Zeiterfassung

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Bericht zum Event, druckbar | ja | Datum, Ort, PL, Teilnehmer W/M/D, Team, Text |
| Admin bestätigt verrechnete Zeiteinträge | offen | |

## Personen und Notfallkontakte

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Notfallkontakt nur für Berechtigte | ja | Projekt-Admins legen Rollen und Funktionen fest, jeder Abruf protokolliert |
| Gast als Person mit gleichen Angaben, Email freiwillig | ja | Übernimmt beim Login mit derselben Email das Konto |
| Personen bearbeiten und löschen | ja | Nur globale Admins, protokolliert. Check-ins bleiben anonymisiert |

## Mandantenfähigkeit (Branch `feat/multitenancy`)

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Startseite: Login, dann Projekt aus eigenen Projekten wählen | ja | Auch in der Kopfzeile |
| „Nach Projekten suchen“ mit Volltextsuche | ja | Name, Region, Beschreibung |
| Midnightsports Aadorf als erstes Projekt | ja | Im Schema angelegt |
| Globale Administration: Projekte erfassen | ja | Name, Logo, Projekt-Admin, Funktionen, erster Ort |
| Logo hochladen, sonst Standardlogo | ja | PNG, JPEG, WebP bis 512 KB, kein SVG |
| Seite Benutzer heisst Teilnehmer | ja | Nur Mitglieder des Projekts |
| Personen global, in der globalen Administration unter Benutzer | ja | |
| Projekt-Link per QR-Code führt zum Projekt | ja | `/?projekt=<kurzname>`, auch über den Login |
| Funktionsbezeichnungen beim Erstellen festlegen | ja | |
| Projekt-Admin sieht nur Personen seines Projekts | ja | |
| Anmeldung beim Projekt nur über Startseite oder Projekt-Link | ja | Nichtmitglieder werden nicht eingecheckt |
| Änderungen an Personen nur durch globale Admins, im Audit | ja | |
| Gesetzlich relevante Vorgänge im Audit | teilweise | Umgesetzt, rechtliche Prüfung und Aufbewahrungsdauer offen |
| Verwarnen und sperren, auch am Event | ja | Meldung beim Check-in, gesperrt: kein Check-in |
| Fristen für Sperren und Verwarnungen | ja | Pro Projekt |
| Verwarnungen nur im auslösenden Projekt sichtbar | ja | |
| Teilen zwischen zwei Projekten, blockiert bis zur Einwilligung | ja | |
| Datenmigration der Produktion | offen | Nach dem Merge |

## Plattform

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Hosting intern (z.B. NAS) | offen | Möglich, nicht empfohlen. Aktuell Cloud Run |
