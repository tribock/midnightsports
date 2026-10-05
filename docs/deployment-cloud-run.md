# Deployment auf Google Cloud Run

Die App läuft als Container auf Cloud Run, die Datenbank auf Cloud SQL (PostgreSQL).
Region: `europe-west6` (Zürich).

## Voraussetzungen

- Google Account mit Zahlungsmethode
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) installiert
- Docker Desktop

## Variablen

Diese Variablen werden in allen folgenden Befehlen verwendet. Vor jeder Sitzung im Terminal setzen:

```sh
export PROJECT_ID=midnightsports-prod      # muss weltweit eindeutig sein
export REGION=europe-west6
export SERVICE=midnightsports
export DB_INSTANCE=midnightsports-db
export IMAGE=$REGION-docker.pkg.dev/$PROJECT_ID/midnightsports/app
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
  sqladmin.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com
```

### 3. Container Registry anlegen

```sh
gcloud artifacts repositories create midnightsports \
  --repository-format=docker --location=$REGION
gcloud auth configure-docker $REGION-docker.pkg.dev
```

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
printf '%s' '<GOOGLE_CLIENT_SECRET>' | gcloud secrets create google-client-secret --data-file=-
openssl rand -base64 32 | tr -d '\n' | gcloud secrets create session-secret --data-file=-
openssl rand -base64 32 | tr -d '\n' | gcloud secrets create qr-signing-key --data-file=-
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
gcloud run deploy $SERVICE \
  --image=$IMAGE:v1 \
  --region=$REGION \
  --service-account=$SA \
  --add-cloudsql-instances=$PROJECT_ID:$REGION:$DB_INSTANCE \
  --set-secrets=DATABASE_URL=database-url:latest,GOOGLE_CLIENT_SECRET=google-client-secret:latest,SESSION_SECRET=session-secret:latest,QR_SIGNING_KEY=qr-signing-key:latest \
  --set-env-vars=GOOGLE_CLIENT_ID=<CLIENT_ID>,ADMIN_EMAIL=<ADMIN_EMAIL> \
  --allow-unauthenticated \
  --min-instances=0 \
  --max-instances=3
```

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
- Mit dem Google Account aus `ADMIN_EMAIL` anmelden, die Rolle Admin muss aktiv sein

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
VERSION=v2   # bei jedem Deployment erhöhen

docker buildx build --platform linux/amd64 \
  -f build/Dockerfile -t $IMAGE:$VERSION --push .

gcloud run deploy $SERVICE --image=$IMAGE:$VERSION --region=$REGION
```

- `--platform linux/amd64` ist auf Macs mit Apple Silicon zwingend
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
