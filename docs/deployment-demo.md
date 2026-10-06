# Demo-Deployment: App und PostgreSQL auf Cloud Run

Nur für Demonstrationen. PostgreSQL läuft als zweiter Container (Sidecar) in derselben Cloud-Run-Instanz.

**Einschränkungen:**

- Die Daten liegen im Arbeitsspeicher und gehen bei jedem Neustart der Instanz verloren
- Neustarts passieren bei jedem Deployment und gelegentlich ohne Vorwarnung
- Es läuft genau eine Instanz (keine Skalierung)
- Keine Backups

Für den Betrieb: [Deployment auf Cloud Run](deployment-cloud-run.md) mit Cloud SQL verwenden.

## Kosten

- `minScale: 0`: Ohne Anfragen läuft keine Instanz, es entstehen praktisch keine Kosten
- Nach ca. 15 Minuten ohne Anfragen wird die Instanz gestoppt, die Daten sind dann weg
- Für eine längere Demo-Session vorübergehend `minScale: "1"` setzen (ca. 1–3 USD pro Tag)

## Voraussetzungen

Aus der [Deployment-Anleitung](deployment-cloud-run.md) folgende Schritte ausführen:

- Schritt 1: Projekt anlegen
- Schritt 2: APIs aktivieren (`sqladmin.googleapis.com` ist nicht nötig)
- Schritt 3: Docker Hub anbinden (inkl. Proxy-Repository `dockerhub`)
- Schritt 5: OAuth Client erstellen, JSON als `secrets/midnightsports.json` speichern
- Schritt 6: Secrets anlegen, **ohne** `database-url`
- Schritt 7: Service Account, **ohne** die Rolle `roles/cloudsql.client`
- Google Wallet: Secret `google-wallet` mit der Service-Account-Datei (siehe [Zugangsdaten](zugangsdaten.md#7-google-wallet-optional))

Schritt 4 (Cloud SQL) entfällt. Lokal werden `gcloud`, `docker`, `jq` und `envsubst` (Paket `gettext`) benötigt.

## 1. Konfiguration in `.env`

Die Vorlage [build/deploy/demo.yaml](../build/deploy/demo.yaml) enthält Platzhalter (`${...}`).
`make deploy-demo` ersetzt sie mit Werten aus `.env`, persönliche Angaben landen so nicht im Git.

| Variable | Pflicht | Beschreibung |
|---|---|---|
| `ADMIN_EMAIL` | ja | Admins der Demo, komma-getrennt |
| `GOOGLE_WALLET_ISSUER_ID` | ja | Issuer ID aus der Google Pay & Wallet Console |
| `DEMO_GOOGLE_CLIENT_ID` | nein | OAuth Client der Demo, Standard: aus `secrets/midnightsports.json` |
| `GCP_PROJECT_ID` | nein | Standard: `midnightsports` |
| `GCP_REGION` | nein | Standard: `europe-west6` |
| `GCP_PROJECT_NUMBER` | nein | Standard: wird mit `gcloud` ermittelt |

Die übrigen Werte aus `.env` (z.B. `GOOGLE_CLIENT_ID` des lokalen OAuth Clients) werden nicht verwendet.

Kontrolle der fertigen Konfiguration (wird nur angezeigt, nicht deployt):

```sh
make demo-yaml
```

## 2. Deployen

```sh
make deploy-demo
```

- Baut das Image, lädt es auf Docker Hub und deployt es auf Cloud Run
- Image-Tag = Version aus Git (`make version`), z.B. `v1.2.0`, `5bd2699` oder `5bd2699-dirty.20261006063011`
- Jedes Deployment erzeugt eine neue Revision, die Daten sind danach weg

Einmalig nach dem ersten Deployment:

```sh
# Öffentlich erreichbar machen
gcloud run services add-iam-policy-binding midnightsports-demo --region=europe-west6 \
  --member=allUsers --role=roles/run.invoker
```

- Im OAuth Client die Weiterleitungs-URI `https://midnightsports-demo-<PROJEKTNUMMER>.europe-west6.run.app/api/auth/callback` eintragen (siehe [Zugangsdaten](zugangsdaten.md#1-google-oauth-client-login-mit-google))
- Öffnen und mit einem Google Account aus `ADMIN_EMAIL` anmelden
- Die laufende Version steht unten auf jeder Seite und unter `/api/version`

Erklärungen zur Vorlage:

- `container-dependencies`: Die App startet erst, wenn PostgreSQL bereit ist (Startup-Probe auf Port 5432)
- `minScale` 0 / `maxScale` 1: höchstens eine Instanz, ohne Anfragen keine
- `emptyDir` im Arbeitsspeicher: Datenbankdateien, max. 256 MB
- `google-wallet`: Secret wird als Datei `/secrets/google-wallet/key.json` eingebunden
- Label `build-date`: erzwingt bei jedem Deployment eine neue Revision
- Nur der Container `app` hat einen Port und ist von aussen erreichbar
- Das Datenbank-Passwort ist fest, weil PostgreSQL nur innerhalb der Instanz erreichbar ist

Ohne Google Wallet: in der Vorlage `GOOGLE_WALLET_KEY_FILE`, `GOOGLE_WALLET_ISSUER_ID`, das Volume `google-wallet` und den `volumeMounts`-Eintrag beim Container `app` entfernen.

## Ein- und ausschalten

```sh
# Ausschalten: keine Instanz startet, keine Kosten
gcloud run services update midnightsports-demo --region=europe-west6 --scaling=0

# Einschalten
gcloud run services update midnightsports-demo --region=europe-west6 --scaling=auto
```

## Fehlersuche

```sh
gcloud run services logs read midnightsports-demo --region=europe-west6 --limit=100
```

- `Fehlt: ...` bei `make deploy-demo`: Variable in `.env` setzen
- `Datenbank nicht erreichbar`: PostgreSQL ist nicht gestartet, Logs des Containers `postgres` prüfen
- `redirect_uri_mismatch` beim Login: Weiterleitungs-URI im OAuth Client fehlt oder ist falsch
- `Image ... not found`: Proxy-Repository `dockerhub` fehlt oder Image wurde nicht gepusht

## Aufräumen

```sh
gcloud run services delete midnightsports-demo --region=europe-west6
```
