package server

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Location struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Street    string `json:"street"`
	Zip       string `json:"zip"`
	City      string `json:"city"`
	IsDefault bool   `json:"isDefault"`
}

func (s *Server) listLocations(w http.ResponseWriter, r *http.Request, u *User) error {
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, street, zip, city, is_default FROM locations ORDER BY is_default DESC, name`)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Location])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

func (s *Server) createLocation(w http.ResponseWriter, r *http.Request, u *User) error {
	return s.saveLocation(w, r, 0)
}

func (s *Server) updateLocation(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	return s.saveLocation(w, r, id)
}

// saveLocation legt einen Ort an (id = 0) oder aktualisiert ihn. Es gibt immer genau einen Standardort.
func (s *Server) saveLocation(w http.ResponseWriter, r *http.Request, id int) error {
	var l Location
	if err := decode(r, &l); err != nil {
		return err
	}
	l.Name = strings.TrimSpace(l.Name)
	if l.Name == "" {
		return errorf(http.StatusBadRequest, "Name fehlt")
	}

	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if l.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE locations SET is_default = false WHERE id <> $1`, id); err != nil {
			return err
		}
	}
	if id == 0 {
		err = tx.QueryRow(ctx,
			`INSERT INTO locations (name, street, zip, city, is_default) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			l.Name, l.Street, l.Zip, l.City, l.IsDefault).Scan(&l.ID)
	} else {
		// Der Standardort kann nur durch Wahl eines anderen Standardorts geändert werden
		err = tx.QueryRow(ctx,
			`UPDATE locations SET name = $2, street = $3, zip = $4, city = $5, is_default = is_default OR $6
			 WHERE id = $1 RETURNING id, is_default`,
			id, l.Name, l.Street, l.Zip, l.City, l.IsDefault).Scan(&l.ID, &l.IsDefault)
	}
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, l)
	return nil
}
