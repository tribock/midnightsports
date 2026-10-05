CREATE TABLE users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email              TEXT NOT NULL UNIQUE,
    google_sub         TEXT UNIQUE,
    role               TEXT NOT NULL DEFAULT 'teilnehmer'
                       CHECK (role IN ('admin', 'projektleiter', 'teilnehmer')),
    first_name         TEXT NOT NULL DEFAULT '',
    last_name          TEXT NOT NULL DEFAULT '',
    birth_date         DATE,
    street             TEXT NOT NULL DEFAULT '',
    zip                TEXT NOT NULL DEFAULT '',
    city               TEXT NOT NULL DEFAULT '',
    emergency_name     TEXT NOT NULL DEFAULT '',
    emergency_phone    TEXT NOT NULL DEFAULT '',
    emergency_relation TEXT NOT NULL DEFAULT '',
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
    cancelled   BOOLEAN NOT NULL DEFAULT false,
    created_by  UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);

CREATE INDEX events_starts_at_idx ON events (starts_at);

CREATE TABLE event_leaders (
    event_id INTEGER NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, user_id)
);

CREATE TABLE registrations (
    event_id   INTEGER NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)
);

-- user_id ist NULL bei Gästen und bei gelöschten Konten (anonymisiert)
CREATE TABLE checkins (
    id               SERIAL PRIMARY KEY,
    event_id         INTEGER NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id          UUID REFERENCES users (id) ON DELETE SET NULL,
    guest_first_name TEXT NOT NULL DEFAULT '',
    guest_last_name  TEXT NOT NULL DEFAULT '',
    checked_in_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    checked_in_by    UUID REFERENCES users (id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX checkins_event_user_idx ON checkins (event_id, user_id) WHERE user_id IS NOT NULL;

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
