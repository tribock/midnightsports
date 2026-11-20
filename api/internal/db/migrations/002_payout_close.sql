-- Abschluss der Auszahlung: die Projektleitung unterschreibt (PNG), erst danach wird das Spesenblatt gedruckt.
-- Bestehende Events bleiben offen (alle Spalten NULL).
ALTER TABLE events
    ADD COLUMN payout_signature BYTEA,
    ADD COLUMN payout_closed_at TIMESTAMPTZ,
    ADD COLUMN payout_closed_by UUID REFERENCES users (id) ON DELETE SET NULL,
    ADD CONSTRAINT events_payout_closed_check CHECK ((payout_signature IS NULL) = (payout_closed_at IS NULL));
