# Deployment auf Google Cloud Run

Die App läuft als Container auf Cloud Run, die Datenbank auf Cloud SQL (PostgreSQL).
Region: `europe-west6` (Zürich).

## Voraussetzungen

- Google Account mit Zahlungsmethode
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) installiert
- Docker Desktop
- Docker Hub Account mit öffentlichem Repository `tribock/midnightsports`

## Variablen

Diese Variablen werden in allen folgenden Befehlen verwendet. Vor jeder Sitzung im Terminal setzen:

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
Die Produktions-URL wird nach Schritt 9 ergänzt.

### 6. Secrets anlegen

```sh
printf '%s' "postgres://midnightsports:$DB_PASSWORD@/midnightsports?host=/cloudsql/$PROJECT_ID:$REGION:$DB_INSTANCE" \
  | gcloud secrets create database-url --data-file=-
jq -r .web.client_secret secrets/google-wallet-key.json | tr -d '\n' \
  | gcloud secrets create google-wallet --data-file=-

gcloud secrets versions add google-wallet --data-file=secrets/google-wallet-key.json

openssl rand -base64 32 | tr -d '\n' | gcloud secrets create session-secret --data-file=-
openssl rand -base64 32 | tr -d '\n' | gcloud secrets create qr-signing-key --data-file=-
export CLIENT_ID=$(jq -r .web.client_id secrets/midnightsports.json)

```

### 7. Service Account für Cloud Run

```sh
gcloud iam service-accounts create midnightsports-run

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA --role=roles/cloudsql.client
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA --role=roles/secretmanager.secretAccessor
```

### 8. Erstes Deployment

Image bauen und hochladen (siehe [Neue Version deployen](#neue-version-deployen)), danach:

```sh
VERSION=$(make -s version)
make build VERSION=$VERSION

gcloud run deploy $SERVICE \
  --image=$RUN_IMAGE:$VERSION \
  --region=$REGION \
  --service-account=$SA \
  --add-cloudsql-instances=$PROJECT_ID:$REGION:$DB_INSTANCE \
  --set-secrets=DATABASE_URL=database-url:latest,GOOGLE_CLIENT_SECRET=google-client-secret:latest,SESSION_SECRET=session-secret:latest,QR_SIGNING_KEY=qr-signing-key:latest \
  --set-env-vars='^;^GOOGLE_CLIENT_ID=<CLIENT_ID>;ADMIN_EMAIL=<ADMIN_EMAIL>' \
  --allow-unauthenticated \
  --min-instances=0 \
  --max-instances=3
```

- `ADMIN_EMAIL` kann mehrere Adressen enthalten, komma-getrennt (z.B. `a@gmail.com,b@gmail.com`)
- `^;^` am Anfang setzt `;` als Trennzeichen zwischen den Variablen, damit die Kommas in `ADMIN_EMAIL` erhalten bleiben

### 9. Redirect-URL setzen

```sh
URL=$(gcloud run services describe $SERVICE --region=$REGION --format='value(status.url)')
echo $URL

gcloud run services update $SERVICE --region=$REGION \
  --update-env-vars=OAUTH_REDIRECT_URL=$URL/api/auth/callback
```

Danach `$URL/api/auth/callback` im OAuth Client als Weiterleitungs-URI eintragen.

### 10. Testen

- `$URL` im Browser öffnen
- Mit einem Google Account aus `ADMIN_EMAIL` anmelden, die Rolle Admin muss aktiv sein

### 11. Wallet aktivieren (optional)

Zertifikate erstellen: siehe [Zugangsdaten](zugangsdaten.md#6-apple-wallet-optional).
Die Dateien werden als Secrets in den Container gemountet.

```sh
gcloud secrets create apple-pass-pem --data-file=secrets/apple-pass.pem
gcloud secrets create google-wallet-key --data-file=secrets/google-wallet-key.json

gcloud run services update $SERVICE --region=$REGION \
  --update-secrets=/secrets/apple/pass.pem=apple-pass-pem:latest,/secrets/google/key.json=google-wallet-key:latest \
  --update-env-vars=APPLE_PASS_TYPE_ID=<PASS_TYPE_ID>,APPLE_TEAM_ID=<TEAM_ID>,APPLE_PASS_PEM_FILE=/secrets/apple/pass.pem,GOOGLE_WALLET_ISSUER_ID=<ISSUER_ID>,GOOGLE_WALLET_KEY_FILE=/secrets/google/key.json
```

Nur eine der beiden Wallets nutzen: die Variablen und Secrets der anderen weglassen.

## Neue Version deployen

```sh
git tag v1.0.0               # optional: Release-Version vergeben
VERSION=$(make -s version)   # z.B. v1.0.0, 5bd2699 oder 5bd2699-dirty.20261006063011

make build VERSION=$VERSION
gcloud run deploy $SERVICE --image=$RUN_IMAGE:$VERSION --region=$REGION
```

- Die Version kommt aus Git (`git describe`): Tag, sonst Commit-Hash, `-dirty` bei uncommitteten Änderungen
- `make build` baut für `linux/amd64` und `linux/arm64`, lädt das Image als `$IMAGE:<version>` auf Docker Hub und schreibt die Version in die App
- Die laufende Version steht unten auf jeder Seite und unter `/api/version`
- Jede Version hat einen eigenen Tag: so ist klar, welche Version läuft, und ein Rollback ist möglich
- Einstellungen und Secrets aus dem ersten Deployment bleiben erhalten
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
- Nach dem Einrichten `OAUTH_REDIRECT_URL` und den OAuth Client auf die neue Domain anpassen

## Kosten

Siehe Schätzung: ca. 15–45 USD pro Monat (Cloud Run fast immer im Gratis-Kontingent, Cloud SQL ist der Hauptkostenpunkt).
Budget-Alarm empfohlen: Console → **Abrechnung → Budgets & Benachrichtigungen**.
