package auth

import (
	"context"
	"crypto/subtle"
	"errors"
)

var ErrCurrentPassword = errors.New("current password does not match")

func (m *UserModel) UpdateAccount(ctx context.Context, id int64, name, currentPassword, newPassword string) (User, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	var username, password string
	err = tx.QueryRow(ctx, `SELECT "USERNAME", COALESCE("PASSWORD", '')
		FROM public.user_app WHERE "ID_USER"=$1 AND COALESCE("ACCOUNTSTATUS",0)=0
		FOR UPDATE`, id).Scan(&username, &password)
	if err != nil {
		return User{}, err
	}
	if newPassword != "" && subtle.ConstantTimeCompare([]byte(password), []byte(currentPassword)) != 1 {
		return User{}, ErrCurrentPassword
	}
	// Keep the employee name in sync because the profile reads it before user_app.
	_, err = tx.Exec(ctx, `UPDATE public."Gis_Employees" SET name=$1
		WHERE "id_Gis_Employees"=(SELECT "id_Gis_Employees" FROM public."Gis_Employees"
		WHERE LOWER(BTRIM(username))=LOWER(BTRIM($2)) ORDER BY "id_Gis_Employees" DESC LIMIT 1)`, name, username)
	if err != nil {
		return User{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE public.user_app
		SET "FIRSTNAME"=$1, "MIDDLENAME"='', "LASTNAME"='',
		"PASSWORD"=CASE WHEN $2<>'' THEN $2 ELSE "PASSWORD" END,
		"MODIFIED_BY"=$3, "DATE_MODIFIED"=NOW()::text WHERE "ID_USER"=$4`, name, newPassword, username, id)
	if err != nil {
		return User{}, err
	}
	user, err := scanUser(tx.QueryRow(ctx, `SELECT `+userFields+`
		FROM public.user_app u `+employeeAndDepartmentJoins+` WHERE u."ID_USER"=$1`, id))
	if err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}
