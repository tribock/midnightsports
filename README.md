# Midnightsports

Verwaltung von Events für Midnightsports und Jugendtreffs, mit mehreren Projekten (Mandanten). Jedes Projekt
(z.B. Midnightsports Aadorf, Pleno) hat eigene Events, Orte, Funktionen und Mitglieder. Erstes Projekt:
Midnightsports Aadorf, wiederkehrende Events in den Turnhallen Guntershausen, Aadorf und Ettenhausen.

> **Nutzung nur mit Vertrag.** Der Code ist öffentlich einsehbar, aber nicht Open Source. Ausführen, Betreiben,
> Kopieren oder Weiterentwickeln nur mit schriftlichem Vertrag. Siehe [Lizenz und Nutzung](#lizenz-und-nutzung).

Rollen gelten pro Projekt: Eine Person kann in einem Projekt Teilnehmerin, in einem anderen Mitarbeiterin sein.

## Teilnehmer

- Profil erfassen (Name, Vorname, Email, Geburtsdatum, Geschlecht, Adresse, Notfallkontakt), gilt für alle Projekte
- Projekte suchen und beitreten: Startseite oder Projekt-Link (z.B. QR-Code auf einem Plakat)
- Nach dem Login Projekt wählen (Startseite oder Kopfzeile)
- Kommende Events ansehen, an- und abmelden
- Eigenen QR-Code zum Einchecken zeigen, optional in Apple oder Google Wallet
- Eigenes Konto löschen

Details:
- Beim ersten Login muss das Profil vollständig sein
- Geschlecht: Weiblich, Männlich, Divers
- Status pro Projekt, gewählt beim Beitreten: „Schüler-/in aus <Region des Projekts>“ oder „Gast extern“
- Strasse freiwillig, PLZ und Ort Pflicht
- Notfallkontakt: Name, Telefon, Beziehung (Vater, Mutter, Andere)
- Keine Altersgrenze, keine Einwilligung der Eltern in der App
- Abmelden bis zum Start des Events, keine maximale Teilnehmerzahl

## Mitarbeiter

- Teilnehmerliste pro Event ansehen
- Einchecken per QR-Scanner oder Suche, Gäste erfassen
- Einsatz (Team) ansehen, ohne Lohn
- Keine Events erfassen, keine Auszahlung, keine Berichte, keine Zeiterfassung, nicht als Projektleiter einteilbar

### Check-in

- Teilnehmer meldet sich vorher für den Event an und zeigt vor Ort seinen QR-Code
- Scan mit dem Handy (Kamera in der Web-App), danach alle Angaben zur Person, Anwesenheit bestätigt
- Fester QR-Code pro Person, gültig für alle Events, enthält einen signierten Token statt der ID
- Pro Event nur einmal eingecheckt
- Nicht für den Event angemeldete Personen können eingecheckt werden, müssen aber Mitglied des Projekts sein
- Verwarnt: eingecheckt, gelbe Meldung mit Grund. Gesperrt: nicht eingecheckt, rote Meldung
- Suche nach Mitgliedern des Projekts
- Gäste ohne Konto: gleiche Pflichtfelder wie im Profil, Email freiwillig. Werden als Person (Kennzeichen „Gast“)
  und Mitglied des Projekts gespeichert, an späteren Events über die Suche auffindbar. Meldet sich der Gast später
  mit derselben Email an, übernimmt er das Konto

## Projektleiter

Alles von Mitarbeiter, zusätzlich:

- **Events** anlegen (Datum, Zeitraum, Ort, Standard: Standardort des Projekts), bearbeiten, absagen. Einzeln,
  keine Serien, nicht löschbar. Genau ein verantwortlicher Projektleiter pro Event
- **Team**: Seite mit allen Projektleitern, Mitarbeitern und Admins, Anzahl pro Funktion, ohne Funktion und total.
  Jede Person hat eine Funktion (z.B. JC, SC), vergeben durch Projekt-Admins
- **Einsatz am Event**: Team-Mitglieder auswählen, Betrag kommt aus der Funktion (nicht editierbar). Funktionen ohne
  Lohn (z.B. PL) über die Zeiterfassung. Der Betrag wird beim Einteilen festgehalten, spätere Lohnänderungen ändern
  vergangene Events nicht
- **Auszahlung**: alle eingeteilten Personen mit Funktion und Betrag. Empfang mit handschriftlicher Unterschrift
  (Finger, Stift, Maus), versehentliche Unterschrift entfernbar. Bleibt eine Person mit derselben Funktion
  eingeteilt, bleibt die Unterschrift erhalten. **Drucken** (oder als PDF): Spesenblatt mit Name, Funktion, Einsatz,
  Unterschrift, leeren Zeilen, Total pro Funktion und Unterschrift der Projektleitung. Über die Zeiterfassung
  abgerechnete Personen separat
- **Bericht**: einer pro Event (Freitext), Teilnehmerzahl automatisch. **Drucken**: Datum, Ort, Projektleiter,
  Teilnehmer vor Ort (W/M/D), anwesendes Team mit Funktion, Bericht
- **Zeiterfassung**: Datum, Stunden, Beschreibung, optional Event. Eigene Einträge, Projekt-Admins sehen alle
- **Verwarnen und sperren** mit Grund: Seite Teilnehmer oder Teilnehmerliste am Event (Warndreieck)
- **Notfallkontakte**, wenn die Rolle Zugriff hat (Standard)

## Projekt-Admin

Alles von Projektleiter im eigenen Projekt, zusätzlich:

- **Teilnehmer**: nur Mitglieder dieses Projekts. Rolle und Status vergeben, Notfallkontakt (Telefon-Symbol),
  verwarnen und sperren (Warndreieck)
- **Administration**:
  - Projekt-Link mit QR-Code
  - Funktionen mit Lohn pro Einsatz (Aadorf: JC 20, JSC 40, SC 80, PL über Zeiterfassung). Funktionen, die an
    Events eingesetzt sind, sind nicht löschbar
  - Zugriff auf Notfallkontakte: Rollen und Funktionen wählen (Standard: Projektleiter)
  - Fristen: Sperre nach X Tagen aufheben oder in Verwarnung umwandeln, Verwarnung nach Y Tagen löschen
  - Geteilte Verwarnungen: Anfragen anderer Projekte einwilligen oder ablehnen, Teilen beenden
  - Veranstaltungsorte (Aadorf: Guntershausen, Aadorf, Ettenhausen mit Adressen; neue Projekte erhalten beim
    Onboarding einen ersten Ort)
- Zeiteinträge als CSV exportieren
- Profile ändern und Personen löschen kann nur der globale Admin

## Globaler Admin

- Festgelegt über `ADMIN_EMAIL` (komma-getrennt), in jedem Projekt Projekt-Admin
- **Alle Projekte**: Onboarding mit Name, Kurzname, Region, Logo, Projekt-Admin per Email, Funktionen mit
  Bezeichnung und Lohn (z.B. JC, JSC, SC, PL), erstem Ort. Festlegen, mit welchen Projekten Verwarnungen geteilt
  werden: das neue Projekt bleibt gesperrt, bis die Projekt-Admins der Partner eingewilligt haben
- **Benutzer**: alle Personen über alle Projekte, Profile bearbeiten (bei Gästen auch die Email), Personen löschen
- **Audit**: Protokoll datenschutzrelevanter Vorgänge (Zugriffe auf Notfallkontakte und Profile, Änderungen,
  Löschungen, Exporte, Rollen, Verwarnungen, Teilen)

## Verwarnungen und Sperren

- Verwarnt: Check-in mit Meldung. Gesperrt: kein Check-in, keine Anmeldung zu Events
- Fristen pro Projekt (siehe Projekt-Admin)
- Sichtbar nur im erfassenden Projekt und in Partnerprojekten mit Einwilligung, ändern nur im erfassenden

## Notfallkontakte

- Button „Notfallkontakt“ pro Person auf der Seite Teilnehmer, jeder Abruf im Audit-Protokoll
- Zugriff: Rollen und Funktionen gemäss Administration, Projekt-Admins und globale Admins immer, Teilnehmer nie
- Ohne Zugriff auch beim Check-in (Scan) nicht sichtbar

## Authentifizierung

- Login mit Google oder Apple ID (Apple optional, wenn konfiguriert): Google OAuth 2.0 bzw. Sign in with Apple
- Session als HttpOnly-Cookie, Sessions in der Datenbank
- Beim Beitreten zu einem Projekt ist man Teilnehmer

## Datenschutz

- Teilnehmer sehen nur ihre eigenen Daten
- Projektleiter und Projekt-Admins sehen nur Mitglieder ihres Projekts
- Datenschutzrelevante Vorgänge im Audit-Protokoll
- Bei Kontolöschung bleiben Check-ins anonymisiert für die Statistik
- Keine automatische Löschung nach Frist

## Betrieb

3-Tier: Frontend in TypeScript, Backend in Go, PostgreSQL für alle Daten.

- PostgreSQL: Open Source, lokal, in Cloud SQL und auf Kubernetes lauffähig
- Frontend: React mit Vite, PWA für Smartphones, nur Deutsch
- Backend: REST-API unter `/api`, liefert das gebaute Frontend aus
- Ein Docker-Image für die App, PostgreSQL als separater Container
- Konfiguration über Umgebungsvariablen (OAuth, Datenbank-URL, `ADMIN_EMAIL`)
- HTTPS und Domain über Reverse Proxy bzw. Cloud Run
- Unit-Tests für Backend und Frontend, CI über GitHub Actions

## Projektstruktur

- `api/`: Backend, `web/`: Frontend, `build/`: Dockerfile und Deployment, `docs/`: Anleitungen
- `Makefile`: `make dev` (Hot Reload), `make run` (Docker), `make build` (Image, Tag = Version aus Git),
  `make deploy` bzw. `make deploy-demo` (Cloud Run, Werte aus `.env`)
- Version unter `/api/version` und unten auf jeder Seite

## Dokumentation

- [Architektur](docs/architektur.md): Rollen, Datenmodell, Regeln, Abläufe
- [Anforderungen und Stand](docs/anforderungen.md)
- [Lokale Entwicklung](docs/lokale-entwicklung.md)
- [Deployment auf Cloud Run](docs/deployment-cloud-run.md), [Demo-Deployment](docs/deployment-demo.md)
- [Zugangsdaten, Tokens und Secrets](docs/zugangsdaten.md)

## Lizenz und Nutzung

Copyright (c) 2026 Louis Kern (tribock). Alle Rechte vorbehalten.

- Quellcode öffentlich einsehbar, **nicht** Open Source
- Jede Nutzung braucht einen schriftlichen Vertrag: Ausführen und Betreiben (auch als Test oder Demo), Anbieten als
  Dienst für Dritte, Kopieren, Ändern, Weiterentwickeln, Weitergeben
- Ein Vertrag gilt für die genannte Organisation, den Zweck und die Dauer
- Komponenten Dritter (`api/go.mod`, `web/package.json`) stehen unter ihren eigenen Lizenzen
- Anfragen über [github.com/tribock](https://github.com/tribock)

Verbindlich ist der Text in [LICENSE](LICENSE).
