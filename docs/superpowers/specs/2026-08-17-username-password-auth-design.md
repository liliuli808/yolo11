# Username/Password Auth + Cloudflare Turnstile Design

Date: 2026-08-17

## Overview

Replace the email-code authentication flow with username/password across the whole
stack (Go API, Android client, admin web). Both register and login require a
Cloudflare Turnstile token validated server-side via the Siteverify API.

Current flow (to be replaced):

- `POST /v1/auth/email-codes` — send 6-digit code
- `POST /v1/auth/email-sessions` — exchange code for session

New flow:

- `POST /v1/auth/register` — `{username, password, turnstileToken}` -> session
- `POST /v1/auth/login` — `{username, password, turnstileToken}` -> session

## Rules

- Username: `^[A-Za-z][A-Za-z0-9_]{2,19}$` (3-20 chars, letter first, alphanumeric + underscore), case-insensitive uniqueness.
- Password: at least 8 characters, no composition requirement.
- Stored with bcrypt (Golang `golang.org/x/crypto/bcrypt`).
- Turnstile: server calls `https://challenges.cloudflare.com/turnstile/v0/siteverify`
  with the secret key + token. In development, Cloudflare test secret
  `1x0000000000000000000000000000000AA` always passes; `2x0000000000000000000000000000000AA` always fails.

## Database (migration `auth/003_username_password`)

- `users` gains `username TEXT` (unique) and `password_hash TEXT`.
- `email_normalized` becomes nullable (registration no longer collects email; email
  kept as an optional future recovery channel).
- `email_codes` table retained for the legacy email-change flow (unreachable for new
  users; endpoints stay dormant).

## Backend (Go)

### Config

- Add required `TurnstileSecretKey` (`TURNSTILE_SECRET_KEY`).
- Add `TurnstileSiteKey` (`TURNSTILE_SITE_KEY`), used by admin web and documented in env.
- Replace `StaffEmails` with `StaffUsernames` (`STAFF_USERNAMES`).
- Update `infra/.env.example` accordingly (add `TURNSTILE_SECRET_KEY`,
  `TURNSTILE_SITE_KEY`; rename `STAFF_EMAILS` -> `STAFF_USERNAMES`).

### Repository

- `CreateUser(username, passwordHash) (*User, error)`
- `GetUserByUsername(username) (*User, error)` — returns password hash
- Remove `FindOrCreateUserByEmail` and its test references.
- New `TurnstileVerifier` interface; production impl POSTs to Siteverify and checks
  `success` and `hostname`. Tests use a fake verifier (always pass / always fail).

### Service

- `Register(ctx, username, password, turnstileToken, ip, fingerprint, userAgent) (*TokenResponse, error)`
  — validate username format + uniqueness, hash password, create user, create session.
- `Login(ctx, username, password, turnstileToken, ip, fingerprint, userAgent) (*TokenResponse, error)`
  — lookup by username, bcrypt compare, create session.
- `VerifyPasswordForDeletion(ctx, userID, password) error` — replaces email-code
  confirmation for `DELETE /v1/me`.
- `isStaff(username)` based on `StaffUsernames`.
- Rate limits keyed by username + IP + fingerprint on both register and login.

### Handler

- New `POST /auth/register`, `POST /auth/login`; remove `SendEmailCode` and
  `CreateEmailSession`.
- `DeleteMe` uses password confirmation.
- New domain errors: `ErrUsernameTaken` (409 `AUTH.USERNAME_TAKEN`),
  `ErrInvalidCredentials` (401 `AUTH.INVALID_CREDENTIALS`),
  `ErrCaptchaFailed` (400 `AUTH.CAPTCHA_FAILED`).

## Android

- New `TurnstileWebView` composable: `AndroidView` wrapping a WebView, JS + DOM
  storage enabled, stable user agent, loads the official Cloudflare hosted page,
  `addJavascriptInterface` bridge returns the token. States: Loading / Ready(token) / Error.
- `LoginScreen`: username + password fields, Turnstile, submit; link to register.
- `RegisterScreen`: username + password + confirm password + terms checkbox +
  Turnstile; link to login.
- Delete `EmailSignInScreen.kt`, `VerificationScreen.kt`, `VerificationCodeInput.kt`.
- Routes: add `Register`, remove `EmailSignIn`/`Verification`.
- Network (`ApiClient`/`LanternApiService`): `register(username, password, turnstileToken)`
  and `login(username, password, turnstileToken)`, responses aligned to the backend
  `Session` (accessToken/refreshToken/personaId/isStaff).
- Navigation: successful login or register navigates to Feed; `SessionViewModel`
  handles `isAuthenticated`.

## Admin Web

- `client.ts`: replace `sendEmailCode`/`createEmailSession` with
  `login(username, password, turnstileToken)`.
- `Login.tsx`: username/password form + official Turnstile JS widget
  (`https://challenges.cloudflare.com/turnstile/v0/api.js`), non-interactive mode,
  token submitted with the request.

## OpenAPI contract

- Add `/v1/auth/register`, `/v1/auth/login`; remove `/v1/auth/email-codes` and
  `/v1/auth/email-sessions`.
- Add `RegisterRequest` / `LoginRequest` (with `turnstileToken`); `Session` unchanged.
- `DELETE /v1/me` request: replace `verificationCode` with `password`.

## Testing

- Backend: rewrite `service_test.go` / `handler_test.go` for register/login (rate
  limit, username taken, wrong password, captcha failure). Update auth-token helper
  in content/identity/moderation handler tests to username/password + fake turnstile.
- CI (`go vet`, `go test`, `make validate-contract`) is the regression gate.
- Android: adapt `RootNavigationTest`, `DesignComponentsTest` to new screens.
