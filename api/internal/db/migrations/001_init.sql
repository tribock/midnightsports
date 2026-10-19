-- Funktionen im Team mit Lohn pro Einsatz.
-- salary NULL = Abrechnung über die Zeiterfassung (z.B. PL), kein Betrag am Event.
CREATE TABLE functions (
    id     SERIAL PRIMARY KEY,
    name   TEXT NOT NULL UNIQUE,
    salary NUMERIC(8, 2) CHECK (salary >= 0)
);

INSERT INTO functions (name, salary) VALUES
    ('JC', 20),
    ('JSC', 40),
    ('SC', 80),
    ('PL', NULL);

-- Zugriff auf Notfallkontakte, festgelegt durch Admins unter Administration:
-- Personen mit einer dieser Rollen oder Funktionen dürfen Notfallkontakte einsehen.
-- Admins haben immer Zugriff, Teilnehmer nie.
CREATE TABLE emergency_access_roles (
    role TEXT PRIMARY KEY CHECK (role IN ('projektleiter', 'mitarbeiter'))
);

INSERT INTO emergency_access_roles (role) VALUES ('projektleiter');

CREATE TABLE emergency_access_functions (
    function_id INTEGER PRIMARY KEY REFERENCES functions (id) ON DELETE CASCADE
);

-- Gäste (guest = true) werden ohne Login erfasst, ihre Email ist freiwillig.
-- Meldet sich ein Gast später mit derselben Email an, übernimmt er das Konto.
CREATE TABLE users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email              TEXT UNIQUE,
    google_sub         TEXT UNIQUE,
    apple_sub          TEXT UNIQUE,
    guest              BOOLEAN NOT NULL DEFAULT false,
    role               TEXT NOT NULL DEFAULT 'teilnehmer'
                       CHECK (role IN ('admin', 'projektleiter', 'mitarbeiter', 'teilnehmer')),
    function_id        INTEGER REFERENCES functions (id) ON DELETE SET NULL,
    first_name         TEXT NOT NULL DEFAULT '',
    last_name          TEXT NOT NULL DEFAULT '',
    birth_date         DATE,
    gender             TEXT NOT NULL DEFAULT '' CHECK (gender IN ('', 'w', 'm', 'd')),
    status             TEXT NOT NULL DEFAULT '' CHECK (status IN ('', 'schueler_aadorf', 'gast_extern', 'mitarbeiter')),
    street             TEXT NOT NULL DEFAULT '',
    zip                TEXT NOT NULL DEFAULT '',
    city               TEXT NOT NULL DEFAULT '',
    emergency_name     TEXT NOT NULL DEFAULT '',
    emergency_phone    TEXT NOT NULL DEFAULT '',
    emergency_relation TEXT NOT NULL DEFAULT '' CHECK (emergency_relation IN ('', 'vater', 'mutter', 'andere')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash BYTEA PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE locations (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    street     TEXT NOT NULL DEFAULT '',
    zip        TEXT NOT NULL DEFAULT '',
    city       TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false
);

-- Strassen sind noch zu ergänzen (Admin → Orte)
INSERT INTO locations (name, zip, city, is_default) VALUES
    ('Turnhalle Guntershausen', '8357', 'Guntershausen bei Aadorf', true),
    ('Turnhalle Aadorf', '8355', 'Aadorf', false),
    ('Turnhalle Ettenhausen', '8356', 'Ettenhausen', false);

CREATE TABLE events (
    id          SERIAL PRIMARY KEY,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    location_id INTEGER NOT NULL REFERENCES locations (id),
    leader_id   UUID REFERENCES users (id) ON DELETE SET NULL,
    cancelled   BOOLEAN NOT NULL DEFAULT false,
    created_by  UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);

CREATE INDEX events_starts_at_idx ON events (starts_at);

-- Einsatz einer Person an einem Event. Funktion und Betrag werden beim Einteilen festgehalten,
-- spätere Änderungen an der Funktion verändern vergangene Events nicht.
-- Auszahlung: die Person bestätigt den Empfang ihres Betrags mit ihrer Unterschrift (PNG).
CREATE TABLE event_staff (
    event_id    INTEGER NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    function_id INTEGER NOT NULL REFERENCES functions (id) ON DELETE RESTRICT,
    salary      NUMERIC(8, 2) CHECK (salary >= 0),
    signature   BYTEA,
    signed_at   TIMESTAMPTZ,
    signed_by   UUID REFERENCES users (id) ON DELETE SET NULL,
    PRIMARY KEY (event_id, user_id),
    CHECK ((signature IS NULL) = (signed_at IS NULL))
);

CREATE TABLE registrations (
    event_id   INTEGER NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)
);

-- user_id ist NULL bei gelöschten Konten (anonymisiert für die Statistik)
CREATE TABLE checkins (
    id            SERIAL PRIMARY KEY,
    event_id      INTEGER NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id       UUID REFERENCES users (id) ON DELETE SET NULL,
    checked_in_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    checked_in_by UUID REFERENCES users (id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX checkins_event_user_idx ON checkins (event_id, user_id) WHERE user_id IS NOT NULL;
-- Anzahl und Liste der Check-ins pro Event (Eventliste, Teilnehmer, Bericht).
-- Der Unique-Index oben gilt nur für Zeilen mit user_id und wird dafür nicht verwendet.
CREATE INDEX checkins_event_idx ON checkins (event_id);

CREATE TABLE reports (
    event_id   INTEGER PRIMARY KEY REFERENCES events (id) ON DELETE CASCADE,
    body       TEXT NOT NULL DEFAULT '',
    author_id  UUID REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE time_entries (
    id          SERIAL PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date        DATE NOT NULL,
    hours       NUMERIC(5, 2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    description TEXT NOT NULL DEFAULT '',
    event_id    INTEGER REFERENCES events (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX time_entries_user_idx ON time_entries (user_id, date);
