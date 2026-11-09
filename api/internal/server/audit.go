package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx ist der Pool oder eine Transaktion. Audit-Einträge werden in derselben Transaktion geschrieben wie die
// Änderung: ohne Eintrag keine Änderung.
type dbtx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Aktionen im Audit-Protokoll. Die Auswahl orientiert sich an der Rechenschaftspflicht und Protokollierung im
// Datenschutzrecht (revDSG mit DSV Art. 4, DSGVO Art. 5 Abs. 2), bei Daten Minderjähriger besonders streng:
// Zugriffe auf heikle Daten (Notfallkontakt, vollständiges Profil), Änderungen an Personendaten und
// Berechtigungen, Löschungen, Exporte, Verwarnungen und Sperren sowie das Teilen von Daten zwischen Projekten.
// Eine rechtliche Prüfung ersetzt das nicht (siehe docs/architektur.md, Audit).
// Die Bezeichnungen für die Anzeige stehen im Frontend (format.ts, auditLabels).
const (
	actionJoined           = "mitglied.angemeldet"
	actionRoleChanged      = "mitglied.rolle"
	actionStatusChanged    = "mitglied.status"
	actionFunctionChanged  = "mitglied.funktion"
	actionGuestCreated     = "gast.erfasst"
	actionEmergencyViewed  = "notfallkontakt.angesehen"
	actionProfileViewed    = "profil.angesehen"
	actionProfileUpdated   = "profil.geaendert"
	actionUserDeleted      = "benutzer.geloescht"
	actionAccountDeleted   = "konto.geloescht"
	actionExport           = "export"
	actionSanctionSet      = "sanktion.erfasst"
	actionSanctionRemoved  = "sanktion.aufgehoben"
	actionSanctionRules    = "sanktion.fristen"
	actionEmergencyAccess  = "notfallzugriff.geaendert"
	actionProjectCreated   = "projekt.erstellt"
	actionProjectUpdated   = "projekt.geaendert"
	actionShareRequested   = "teilen.angefragt"
	actionShareConsented   = "teilen.eingewilligt"
	actionShareDeclined    = "teilen.abgelehnt"
	actionShareRemoved     = "teilen.entfernt"
	deletedAccountAuditTag = "Gelöschtes Konto"
)

type auditEntry struct {
	Actor     *User
	ProjectID *int
	Action    string
	// Target: betroffene Person (optional)
	Target  *User
	Details map[string]any
}

// audit schreibt einen Eintrag ins Audit-Protokoll. Namen werden als Text gespeichert, damit der Eintrag
// auch nach Änderungen lesbar bleibt.
func (s *Server) audit(ctx context.Context, q dbtx, e auditEntry) error {
	var actorID, targetID *string
	var actorName, targetName string
	if e.Actor != nil {
		actorID, actorName = &e.Actor.ID, e.Actor.label()
	}
	if e.Target != nil {
		targetID, targetName = &e.Target.ID, e.Target.label()
	}
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	_, err := q.Exec(ctx,
		`INSERT INTO audit_log (actor_id, actor_name, project_id, action, target_user_id, target_name, details)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		actorID, actorName, e.ProjectID, e.Action, targetID, targetName, e.Details)
	return err
}

// anonymizeAudit ersetzt den Namen einer gelöschten Person im Audit-Protokoll. Die ID bleibt als Pseudonym,
// damit nachvollziehbar bleibt, welche Einträge dieselbe Person betreffen.
func anonymizeAudit(ctx context.Context, q dbtx, userID string) error {
	if _, err := q.Exec(ctx,
		`UPDATE audit_log SET actor_name = $2 WHERE actor_id = $1`, userID, deletedAccountAuditTag); err != nil {
		return err
	}
	_, err := q.Exec(ctx,
		`UPDATE audit_log SET target_name = $2 WHERE target_user_id = $1`, userID, deletedAccountAuditTag)
	return err
}

type AuditRecord struct {
	ID          int64          `json:"id"`
	At          time.Time      `json:"at"`
	ActorID     *string        `json:"actorId"`
	ActorName   string         `json:"actorName"`
	ProjectName *string        `json:"projectName"`
	Action      string         `json:"action"`
	TargetID    *string        `json:"targetId"`
	TargetName  string         `json:"targetName"`
	Details     map[string]any `json:"details"`
}

const auditPageSize = 100

// listAudit liefert das Audit-Protokoll (globale Admins), neuste zuerst, in Seiten zu 100 Einträgen.
// Filter: q (Name der handelnden oder betroffenen Person), project (Kurzname), action (Präfix, z.B. "sanktion"),
// before (ID für die nächste Seite).
func (s *Server) listAudit(w http.ResponseWriter, r *http.Request, u *User) error {
	query := r.URL.Query()
	var before *int64
	if v := query.Get("before"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return errorf(http.StatusBadRequest, "Ungültige Seite")
		}
		before = &n
	}
	rows, err := s.db.Query(r.Context(),
		`SELECT a.id, a.at, a.actor_id::text, a.actor_name, p.name, a.action, a.target_user_id::text, a.target_name, a.details
		 FROM audit_log a
		 LEFT JOIN projects p ON p.id = a.project_id
		 WHERE ($1::bigint IS NULL OR a.id < $1)
		   AND ($2 = '' OR a.actor_name ILIKE '%' || $2 || '%' OR a.target_name ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR p.slug = $3)
		   AND ($4 = '' OR a.action LIKE $4 || '%')
		 ORDER BY a.id DESC
		 LIMIT $5`,
		before, strings.TrimSpace(query.Get("q")), query.Get("project"), query.Get("action"), auditPageSize)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[AuditRecord])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}
