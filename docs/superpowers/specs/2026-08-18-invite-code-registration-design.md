# Invite-Code Registration Design

Date: 2026-08-18

## Overview

Registration (`POST /v1/auth/register`) requires a single-use invite code in
addition to username/password + Turnstile. Invite codes are stored in a database
table and managed through staff-only REST endpoints (no admin web UI in this
phase). The code is consumed atomically inside the registration transaction so a
failed registration (e.g. duplicate username) does not burn the code.

## Rules

- Every registration must provide a valid, unused, non-expired invite code.
- A code is single-use: once consumed it can never be used again.
- Codes are case-insensitive when matched, but stored/displayed in uppercase.
- Code value: 24 random bytes from rand.Reader formatted as
  `LANTERN-XXXX-XXXX-XXXX-XXXX` (Base32 crockford alphabet, no look-alike chars).
- Management endpoints are staff-only (same guard as moderation).

## Database (migration `auth/004_invite_codes`)

`invite_codes`:

| column       | type        | notes                                   |
|--------------|-------------|-----------------------------------------|
| id           | uuid PK     |                                         |
| code         | text        | normalized uppercase, immutable         |
| created_by   | uuid        | FK `users(id)`, the staff member        |
| used_by      | uuid NULL   | FK `users(id)`, set on consumption      |
| used_at      | timestamptz NULL | consumption timestamp                |
| expires_at   | timestamptz NULL | NULL = never expires                 |
| created_at   | timestamptz | default now()                           |

- Unique index on `code`.
- Index on `used_by IS NULL` (partial) to speed active-code lookup.

## Backend (Go, auth module)

### Repository

- `CreateInviteCode(ctx, createdBy, expiresAt) (*InviteCode, error)` — generates and inserts.
- `ListInviteCodes(ctx, limit, cursor) ([]InviteCode, nextCursor, error)` — newest first.
- `GetInviteCode(ctx, code) (*InviteCode, error)` — by normalized uppercase code.
- `ConsumeInviteCode(ctx, code, usedBy) error` — single atomic UPDATE
  `SET used_by=$2, used_at=now() WHERE code=$1 AND used_by IS NULL AND (expires_at IS NULL OR expires_at > now())`
  arguing `0` rows => error (invalid/used/expired resolved by prior read; the UPDATE is the
  concurrency safety net).
- `RevokeInviteCode(ctx, id) error` — marks used? No: revokes by DELETE; refuses when already used
  (re-read before delete in service).

### Service

- `Register(ctx, username, password, turnstileToken, inviteCode, ...)`: inside the existing
  registration transaction — validate username format, create user, then consume the invite
  code; any step failing rolls back, so the code is not consumed.
  Errors: `INVITE.CODE_INVALID` (not found), `INVITE.CODE_USED` (already consumed),
  `INVITE.CODE_EXPIRED` (expires_at passed). All map to HTTP 400.
- New `InviteCodeService` (same package or sibling): `Create`, `List`, `Revoke` — staff only.

### Handler + routes

Staff-guarded (auth middleware, same pattern as moderation guard):

- `POST /v1/invite-codes` — body `{expiresAt?: RFC3339}` -> 201 `{id, code, expiresAt, createdAt}`.
- `GET /v1/invite-codes?limit&cursor` -> `{data: [...], pagination}`.
- `DELETE /v1/invite-codes/{id}` -> 204 on success; 409 if already used.

`POST /v1/auth/register` body gains required `inviteCode: string`.

## Contract (openapi.yaml)

- `RegisterRequest` adds required `inviteCode`.
- New paths under tag `invite-codes` (staff): create/list/delete as above.
- Error schema entries for `INVITE.CODE_INVALID`, `INVITE.CODE_USED`, `INVITE.CODE_EXPIRED`.
- `make validate-contract` must pass.

## Android

- `RegisterRequest` gains `inviteCode` (`@SerialName("inviteCode")`).
- `AuthViewModel.register` accepts and forwards invite code.
- `RegisterScreen` adds an "邀请码" input field (testTag `register-invite`); submit enabled only
  when the code is non-blank. No client-side format validation beyond non-blank.

## Admin web

- No changes (API-only per decision). The existing login flow is unchanged.

## Tests

- Invite repository: create/list/consume/revoke; consume rejects already-used and expired.
- Service registration: valid code succeeds and consumes; invalid/used/expired codes rejected;
  duplicate-username failure does not consume the code (transactionality).
- Concurrency: two parallel registrations with the same code -> exactly one succeeds.
- Handler: staff vs non-staff access to management endpoints; register error mapping (400 codes).
- Contract validation still passes (covered by make target).

## Dev seeding / manual test

With the API running and a staff token:

```bash
TOKEN=$(curl -s -X POST http://localhost:8081/v1/auth/login \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: seed-invite-1' \
  -d '{"username":"admin","password":"<pw>","turnstileToken":"1x0000000000000000000000000000000AA"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["accessToken"])')

curl -s -X POST http://localhost:8081/v1/invite-codes \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: gen-1' -d '{}'
```

The response contains the code to register with on Android.