# Midnightsports

Ein Tool zur Verwaltung von Events von Midnightsports

Genutzt wird das Tool von Projektleitern und Teilnehmern.
Die Events sind wiederkehrende Veranstaltungen in der Turnhalle Guntershausen.
Alternativ kann die Veranstaltung auch in den Turnhallen in Aadorf oder in Ettenhausen durchgeführt werden.


## Was ein Teilnehmer kann

- eigene Daten erfassen (Name, Vorname, Email, Geburtsdatum, Adresse und Notfallkontakt)
- Anzeigen seines QR-Codes, um sich beim Eintreffen am Event zu registrieren
- QR-Code zu Apple Wallet oder Google Wallet hinzufügen (optional, wenn konfiguriert)
- kommende Events anzeigen
- sich für Events an- und abmelden
- eigenes Konto löschen

### Details

- Beim ersten Login muss das Profil vollständig ausgefüllt werden
- Notfallkontakt: Name, Telefonnummer, Beziehung
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
- Personen ohne Konto können als Gast (Name, Vorname) erfasst werden

### Events

- Events werden einzeln angelegt (keine Serien)
- Events können bearbeitet und abgesagt, aber nicht gelöscht werden
- Einem Event können mehrere Projektleiter zugewiesen werden

### Zeiterfassung

- Eintrag: Datum, Dauer (Stunden), Beschreibung, optional Event
- Projektleiter sehen nur eigene Einträge, Admins sehen alle

### Berichte

- Ein Bericht pro Event (Freitext)
- Teilnehmerzahl wird automatisch übernommen
- Lesbar für alle Projektleiter und Admins

## Was ein Admin kann

- alles, was ein Projektleiter kann
- Rollen vergeben und entziehen
- Veranstaltungsorte verwalten
- Zeiteinträge als CSV exportieren

## Veranstaltungsorte

- Orte werden in der Datenbank gespeichert
- Initial angelegt: Guntershausen, Aadorf, Ettenhausen (inkl. Adressen der Turnhallen)
- Orte können bearbeitet und neue Orte erfasst werden

## Authentifizierung

- Alle Benutzer melden sich über ihren Google Account an
- Es gibt die Rollen Admin, Projektleiter und Teilnehmer
- Standardmässig hat der Benutzer die Rolle Teilnehmer
- Die Admins werden über die Umgebungsvariable `ADMIN_EMAIL` festgelegt (komma-getrennt, z.B. `a@gmail.com,b@gmail.com`)
- Google OAuth 2.0, Session über HttpOnly-Cookie, Sessions in der Datenbank

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
