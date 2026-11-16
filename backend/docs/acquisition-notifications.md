# Acquisition email notifications (cron)

The notification worker is separate from the HTTP server. It reads PostgreSQL
and uses the existing SMTP settings from the backend environment file. No HTTP
endpoint, public scheduler, or cron entry is installed automatically.

## Testing mode

```dotenv
NOTIFICATION_TEST_MODE=true
NOTIFICATION_TEST_TO=rizki.hamdani@aepplantations.id
NOTIFICATION_BATCH_SIZE=1
NOTIFICATION_BCC=rizki.hamdani@aepplantations.id,yandi.baraga@aepplantations.id,roy.asmara@aepplantations.id
NOTIFICATION_REJECTED_CC=tadjuddin.noer@angloeastern.co.id
```

Testing is the default if the flag is absent. Testing emails have `[TESTING]` at
the start of the subject and a prominent bilingual testing banner. All email goes
only to `NOTIFICATION_TEST_TO`; sequence recipients, authors, CC, rejection copies,
and BCC are excluded. The testing address is required and must be one valid email;
there is no fallback to production recipients. Approver routing is bypassed during
testing, so missing next-approver addresses do not prevent a test email.
Queue-state changes still follow the normal workflow when `--send` is supplied.
Testing does not create synthetic land data. Once testing is finished, set
`NOTIFICATION_TEST_MODE=false` to restore normal sequence-based recipients and
copies and remove testing markers. Already sent events are not automatically resent.

Omitting `--send` is always a read-only dry-run: no SMTP connection, no email, no
`notif_state`/audit changes. It validates data, recipients, and the rendered email.

## Run from source

From the backend directory:

```sh
# Default: preview the oldest pending event, no email and no state changes.
go run ./cmd/notify-acquisition

# Preview up to 20 events.
go run ./cmd/notify-acquisition --limit 20

# Inspect a specific pending event (use --retry-failed to include state 9).
go run ./cmd/notify-acquisition --process-id 12345

# Actual delivery; subject/banner depend on NOTIFICATION_TEST_MODE.
go run ./cmd/notify-acquisition --send

# Explicitly include failed events for a retry.
go run ./cmd/notify-acquisition --retry-failed --send
```

`--env /absolute/path/to/backend/.env` selects a configuration file independently
of cron's working directory. Environment variables already supplied to the
process take precedence over the file. `--limit` overrides the configured batch
size; valid values are 1–500. Results are JSON lines with process ID, land ID,
mode, subject, To/CC, BCC count, and outcome. BCC addresses are not printed or
exposed in message headers. SMTP passwords are redacted from errors.

## Linux cron example

Build the binary on the target Linux server (or cross-compile for its OS/CPU):

```sh
cd /srv/gis-app/backend
go build -o bin/gis-notify ./cmd/notify-acquisition
```

Then add the entry to the server user's crontab. Replace paths with the actual
deployment directory and make sure the log directory is writable:

```cron
* * * * * /srv/gis-app/backend/bin/gis-notify --env /srv/gis-app/backend/.env --send >> /srv/gis-app/backend/notification-cron.log 2>&1
```

The example sends at most one event per minute with the current default batch
size. Cron reloads the environment file on every invocation, so changing the
notification testing flag does not require restarting the API server.

## Legacy recipient and queue behavior

Pending events are `notif_state=0` or NULL and are processed in ascending process
ID order. `--retry-failed` also includes state 9. Existing unprocessed history is
part of this queue, so inspect the oldest events in dry-run before scheduling.

Routing follows the supplied PHP `getTo` function, using the land's **current**
status (not the process sequence ID): sequence `status+1` supplies To;
`status+2` supplies CC. Candidate employees come from company-0 staff excluding
RM/GM, company staff without an estate restriction, the company's regional GM,
matching estate staff, and explicit employee-company assignments. Officers must
also have an effective process-officer assignment; expired assignments are ignored.

The process author is copied in CC. For rejected land (`status=99`), routing starts
from sequence 0, the author is added to To, and `NOTIFICATION_REJECTED_CC` is copied.
`NOTIFICATION_BCC` applies to production notifications. Duplicate emails are removed,
with To taking precedence over CC, then BCC. Empty addresses are ignored; malformed
addresses fail the event. At least one To recipient is required. Missing company,
estate, or next-approver email is an explicit failure, never a fallback to unrelated
users. Old/completed events may need their data or officer assignments reviewed.

Email contains approval/date/officer/note, land ID, company, estate, owner, wet+dry
area, calculated compensation total, current land status, and a login link using
`FRONTEND_ORIGIN`. HTML values are escaped.

After SMTP accepts the email, the worker sets `notif_state=1`, `date_modified=NOW()`
and `modified_by='notif_mail'`. Failed events become 9 only in send mode; dry-run
never changes their state. Failures result in a nonzero exit status for monitoring.

A PostgreSQL session advisory lock prevents simultaneous runs of this worker;
the lock is released even on errors. Each sent event also uses a transaction and
row lock. Run only one notification implementation against the queue; disable any
old PHP cron when enabling the new worker. As with SMTP/database delivery in
general, a crash after SMTP acceptance but before commit can lead to a duplicate
on retry. Explicit commit failures stop the worker with `accepted_but_unconfirmed`
for manual review. Stable Message-ID values aid diagnosis but do not guarantee
exactly-once delivery.

## Verification

```sh
go test ./internal/notification ./internal/mailer ./cmd/notify-acquisition
```

Unit tests verify testing-only delivery and markers, production mode, recipient routing,
deduplication, BCC privacy, rejection copies, header/HTML injection defenses, and
currency formatting. Dry-run validates real queue queries without sending email.
