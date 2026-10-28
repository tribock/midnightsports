# Lokale Entwicklung

Voraussetzungen: Go, Node.js (LTS), Docker Desktop, make.

## Einrichten und starten

```sh
cp .env.example .env    # Werte ergänzen, siehe zugangsdaten.md. Ohne Google: DEV_LOGIN=true
make dev                # PostgreSQL in Docker, Backend :8080, Frontend mit Hot Reload :5173
make run                # ganze App in Docker wie in Produktion, :8080
```

| Befehl | Beschreibung |
|---|---|
| `make db` | Nur PostgreSQL starten |
| `make down` | Container stoppen |
| `make db-reset` | Datenbank des aktuellen Branches löschen und neu anlegen (nur lokal) |
| `make psql` | Mit der Datenbank des aktuellen Branches verbinden |
| `make containers` | Container und Volumes aller Branches anzeigen |
| `make test` | Tests für Backend und Frontend |

## Datenbank

- PostgreSQL 17 in Docker auf `localhost:5432`, Benutzer, Passwort und Datenbank `midnightsports`
- Migrationen und Startdaten (Projekt Aadorf) laufen beim Start des Backends
- **Pro Branch ein eigenes Volume**: Compose-Projekt `midnightsports` auf `main`, sonst `midnightsports-<branch>`.
  Testdaten bleiben beim Branch-Wechsel erhalten. Vorher `make down`, alle nutzen dieselben Ports
- Nur über `make` wird der Branch-Name verwendet. Alten Branch entfernen: `docker compose -p midnightsports-<branch> down -v`

## Login ohne Google (`DEV_LOGIN`)

- Login mit beliebiger Email, nur lokal (startet nur mit localhost-URL)
- Emails aus `ADMIN_EMAIL` sind globale Admins
- Testen: mit einer zweiten Email anmelden, bei Aadorf beitreten, als globaler Admin unter Teilnehmer eine Rolle vergeben

## QR-Scanner

Kamera nur über HTTPS oder `localhost`. Am Laptop die Webcam über http://localhost:5173 verwenden, mit dem Handy
einen Tunnel (`cloudflared tunnel --url http://localhost:8080`) und `DEV_LOGIN` oder die Tunnel-URL im OAuth Client.
