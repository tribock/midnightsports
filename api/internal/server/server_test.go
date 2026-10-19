package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		msg    string
	}{
		{"API-Fehler", errForbidden, http.StatusForbidden, "Keine Berechtigung"},
		{"nicht gefunden in der Datenbank", fmt.Errorf("query: %w", pgx.ErrNoRows), http.StatusNotFound, "Nicht gefunden"},
		{"unbekannter Fehler", errors.New("boom"), http.StatusInternalServerError, "Interner Fehler"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, tt.err)
			var body map[string]string
			json.NewDecoder(rec.Body).Decode(&body)
			if rec.Code != tt.status || body["error"] != tt.msg {
				t.Errorf("got %d %q, want %d %q", rec.Code, body["error"], tt.status, tt.msg)
			}
		})
	}
}

func TestPgErrors(t *testing.T) {
	fk := fmt.Errorf("insert: %w", &pgconn.PgError{Code: pgForeignKeyViolation})
	unique := &pgconn.PgError{Code: pgUniqueViolation}

	if !isForeignKeyViolation(fk) || isUniqueViolation(fk) {
		t.Error("Fremdschlüssel-Fehler falsch erkannt")
	}
	if !isUniqueViolation(unique) || isForeignKeyViolation(unique) {
		t.Error("Unique-Fehler falsch erkannt")
	}
	if isForeignKeyViolation(nil) || isUniqueViolation(errors.New("x")) {
		t.Error("andere Fehler dürfen nicht erkannt werden")
	}
}

func TestValidUUID(t *testing.T) {
	valid := []string{"0b6f8a2e-1c3d-4e5f-8a9b-0c1d2e3f4a5b", "0B6F8A2E-1C3D-4E5F-8A9B-0C1D2E3F4A5B"}
	invalid := []string{"", "123", "0b6f8a2e1c3d4e5f8a9b0c1d2e3f4a5b", "0b6f8a2e-1c3d-4e5f-8a9b-0c1d2e3f4a5g", "0b6f8a2e-1c3d-4e5f-8a9b_0c1d2e3f4a5b"}
	for _, s := range valid {
		if !validUUID(s) {
			t.Errorf("%q abgelehnt", s)
		}
	}
	for _, s := range invalid {
		if validUUID(s) {
			t.Errorf("%q akzeptiert", s)
		}
	}
}
