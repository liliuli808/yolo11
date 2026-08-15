# Anonymous Social Android Rebuild Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a new, globally deployable Android anonymous-social application that reproduces the observed product capabilities without reusing the original APK's code, branding, assets, accounts, or backend.

**Architecture:** Use a monorepo with a Kotlin/Jetpack Compose Android client, a modular Go API, PostgreSQL, Redis, S3-compatible storage, and an internal moderation console. The first releasable vertical slice covers email authentication, real profiles, anonymous personas, posts, topics, comments, reports, blocks, and moderation; chat, audio rooms, and payments are later slices.

**Tech Stack:** Kotlin, Jetpack Compose, Hilt, Retrofit/OkHttp, Room, DataStore, Go, Chi, pgx, PostgreSQL, Redis, MinIO for local development, S3-compatible object storage for production, OpenAPI, Docker Compose, GitHub Actions.

## Global Constraints

- Do not copy source code, assets, visual branding, account data, API credentials, endpoints, or protected copy from `一罐.apk`.
- Treat the APK only as behavior and navigation evidence. Record observations in `docs/reverse/`.
- Use a new development identifier, `app.rebuild.social`; make the final Android application ID, display name, domain, and sender address configuration values.
- All public content must be associated with an anonymous `persona`, never directly with an email address or real profile.
- Every public API lives under `/v1`; pagination is cursor based; mutating post APIs accept an idempotency key.
- Never commit secrets. Keep production settings in the deployment platform's secret store and document names only in `.env.example`.
- Database migrations use reserved numeric ranges: authentication `001-009`, identity `010-019`, content `020-029`, moderation `030-039`.
- Only the interface-contract agent edits `contracts/openapi/openapi.yaml`.
- Do not enable payments, withdrawals, or public audio rooms until their dedicated compliance and safety work packages are complete.

## Repository Layout

```text
apps/android/
apps/admin/
services/api/
contracts/openapi/
infra/
docs/reverse/
docs/architecture/
```

## Dependency Order

```text
APK evidence + product map + service scaffold
                    |
                    v
             OpenAPI/domain contract
                    |
       +------------+------------+
       |            |            |
       v            v            v
     auth       identity       content/media
       |            |            |
       +------------+------------+
                    |
                    v
          moderation + Android shell
                    |
                    v
          Android feature integration
                    |
                    v
        deployment, monitoring, E2E tests
```

## Work Package 0: Runtime APK Evidence

**Owner:** Agent A

**Files:**
- Create: `docs/reverse/page-map.md`
- Create: `docs/reverse/screen-inventory.md`
- Create: `docs/reverse/behavior-matrix.md`
- Create: `docs/reverse/screens/`

**Scope:** Finish the temporary Android SDK installation in `/tmp`, launch the APK in an isolated AVD, and document behavior. Do not create product source code and do not use original credentials or bypass authentication.

- [ ] Finish installing `platform-tools`, `platforms;android-35`, and `system-images;android-35;google_apis;x86_64` below `/tmp/android-sdk`.
- [ ] Create an AVD below `/tmp/android-avd` and boot it with networking enabled.
- [ ] Install `/home/jichi/yiguan/一罐.apk` with `adb install`.
- [ ] Capture each reachable screen at a fixed portrait resolution, including loading, empty, error, permission, and back-navigation behavior.
- [ ] Record every discovered route with: entry condition, visible data, allowed actions, back target, network dependency, and privacy-sensitive behavior.
- [ ] Produce a screen inventory covering startup, login, persona, feed, topic, post detail, profile, chat, audio room, wallet, and settings routes.

**Acceptance:** Every reachable route is documented with at least one screenshot and a behavior description. The documents contain no extracted source, proprietary graphics, or copied long-form copy.

## Work Package 1: Product Map and Original Visual Direction

**Owner:** Agent B

**Files:**
- Create: `docs/architecture/product-map.md`
- Create: `docs/architecture/design-tokens.md`
- Create: `docs/architecture/content-policy.md`

**Consumes:** `docs/reverse/page-map.md` and `docs/reverse/behavior-matrix.md`.

**Produces:** A page-by-page clean-room product map and a design-token contract consumed by Android and the admin console.

- [ ] Convert the reverse-engineering evidence into a new information architecture with original names and copy.
- [ ] Define tokens for color, typography, spacing, elevation, radius, icons, empty states, and moderation state treatments.
- [ ] Define the first-slice navigation: welcome, email login, persona switcher, feed, topic discovery, composer, post detail, profile, privacy, report, and block.
- [ ] Define content rules for prohibited content, reports, blocks, moderation outcomes, user deletion, and data export.
- [ ] Add screenshot acceptance criteria for every first-slice page.

**Acceptance:** Android implementers can build each page from this document without referencing the original brand or assets.

## Work Package 2: Service Scaffold and Local Infrastructure

**Owner:** Agent C

**Files:**
- Create: `services/api/go.mod`
- Create: `services/api/cmd/api/main.go`
- Create: `services/api/internal/platform/config/config.go`
- Create: `services/api/internal/platform/httpx/response.go`
- Create: `services/api/internal/platform/httpx/middleware.go`
- Create: `services/api/internal/platform/database/postgres.go`
- Create: `services/api/internal/platform/cache/redis.go`
- Create: `services/api/migrations/`
- Create: `infra/docker-compose.yml`
- Create: `infra/.env.example`
- Create: `Makefile`

**Interfaces:**
- `GET /healthz` returns `{ "status": "ok" }`.
- All API errors return `{ "code": "string", "message": "string", "requestId": "string" }`.

- [ ] Create a Go API using Chi with graceful shutdown, request IDs, structured logs, recovery middleware, CORS, and per-route rate-limit hooks.
- [ ] Start PostgreSQL, Redis, MinIO, and a local SMTP capture service with Docker Compose.
- [ ] Add configuration validation for `DATABASE_URL`, `REDIS_URL`, `S3_ENDPOINT`, `S3_BUCKET`, `JWT_SIGNING_KEY`, `EMAIL_FROM`, and `PUBLIC_BASE_URL`.
- [ ] Add migration tooling and a test database target.
- [ ] Add unit tests for configuration validation and HTTP error response shape.
- [ ] Add `make dev`, `make test`, `make migrate-up`, and `make lint` targets.

**Acceptance:** `make dev` starts the API and local dependencies; `GET /healthz` succeeds; missing required configuration causes startup failure with a clear error.

## Work Package 3: API Contract and Domain Model

**Owner:** Agent D

**Files:**
- Create: `contracts/openapi/openapi.yaml`
- Create: `docs/architecture/domain-model.md`
- Create: `docs/architecture/api-errors.md`

**Consumes:** Product map and service error response from Work Packages 1 and 2.

**Produces:** The canonical request/response contract consumed by all Go and Android feature agents.

- [ ] Define schemas for `User`, `Session`, `Persona`, `RealProfile`, `Topic`, `Post`, `MediaAsset`, `Comment`, `Reaction`, `Report`, `Block`, and `ModerationCase`.
- [ ] Define authentication endpoints: `POST /v1/auth/email-codes`, `POST /v1/auth/email-sessions`, `POST /v1/auth/refresh`, `DELETE /v1/auth/session`, and `DELETE /v1/me`.
- [ ] Define identity endpoints for profile and persona creation, update, list, switch, and deletion.
- [ ] Define topic, post, media-upload, comment, reaction, report, and block endpoints.
- [ ] Define cursor pagination, idempotency-key behavior, authorization rules, and exact error codes.
- [ ] Validate the OpenAPI document in CI.

**Acceptance:** The contract validates and exposes no field that maps an anonymous persona to email or real-profile data.

## Work Package 4: Email Authentication and Account Security

**Owner:** Agent E

**Files:**
- Create: `services/api/internal/auth/service.go`
- Create: `services/api/internal/auth/repository.go`
- Create: `services/api/internal/auth/handler.go`
- Create: `services/api/internal/auth/mailer.go`
- Create: `services/api/migrations/001_auth.sql`
- Create: `services/api/internal/auth/service_test.go`
- Create: `services/api/internal/auth/handler_test.go`

**Consumes:** OpenAPI authentication definitions and platform services.

**Produces:** Authenticated request context exposing `UserID` and session identity to later modules.

- [ ] Generate a six-digit email code, store only a keyed hash, expire it after ten minutes, and invalidate it after successful use.
- [ ] Apply independent limits by normalized email, IP address, and device fingerprint.
- [ ] Issue short-lived access tokens and rotating refresh tokens; store refresh-token hashes and session metadata.
- [ ] Implement logout, session revocation, account deletion request, and audit events.
- [ ] Provide a local SMTP adapter and a production SMTP-compatible adapter selected through configuration.
- [ ] Write unit tests for expiry, replay rejection, rate limits, token rotation, and revoked-session rejection.

**Acceptance:** Reusing a code fails, rate-limited requests return the contract error code, and deleting an account revokes every session.

## Work Package 5: Real Profiles and Anonymous Personas

**Owner:** Agent F

**Files:**
- Create: `services/api/internal/identity/service.go`
- Create: `services/api/internal/identity/repository.go`
- Create: `services/api/internal/identity/handler.go`
- Create: `services/api/migrations/010_identity.sql`
- Create: `services/api/internal/identity/service_test.go`

**Consumes:** Authenticated request context and identity OpenAPI definitions.

**Produces:** `PersonaID` selection for public actions and privacy-safe persona projection methods for content services.

- [ ] Store real profiles and personas in separate tables with separate response types.
- [ ] Implement persona creation, update, activation, deactivation, and current-persona selection.
- [ ] Ensure persona read models never include `user_id`, email, real-profile fields, or internal moderation notes.
- [ ] Add account export and deletion jobs with auditable lifecycle states.
- [ ] Test that a public persona API response cannot be joined to a real profile by API consumers.

**Acceptance:** A user can maintain multiple personas, but all public content APIs expose only the selected persona.

## Work Package 6: Topics, Posts, Media, and Interaction

**Owner:** Agent G

**Files:**
- Create: `services/api/internal/content/service.go`
- Create: `services/api/internal/content/repository.go`
- Create: `services/api/internal/content/handler.go`
- Create: `services/api/internal/media/service.go`
- Create: `services/api/internal/media/storage.go`
- Create: `services/api/migrations/020_content.sql`
- Create: `services/api/internal/content/service_test.go`
- Create: `services/api/internal/content/integration_test.go`

**Consumes:** Authenticated user, active persona, content OpenAPI definitions, and S3-compatible storage.

**Produces:** Read APIs consumed by the Android feed, topic, composer, detail, and profile features.

- [ ] Implement upload-intent creation with MIME type, file size, checksum, and signed-upload constraints.
- [ ] Model post lifecycle: `draft`, `pending_review`, `published`, `rejected`, `withdrawn`, and `deleted`.
- [ ] Implement topics, post creation, cursor feeds, post detail, comments, reactions, saves, and follow-aware visibility.
- [ ] Require an idempotency key for post creation and prevent duplicate publish operations.
- [ ] Filter hidden, blocked, deleted, and non-published content before every feed response.
- [ ] Test persona isolation, idempotent publish, cursor stability, blocked-author filtering, and moderation state transitions.

**Acceptance:** A user can publish a text/image post as a persona; it is visible only after moderation marks it published.

## Work Package 7: Reports, Blocks, and Moderation Console

**Owner:** Agent H

**Files:**
- Create: `services/api/internal/moderation/service.go`
- Create: `services/api/internal/moderation/repository.go`
- Create: `services/api/internal/moderation/handler.go`
- Create: `services/api/migrations/030_moderation.sql`
- Create: `apps/admin/package.json`
- Create: `apps/admin/src/main.tsx`
- Create: `apps/admin/src/features/moderation/`
- Create: `apps/admin/src/features/auth/`

**Consumes:** Content, persona, report, block, and moderation contract definitions.

**Produces:** Internal moderation actions that immediately affect public API visibility.

- [ ] Implement user and content reports with category, free-text reason, evidence references, reporter identity, and deduplication.
- [ ] Implement directional block relationships and enforce them in feeds, comments, profiles, and future chat interfaces.
- [ ] Implement moderation states for review, hide, restore, warn, suspend, and ban with immutable audit events.
- [ ] Build a minimal admin console for staff sign-in, queue filtering, report details, moderation actions, and audit history.
- [ ] Add end-to-end tests verifying that hidden content disappears from Android-facing APIs.

**Acceptance:** A report creates a queue entry; a moderator hide action makes the target unavailable to normal API callers without deleting audit history.

## Work Package 8: Android Foundation and Design System

**Owner:** Agent I

**Files:**
- Create: `apps/android/settings.gradle.kts`
- Create: `apps/android/build.gradle.kts`
- Create: `apps/android/app/build.gradle.kts`
- Create: `apps/android/app/src/main/AndroidManifest.xml`
- Create: `apps/android/app/src/main/java/app/rebuild/social/core/design/`
- Create: `apps/android/app/src/main/java/app/rebuild/social/core/network/`
- Create: `apps/android/app/src/main/java/app/rebuild/social/core/session/`
- Create: `apps/android/app/src/main/java/app/rebuild/social/navigation/`
- Create: `apps/android/app/src/test/`
- Create: `apps/android/app/src/androidTest/`

**Consumes:** Design tokens and OpenAPI contract.

**Produces:** A runnable Compose shell, dependency graph, session persistence, navigation, and test conventions for feature agents.

- [ ] Configure Kotlin, Compose, Hilt, Retrofit/OkHttp, Room, DataStore, and a build-config API base URL.
- [ ] Implement reusable typography, spacing, color, button, card, input, avatar, loading, empty, and error components.
- [ ] Implement a root navigation graph with authenticated and unauthenticated destinations.
- [ ] Implement API error mapping and session-refresh retry behavior.
- [ ] Add Compose tests for navigation, loading state, error state, and session restoration.

**Acceptance:** The Android app starts in an emulator, switches between fake and real API base URLs, and presents consistent loading/error UI.

## Work Package 9: Android Authentication and Identity Features

**Owner:** Agent J

**Files:**
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/auth/`
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/identity/`
- Create: `apps/android/app/src/androidTest/java/app/rebuild/social/feature/auth/`
- Create: `apps/android/app/src/androidTest/java/app/rebuild/social/feature/identity/`

**Consumes:** Android foundation, authentication APIs, and identity APIs.

**Produces:** Authenticated `UserSession` and active `Persona` state consumed by all public-content UI.

- [ ] Implement welcome, email entry, verification-code, session recovery, logout, and account-deletion screens.
- [ ] Implement real-profile editing and persona create/edit/switch/deactivate screens.
- [ ] Store access and refresh tokens only through encrypted Android storage abstractions.
- [ ] Ensure no email or real-profile field appears in persona cards, post composer, feed, or profile render models.
- [ ] Add UI tests for valid login, invalid code, rate-limit message, session refresh, persona switch, and offline error.

**Acceptance:** A user can log in through email, create a persona, switch it, and see only the active persona in public contexts.

## Work Package 10: Android Feed, Topic, Composer, and Profile Features

**Owner:** Agent K

**Files:**
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/feed/`
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/topic/`
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/composer/`
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/profile/`
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/safety/`
- Create: `apps/android/app/src/androidTest/java/app/rebuild/social/feature/feed/`

**Consumes:** Android foundation, active persona state, content APIs, moderation APIs, and the product map.

**Produces:** The first complete Android vertical slice.

- [ ] Implement cursor-paginated home, following, and topic feeds with refresh, loading, empty, and retry states.
- [ ] Implement post detail, comments, reactions, saves, topic exploration, and persona profile screens.
- [ ] Implement text/image composer with upload progress, draft state, selected persona, selected topic, and review status.
- [ ] Implement report, block, privacy, data-export, and account-deletion entry points.
- [ ] Compare every page against the new screenshot acceptance criteria from Work Package 1.
- [ ] Add UI tests for the complete journey: login, persona creation, post creation, moderation-visible feed, comment, report, and block.

**Acceptance:** The Android app completes the first-slice user journey against the real local API without fixture-only behavior.

## Work Package 11: Release Engineering and Public-Readiness Gate

**Owner:** Agent L

**Files:**
- Create: `infra/production/`
- Create: `docs/operations/runbook.md`
- Create: `docs/operations/incident-response.md`
- Create: `docs/legal/privacy-policy.md`
- Create: `docs/legal/terms-of-service.md`
- Create: `docs/legal/community-guidelines.md`
- Create: `.github/workflows/ci.yml`

**Consumes:** Working Android app, API, admin console, contract tests, and first-slice E2E flow.

- [ ] Configure production environment variables, S3-compatible storage, transactional email, domain, TLS, backups, and secret management.
- [ ] Add API metrics, structured log aggregation, error alerts, database backup verification, and object-storage lifecycle rules.
- [ ] Add CI checks for Go tests, Android unit/UI tests, OpenAPI validation, migration checks, and container builds.
- [ ] Document privacy, terms, content rules, deletion/export, moderation escalation, and on-call recovery steps.
- [ ] Run the full first-slice E2E scenario in a staging environment.

**Acceptance:** A new public user can register, post, be moderated, report/block another user, and request deletion; operators can diagnose failures from logs and metrics.

## Later Vertical Slices

Implement each of these only after Work Package 11 is accepted. Each gets its own contract, migration range, Android feature module, moderation rules, and E2E suite.

1. Realtime chat, flash chat, group chat, message persistence, read receipts, and WebSocket gateway.
2. Follows, discovery, nearby/city experiences, albums, recommendations, and notification center.
3. Audio rooms using a dedicated SFU; Go owns room policy, tokens, presence, moderation, and signaling, not media forwarding.
4. Virtual gifts, wallet ledger, balances, and creator income reporting.
5. Payments and withdrawals only after payment-provider, KYC, tax, AML, regional compliance, refund, and dispute requirements are approved.

## Agent Coordination Rules

- Run Work Packages 0, 1, and 2 in parallel.
- Start Work Package 3 after Work Packages 1 and 2 have produced their documents.
- Start Work Packages 4 through 8 after Work Package 3 is accepted; they may run in parallel only within their declared path boundaries.
- Start Work Packages 9 and 10 after their API dependencies are merged into a staging environment.
- Start Work Package 11 after the Android first-slice E2E test passes.
- Cross-module changes start with a proposed OpenAPI or domain-model change, then the contract owner approves it before implementation agents change their modules.
