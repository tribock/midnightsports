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

func (s *Server) listLocations(w http.ResponseWriter, r *http.Request, m *Member) error {
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, street, zip, city, is_default FROM locations WHERE project_id = $1 ORDER BY is_default DESC, name`,
		m.Project.ID)
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

func (s *Server) createLocation(w http.ResponseWriter, r *http.Request, m *Member) error {
	return s.saveLocation(w, r, m, 0)
}

func (s *Server) updateLocation(w http.ResponseWriter, r *http.Request, m *Member) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	return s.saveLocation(w, r, m, id)
}

// saveLocation legt einen Ort des Projekts an (id = 0) oder aktualisiert ihn. Es gibt pro Projekt genau einen Standardort.
func (s *Server) saveLocation(w http.ResponseWriter, r *http.Request, m *Member, id int) error {
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
		if _, err := tx.Exec(ctx, `UPDATE locations SET is_default = false WHERE project_id = $1 AND id <> $2`,
			m.Project.ID, id); err != nil {
			return err
		}
	}
	if id == 0 {
		// Der erste Ort eines Projekts ist immer der Standardort
		err = tx.QueryRow(ctx,
			`INSERT INTO locations (project_id, name, street, zip, city, is_default)
			 VALUES ($1, $2, $3, $4, $5, $6 OR NOT EXISTS (SELECT 1 FROM locations WHERE project_id = $1))
			 RETURNING id, is_default`,
			m.Project.ID, l.Name, l.Street, l.Zip, l.City, l.IsDefault).Scan(&l.ID, &l.IsDefault)
	} else {
		// Der Standardort kann nur durch Wahl eines anderen Standardorts geändert werden
		err = tx.QueryRow(ctx,
			`UPDATE locations SET name = $3, street = $4, zip = $5, city = $6, is_default = is_default OR $7
			 WHERE id = $1 AND project_id = $2 RETURNING id, is_default`,
			id, m.Project.ID, l.Name, l.Street, l.Zip, l.City, l.IsDefault).Scan(&l.ID, &l.IsDefault)
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
