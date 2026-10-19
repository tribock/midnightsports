package server

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Fehlercodes von PostgreSQL
const (
	pgForeignKeyViolation = "23503"
	pgUniqueViolation     = "23505"
)

// execOne führt eine Änderung aus. Betrifft sie keinen Datensatz, wird errNotFound geliefert.
func (s *Server) execOne(ctx context.Context, sql string, args ...any) error {
	tag, err := s.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}

// loadUser lädt einen Benutzer über seine ID. Ungültige IDs gelten als nicht gefunden.
func (s *Server) loadUser(ctx context.Context, id string) (*User, error) {
	if !validUUID(id) {
		return nil, pgx.ErrNoRows
	}
	return scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id = $1`, id))
}

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func isUniqueViolation(err error) bool { return pgErrorCode(err) == pgUniqueViolation }

func isForeignKeyViolation(err error) bool { return pgErrorCode(err) == pgForeignKeyViolation }
