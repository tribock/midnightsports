-- Spesen einer Person im Projekt, optional einem Event zugeordnet (wie time_entries)
CREATE TABLE expenses (
    id         SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date       DATE NOT NULL,
    amount     NUMERIC(8, 2) NOT NULL CHECK (amount > 0),
    reason     TEXT NOT NULL DEFAULT '',
    event_id   INTEGER REFERENCES events (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX expenses_project_user_idx ON expenses (project_id, user_id, date);
