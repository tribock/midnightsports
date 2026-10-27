# Lokale Entwicklung

## Voraussetzungen

- Go (aktuelle Version)
- Node.js (LTS) und npm
- Docker Desktop (inkl. Docker Compose)
- make

## Einrichtung

1. Repository klonen
2. Konfiguration anlegen:

   ```sh
   cp .env.example .env
   ```

3. Werte in `.env` ergänzen (siehe [Zugangsdaten](zugangsdaten.md))
   - Ohne Google OAuth: `DEV_LOGIN=true` setzen (siehe unten)

## Starten

Es gibt zwei Varianten:

| Befehl | Zweck | URL |
|---|---|---|
| `make run` | Ganze App in Docker (wie in Produktion) | http://localhost:8080 |
| `make dev` | Entwicklung mit Hot Reload | http://localhost:5173 |

### `make run`

- Startet App und PostgreSQL via Docker Compose
- Baut das Image bei jedem Start neu
- Stoppen: `Ctrl+C`, danach `make down`

### `make dev`

- Startet PostgreSQL in Docker
- Startet das Backend mit `go run` auf Port 8080
- Startet das Frontend mit Vite auf Port 5173
- Vite leitet Anfragen an `/api` an das Backend weiter
- Änderungen im Frontend sind sofort sichtbar, das Backend muss bei Änderungen neu gestartet werden

## Weitere Befehle

| Befehl | Beschreibung |
|---|---|
| `make db` | Nur PostgreSQL starten |
| `make down` | Alle Container stoppen |
| `make db-reset` | Lokale Datenbank löschen und neu anlegen (alle Daten gehen verloren). Nur lokal, nie in Produktion |
| `make test` | Tests für Backend und Frontend ausführen |
| `make version` | Aktuelle Version aus Git anzeigen |
| `make build` | Docker-Image bauen und auf Docker Hub pushen (Tag = Version) |
| `make deploy-demo` | Demo auf Cloud Run deployen (siehe [Demo-Deployment](deployment-demo.md)) |

## Datenbank

- Lokal wird kein Cloud SQL und kein Google-Konto benötigt
- Cloud SQL ist ein verwaltetes PostgreSQL, lokal läuft dieselbe Version (PostgreSQL 17) in Docker
- Die App unterscheidet nur über `DATABASE_URL`, der Code ist identisch
- Läuft im Container `postgres` auf `localhost:5432`
- Benutzer, Passwort und Datenbank: `midnightsports`
- Daten bleiben in einem Docker-Volume erhalten
- Migrationen und Startdaten (Veranstaltungsorte) laufen automatisch beim Start des Backends
- Zugriff mit psql:

  ```sh
  docker compose exec postgres psql -U midnightsports
  ```

## Login ohne Google (`DEV_LOGIN`)

- Mit `DEV_LOGIN=true` zeigt die Login-Seite ein Feld für eine beliebige Email-Adresse
- Damit lassen sich mehrere Benutzer (Teilnehmer, Projektleiter) einfach testen
- Die Emails aus `ADMIN_EMAIL` (komma-getrennt) erhalten die Rolle Admin
- Nur lokal verwenden, nie in Produktion setzen
- Das Backend verweigert den Start, wenn `DEV_LOGIN=true` und keine localhost-URL konfiguriert ist

## QR-Scanner testen

- Der Browser erlaubt Kamerazugriff nur über HTTPS oder `localhost`
- Am Laptop: Webcam über http://localhost:5173 verwenden
- QR-Code des Teilnehmers in einem zweiten Browserfenster (anderer Benutzer) anzeigen
- Test mit dem Handy: HTTPS-Tunnel verwenden, z.B.

  ```sh
  cloudflared tunnel --url http://localhost:8080
  ```

  Die Tunnel-URL muss als Weiterleitungs-URI im OAuth Client eingetragen werden (oder `DEV_LOGIN` verwenden).
