package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"gis-app/backend/internal/database"
	"gis-app/backend/internal/notification"
	"github.com/joho/godotenv"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() { os.Exit(run()) }

func run() int {
	env := flag.String("env", ".env", "Path to the backend environment file")
	send := flag.Bool("send", false, "Send email and update notif_state (default: read-only dry-run)")
	retry := flag.Bool("retry-failed", false, "Include notif_state=9 as well as pending notifications")
	limit := flag.Int("limit", 0, "Maximum events per run; defaults to NOTIFICATION_BATCH_SIZE")
	processID := flag.Int64("process-id", 0, "Only process this pending/failed process ID")
	flag.Parse()
	if *processID < 0 || *limit < 0 {
		fmt.Fprintln(os.Stderr, "Invalid process ID or limit")
		return 2
	}
	if err := godotenv.Load(*env); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot load environment file:", err)
		return 2
	}
	config, err := notification.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Notification configuration:", err)
		return 2
	}
	if *limit == 0 {
		*limit = config.BatchSize
	}
	if *limit > 500 {
		fmt.Fprintln(os.Stderr, "Limit must not exceed 500")
		return 2
	}
	db, err := database.Connect()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot connect to notification database")
		return 1
	}
	defer db.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(*limit)*25*time.Second+10*time.Second)
	defer cancel()
	encoder := json.NewEncoder(os.Stdout)
	redact := func(value string) string {
		if config.Mail.Password != "" {
			return strings.ReplaceAll(value, config.Mail.Password, "[redacted]")
		}
		return value
	}
	worker := notification.Worker{DB: db, Config: config}
	err = worker.Run(ctx, notification.Options{Send: *send, RetryFailed: *retry, Limit: *limit, ProcessID: *processID}, func(result notification.Result) {
		result.Error = redact(result.Error)
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintln(os.Stderr, "Cannot write notification result")
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, redact(err.Error()))
		return 1
	}
	return 0
}
