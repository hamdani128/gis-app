package notification

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Worker struct {
	DB     *pgxpool.Pool
	Config Config
}

type Options struct {
	Send, RetryFailed bool
	Limit             int
	ProcessID         int64
}
type Result struct {
	ProcessID int64    `json:"process_id"`
	LandID    int64    `json:"land_id,omitempty"`
	Mode      string   `json:"mode"`
	Action    string   `json:"action"`
	Subject   string   `json:"subject,omitempty"`
	To        []string `json:"to,omitempty"`
	CC        []string `json:"cc,omitempty"`
	BCCCount  int      `json:"bcc_count"`
	Error     string   `json:"error,omitempty"`
}

const workerLockID int64 = 7349278142

func (w Worker) Run(ctx context.Context, options Options, report func(Result)) error {
	if options.Limit < 1 || options.Limit > 500 {
		return errors.New("limit must be between 1 and 500")
	}
	conn, err := w.DB.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if options.Send {
		var locked bool
		if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, workerLockID).Scan(&locked); err != nil {
			return err
		}
		if !locked {
			return errors.New("another notification worker is already running")
		}
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var unlocked bool
			if err := conn.QueryRow(cleanup, `SELECT pg_advisory_unlock($1)`, workerLockID).Scan(&unlocked); err != nil || !unlocked {
				// Never return a connection with a session lock to the pool.
				_ = conn.Conn().Close(cleanup)
			}
		}()
	}
	rows, err := conn.Query(ctx, `SELECT "id_Gis_Aquisition_Processes" FROM public."Gis_Aquisition_Processes"
		WHERE (notif_state IS NULL OR notif_state=0 OR ($1::boolean AND notif_state=9))
		AND ($2::bigint=0 OR "id_Gis_Aquisition_Processes"=$2)
		ORDER BY "id_Gis_Aquisition_Processes" ASC LIMIT $3`, options.RetryFailed, options.ProcessID, options.Limit)
	if err != nil {
		return fmt.Errorf("notification queue: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		report(Result{Mode: w.mode(options.Send), Action: "queue_empty"})
		return nil
	}
	failed := 0
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		itemCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		result, accepted, err := w.process(itemCtx, conn, options, id)
		cancel()
		if err != nil {
			result.Error = err.Error()
			result.Action = "failed"
			failed++
			if accepted {
				result.Action = "accepted_but_unconfirmed"
				report(result)
				return errors.New("SMTP accepted a notification but its state could not be committed; review the process before retrying")
			}
			if options.Send {
				markCtx, markCancel := context.WithTimeout(ctx, 5*time.Second)
				_, markErr := conn.Exec(markCtx, `UPDATE public."Gis_Aquisition_Processes" SET notif_state=9,
				date_modified=NOW(),modified_by='notif_mail' WHERE "id_Gis_Aquisition_Processes"=$1
				AND (notif_state IS NULL OR notif_state=0 OR ($2::boolean AND notif_state=9))`, id, options.RetryFailed)
				markCancel()
				if markErr != nil {
					report(result)
					return fmt.Errorf("notification failure state: %w", markErr)
				}
			}
		}
		report(result)
	}
	if failed > 0 {
		return fmt.Errorf("%d notification(s) failed", failed)
	}
	return nil
}

func (w Worker) mode(send bool) string {
	if !send {
		if w.Config.TestMode {
			return "dry-run/testing"
		}
		return "dry-run/production"
	}
	if w.Config.TestMode {
		return "testing"
	}
	return "production"
}

func (w Worker) process(ctx context.Context, conn *pgxpool.Conn, options Options, id int64) (Result, bool, error) {
	result := Result{ProcessID: id, Mode: w.mode(options.Send)}
	txOptions := pgx.TxOptions{}
	if !options.Send {
		txOptions.AccessMode = pgx.ReadOnly
	}
	tx, err := conn.BeginTx(ctx, txOptions)
	if err != nil {
		return result, false, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	var event Event
	query := `SELECT ap."id_Gis_Aquisition_Processes",ap."Gis_Compensated_Lands_id",
		COALESCE(ap.approval,-1),COALESCE(ap.date_created::text,''),COALESCE(ap.note,''),COALESCE(ap.username,'')
		FROM public."Gis_Aquisition_Processes" ap WHERE ap."id_Gis_Aquisition_Processes"=$1
		AND (ap.notif_state IS NULL OR ap.notif_state=0 OR ($2::boolean AND ap.notif_state=9))`
	if options.Send {
		query += ` FOR UPDATE SKIP LOCKED`
	}
	err = tx.QueryRow(ctx, query, id, options.RetryFailed).Scan(&event.ID, &event.LandID, &event.Approval, &event.Date, &event.Note, &event.Username)
	if errors.Is(err, pgx.ErrNoRows) {
		result.Action = "skipped"
		return result, false, nil
	}
	if err != nil {
		return result, false, fmt.Errorf("read process: %w", err)
	}
	result.LandID = event.LandID
	if err := loadEvent(ctx, tx, &event); err != nil {
		return result, false, err
	}
	recipients := Recipients{}
	if !w.Config.TestMode {
		recipients, err = loadRecipients(ctx, tx, event)
		if err != nil {
			return result, false, err
		}
	}
	message, err := BuildMessage(event, recipients, w.Config)
	if err != nil {
		return result, false, err
	}
	result.Subject = message.Subject
	result.To = message.To
	result.CC = message.CC
	result.BCCCount = len(message.BCC)
	if !options.Send {
		result.Action = "would_send"
		return result, false, nil
	}
	if err := w.Config.Mail.SendHTML(ctx, message); err != nil {
		return result, false, err
	}
	_, err = tx.Exec(ctx, `UPDATE public."Gis_Aquisition_Processes" SET notif_state=1,
		date_modified=NOW(),modified_by='notif_mail' WHERE "id_Gis_Aquisition_Processes"=$1`, id)
	if err != nil {
		return result, true, err
	}
	if err := tx.Commit(ctx); err != nil {
		return result, true, err
	}
	result.Action = "sent"
	return result, true, nil
}

func loadEvent(ctx context.Context, tx pgx.Tx, e *Event) error {
	err := tx.QueryRow(ctx, `SELECT COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id",0),COALESCE(p."id_Gis_Potential_Lands",0),
		COALESCE(NULLIF(BTRIM(co.name),''),NULLIF(BTRIM(co.com_name),''),''),COALESCE(p.estate_name,''),COALESCE(o.name,''),COALESCE(l.status,0),
		(COALESCE(l.dry_land_wide,0)+COALESCE(l.wet_land_wide,0))::double precision,
		(COALESCE(l.dry_land_wide,0)*COALESCE(l.dry_land_perha_price,0)+COALESCE(l.wet_land_wide,0)*COALESCE(l.wet_land_perha_price,0)
		+COALESCE(l.admin_price,0)+COALESCE(l.plant_price,0))::double precision,
		COALESCE(employee.name,''),COALESCE(employee.email,''),
		COALESCE((SELECT seq.name FROM public."Gis_Aquisition_Process_Sequences" seq WHERE seq.sequence=l.status ORDER BY seq."id_Gis_Aquisition_Process_Sequences" LIMIT 1),
		CASE l.status WHEN 10 THEN 'Compensated Land (old)' WHEN 99 THEN 'Rejected' ELSE 'Status '||COALESCE(l.status::text,'0') END)
		FROM public."Gis_Compensated_Lands" l
		LEFT JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		LEFT JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		LEFT JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
		LEFT JOIN public."Gis_Companies" co ON co."id_Gis_Companies"=COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")
		LEFT JOIN LATERAL (SELECT emp.name,emp.email FROM public."Gis_Employees" emp
		  WHERE LOWER(BTRIM(emp.username))=LOWER(BTRIM($2)) ORDER BY emp."id_Gis_Employees" DESC LIMIT 1) employee ON TRUE
		WHERE l."id_Gis_Compensated_Lands"=$1`, e.LandID, e.Username).Scan(&e.CompanyID, &e.EstateID, &e.Company, &e.Estate, &e.Owner, &e.Status, &e.Area, &e.Total, &e.OfficerName, &e.OfficerEmail, &e.StatusName)
	if err != nil {
		return fmt.Errorf("land information: %w", err)
	}
	if e.CompanyID <= 0 || e.EstateID <= 0 {
		return errors.New("land is missing its company or estate")
	}
	return nil
}

func loadRecipients(ctx context.Context, tx pgx.Tx, e Event) (Recipients, error) {
	base := e.Status
	if base == 99 {
		base = 0
	}
	rows, err := tx.Query(ctx, `WITH candidates AS (
		SELECT e."id_Gis_Employees" FROM public."Gis_Employees" e
		WHERE (e."Gis_Companies_id"=0 AND e.dept IS NOT NULL AND NOT EXISTS
		  (SELECT 1 FROM public.cg_ref_codes dept WHERE dept.id=e.dept AND UPPER(BTRIM(dept.low_value)) IN ('RM','GM')))
		OR (e."Gis_Companies_id"=$1 AND e."Gis_Potential_Lands_id" IS NULL)
		OR e."id_Gis_Employees" IN (SELECT r."Gis_Employees_id" FROM public."Gis_regions" r
		  JOIN public."Gis_Companies" co ON co."Gis_regions_id"=r."idGis_regions" WHERE co."id_Gis_Companies"=$1)
		OR e."Gis_Potential_Lands_id"=$2
		OR EXISTS (SELECT 1 FROM public."Gis_Employee_Companies" ec WHERE ec."Gis_Employees_id"=e."id_Gis_Employees" AND ec."Gis_Companies_id"=$1)
	)
	SELECT DISTINCT seq.sequence,BTRIM(e.email) FROM candidates c
	JOIN public."Gis_Employees" e ON e."id_Gis_Employees"=c."id_Gis_Employees"
	JOIN public."Gis_Process_Officers" po ON po."Gis_Employees_id_Gis_Employees"=e."id_Gis_Employees"
	JOIN public."Gis_Aquisition_Process_Sequences" seq ON seq."id_Gis_Aquisition_Process_Sequences"=po."Gis_Aquisition_Process_Sequences_id"
	WHERE seq.sequence IN ($3::integer+1,$3::integer+2) AND NULLIF(BTRIM(e.email),'') IS NOT NULL
	AND (po.valid_from IS NULL OR po.valid_from<=NOW())
	AND (NULLIF(BTRIM(po.valid_until),'') IS NULL OR CASE WHEN BTRIM(po.valid_until) ~ '^\d{4}-\d{2}-\d{2}' THEN LEFT(BTRIM(po.valid_until),10)::date>=CURRENT_DATE ELSE TRUE END)
	ORDER BY seq.sequence,BTRIM(e.email)`, e.CompanyID, e.EstateID, base)
	if err != nil {
		return Recipients{}, fmt.Errorf("notification recipients: %w", err)
	}
	defer rows.Close()
	result := Recipients{}
	for rows.Next() {
		var sequence int
		var email string
		if err := rows.Scan(&sequence, &email); err != nil {
			return result, err
		}
		if sequence == base+1 {
			result.To = append(result.To, email)
		} else {
			result.CC = append(result.CC, email)
		}
	}
	return result, rows.Err()
}
