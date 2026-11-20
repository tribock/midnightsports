package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// Seitenweise Listen (Teilnehmer, Benutzer): ?offset=50&limit=50.
const (
	defaultPageSize = 50
	maxPageSize     = 100
)

type pageRequest struct {
	offset, limit int
}

// readPage liest offset und limit aus der URL. Ungültige oder fehlende Werte ergeben die erste Seite mit 50 Einträgen,
// mehr als 100 pro Seite gibt es nicht.
func readPage(r *http.Request) pageRequest {
	p := pageRequest{limit: defaultPageSize}
	if n, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && n > 0 {
		p.offset = n
	}
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 {
		p.limit = min(n, maxPageSize)
	}
	return p
}

// sql liefert LIMIT und OFFSET mit den Platzhaltern $first und $first+1, args die passenden Werte.
func (p pageRequest) sql(first int) string {
	return fmt.Sprintf(" LIMIT $%d OFFSET $%d", first, first+1)
}

func (p pageRequest) args() []any { return []any{p.limit, p.offset} }

// pageTotal: Gesamtzahl aller Treffer als letzte Spalte, berechnet vor LIMIT und OFFSET.
const pageTotal = `count(*) OVER ()`

// Page ist eine Seite einer Liste mit der Gesamtzahl aller Treffer.
type Page[T any] struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Items  []T `json:"items"`
}

// collectPage liest die Zeilen einer Abfrage mit pageTotal als letzter Spalte. fields nennt die Ziele der übrigen
// Spalten in ihrer Reihenfolge. Hinter der letzten Seite ist die Liste leer und total 0.
func collectPage[T any](rows pgx.Rows, p pageRequest, fields func(*T) []any) (Page[T], error) {
	defer rows.Close()
	page := Page[T]{Offset: p.offset, Limit: p.limit, Items: []T{}}
	for rows.Next() {
		var item T
		if err := rows.Scan(append(fields(&item), &page.Total)...); err != nil {
			return page, err
		}
		page.Items = append(page.Items, item)
	}
	return page, rows.Err()
}
