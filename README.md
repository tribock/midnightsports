# Midnightsports

Ein Tool zur Verwaltung von Events von Midnightsports

Genutzt wird das Tool von Projektleitern und Teilnehmern.
Die Events sind wiederkehrende Veranstaltungen in der Turnhalle Guntershausen.
Alternativ kann die Veranstaltung auch in den Turnhallen in Aadorf oder in Ettenhausen durchgeführt werden.

> **Nutzung nur mit Vertrag.** Der Code ist öffentlich einsehbar, aber nicht Open Source.
> Ausführen, Betreiben, Kopieren oder Weiterentwickeln ist nur mit einem schriftlichen Vertrag erlaubt.
> Siehe [Lizenz und Nutzung](#lizenz-und-nutzung).


## Was ein Teilnehmer kann

- eigene Daten erfassen (Name, Vorname, Email, Geburtsdatum, Geschlecht, Status, Adresse und Notfallkontakt)
- Anzeigen seines QR-Codes, um sich beim Eintreffen am Event zu registrieren
- QR-Code zu Apple Wallet oder Google Wallet hinzufügen (optional, wenn konfiguriert)
- kommende Events anzeigen
- sich für Events an- und abmelden
- eigenes Konto löschen

### Details

- Beim ersten Login muss das Profil vollständig ausgefüllt werden
- Geschlecht: Auswahl Weiblich / Männlich / Divers
- Status: Auswahl „Schüler-/in aus Aadorf" / „Gast extern"
- Strasse ist freiwillig, PLZ und Ort sind Pflicht
- Notfallkontakt: Name, Telefonnummer, Beziehung (Auswahl Vater / Mutter / Andere)
- Keine Altersgrenze, keine Einwilligung der Eltern in der App
- Abmeldung ist bis zum Start des Events möglich
- Keine maximale Teilnehmerzahl pro Event

## Was ein Projektleiter kann

- Zeiterfassung (eigene Aufwände)
- Berichte für Events verfassen
- Teilnehmererfassung (über QR-Scanner oder über Eingabemaske in der App)
- Anlegen von Events (Datum, Zeitraum, Veranstaltungsort <Standard: Guntershausen>)
- Liste der angemeldeten Teilnehmer pro Event anzeigen

### Check-in

- Teilnehmer meldet sich vorgängig für einen Event an
- Vor Ort zeigt der Teilnehmer seinen QR-Code dem Projektleiter
- Projektleiter scannt den QR-Code mit seinem Handy (Kamera über die Web-App)
- Nach dem Scan sieht der Projektleiter alle Angaben zur Person, die Anwesenheit ist bestätigt
- Jeder Teilnehmer hat einen festen QR-Code, gültig für alle Events
- Der QR-Code enthält einen signierten Token, nicht die nackte ID
- Ein Teilnehmer kann pro Event nur einmal eingecheckt werden
- Nicht angemeldete Teilnehmer können vor Ort trotzdem eingecheckt werden
- Eingabemaske: Suche nach bestehendem Teilnehmer
- Personen ohne Konto können als Gast erfasst werden
  - gleiche Pflichtfelder wie im Profil, nur die Email ist freiwillig
  - Gäste werden als Benutzer gespeichert (Kennzeichen „Gast") und sind an späteren Events über die Suche auffindbar
  - meldet sich ein Gast später mit derselben Email an, übernimmt er das Konto

### Events

- Events werden einzeln angelegt (keine Serien)
- Events können bearbeitet und abgesagt, aber nicht gelöscht werden
- Jeder Event hat genau einen verantwortlichen Projektleiter

### Team und Einsatz

- Seite **Team** (Projektleiter, Admins): zeigt alle Projektleiter, Mitarbeiter und Admins
- Oben eine Übersicht: Anzahl Personen pro Funktion, ohne Funktion und total
- Jede Person im Team hat genau eine Funktion (z.B. JC, SC), festgelegt durch Admins
- Beim Erstellen/Bearbeiten eines Events:
  - genau ein verantwortlicher Projektleiter (Projektleiter oder Admin)
  - Team-Mitglieder auswählen, ihre Funktion wird angezeigt
  - Betrag wird aus der Funktion übernommen und ist nicht editierbar
  - Funktionen ohne festen Lohn (z.B. PL) erhalten keinen Betrag am Event, sie werden über die Zeiterfassung abgerechnet
- Der Betrag wird beim Einteilen festgehalten: wer mit derselben Funktion eingeteilt bleibt, behält ihn, spätere Lohnänderungen verändern vergangene Events nicht
- Team und Lohn am Event festlegen nur durch Projektleiter und Admins, Mitarbeiter sehen den Einsatz ohne Lohn

### Auszahlung

- Button **Auszahlung** am Event (Projektleiter, Admins) öffnet die Auszahlungsseite
- Zeigt alle eingeteilten Personen mit Name, Funktion und Betrag
- Jede Person bestätigt den Empfang mit ihrer handschriftlichen Unterschrift im Eingabefeld (Finger, Stift oder Maus)
- Eine versehentliche Unterschrift kann entfernt und neu erfasst werden
- **Drucken** (oder im Druckdialog als PDF sichern): Spesenblatt mit Name, Funktion, Einsatz und Unterschrift, leeren Zeilen für Ergänzungen, Total pro Funktion und Feld für die Unterschrift der Projektleitung
- Personen, die über die Zeiterfassung abgerechnet werden (z.B. PL), werden separat aufgeführt
- Bleibt eine Person mit derselben Funktion eingeteilt, bleibt ihre Unterschrift beim Bearbeiten des Events erhalten

### Zeiterfassung

- Eintrag: Datum, Dauer (Stunden), Beschreibung, optional Event
- Projektleiter sehen nur eigene Einträge, Admins sehen alle

### Berichte

- Ein Bericht pro Event (Freitext)
- Teilnehmerzahl wird automatisch übernommen
- **Bericht drucken** (oder im Druckdialog als PDF sichern): Datum, Ort, zuständiger Projektleiter, Teilnehmer vor Ort (W/M/D), anwesendes Team mit Funktion und Bericht
- Lesbar für alle Projektleiter und Admins

## Was ein Mitarbeiter kann

- alles, was ein Projektleiter kann, ausser:
  - keine Berichte (weder lesen noch schreiben)
  - keine Zeiterfassung
- kann nicht als Projektleiter einem Event zugewiesen werden

## Was ein Admin kann

- alles, was ein Projektleiter kann
- Rollen vergeben und entziehen
- Benutzer löschen (auch Gäste), Check-ins bleiben anonymisiert erhalten
- Seite **Benutzer** als Tabelle (Name, Rolle, Aktionen): Notfallkontakt (Telefon-Symbol), Bearbeiten (Stift) und Löschen (Papierkorb)
- Profile bearbeiten: alle Profilangaben, bei Gästen zusätzlich die Email. Meldet sich der Gast danach mit dieser Email über Google an, übernimmt er das Konto

## Notfallkontakte

- Seite **Benutzer** mit Button „Notfallkontakt" pro Person
- Zugriff: Personen, deren Rolle oder Funktion Zugriff hat
- Welche Rollen und Funktionen Zugriff haben, legen Admins unter **Administration** fest (Standard: Projektleiter)
- Admins haben immer Zugriff, Teilnehmer nie
- Rollen ändern, Benutzer bearbeiten und löschen bleibt Admins vorbehalten
- Ohne Zugriff wird der Notfallkontakt auch beim Check-in (Scan) nicht angezeigt
- Seite **Administration**: Funktionen mit Lohn pro Einsatz und Veranstaltungsorte verwalten
  - Startwerte: JC (20 CHF), JSC (40 CHF), SC (80 CHF), PL (über Zeiterfassung)
  - Ohne Lohn: Abrechnung über die Zeiterfassung
  - Zugriff auf Notfallkontakte: Rollen und Funktionen auswählen
  - Funktionen, die an Events eingesetzt sind, können nicht gelöscht werden
- Zeiteinträge als CSV exportieren

## Veranstaltungsorte

- Orte werden in der Datenbank gespeichert
- Initial angelegt: Guntershausen, Aadorf, Ettenhausen (inkl. Adressen der Turnhallen)
- Orte können unter **Administration** bearbeitet und neue Orte erfasst werden

## Authentifizierung

- Alle Benutzer melden sich über ihren Google Account oder ihre Apple ID an (Apple optional, wenn konfiguriert)
- Es gibt die Rollen Admin, Projektleiter, Mitarbeiter und Teilnehmer
- Standardmässig hat der Benutzer die Rolle Teilnehmer
- Die Admins werden über die Umgebungsvariable `ADMIN_EMAIL` festgelegt (komma-getrennt, z.B. `a@gmail.com,b@gmail.com`)
- Google OAuth 2.0 bzw. Sign in with Apple, Session über HttpOnly-Cookie, Sessions in der Datenbank

## Datenschutz

- Teilnehmer sehen nur ihre eigenen Daten
- Projektleiter und Admins sehen die Daten aller Teilnehmer
- Bei Kontolöschung bleiben Check-ins anonymisiert für die Statistik erhalten
- Keine automatische Löschung nach Frist

## Betrieb der Applikation

Die Applikation ist eine klassische 3-Tier Applikation.
Sie verfügt über ein Frontend in TypeScript, ein Backend in Golang und kommuniziert mit einer PostgreSQL Datenbank, welche sie für die Persistierung sämtlicher Informationen benötigt.

- Nur eine Datenbank (PostgreSQL): Open Source, skalierbar, lokal und auf Kubernetes lauffähig
- Frontend: React mit Vite, als PWA für Smartphones optimiert
- Oberfläche nur auf Deutsch
- Backend stellt eine REST-API unter `/api` bereit und liefert das gebaute Frontend aus
- Ein Docker-Image für die App, PostgreSQL als separater Container
- Konfiguration über Umgebungsvariablen (OAuth-Credentials, Datenbank-URL, `ADMIN_EMAIL`)
- HTTPS und Domain über vorgelagerten Reverse Proxy / Ingress
- Unit-Tests für Backend und Frontend, CI über GitHub Actions

## Dokumentation

- [Architektur](docs/architektur.md): Systemübersicht, Anwendungsfälle, Datenmodell (ERD) und Abläufe (Sequenzdiagramme)
- [Anforderungen und Stand](docs/anforderungen.md): Anforderungen des Auftraggebers und was davon umgesetzt ist
- [Lokale Entwicklung](docs/lokale-entwicklung.md)
- [Deployment auf Cloud Run](docs/deployment-cloud-run.md)
- [Demo-Deployment mit PostgreSQL auf Cloud Run](docs/deployment-demo.md)
- [Zugangsdaten, Tokens und Secrets](docs/zugangsdaten.md)

## Struktur des Projekts

- Makefile (make run, make build)
  - `make run`: startet App und PostgreSQL lokal via Docker Compose
  - `make dev`: startet PostgreSQL in Docker, Backend und Frontend mit Hot Reload
  - `make build`: baut das Docker-Image, Tag = Version aus Git (`make version`)
  - `make deploy-demo`: deployt die Demo auf Cloud Run, Werte aus `.env`
- Die Version der App ist unter `/api/version` abrufbar und steht unten auf jeder Seite
- docs (Ordner mit Anleitungen)
- build (Ordner mit Dockerfile und benötigten Build-Instruktionen)
- web (Ordner, der das Web UI beinhaltet)
- api (Ordner, der das Backend beinhaltet)

## Lizenz und Nutzung

Copyright (c) 2026 Louis Kern (tribock). Alle Rechte vorbehalten.

- Der Quellcode ist öffentlich, damit er eingesehen werden kann. Er ist **nicht** Open Source
- Jede Nutzung braucht einen schriftlichen Vertrag: Ausführen und Betreiben (auch als Test oder Demo),
  Anbieten als Dienst für Dritte, Kopieren, Ändern, Weiterentwickeln und Weitergeben
- Ein Vertrag gilt für die darin genannte Organisation, den Zweck und die Dauer
- Komponenten Dritter (Bibliotheken in `api/go.mod` und `web/package.json`) stehen unter ihren eigenen Lizenzen
- Anfragen: über [github.com/tribock](https://github.com/tribock)

Verbindlich ist der Text in [LICENSE](LICENSE).
