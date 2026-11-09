-- Mandantenfähiges Schema (Branch feat/multitenancy): jedes Projekt (z.B. Midnightsports Aadorf, Pleno)
-- hat eigene Events, Orte, Funktionen und Mitglieder. Personen (users) sind global.
-- Die Übernahme der Daten aus der bisherigen Produktion wird erst nach dem Merge geplant.

-- Gäste (guest = true) werden ohne Login erfasst, ihre Email ist freiwillig.
-- Meldet sich ein Gast später mit derselben Email an, übernimmt er das Konto.
-- global_admin: Emails aus ADMIN_EMAIL, verwalten Projekte, Benutzer und das Audit-Protokoll.
CREATE TABLE users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email              TEXT UNIQUE,
    google_sub         TEXT UNIQUE,
    apple_sub          TEXT UNIQUE,
    guest              BOOLEAN NOT NULL DEFAULT false,
    global_admin       BOOLEAN NOT NULL DEFAULT false,
    first_name         TEXT NOT NULL DEFAULT '',
    last_name          TEXT NOT NULL DEFAULT '',
    birth_date         DATE,
    gender             TEXT NOT NULL DEFAULT '' CHECK (gender IN ('', 'w', 'm', 'd')),
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

-- Projekt (Mandant). slug erscheint in der URL (/p/aadorf, /?projekt=aadorf).
-- region: Einzugsgebiet für den Status "Schüler/in aus <region>" und die Suche.
-- logo: hochgeladenes Logo (PNG, JPEG, WebP), NULL = Standardlogo.
-- Fristen für Verwarnungen und Sperren (NULL = ohne Frist):
--   ban_days: nach so vielen Tagen wird eine Sperre aufgehoben (ban_action 'entsperren')
--             oder in eine Verwarnung umgewandelt (ban_action 'verwarnen')
--   warning_days: nach so vielen Tagen wird eine Verwarnung gelöscht
CREATE TABLE projects (
    id           SERIAL PRIMARY KEY,
    slug         TEXT NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    name         TEXT NOT NULL,
    region       TEXT NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',
    logo         BYTEA,
    logo_type    TEXT,
    ban_days     INTEGER CHECK (ban_days > 0),
    ban_action   TEXT NOT NULL DEFAULT 'entsperren' CHECK (ban_action IN ('entsperren', 'verwarnen')),
    warning_days INTEGER CHECK (warning_days > 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((logo IS NULL) = (logo_type IS NULL))
);

-- Gemeinsame Verwarnungen und Sperren zweier Projekte. Der globale Admin legt die Anfrage beim Erstellen
-- von project_id an, der Projekt-Admin von partner_id willigt ein. Bis dahin ist project_id ausstehend
-- (nur für globale Admins nutzbar). Abgelehnte Anfragen bleiben ausstehend, bis der globale Admin sie entfernt.
CREATE TABLE project_shares (
    project_id   INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    partner_id   INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    consented_at TIMESTAMPTZ,
    consented_by UUID REFERENCES users (id) ON DELETE SET NULL,
    declined_at  TIMESTAMPTZ,
    PRIMARY KEY (project_id, partner_id),
    CHECK (project_id <> partner_id),
    CHECK (consented_at IS NULL OR declined_at IS NULL)
);

-- Funktionen im Team mit Lohn pro Einsatz, pro Projekt (Bezeichnungen legt der globale Admin beim Erstellen fest).
-- salary NULL = Abrechnung über die Zeiterfassung (z.B. PL), kein Betrag am Event.
CREATE TABLE functions (
    id         SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    salary     NUMERIC(8, 2) CHECK (salary >= 0),
    UNIQUE (project_id, name),
    -- für den zusammengesetzten Fremdschlüssel der Mitglieder (Funktion aus demselben Projekt)
    UNIQUE (project_id, id)
);

-- Mitgliedschaft einer Person in einem Projekt. Rollen: Teilnehmer < Mitarbeiter < Projektleiter < Admin (Projekt-Admin).
-- status: Herkunft im Projekt, gewählt beim Anmelden. Mitarbeiter vergeben nur Projekt-Admins.
CREATE TABLE project_members (
    project_id  INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'teilnehmer'
                CHECK (role IN ('admin', 'projektleiter', 'mitarbeiter', 'teilnehmer')),
    function_id INTEGER,
    status      TEXT NOT NULL DEFAULT '' CHECK (status IN ('', 'schueler', 'gast_extern', 'mitarbeiter')),
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id),
    FOREIGN KEY (project_id, function_id) REFERENCES functions (project_id, id) ON DELETE SET NULL (function_id)
);

CREATE INDEX project_members_user_idx ON project_members (user_id);

-- Zugriff auf Notfallkontakte pro Projekt, festgelegt durch Projekt-Admins unter Administration:
-- Personen mit einer dieser Rollen oder Funktionen dürfen Notfallkontakte der Mitglieder einsehen.
-- Projekt-Admins und globale Admins haben immer Zugriff, Teilnehmer nie.
CREATE TABLE emergency_access_roles (
    project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('projektleiter', 'mitarbeiter')),
    PRIMARY KEY (project_id, role)
);

CREATE TABLE emergency_access_functions (
    function_id INTEGER PRIMARY KEY REFERENCES functions (id) ON DELETE CASCADE
);

CREATE TABLE locations (
    id         SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    street     TEXT NOT NULL DEFAULT '',
    zip        TEXT NOT NULL DEFAULT '',
    city       TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (project_id, id)
);

CREATE TABLE events (
    id          SERIAL PRIMARY KEY,
    project_id  INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    location_id INTEGER NOT NULL,
    leader_id   UUID REFERENCES users (id) ON DELETE SET NULL,
    cancelled   BOOLEAN NOT NULL DEFAULT false,
    created_by  UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at),
    FOREIGN KEY (project_id, location_id) REFERENCES locations (project_id, id)
);

CREATE INDEX events_project_starts_at_idx ON events (project_id, starts_at);

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
    project_id  INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date        DATE NOT NULL,
    hours       NUMERIC(5, 2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    description TEXT NOT NULL DEFAULT '',
    event_id    INTEGER REFERENCES events (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX time_entries_project_user_idx ON time_entries (project_id, user_id, date);

-- Verwarnung oder Sperre einer Person in einem Projekt (höchstens eine pro Projekt, eine Sperre ersetzt
-- eine Verwarnung). Sichtbar im auslösenden Projekt und in Projekten mit eingewilligtem Teilen (project_shares).
-- Ob sie nach der Frist noch gilt, berechnet die App aus den Fristen des auslösenden Projekts.
CREATE TABLE sanctions (
    project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    level      TEXT NOT NULL CHECK (level IN ('verwarnung', 'sperre')),
    reason     TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX sanctions_user_idx ON sanctions (user_id);

-- Audit-Protokoll für Vorgänge mit Personendaten (Minderjährige, revDSG / DSGVO): wer hat wann was mit
-- wessen Daten gemacht. Ohne Fremdschlüssel auf users, damit Einträge das Löschen einer Person überdauern.
-- Beim Löschen einer Person werden ihre Namen hier durch "Gelöschtes Konto" ersetzt (Löschpflicht),
-- die IDs bleiben als Pseudonym für die Nachvollziehbarkeit.
CREATE TABLE audit_log (
    id             BIGSERIAL PRIMARY KEY,
    at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id       UUID,
    actor_name     TEXT NOT NULL DEFAULT '',
    project_id     INTEGER REFERENCES projects (id) ON DELETE SET NULL,
    action         TEXT NOT NULL,
    target_user_id UUID,
    target_name    TEXT NOT NULL DEFAULT '',
    details        JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX audit_log_target_idx ON audit_log (target_user_id);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_id);
CREATE INDEX audit_log_project_idx ON audit_log (project_id, id);

-- Erstes Projekt: Midnightsports Aadorf mit den bisherigen Funktionen, Orten und dem Zugriff auf Notfallkontakte
INSERT INTO projects (slug, name, region, description) VALUES
    ('aadorf', 'Midnightsports Aadorf', 'Aadorf',
     'Offene Sportnächte für Jugendliche in den Turnhallen Guntershausen, Aadorf und Ettenhausen');

INSERT INTO functions (project_id, name, salary)
SELECT p.id, f.name, f.salary
FROM projects p, (VALUES ('JC', 20), ('JSC', 40), ('SC', 80), ('PL', NULL)) AS f (name, salary)
WHERE p.slug = 'aadorf';

INSERT INTO emergency_access_roles (project_id, role)
SELECT id, 'projektleiter' FROM projects WHERE slug = 'aadorf';

-- Strassen sind noch zu ergänzen (Administration → Orte)
INSERT INTO locations (project_id, name, zip, city, is_default)
SELECT p.id, l.name, l.zip, l.city, l.is_default
FROM projects p, (VALUES
    ('Turnhalle Guntershausen', '8357', 'Guntershausen bei Aadorf', true),
    ('Turnhalle Aadorf', '8355', 'Aadorf', false),
    ('Turnhalle Ettenhausen', '8356', 'Ettenhausen', false)) AS l (name, zip, city, is_default)
WHERE p.slug = 'aadorf';
