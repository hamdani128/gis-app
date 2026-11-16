package auth

import (
	"context"
	"fmt"
)

// ResetPassword keeps the row locked through delivery, so simultaneous password
// changes cannot overwrite one another. Failed delivery rolls back the update.
func (m *UserModel) ResetPassword(ctx context.Context, username, email, password string, deliver func(string, string) error) error {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("reset transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var id int64
	var matchedUsername, matchedEmail string
	err = tx.QueryRow(ctx, `SELECT u."ID_USER", BTRIM(u."USERNAME"), BTRIM(e.email)
		FROM public.user_app u
		JOIN LATERAL (SELECT employee.email FROM public."Gis_Employees" employee
		  WHERE LOWER(BTRIM(employee.username))=LOWER(BTRIM(u."USERNAME"))
		  ORDER BY employee."id_Gis_Employees" DESC LIMIT 1) e ON TRUE
		WHERE LOWER(BTRIM(u."USERNAME"))=LOWER(BTRIM($1))
		  AND LOWER(BTRIM(e.email))=LOWER(BTRIM($2))
		  AND COALESCE(u."ACCOUNTSTATUS",0)=0
		ORDER BY u."ID_USER" DESC LIMIT 1 FOR UPDATE OF u`, username, email).Scan(&id, &matchedUsername, &matchedEmail)
	if err != nil {
		return fmt.Errorf("reset account lookup: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE public.user_app SET "PASSWORD"=$1,
		"MODIFIED_BY"='forgot-password', "DATE_MODIFIED"=NOW()::text WHERE "ID_USER"=$2`, password, id)
	if err != nil {
		return fmt.Errorf("reset password update: %w", err)
	}
	if err := deliver(matchedEmail, matchedUsername); err != nil {
		return fmt.Errorf("reset email delivery: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("reset commit: %w", err)
	}
	return nil
}
