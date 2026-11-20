package server

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	LevelWarning = "verwarnung"
	LevelBan     = "sperre"

	BanActionUnban = "entsperren"
	BanActionWarn  = "verwarnen"
)

// sanctionRules sind die Fristen eines Projekts für Verwarnungen und Sperren (nil = ohne Frist).
type sanctionRules struct {
	// BanDays: nach so vielen Tagen wird eine Sperre aufgehoben oder in eine Verwarnung umgewandelt (BanAction)
	BanDays   *int   `json:"banDays"`
	BanAction string `json:"banAction"`
	// WarningDays: nach so vielen Tagen wird eine Verwarnung gelöscht
	WarningDays *int `json:"warningDays"`
}

func (r sanctionRules) validate() error {
	if r.BanAction != BanActionUnban && r.BanAction != BanActionWarn {
		return errorf(http.StatusBadRequest, "Ungültige Aktion nach Ablauf der Sperre")
	}
	for _, d := range []*int{r.BanDays, r.WarningDays} {
		if d != nil && (*d < 1 || *d > 3650) {
			return errorf(http.StatusBadRequest, "Fristen müssen zwischen 1 und 3650 Tagen liegen")
		}
	}
	return nil
}

func days(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }

// effectiveSanction berechnet, was von einer Verwarnung oder Sperre (erfasst am since) heute noch gilt.
// Eine abgelaufene Sperre wird je nach Projekt aufgehoben oder zur Verwarnung, die ab dem Ablauf der Sperre
// wieder ihre eigene Frist hat. level "" bedeutet: gilt nicht mehr. until ist das Ende der aktuellen Stufe
// (nil = unbefristet).
func effectiveSanction(level string, since time.Time, r sanctionRules, now time.Time) (string, *time.Time) {
	if level == LevelBan {
		if r.BanDays == nil {
			return LevelBan, nil
		}
		end := since.Add(days(*r.BanDays))
		if now.Before(end) {
			return LevelBan, &end
		}
		if r.BanAction != BanActionWarn {
			return "", nil
		}
		level, since = LevelWarning, end
	}
	if r.WarningDays == nil {
		return LevelWarning, nil
	}
	end := since.Add(days(*r.WarningDays))
	if now.Before(end) {
		return LevelWarning, &end
	}
	return "", nil
}

// Sanction ist eine geltende Verwarnung oder Sperre, wie sie ein Projekt sieht.
type Sanction struct {
	Level       string     `json:"level"`
	Reason      string     `json:"reason"`
	Since       time.Time  `json:"since"`
	Until       *time.Time `json:"until"`
	ProjectName string     `json:"projectName"`
	// Own: im eigenen Projekt erfasst (änderbar), sonst von einem Partnerprojekt geteilt (nur lesen)
	Own bool `json:"own"`
}

// visibleSanctions lädt die geltenden Verwarnungen und Sperren der Personen, wie sie Projekt projectID sieht:
// eigene und die von Partnerprojekten mit eingewilligtem Teilen. Pro Person die schwerste zuerst.
func (s *Server) visibleSanctions(ctx context.Context, q dbtx, projectID int, userIDs []string) (map[string][]Sanction, error) {
	result := map[string][]Sanction{}
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := q.Query(ctx,
		`SELECT s.user_id::text, s.level, s.reason, s.created_at, p.name, s.project_id = $1,
		        p.ban_days, p.ban_action, p.warning_days
		 FROM sanctions s
		 JOIN projects p ON p.id = s.project_id
		 WHERE s.user_id = ANY($2::uuid[])
		   AND (s.project_id = $1 OR EXISTS (
		       SELECT 1 FROM project_shares sh
		       WHERE sh.consented_at IS NOT NULL
		         AND ((sh.project_id = $1 AND sh.partner_id = s.project_id)
		           OR (sh.partner_id = $1 AND sh.project_id = s.project_id))))`, projectID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := time.Now()
	for rows.Next() {
		var userID, level string
		var sc Sanction
		var rules sanctionRules
		if err := rows.Scan(&userID, &level, &sc.Reason, &sc.Since, &sc.ProjectName, &sc.Own,
			&rules.BanDays, &rules.BanAction, &rules.WarningDays); err != nil {
			return nil, err
		}
		sc.Level, sc.Until = effectiveSanction(level, sc.Since, rules, now)
		if sc.Level == "" {
			continue
		}
		result[userID] = append(result[userID], sc)
	}
	// Sperren vor Verwarnungen, damit die erste die wichtigste ist
	for _, list := range result {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Level == LevelBan && list[j].Level != LevelBan })
	}
	return result, rows.Err()
}

// banned: eine der geltenden Sanktionen ist eine Sperre.
func banned(list []Sanction) bool {
	return len(list) > 0 && list[0].Level == LevelBan
}

// setSanction verwarnt oder sperrt ein Mitglied im eigenen Projekt (Projektleiter und Admins).
// Eine bestehende Verwarnung oder Sperre dieses Projekts wird ersetzt und die Frist beginnt neu.
func (s *Server) setSanction(w http.ResponseWriter, r *http.Request, m *Member) error {
	var req struct {
		Level  string `json:"level"`
		Reason string `json:"reason"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Level != LevelWarning && req.Level != LevelBan {
		return errorf(http.StatusBadRequest, "Ungültige Stufe")
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		return errorf(http.StatusBadRequest, "Bitte einen Grund angeben")
	}
	target, err := s.projectMemberUser(r, m)
	if err != nil {
		return err
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`INSERT INTO sanctions (project_id, user_id, level, reason, created_by) VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (project_id, user_id) DO UPDATE SET level = $3, reason = $4, created_by = $5, created_at = now()`,
		m.Project.ID, target.ID, req.Level, req.Reason, m.ID); err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionSanctionSet,
		Target: target, Details: map[string]any{"stufe": req.Level}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

// removeSanction hebt die Verwarnung oder Sperre des eigenen Projekts auf.
func (s *Server) removeSanction(w http.ResponseWriter, r *http.Request, m *Member) error {
	target, err := s.projectMemberUser(r, m)
	if err != nil {
		return err
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM sanctions WHERE project_id = $1 AND user_id = $2`, m.Project.ID, target.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionSanctionRemoved,
		Target: target}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

// --- Einstellungen des Projekts (Projekt-Admins) ---

func (s *Server) getSanctionRules(w http.ResponseWriter, r *http.Request, m *Member) error {
	writeJSON(w, http.StatusOK, m.Project.sanctionRules)
	return nil
}

// setSanctionRules legt fest, ob und wann Sperren aufgehoben bzw. zu Verwarnungen und Verwarnungen gelöscht werden.
// Die Fristen gelten auch für bestehende Verwarnungen und Sperren.
func (s *Server) setSanctionRules(w http.ResponseWriter, r *http.Request, m *Member) error {
	var rules sanctionRules
	if err := decode(r, &rules); err != nil {
		return err
	}
	if err := rules.validate(); err != nil {
		return err
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE projects SET ban_days = $2, ban_action = $3, warning_days = $4 WHERE id = $1`,
		m.Project.ID, rules.BanDays, rules.BanAction, rules.WarningDays); err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionSanctionRules,
		Details: map[string]any{"sperreTage": rules.BanDays, "nachSperre": rules.BanAction, "verwarnungTage": rules.WarningDays}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, rules)
	return nil
}

// projectMemberUser lädt das Mitglied {id} des Projekts. Personen anderer Projekte gelten als nicht gefunden.
func (s *Server) projectMemberUser(r *http.Request, m *Member) (*User, error) {
	id, err := pathUUID(r, "id")
	if err != nil {
		return nil, err
	}
	return scanUser(s.db.QueryRow(r.Context(),
		`SELECT `+userColumns+` FROM users u
		 JOIN project_members pm ON pm.user_id = u.id AND pm.project_id = $2
		 WHERE u.id = $1`, id, m.Project.ID))
}

// loadSanctionsFor ist visibleSanctions für eine einzelne Person.
func (s *Server) loadSanctionsFor(ctx context.Context, projectID int, userID string) ([]Sanction, error) {
	all, err := s.visibleSanctions(ctx, s.db, projectID, []string{userID})
	if err != nil {
		return nil, err
	}
	return all[userID], nil
}
