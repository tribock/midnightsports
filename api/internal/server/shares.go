package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// listShares liefert die geteilten Verwarnungen des Projekts mit Partnern (Projekt-Admins).
func (s *Server) listShares(w http.ResponseWriter, r *http.Request, m *Member) error {
	shares, err := loadShares(r.Context(), s.db, m.Project.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, shares)
	return nil
}

func (s *Server) consentShare(w http.ResponseWriter, r *http.Request, m *Member) error {
	return s.answerShare(w, r, m, true)
}

func (s *Server) declineShare(w http.ResponseWriter, r *http.Request, m *Member) error {
	return s.answerShare(w, r, m, false)
}

// answerShare: der Projekt-Admin willigt ein, Verwarnungen und Sperren mit dem anfragenden Projekt zu teilen,
// oder lehnt ab. Mit der letzten Einwilligung wird das anfragende Projekt aktiv. Eine Ablehnung lässt es
// ausstehend, bis der globale Admin die Anfrage entfernt.
func (s *Server) answerShare(w http.ResponseWriter, r *http.Request, m *Member, consent bool) error {
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	const open = `FROM projects p WHERE p.id = sh.project_id AND p.slug = $1 AND sh.partner_id = $2 AND sh.consented_at IS NULL`
	partner := r.PathValue("partner")
	var tag pgconn.CommandTag
	action := actionShareDeclined
	if consent {
		action = actionShareConsented
		tag, err = tx.Exec(ctx,
			`UPDATE project_shares sh SET consented_at = now(), consented_by = $3, declined_at = NULL `+open,
			partner, m.Project.ID, m.ID)
	} else {
		tag, err = tx.Exec(ctx, `UPDATE project_shares sh SET declined_at = now() `+open, partner, m.Project.ID)
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorf(http.StatusNotFound, "Keine offene Anfrage dieses Projekts")
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: action,
		Details: map[string]any{"partner": partner}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

// endShare beendet ein eingewilligtes Teilen (Projekt-Admins beider Seiten). Offene Anfragen entfernt nur
// der globale Admin, damit ein ausstehendes Projekt nicht an der Prüfung vorbei aktiv wird.
func (s *Server) endShare(w http.ResponseWriter, r *http.Request, m *Member) error {
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx,
		`DELETE FROM project_shares sh USING projects q
		 WHERE q.slug = $2 AND sh.consented_at IS NOT NULL
		   AND ((sh.project_id = $1 AND sh.partner_id = q.id) OR (sh.project_id = q.id AND sh.partner_id = $1))`,
		m.Project.ID, r.PathValue("partner"))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionShareRemoved,
		Details: map[string]any{"partner": r.PathValue("partner")}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}
