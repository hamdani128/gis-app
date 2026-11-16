# Forgot password

`POST /api/auth/forgot-password`, public endpoint; requires JSON:

```json
{ "username": "employee.username", "email": "employee@example.com" }
```

Username and email are trimmed and matched without case sensitivity against an
active `user_app` account and its latest `Gis_Employees` email. No other account
fields are accepted. Matching requests generate a cryptographically random
16-character password and send an HTML email to the email from the employee
record. The existing password format remains compatible with current login and
the legacy app. Users should change this password in Account Settings after login.

Both matching and nonmatching requests return the same 200 envelope:

```json
{ "status": 200, "message": "Jika username dan email cocok dengan akun aktif, password baru akan dikirim ke email terdaftar.", "data": null }
```

Errors: 400 invalid/missing input, 415 non-JSON, 429 rate limited (with Retry-After),
503 email unavailable or reset processing failure. Passwords are never included in
API responses or logs. The database update and delivery run under a row lock in a
transaction; failed delivery rolls back the password change. SMTP acceptance is
not proof of inbox delivery. If the database commit fails after SMTP accepts an
email, the delivered password will not take effect; ICT must retry/reset the account.

## Mail setup

Configure `SMTP_HOST`, `SMTP_PORT`, `SMTP_TLS_MODE`, `SMTP_FROM` and optionally
`SMTP_USERNAME` / `SMTP_PASSWORD` in the backend environment. `SMTP_TLS_MODE`:
`starttls` (default, usually port 587), `tls` (implicit TLS, usually 465), or `none`
for a trusted internal relay without authentication. TLS certificates are verified;
authenticated SMTP always requires TLS. Configure `FRONTEND_ORIGIN` to the public
frontend URL so email login links resolve correctly. Restart the backend after
changing environment values. No mail server or credentials are hardcoded.

SMTP operations have an 8-second total deadline; reset handling has a 12-second
deadline. Missing mail configuration returns 503 before any database change.

## Request limits

In-memory limits: one reset request per normalized username per minute and five
per client IP per 15 minutes. Expired entries are removed, and the cache is bounded.
The client IP comes from `RemoteAddr`, never untrusted forwarded headers. Behind a
reverse proxy, the IP limit applies to the proxy address. Limits reset on restart
and are per instance; deployments with multiple instances should use a shared
limiter/proxy rule and a trusted client IP policy.

## Verification scope

Unit tests cover malformed requests, field validation, password generation,
request limits, missing SMTP configuration, MIME/HTML formatting, HTML escaping,
header injection, and TLS configuration. No production email is sent by tests.
Database writes and delivery against the organization's SMTP relay need a configured
test account and mail environment for end-to-end verification.

Run `go run ./cmd/check-auth` from the backend directory to check SMTP DNS,
connection, TLS, authentication, and relevant database column types. This command
does not send email or modify passwords. Reset failures also record the failing
stage in backend logs, with SMTP and generated passwords redacted. A DNS lookup
failure must be resolved with the correct SMTP hostname or the organization's DNS
configuration before authentication or delivery can be checked.
