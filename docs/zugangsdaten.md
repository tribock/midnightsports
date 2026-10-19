# Zugangsdaten, Tokens und Secrets

Anleitung zum Erstellen aller benötigten Zugangsdaten.

## Übersicht

| Variable | Beschreibung | Lokal | Cloud Run |
|---|---|---|---|
| `DATABASE_URL` | Verbindung zu PostgreSQL | `.env` | Secret Manager |
| `GOOGLE_CLIENT_ID` | OAuth Client-ID | `.env` | Umgebungsvariable |
| `GOOGLE_CLIENT_SECRET` | OAuth Client-Secret | `.env` | Secret Manager |
| `OAUTH_REDIRECT_URL` | Callback-URL nach Google-Login | `.env` | Umgebungsvariable |
| `SESSION_SECRET` | Signiert Session-Cookies | `.env` | Secret Manager |
| `QR_SIGNING_KEY` | Signiert die QR-Code-Tokens | `.env` | Secret Manager |
| `ADMIN_EMAIL` | Google-Accounts der Admins, komma-getrennt | `.env` | Umgebungsvariable |
| `DEV_LOGIN` | Login ohne Google (nur lokal!) | `.env` | nie setzen |
| `APPLE_PASS_TYPE_ID` | Apple Wallet: Pass Type ID (optional) | `.env` | Umgebungsvariable |
| `APPLE_TEAM_ID` | Apple: Team ID für Wallet und Login (optional) | `.env` | Umgebungsvariable |
| `APPLE_PASS_PEM_FILE` | Apple Wallet: Pfad zu Zertifikat und Schlüssel (optional) | `secrets/` | Secret als Datei |
| `APPLE_LOGIN_CLIENT_ID` | Login mit Apple: Services ID (optional) | `.env` | Umgebungsvariable |
| `APPLE_LOGIN_KEY_ID` | Login mit Apple: Key ID (optional) | `.env` | Umgebungsvariable |
| `APPLE_LOGIN_KEY_FILE` | Login mit Apple: Pfad zum Schlüssel `.p8` (optional) | `secrets/` | Secret als Datei |
| `GOOGLE_WALLET_ISSUER_ID` | Google Wallet: Issuer ID (optional) | `.env` | Umgebungsvariable |
| `GOOGLE_WALLET_KEY_FILE` | Google Wallet: Pfad zum Service-Account-Schlüssel (optional) | `secrets/` | Secret als Datei |

Regeln:

- Secrets nie ins Git committen (`.env` ist in `.gitignore`)
- Lokal und Produktion verwenden unterschiedliche Secrets
- Ein geändertes `QR_SIGNING_KEY` macht alle bestehenden QR-Codes ungültig
- Ein geändertes `SESSION_SECRET` meldet alle Benutzer ab
- Zertifikate und Schlüsseldateien lokal im Ordner `secrets/` ablegen (ist in `.gitignore`)
- Ohne Wallet-Variablen erscheinen die Wallet-Buttons nicht, ohne `APPLE_LOGIN_*` der Button «Mit Apple anmelden»

## 1. Google OAuth Client (Login mit Google)

Voraussetzung: Google-Cloud-Projekt.

- Für den OAuth Client ist keine Zahlungsmethode nötig, das Projekt ist kostenlos
- Projekt anlegen: Console → Projektauswahl → **Neues Projekt**
- Für die lokale Entwicklung und die Produktion kann dasselbe Projekt verwendet werden
- Lokal kann ganz auf Google verzichtet werden: `DEV_LOGIN=true` (siehe [Lokale Entwicklung](lokale-entwicklung.md))

1. [Google Cloud Console](https://console.cloud.google.com) öffnen, Projekt auswählen
2. Menü **APIs & Dienste → OAuth-Zustimmungsbildschirm** (bzw. **Google Auth Platform**)
3. **Branding**: App-Name `Midnightsports`, Support-Email, Kontakt-Email eintragen
4. **Zielgruppe**: `Extern` wählen
5. **Datenzugriff**: nur die Scopes `openid`, `email`, `profile` hinzufügen
6. **Clients → Client erstellen**
   - Anwendungstyp: `Webanwendung`
   - Name: z.B. `midnightsports-web`
   - Autorisierte Weiterleitungs-URIs:
     - `http://localhost:8080/api/auth/callback` (lokal, `make run`)
     - `http://localhost:5173/api/auth/callback` (lokal, `make dev`)
     - `https://<cloud-run-url>/api/auth/callback` (Produktion, nach dem ersten Deployment ergänzen)
7. Client-ID und Client-Secret **sofort** kopieren oder als JSON herunterladen
   - Das Secret wird von Google nur einmal angezeigt
   - Bei Verlust: im Client ein neues Secret hinzufügen, das alte löschen

Hinweise:

- Im Status `Testen` können sich nur eingetragene Testbenutzer (max. 100) anmelden
- Für den Betrieb unter **Zielgruppe → App veröffentlichen** wählen
- Mit den Scopes `openid`, `email`, `profile` ist keine Prüfung durch Google nötig

## 2. Session-Secret und QR-Signing-Key

Zufällige Werte mit mindestens 32 Bytes erzeugen:

```sh
openssl rand -base64 32   # SESSION_SECRET
openssl rand -base64 32   # QR_SIGNING_KEY
```

Jeden Befehl separat ausführen, beide Werte müssen verschieden sein.

## 3. Datenbank-Passwort

```sh
openssl rand -base64 24 | tr -d '/+='
```

Sonderzeichen werden entfernt, damit das Passwort ohne Escaping in der `DATABASE_URL` funktioniert.

Format der `DATABASE_URL`:

```
# Lokal
postgres://midnightsports:midnightsports@localhost:5432/midnightsports?sslmode=disable

# Cloud Run (Verbindung über Cloud SQL Unix-Socket)
postgres://midnightsports:<PASSWORT>@/midnightsports?host=/cloudsql/<PROJEKT>:europe-west6:midnightsports-db
```

## 4. Secrets in Google Secret Manager ablegen

```sh
printf '%s' 'WERT' | gcloud secrets create NAME --data-file=-
```

Neuen Wert für ein bestehendes Secret setzen (neue Version):

```sh
printf '%s' 'NEUER_WERT' | gcloud secrets versions add NAME --data-file=-
```

Danach Cloud Run neu deployen, damit die neue Version geladen wird.

## 5. Secrets rotieren

- `GOOGLE_CLIENT_SECRET`: in der Console neues Secret erstellen, im Secret Manager hinterlegen, deployen, altes Secret löschen
- `APPLE_LOGIN_KEY_FILE`: im Apple-Konto neuen Key erstellen, Datei und `APPLE_LOGIN_KEY_ID` ersetzen, deployen, alten Key widerrufen
- `SESSION_SECRET`: neuen Wert setzen und deployen, alle Benutzer müssen sich neu anmelden
- `QR_SIGNING_KEY`: nur bei Verdacht auf Missbrauch rotieren, alle QR-Codes werden ungültig
- Datenbank-Passwort: `gcloud sql users set-password`, danach `DATABASE_URL` aktualisieren und deployen

## 6. Apple Wallet (optional)

Voraussetzung: Mitgliedschaft im [Apple Developer Program](https://developer.apple.com/programs/) (99 USD/Jahr).

1. Schlüssel und Zertifikatsanfrage (CSR) erstellen:

   ```sh
   mkdir -p secrets && cd secrets
   openssl req -new -newkey rsa:2048 -nodes \
     -keyout apple-pass.key -out apple-pass.csr -subj "/CN=Midnightsports Pass"
   ```

2. [Apple Developer](https://developer.apple.com/account/resources/identifiers/list/passTypeId) → **Identifiers → + → Pass Type IDs**
   - Beschreibung: `Midnightsports`
   - Identifier: z.B. `pass.ch.midnightsports` → `APPLE_PASS_TYPE_ID`
3. Pass Type ID öffnen → **Create Certificate** → `apple-pass.csr` hochladen → `pass.cer` herunterladen
4. Apple WWDR-Zertifikat (G4) herunterladen: <https://www.apple.com/certificateauthority/AppleWWDRCAG4.cer>
5. Alles in eine PEM-Datei zusammenfügen:

   ```sh
   openssl x509 -inform DER -in pass.cer -out apple-pass.crt
   openssl x509 -inform DER -in AppleWWDRCAG4.cer -out apple-wwdr.crt
   cat apple-pass.crt apple-pass.key apple-wwdr.crt > apple-pass.pem
   ```

6. Team ID: [Membership Details](https://developer.apple.com/account#MembershipDetailsCard) → `APPLE_TEAM_ID`
7. In `.env`:

   ```
   APPLE_PASS_TYPE_ID=pass.ch.midnightsports
   APPLE_TEAM_ID=ABCDE12345
   APPLE_PASS_PEM_FILE=../secrets/apple-pass.pem
   ```

Hinweise:

- Der Pfad ist relativ zum Ordner `api` (`make dev`), bei `make run` wird `secrets/` nach `/secrets` gemountet
- Das Pass-Zertifikat ist ein Jahr gültig, danach Schritte 1–5 wiederholen
- Bestehende Pässe bleiben gültig, der QR-Code ändert sich nicht

## 7. Google Wallet (optional)

Kostenlos. Voraussetzung: Google-Cloud-Projekt (dasselbe wie für den OAuth Client).

1. [Google Pay & Wallet Console](https://pay.google.com/business/console) → **Google Wallet API** → Issuer-Konto erstellen 
   - Issuer ID notieren → `GOOGLE_WALLET_ISSUER_ID`
2. Google Wallet API aktivieren und Service Account erstellen:

   ```sh
   gcloud services enable walletobjects.googleapis.com
   gcloud iam service-accounts create midnightsports-wallet
   gcloud iam service-accounts keys create secrets/google-wallet-key.json \
     --iam-account=midnightsports-wallet@$PROJECT_ID.iam.gserviceaccount.com
   ```

3. Wallet Console → **Benutzer** → Email des Service Accounts mit Rolle **Entwickler** hinzufügen
4. In `.env`:

   ```
   GOOGLE_WALLET_ISSUER_ID=3388000000012345678
   GOOGLE_WALLET_KEY_FILE=../secrets/google-wallet-key.json
   ```

Hinweise:

- Im Demo-Modus können nur Testbenutzer (in der Wallet Console eingetragen) Pässe speichern
- Für alle Benutzer: in der Wallet Console **Publishing-Zugang** beantragen
- Das Logo erscheint nur, wenn die App über HTTPS erreichbar ist (nicht lokal)
- Organisationsrichtlinien können das Erstellen von Service-Account-Schlüsseln verbieten

## 8. Login mit Apple (optional)

Voraussetzung: Mitgliedschaft im [Apple Developer Program](https://developer.apple.com/programs/) (99 USD/Jahr,
dasselbe Konto wie für Apple Wallet). Apple akzeptiert nur HTTPS-Adressen als Callback, kein `localhost`.

1. **App ID** anlegen: [Identifiers](https://developer.apple.com/account/resources/identifiers/list) → **+ → App IDs → App**
   - Beschreibung: `Midnightsports`
   - Bundle ID (Explicit): z.B. `ch.midnightsports.app`
   - Unter **Capabilities** **Sign In with Apple** aktivieren → **Continue → Register**
2. **Services ID** anlegen (das ist die Client-ID für das Web): **Identifiers → + → Services IDs**
   - Beschreibung: `Midnightsports` (erscheint im Anmeldedialog von Apple)
   - Identifier: z.B. `ch.midnightsports.login` → `APPLE_LOGIN_CLIENT_ID`
   - Services ID öffnen → **Sign In with Apple** aktivieren → **Configure**
     - Primary App ID: die App ID aus Schritt 1
     - Domains and Subdomains: Domain der App ohne `https://`, z.B. `midnightsports.example`
     - Return URLs: `https://<domain>/api/auth/apple/callback`
   - **Done → Continue → Save**
3. **Key** erstellen: [Keys](https://developer.apple.com/account/resources/authkeys/list) → **+**
   - Name: `Midnightsports Login`, **Sign In with Apple** aktivieren → **Configure** → Primary App ID aus Schritt 1
   - **Continue → Register → Download**: Datei `AuthKey_<KEY_ID>.p8` nach `secrets/apple-login.p8` kopieren
   - Die Datei kann **nur einmal** heruntergeladen werden. Bei Verlust: neuen Key erstellen, alten widerrufen
   - Key ID (10 Zeichen) notieren → `APPLE_LOGIN_KEY_ID`
4. Team ID: [Membership Details](https://developer.apple.com/account#MembershipDetailsCard) → `APPLE_TEAM_ID`
5. In `.env`:

   ```
   APPLE_TEAM_ID=ABCDE12345
   APPLE_LOGIN_CLIENT_ID=ch.midnightsports.login
   APPLE_LOGIN_KEY_ID=XYZ9876543
   APPLE_LOGIN_KEY_FILE=../secrets/apple-login.p8
   ```

Hinweise:

- Der Pfad ist relativ zum Ordner `api` (`make dev`), bei `make run` `/secrets/apple-login.p8`
- Die App leitet die Callback-URL aus `OAUTH_REDIRECT_URL` ab (gleicher Host, Pfad `/api/auth/apple/callback`).
  Die Return URL in Schritt 2 muss genau dazu passen
- Lokal testen: über einen HTTPS-Tunnel (z.B. `cloudflared tunnel --url http://localhost:8080`), die Tunnel-Adresse
  als Domain und Return URL in der Services ID eintragen und `OAUTH_REDIRECT_URL` auf die Tunnel-Adresse setzen.
  `DEV_LOGIN` funktioniert dann nicht (nur mit localhost)
- Das Client-Secret erzeugt die App bei jeder Anmeldung selbst aus dem `.p8`-Schlüssel, es muss nicht erneuert werden
- Benutzer können ihre Email verbergen («E-Mail-Adresse verbergen»). Die App erhält dann eine Adresse
  `…@privaterelay.appleid.com`. Ein bestehendes Konto (Google oder Gast) mit der echten Email wird dann nicht
  gefunden, es entsteht ein zweites Konto
- Den Namen schickt Apple nur bei der allerersten Anmeldung. Fehlt er, muss ihn die Person im Profil ergänzen
  (das Profil verlangt ihn ohnehin)
