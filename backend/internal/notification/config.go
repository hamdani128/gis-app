package notification

import (
	"fmt"
	"gis-app/backend/internal/mailer"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Mail            mailer.Config
	TestMode        bool
	TestTo          string
	BCC, RejectedCC []string
	BatchSize       int
}

func LoadConfig() (Config, error) {
	mail, err := mailer.LoadConfig()
	if err != nil {
		return Config{}, err
	}
	mode := strings.TrimSpace(os.Getenv("NOTIFICATION_TEST_MODE"))
	if mode == "" {
		mode = "true"
	}
	test, err := strconv.ParseBool(mode)
	if err != nil {
		return Config{}, fmt.Errorf("NOTIFICATION_TEST_MODE must be true or false")
	}
	testTo := strings.TrimSpace(os.Getenv("NOTIFICATION_TEST_TO"))
	if test {
		addresses, err := mailer.NormalizeRecipients([]string{testTo}, nil, nil)
		if err != nil || len(addresses.To) != 1 {
			return Config{}, fmt.Errorf("NOTIFICATION_TEST_TO must contain one valid email address in testing mode")
		}
		testTo = addresses.To[0]
	}
	size := strings.TrimSpace(os.Getenv("NOTIFICATION_BATCH_SIZE"))
	if size == "" {
		size = "1"
	}
	batch, err := strconv.Atoi(size)
	if err != nil || batch < 1 || batch > 500 {
		return Config{}, fmt.Errorf("NOTIFICATION_BATCH_SIZE must be between 1 and 500")
	}
	recipients, err := mailer.NormalizeRecipients(nil, nil, splitEmails(os.Getenv("NOTIFICATION_BCC")))
	if err != nil {
		return Config{}, err
	}
	rejected, err := mailer.NormalizeRecipients(nil, splitEmails(os.Getenv("NOTIFICATION_REJECTED_CC")), nil)
	if err != nil {
		return Config{}, err
	}
	return Config{Mail: mail, TestMode: test, TestTo: testTo, BCC: recipients.BCC, RejectedCC: rejected.CC, BatchSize: batch}, nil
}

func splitEmails(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '\n' })
}
