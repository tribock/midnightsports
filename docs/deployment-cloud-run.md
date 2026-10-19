# Deployment auf Google Cloud Run

Die App läuft als Container auf Cloud Run, die Datenbank auf Cloud SQL (PostgreSQL).
Region: `europe-west6` (Zürich).

## Voraussetzungen

- Google Account mit Zahlungsmethode
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) installiert
- Docker Desktop
- Docker Hub Account mit öffentlichem Repository `tribock/midnightsports`

## Variablen

Deployt wird mit `make deploy`: die Vorlage [build/deploy/deploy.yaml](../build/deploy/deploy.yaml) wird mit
Werten aus `.env` ausgefüllt (wie bei der [Demo](deployment-demo.md)). Für die einmalige Einrichtung (Schritte 1–7)
werden diese Variablen im Terminal verwendet:

```sh
export PROJECT_ID=midnightsports      # muss weltweit eindeutig sein
export REGION=europe-west6
export SERVICE=midnightsports
export DB_INSTANCE=midnightsports-db
export IMAGE=docker.io/tribock/midnightsports                                  # Push (Docker Hub)
export RUN_IMAGE=$REGION-docker.pkg.dev/$PROJECT_ID/dockerhub/tribock/midnightsports  # Deploy (via Proxy)
export SA=midnightsports-run@$PROJECT_ID.iam.gserviceaccount.com
```

## Ersteinrichtung (einmalig)

### 1. Projekt anlegen

```sh
gcloud auth login
gcloud projects create $PROJECT_ID
gcloud config set project $PROJECT_ID
gcloud billing accounts list
gcloud billing projects link $PROJECT_ID --billing-account=<BILLING_ACCOUNT_ID>
```

### 2. APIs aktivieren

```sh
gcloud services enable \
  run.googleapis.com \
  secretmanager.googleapis.com \
  artifactregistry.googleapis.com \
  sqladmin.googleapis.com
```

### 3. Docker Hub anbinden

```sh
docker login

gcloud artifacts repositories create dockerhub \
  --repository-format=docker --location=$REGION \
  --mode=remote-repository --remote-docker-repo=DOCKER-HUB
```

- Das Image wird auf Docker Hub veröffentlicht (`$IMAGE`)
- Cloud Run lädt es über ein Artifact Registry Remote Repository, das als Proxy für Docker Hub dient (`$RUN_IMAGE`)
- Direkt mit `docker.io/...` deployen funktioniert nicht zuverlässig: Cloud Run lädt solche Images über `mirror.gcr.io`, dort fehlen eigene Images oft (`Image not found`)
- Das Repository auf Docker Hub muss **öffentlich** sein
- Das Image enthält keine Secrets (`.env` und `secrets/` sind in `.dockerignore`)

### 4. Datenbank anlegen

Dauert einige Minuten.

```sh
gcloud sql instances create $DB_INSTANCE \
  --database-version=POSTGRES_17 \
  --edition=ENTERPRISE \
  --tier=db-f1-micro \
  --region=$REGION \
  --storage-size=10 \
  --backup-start-time=02:00

gcloud sql databases create midnightsports --instance=$DB_INSTANCE

DB_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=')
gcloud sql users create midnightsports --instance=$DB_INSTANCE --password=$DB_PASSWORD
```

- `db-f1-micro` ist die günstigste Klasse (ohne SLA), bei Bedarf später auf `db-g1-small` wechseln
- Tägliche Backups sind aktiviert

### 5. OAuth Client erstellen

Siehe [Zugangsdaten](zugangsdaten.md#1-google-oauth-client-login-mit-google).
JSON als `secrets/midnightsports.json` speichern, `make deploy` liest die Client-ID daraus.
Die Produktions-URL wird in Schritt 9 ergänzt.

### 6. Secrets anlegen

```sh
printf '%s' "postgres://midnightsports:$DB_PASSWORD@/midnightsports?host=/cloudsql/$PROJECT_ID:$REGION:$DB_INSTANCE" \
  | gcloud secrets create database-url --data-file=-
jq -r .web.client_secret secrets/midnightsports.json | tr -d '\n' \
  | gcloud secrets create google-client-secret --data-file=-

openssl rand -base64 32 | tr -d '\n' | gcloud secrets create session-secret --data-file=-
openssl rand -base64 32 | tr -d '\n' | gcloud secrets create qr-signing-key --data-file=-
```

- Existiert ein Secret schon, stattdessen `gcloud secrets versions add NAME --data-file=-` verwenden
- Gratis sind 6 aktive Secret-Versionen pro Abrechnungskonto. Alte Versionen nach dem Ersetzen löschen:
  `gcloud secrets versions destroy <VERSION> --secret=NAME`

### 7. Service Account für Cloud Run

```sh
gcloud iam service-accounts create midnightsports-run

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA --role=roles/cloudsql.client
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA --role=roles/secretmanager.secretAccessor
```

- Wurde der Service Account für die [Demo](deployment-demo.md) angelegt, fehlt `roles/cloudsql.client`.
  Ohne diese Rolle startet die App nicht (`NOT_AUTHORIZED ... cloudsql.instances.get` im Log)

### 8. Konfiguration in `.env`

| Variable | Pflicht | Beschreibung |
|---|---|---|
| `ADMIN_EMAIL` | ja | Admins, komma-getrennt (gleiche Variable wie lokal und in der Demo) |
| `GOOGLE_WALLET_ISSUER_ID` | ja | Google Wallet, Issuer ID (gleiche Variable wie in der Demo) |
| `PROD_GOOGLE_CLIENT_ID` | nein | OAuth Client, Standard: aus `secrets/midnightsports.json` |
| `PROD_URL` | nein | Öffentliche Adresse ohne `/` am Ende, Standard: `https://<SERVICE>-<PROJEKTNUMMER>.<REGION>.run.app` |
| `SERVICE` | nein | Name des Cloud Run Service, Standard: `midnightsports` |
| `DB_INSTANCE` | nein | Cloud-SQL-Instanz, Standard: `midnightsports-db` |
| `GCP_PROJECT_ID` | nein | Standard: `midnightsports` |
| `GCP_REGION` | nein | Standard: `europe-west6` |
| `GCP_PROJECT_NUMBER` | nein | Standard: wird mit `gcloud` ermittelt |

Secrets stehen nicht in `.env`, die Vorlage verweist auf die Secrets aus Schritt 6.
Kontrolle der fertigen Konfiguration (wird nur angezeigt, nicht deployt):

```sh
make deploy-yaml
```

### 9. Erstes Deployment

```sh
make deploy
```

- `make deploy` macht den Service öffentlich (`allUsers` als `roles/run.invoker`, ersetzt `--allow-unauthenticated`)
  und prüft danach, dass `/api/health` antwortet

- Die Adresse der App ist `PROD_URL`, siehe `make deploy-yaml` bei `OAUTH_REDIRECT_URL`
- `<PROD_URL>/api/auth/callback` im OAuth Client als Weiterleitungs-URI eintragen

Erklärungen zur Vorlage:

- `google-wallet`: Secret mit dem Service-Account-Schlüssel, eingebunden als `/secrets/google-wallet/key.json`.
  Anlegen: siehe [Zugangsdaten](zugangsdaten.md#7-google-wallet-optional),
  `gcloud secrets create google-wallet --data-file=secrets/google-wallet-key.json`
- `cloudsql-instances`: verbindet die Instanz über den Unix-Socket `/cloudsql/...`, den `database-url` verwendet
- `minScale` 0 / `maxScale` 3: ohne Anfragen keine Instanz, höchstens 3. `run.googleapis.com/maxScale` beim Service
  begrenzt den ganzen Service, `autoscaling.knative.dev/maxScale` die Revision, beide müssen gesetzt sein. Für einen Event-Abend ohne Kaltstart
  vorübergehend `minScale: "1"` setzen
- Label `build-date`: erzwingt bei jedem Deployment eine neue Revision
- `make deploy` ersetzt die ganze Konfiguration des Service. Änderungen in der Console oder mit
  `gcloud run services update` gehen beim nächsten Deployment verloren, deshalb immer die Vorlage anpassen

### 10. Testen

- `PROD_URL` im Browser öffnen
- Mit einem Google Account aus `ADMIN_EMAIL` anmelden, die Rolle Admin muss aktiv sein

### 11. Optionale Funktionen

Apple Wallet und Login mit Apple sind in der Vorlage auskommentiert. Zum Aktivieren:

1. Zugangsdaten erstellen und als Secret ablegen:

   | Funktion | Anleitung | Secret |
   |---|---|---|
   | Apple Wallet | [Zugangsdaten](zugangsdaten.md#6-apple-wallet-optional) | `gcloud secrets create apple-pass-pem --data-file=secrets/apple-pass.pem` |
   | Login mit Apple | [Zugangsdaten](zugangsdaten.md#8-login-mit-apple-optional) | `gcloud secrets create apple-login-key --data-file=secrets/apple-login.p8` |

2. Werte in `.env` setzen:
   - Apple Wallet: `APPLE_PASS_TYPE_ID`, `APPLE_TEAM_ID`
   - Login mit Apple: `APPLE_TEAM_ID`, `PROD_APPLE_LOGIN_CLIENT_ID`, `PROD_APPLE_LOGIN_KEY_ID`.
     In der Services ID bei Apple die Domain von `PROD_URL` und die Return URL `<PROD_URL>/api/auth/apple/callback` eintragen
3. In [build/deploy/deploy.yaml](../build/deploy/deploy.yaml) die Blöcke `# Optional: <Funktion>` an allen drei Stellen
   (`env`, `volumeMounts`, `volumes`) einkommentieren.
   `APPLE_TEAM_ID` nur einmal aufführen, wenn Apple Wallet und Login mit Apple aktiv sind
4. `make deploy-yaml` kontrollieren, dann `make deploy`

## Neue Version deployen

```sh
git tag v1.0.0   # optional: Release-Version vergeben
make deploy
```

- Die Version kommt aus Git (`git describe`): Tag, sonst Commit-Hash, `-dirty` bei uncommitteten Änderungen
- `make deploy` baut für `linux/amd64` und `linux/arm64`, lädt das Image als `$IMAGE:<version>` auf Docker Hub,
  schreibt die Version in die App und deployt es mit der Vorlage
- Die laufende Version steht unten auf jeder Seite und unter `/api/version`
- Jede Version hat einen eigenen Tag: so ist klar, welche Version läuft, und ein Rollback ist möglich
- Datenbank-Migrationen laufen automatisch beim Start

## Rollback

```sh
gcloud run revisions list --service=$SERVICE --region=$REGION
gcloud run services update-traffic $SERVICE --region=$REGION \
  --to-revisions=<REVISION>=100
```

## Logs

```sh
gcloud run services logs read $SERVICE --region=$REGION --limit=100
```

Oder in der Console unter **Cloud Run → midnightsports → Logs**.

## Eigene Domain (optional)

- Cloud Run Domain-Mapping ist nicht in allen Regionen verfügbar
- Alternative: Global External Application Load Balancer (ca. 18 USD/Monat zusätzlich)
- Nach dem Einrichten `PROD_URL` in `.env` setzen, den OAuth Client (und die Services ID bei Apple) auf die neue Domain anpassen und neu deployen

## Kosten

Siehe Schätzung: ca. 15–45 USD pro Monat (Cloud Run fast immer im Gratis-Kontingent, Cloud SQL ist der Hauptkostenpunkt).
Budget-Alarm empfohlen: Console → **Abrechnung → Budgets & Benachrichtigungen**.
