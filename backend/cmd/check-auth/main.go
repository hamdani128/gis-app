// Read-only diagnostics: verifies mail authentication and reset query schema.
// Does not send mail or change a user's password.
package main

import (
	"context"
	"fmt"
	"gis-app/backend/internal/database"
	"gis-app/backend/internal/mailer"
	"github.com/joho/godotenv"
	"os"
	"strings"
	"time"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Environment file could not be loaded")
		os.Exit(1)
	}
	c, err := mailer.LoadConfig()
	if err != nil {
		fmt.Println("SMTP config:", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.CheckConnection(ctx); err != nil {
		detail := err.Error()
		if c.Password != "" {
			detail = strings.ReplaceAll(detail, c.Password, "[redacted]")
		}
		fmt.Println("SMTP check:", detail)
	} else {
		fmt.Println("SMTP TLS and authentication: OK (no email sent)")
	}
	db, err := database.Connect()
	if err != nil {
		fmt.Println("Database connection check: FAILED")
		return
	}
	defer db.Close()
	schemaCtx, schemaCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer schemaCancel()
	rows, err := db.Query(schemaCtx, `SELECT table_name, column_name, data_type, COALESCE(character_maximum_length,0)
		FROM information_schema.columns WHERE table_schema='public' AND
		((table_name='user_app' AND column_name IN ('PASSWORD','MODIFIED_BY','DATE_MODIFIED','ACCOUNTSTATUS'))
		OR (table_name='Gis_Employees' AND column_name IN ('username','email')))
		ORDER BY table_name, column_name`)
	if err != nil {
		fmt.Println("Database schema check: FAILED")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var table, column, kind string
		var length int
		if err := rows.Scan(&table, &column, &kind, &length); err != nil {
			fmt.Println("Schema read failed")
			return
		}
		fmt.Printf("Schema: %s.%s %s length=%d\n", table, column, kind, length)
	}
}
