# Architektur

Überblick über den Aufbau der Applikation: Systemübersicht, Anwendungsfälle, Datenmodell und die wichtigsten Abläufe.
Die Diagramme sind in [Mermaid](https://mermaid.js.org) geschrieben und werden auf GitHub und in VS Code (Markdown-Vorschau) direkt dargestellt.

## Systemübersicht

```mermaid
flowchart LR
    subgraph Browser
        SPA["React-App, Vite (web/)"]
    end
    subgraph Server["Go-Server, Cloud Run (api/)"]
        Static["Frontend ausliefern<br/>(gzip, Cache)"]
        API["REST-API /api/*<br/>Sessions und Rollen"]
    end
    DB[("PostgreSQL")]
    Google["Google OAuth"]
    Wallet["Apple / Google Wallet"]

    SPA -- "HTTPS, Session-Cookie" --> API
    SPA -- "index.html, /assets/*" --> Static
    API -- "pgx" --> DB
    API -- "Login" --> Google
    API -- "QR-Pass" --> Wallet
```

- Ein einziger Go-Prozess liefert das gebaute Frontend aus und stellt die API bereit.
- Die Datenbank wird beim Start migriert (`api/internal/db/migrations`).
- Angemeldet wird über Google. Die Session liegt in einem HttpOnly-Cookie, in der Datenbank wird nur ein HMAC des Tokens gespeichert.
- Rollen werden pro Route im Server geprüft (`route(...)` in `server.go`), das Frontend blendet nur aus, was nicht erlaubt ist.

## Anwendungsfälle

Jede Rolle kann alles, was die Rolle darunter kann (Teilnehmer < Mitarbeiter < Projektleiter < Admin).
Pro Rolle sind nur die Anwendungsfälle aufgeführt, die neu dazukommen.

```mermaid
flowchart LR
    subgraph RT["Teilnehmer"]
        T(["👤 Teilnehmer"])
        T1([Mit Google anmelden])
        T2([Profil pflegen, Konto löschen])
        T3([Events ansehen, an- und abmelden])
        T4([QR-Code / Wallet-Pass anzeigen])
        T --- T1 & T2 & T3 & T4
    end

    subgraph RM["Mitarbeiter"]
        MA(["👤 Mitarbeiter<br/>+ alles von Teilnehmer"])
        M1([Teilnehmerliste und Einsatz ansehen])
        M2([QR-Code scannen: Check-in])
        M3([Teilnehmer manuell oder als Gast erfassen])
        MA --- M1 & M2 & M3
    end

    subgraph RP["Projektleiter"]
        PL(["👤 Projektleiter<br/>+ alles von Mitarbeiter"])
        P0([Events erstellen, bearbeiten, absagen])
        P1([Team einteilen])
        P2([Auszahlung mit Unterschrift, Spesenblatt drucken])
        P3([Bericht schreiben und drucken])
        P4([Zeiterfassung])
        P5([Notfallkontakte einsehen ¹])
        PL --- P0 & P1 & P2 & P3 & P4 & P5
    end

    subgraph RA["Admin"]
        A(["👤 Admin<br/>+ alles von Projektleiter"])
        A1([Benutzer verwalten: Rolle, Profil, löschen])
        A2([Funktionen und Löhne festlegen])
        A3([Zugriff auf Notfallkontakte festlegen])
        A4([Veranstaltungsorte verwalten])
        A5([Zeiteinträge als CSV exportieren])
        A --- A1 & A2 & A3 & A4 & A5
    end
```

¹ Standard: Projektleiter. Admins legen unter Administration fest, welche weiteren Rollen und Funktionen (z.B. SC) Zugriff haben. Admins haben immer Zugriff, Teilnehmer nie.

## Datenmodell

```mermaid
erDiagram
    users ||--o{ sessions : "hat"
    functions |o--o{ users : "Funktion im Team"
    functions ||--o| emergency_access_functions : "Zugriff auf Notfallkontakte"
    locations ||--o{ events : "findet statt in"
    users |o--o{ events : "verantwortlich (leader_id)"
    events ||--o{ event_staff : "Einsatz"
    users ||--o{ event_staff : "eingeteilt"
    functions ||--o{ event_staff : "Funktion beim Einsatz"
    events ||--o{ registrations : "Anmeldungen"
    users ||--o{ registrations : "meldet sich an"
    events ||--o{ checkins : "vor Ort"
    users |o--o{ checkins : "eingecheckt (NULL = gelöscht)"
    events ||--o| reports : "Bericht"
    users ||--o{ time_entries : "erfasst"
    events |o--o{ time_entries : "optional zu Event"

    users {
        uuid id PK
        text email UK "NULL bei Gästen ohne Email"
        text google_sub UK
        text apple_sub UK
        bool guest
        text role "admin, projektleiter, mitarbeiter, teilnehmer"
        int function_id FK
        text first_name
        text last_name
        date birth_date
        text gender "w, m, d"
        text status "schueler_aadorf, gast_extern, mitarbeiter"
        text zip
        text city
        text emergency_name
        text emergency_phone
        text emergency_relation "vater, mutter, andere"
    }
    sessions {
        bytea token_hash PK "HMAC des Tokens"
        uuid user_id FK
        timestamptz expires_at
    }
    functions {
        int id PK
        text name UK "JC, JSC, SC, PL"
        numeric salary "NULL = über Zeiterfassung"
    }
    emergency_access_roles {
        text role PK "projektleiter, mitarbeiter"
    }
    emergency_access_functions {
        int function_id PK, FK
    }
    locations {
        int id PK
        text name
        text street
        text zip
        text city
        bool is_default "genau ein Standardort"
    }
    events {
        int id PK
        timestamptz starts_at
        timestamptz ends_at
        int location_id FK
        uuid leader_id FK
        bool cancelled
        uuid created_by FK
    }
    event_staff {
        int event_id PK, FK
        uuid user_id PK, FK
        int function_id FK "beim Einteilen festgehalten"
        numeric salary "beim Einteilen festgehalten"
        bytea signature "PNG, Empfang bestätigt"
        timestamptz signed_at
        uuid signed_by FK
    }
    registrations {
        int event_id PK, FK
        uuid user_id PK, FK
        timestamptz created_at
    }
    checkins {
        int id PK
        int event_id FK
        uuid user_id FK
        timestamptz checked_in_at
        uuid checked_in_by FK
    }
    reports {
        int event_id PK, FK
        text body
        uuid author_id FK
        timestamptz updated_at
    }
    time_entries {
        int id PK
        uuid user_id FK
        date date
        numeric hours
        text description
        int event_id FK
    }
```

- **Gäste** sind normale Benutzer mit `guest = true`. Meldet sich jemand mit derselben Email über Google an, übernimmt er das Konto.
- **Gelöschte Konten**: Check-ins bleiben mit `user_id = NULL` für die Statistik erhalten, alles andere wird gelöscht.
- **Einsatz** (`event_staff`): Funktion und Betrag werden beim Einteilen kopiert, damit spätere Lohnänderungen vergangene Events nicht verändern. Die Unterschrift der Auszahlung gehört zur selben Zeile.
- **Admins** haben immer Zugriff auf Notfallkontakte und stehen deshalb nicht in `emergency_access_roles`.

## Abläufe

### Anmeldung mit Google

```mermaid
sequenceDiagram
    actor U as Benutzer
    participant B as Browser (React)
    participant S as Go-Server
    participant G as Google
    participant DB as PostgreSQL

    U->>B: "Mit Google anmelden"
    B->>S: GET /api/auth/login
    S-->>B: Redirect zu Google + State-Cookie
    B->>G: Anmeldung, Konto wählen
    G-->>B: Redirect /api/auth/callback?code&state
    B->>S: GET /api/auth/callback
    S->>S: State prüfen
    S->>G: Code gegen Token tauschen, Userinfo laden
    G-->>S: sub, email, Name (Email verifiziert)
    S->>DB: Benutzer über google_sub oder Email suchen, sonst anlegen
    Note over S,DB: Gast mit derselben Email wird übernommen.<br/>Emails aus ADMIN_EMAIL werden Admin.
    S->>DB: Session speichern (HMAC des Tokens)
    S-->>B: Session-Cookie, Redirect /
    B->>S: GET /api/me
    S-->>B: Benutzer mit Rolle
    alt Profil unvollständig
        B->>U: Profilseite
    else
        B->>U: Events
    end
```

Login mit Apple (optional) läuft gleich ab, mit diesen Unterschieden:

- Apple schickt das Resultat per `POST /api/auth/apple/callback` (`form_post`) von `appleid.apple.com`.
  Die Route ist deshalb vom CSRF-Schutz ausgenommen, State- und Nonce-Cookie sind `SameSite=None`
- Statt PKCE bindet eine Nonce im `id_token` den Code an den Browser
- Das Client-Secret ist ein JWT, das der Server bei jeder Anmeldung mit dem `.p8`-Schlüssel signiert
- Statt Userinfo liefert das `id_token` vom Token-Endpunkt `sub` und Email, der Name kommt nur bei der ersten Anmeldung
- Gesucht wird über `apple_sub` oder die Email. Gleiche Email wie bei Google: dasselbe Konto

### Check-in per QR-Code

```mermaid
sequenceDiagram
    actor T as Teilnehmer
    actor M as Mitarbeiter
    participant B as Browser (Scan-Seite)
    participant S as Go-Server
    participant DB as PostgreSQL

    T->>M: zeigt QR-Code (App oder Wallet)
    M->>B: Kamera scannt Code
    B->>S: POST /api/events/{id}/checkins {token}
    S->>S: Rolle prüfen (Mitarbeiter, Projektleiter, Admin)
    S->>S: Signatur des Tokens prüfen → Benutzer-ID
    S->>DB: Event abgesagt?
    S->>DB: Benutzer laden (inkl. Zugriff auf Notfallkontakte)
    S->>DB: INSERT Check-in ON CONFLICT DO NOTHING<br/>RETURNING id, Zeit, angemeldet?
    alt bereits eingecheckt
        S->>DB: bestehenden Check-in lesen
    end
    S-->>B: Person, angemeldet, bereits eingecheckt
    Note over S,B: Notfallkontakt nur mit Zugriff
    B->>M: grün = eingecheckt, gelb = Hinweis
```

### Auszahlung mit Unterschrift

```mermaid
sequenceDiagram
    actor PL as Projektleiter
    actor C as Coach
    participant B as Browser (Auszahlung)
    participant S as Go-Server
    participant DB as PostgreSQL

    PL->>B: Event → "Auszahlung"
    B->>S: GET /api/events/{id}/payout
    S->>DB: Einsatz mit Betrag und Unterschrift
    S-->>B: Liste
    PL->>C: übergibt Betrag und Gerät
    C->>B: unterschreibt im Feld, "Empfang bestätigen"
    B->>S: PUT /api/events/{id}/payout/{userId} {PNG}
    S->>S: PNG prüfen (Format, max. 512 KB)
    S->>DB: UPDATE … WHERE signature IS NULL
    alt bereits unterschrieben
        S-->>B: 409 Empfang bereits bestätigt
    else
        S-->>B: 204
    end
    PL->>B: "Drucken"
    B->>PL: Spesenblatt (Druckansicht, als PDF speicherbar)
```

## Leistung

Bei der Optimierung berücksichtigte Punkte, damit sie bei Änderungen erhalten bleiben:

- **IDs direkt vergleichen**: Benutzer-IDs werden in SQL mit der uuid-Spalte verglichen (`id = $1`), nie als Text (`id::text = $1`), sonst kann Postgres den Index nicht nutzen. Ungültige IDs fängt `validUUID` vorher ab.
- **Index `checkins (event_id)`**: wird von Eventliste, Teilnehmerliste und Bericht verwendet. Der Unique-Index auf `(event_id, user_id)` gilt nur für Zeilen mit Benutzer und hilft hier nicht.
- **Einsatz speichern**: alle gewählten Personen in einer Abfrage laden, die Einträge als Batch schreiben.
- **Frontend**: Dateien in `/assets/` werden einmal komprimiert (gzip) und im Speicher gehalten, der Browser speichert sie dauerhaft. Die QR-Seiten (Scanner, QR-Code) werden erst bei Bedarf geladen.
