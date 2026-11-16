# Account settings

`PUT /api/auth/me` updates the active user identified by the signed session cookie.
Send `Content-Type: application/json` and include credentials.

```json
{
  "name": "System Admin",
  "current_password": "existing-password",
  "new_password": "replacement-password"
}
```

`name` is required, trimmed, and limited to 100 characters. Omit both password
fields to update only the display name. To change the password, supply the current
password and a different new password containing 8–100 characters. Passwords are
not trimmed. Names are synchronized with the employee record used by the profile.
Username, email, roles, and company access cannot be changed through this endpoint.
All writes happen in one transaction; an incorrect current password saves nothing.

Success returns the usual `{ "status": 200, "message": "...", "data": {...} }`
envelope with the updated public profile, matching `GET /api/auth/me`. Passwords
are never returned. Errors: 400 for invalid input or an incorrect current password,
401 for a missing/invalid/expired session or inactive account, 415 for a non-JSON
request, and 500 for a server error. Unknown input fields are rejected.
