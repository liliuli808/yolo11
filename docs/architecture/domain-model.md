# Domain Model — Lantern Anonymous Social Client

## Scope

This document describes the domain model for the first slice of Lantern. It maps the
product concepts from `product-map.md` onto the data entities consumed and produced by
the API contract in `contracts/openapi/openapi.yaml`.

## Core concepts

| Concept | Description |
|---------|-------------|
| **Real profile** | The email-backed account. Owns personas, sessions, reports, and data exports. Never appears in public content. |
| **Persona** | The anonymous public mask. Authors posts, replies, reactions, and blocks. The only identity visible in public surfaces. |
| **Topic / Channel** | A curated thematic bucket. Users follow topics; posts are tagged with one topic. |
| **Post / Note** | A short public post authored by a persona and tagged with a topic. |
| **Comment / Reply** | A response to a post, authored by a persona. |
| **Reaction** | A lightweight engagement (first slice: `like`) attached to a post or reply. |
| **Block** | A directional, private relationship between two personas. |
| **Report** | A user-submitted complaint against a post, reply, or persona. |
| **Moderation case** | A moderator-facing record that groups reports and records an outcome. |

## Entity relationships

```text
RealProfile ||--o{ Persona       : owns
RealProfile ||--o{ Session       : has
RealProfile ||--o{ Report        : submits
RealProfile ||--o{ DataExport    : requests
Persona     ||--o{ Post          : authors
Persona     ||--o{ Comment       : authors
Persona     ||--o{ Reaction      : creates
Persona     ||--o{ Block         : initiates
Persona     ||--o{ Block         : targets
Topic       ||--o{ Post          : contains
Post        ||--o{ Comment       : has
Post        ||--o{ Reaction      : receives
Comment     ||--o{ Reaction      : receives
Report      }o--|| ModerationCase : grouped into
```

## Tables / collections

### `real_profiles`

The email-backed account root. Migration range: `010-019`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | Public identifier used only in private `/v1/me` contexts. |
| `email_hash` | String | Deterministic hash used for uniqueness and login lookup. Email cleartext is never stored unencrypted. |
| `email_encrypted` | Bytes | Encrypt-then-store email for export, deletion confirmation, and moderation legal requests. |
| `status` | Enum | `active`, `deleting`, `suspended`, `banned`. |
| `deletion_requested_at` | Timestamp | Null unless deletion has been confirmed. |
| `deletion_grace_period_ends_at` | Timestamp | 30 days after confirmation. |
| `created_at` | Timestamp | |
| `updated_at` | Timestamp | |

### `personas`

Anonymous public identities. Migration range: `010-019`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | Public identifier. |
| `real_profile_id` | UUID FK | Private; never exposed in public APIs or public persona objects. |
| `alias` | String | Unique display name, 1-32 chars. |
| `bio` | String | Optional, max 160 chars. |
| `avatar_seed` | String | Deterministic seed for generated avatar. |
| `avatar_color` | String | Hex color for generated avatar. |
| `status` | Enum | `active`, `restricted`, `archived`. |
| `is_default` | Boolean | Only one active persona per real profile may be default. |
| `note_count` | Integer | Denormalized counter, maintained by post lifecycle events. |
| `created_at` | Timestamp | |
| `updated_at` | Timestamp | |
| `archived_at` | Timestamp | Null unless archived. |

Indexes: `real_profile_id`, unique `(real_profile_id, alias)` where `archived_at IS NULL`.

### `sessions`

Bearer/refresh token sessions. Migration range: `001-009`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `real_profile_id` | UUID FK | |
| `refresh_token_hash` | String | Hash of the refresh token presented by the client. |
| `expires_at` | Timestamp | Refresh token expiry. |
| `ip_hash` | String | Optional hash of originating IP for abuse signals. |
| `user_agent_hash` | String | Optional hash of originating user agent. |
| `created_at` | Timestamp | |
| `revoked_at` | Timestamp | |

### `email_codes`

Short-lived verification codes for login, email change, and deletion. Migration range: `001-009`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `email_hash` | String | Lookup hash of the target email. |
| `code_hash` | String | Hash of the 6-digit code. |
| `purpose` | Enum | `login`, `email_change`, `deletion`. |
| `attempts` | Integer | Incremented on failed verification. |
| `max_attempts` | Integer | Hard cap before the code is invalidated. |
| `expires_at` | Timestamp | |
| `created_at` | Timestamp | |

### `topics`

Curated channels/topics. Migration range: `020-029`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `name` | String | Max 64 chars. |
| `description` | String | Max 256 chars. |
| `category` | String | e.g. `Everyday`, `Reflection`, `Creative`. |
| `status` | Enum | `active`, `hidden`. |
| `note_count` | Integer | Denormalized. |
| `follower_count` | Integer | Denormalized. |
| `created_at` | Timestamp | |
| `updated_at` | Timestamp | |

### `topic_follows`

Many-to-many follow relationship between personas and topics. Migration range: `020-029`.

| Field | Type | Notes |
|-------|------|-------|
| `persona_id` | UUID FK | |
| `topic_id` | UUID FK | |
| `created_at` | Timestamp | |
| PK | `(persona_id, topic_id)` | |

### `posts`

Notes/posts. Migration range: `020-029`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `persona_id` | UUID FK | Public author identity. |
| `topic_id` | UUID FK | |
| `content` | String | Max 2000 chars. |
| `moderation_state` | Enum | `published`, `pendingReview`, `rejected`, `hidden`, `deleted`. |
| `reaction_counts` | JSONB | Map of reaction type to count, e.g. `{"like": 12}`. |
| `reply_count` | Integer | Denormalized. |
| `created_at` | Timestamp | |
| `updated_at` | Timestamp | |
| `deleted_at` | Timestamp | Soft delete. |

Indexes: `persona_id`, `topic_id`, `created_at DESC`.

### `comments`

Replies/comments. Migration range: `020-029`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `post_id` | UUID FK | |
| `persona_id` | UUID FK | Public author identity. |
| `content` | String | Max 2000 chars. |
| `moderation_state` | Enum | `published`, `pendingReview`, `rejected`, `hidden`, `deleted`. |
| `reaction_counts` | JSONB | |
| `created_at` | Timestamp | |
| `updated_at` | Timestamp | |
| `deleted_at` | Timestamp | Soft delete. |

Indexes: `post_id`, `persona_id`.

### `reactions`

Engagements on posts and replies. Migration range: `020-029`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `target_type` | Enum | `post`, `comment`. |
| `target_id` | UUID | |
| `persona_id` | UUID FK | |
| `type` | Enum | `like` in first slice. |
| `created_at` | Timestamp | |
| `updated_at` | Timestamp | |
| UK | `(target_type, target_id, persona_id)` | One reaction per persona per target. |

### `media_assets`

Media metadata placeholder. Migration range: `020-029`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `persona_id` | UUID FK | Uploader. |
| `url` | String | |
| `mime_type` | String | |
| `width` | Integer | |
| `height` | Integer | |
| `thumbnail_url` | String | |
| `status` | Enum | `pending`, `ready`, `rejected`. |
| `created_at` | Timestamp | |

### `blocks`

Directional persona blocks. Migration range: `030-039`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `blocker_persona_id` | UUID FK | The persona initiating the block. |
| `blocked_persona_id` | UUID FK | The target persona. |
| `created_at` | Timestamp | |
| UK | `(blocker_persona_id, blocked_persona_id)` | |

### `reports`

User-submitted reports. Migration range: `030-039`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `reporter_real_profile_id` | UUID FK | Stored at real-profile level to prevent abuse. |
| `target_type` | Enum | `post`, `comment`, `persona`. |
| `target_id` | UUID | |
| `category` | Enum | Matches report categories in `content-policy.md`. |
| `details` | String | Optional, max 2000 chars. |
| `status` | Enum | `open`, `resolved`. |
| `resolved_at` | Timestamp | |
| `resolution_notice_sent_at` | Timestamp | |
| `created_at` | Timestamp | |

### `moderation_cases`

Moderator-facing cases. Migration range: `030-039`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `target_type` | Enum | `post`, `comment`, `persona`. |
| `target_id` | UUID | |
| `status` | Enum | `open`, `underReview`, `resolved`. |
| `outcome` | Enum | `noAction`, `warn`, `hide`, `remove`, `restrictPersona`, `suspendAccount`, `banAccount`. |
| `notes` | String | Moderator-only internal notes. |
| `created_at` | Timestamp | |
| `updated_at` | Timestamp | |

### `data_exports`

User data export requests. Migration range: `010-019`.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID PK | |
| `real_profile_id` | UUID FK | |
| `status` | Enum | `pending`, `ready`, `expired`. |
| `format` | Enum | `json`, `zip`. |
| `download_url_encrypted` | String | Encrypted download URL, decrypted at delivery time. |
| `ready_at` | Timestamp | |
| `expires_at` | Timestamp | 7 days after ready. |
| `created_at` | Timestamp | |

## Persona isolation rules

1. **Public attribution.** Every post, reply, reaction, and block is attributed to a
   persona. Real-profile identifiers never appear in public feeds, post detail,
   persona profiles, topic listings, or search results.
2. **API contract.** The `Persona` schema in `openapi.yaml` contains only
   `id`, `alias`, `bio`, `avatar`, `createdAt`, `noteCount`, and `isBlocked`. It does
   not contain `email`, `emailHash`, `realProfileId`, or moderation notes.
3. **Private persona context.** `PrivatePersona` (returned from `/v1/me/personas`)
   adds `isDefault`, `status`, and `updatedAt` for the owner, but still no real-profile
   linkage.
4. **Real profile boundary.** Real-profile data (`RealProfile`, `User`) is only
   returned from `/v1/me`. It includes the masked email and persona management
   metadata. Public content is never mixed into this response.
5. **Database enforcement.** Foreign keys from `personas` to `real_profiles` are
   resolved server-side and must never be serialized into public responses.
6. **Moderation visibility.** Moderators see persona aliases and content, not real
   names or emails, unless a legal/safety investigation explicitly requires it.

## Lifecycle states

### Real profile

```text
active ──► deleting ──► deleted (after grace period)
  │           ▲
  ▼           │
suspended ────┘ (login cancels deletion during grace period)
  │
  ▼
banned
```

- `active`: Normal operation.
- `deleting`: Deletion confirmed; 30-day grace period before purge.
- `suspended`: Temporary read-only; login still possible, content visible to owner.
- `banned`: Permanent; no new sessions.
- `deleted`: Final purged state; retained only in anonymized form where required.

### Persona

```text
active ──► restricted ──► active (restriction expiry)
  │
  ▼
archived
```

- `active`: Can author content and react.
- `restricted`: Cannot publish new posts or replies for a period.
- `archived`: Soft-deleted; historical content remains attributed, but no new actions.

### Post / Comment

```text
published ──► pendingReview ──► published / rejected / hidden
   │                              ▲
   └──────────────────────────────┘ (author or moderator can hide)
   │
   ▼
deleted (author action)
```

- `published`: Visible publicly.
- `pendingReview`: Under automated or human review; reduced distribution.
- `rejected`: Removed from public view; visible only to author with a removal notice.
- `hidden`: Hidden by author or moderator.
- `deleted`: Soft-deleted; shows as unavailable.

### Report

```text
open ──► resolved
```

- `open`: Not yet reviewed.
- `resolved`: A moderator has taken action or dismissed the report.

### Moderation case

```text
open ──► underReview ──► resolved
```

- `open`: Created from one or more reports.
- `underReview`: A moderator is actively investigating.
- `resolved`: An outcome has been applied.

## Idempotency

- Mutating endpoints accept an `Idempotency-Key` header (8-128 characters).
- The server caches the response keyed by `(idempotency_key, endpoint, authenticated_principal)`.
- Replays with the same key return the cached response without side effects.
- Keys expire after at least 24 hours.
- A key used for a different request body on the same endpoint returns
  `idempotency.conflict`.

## Cursor pagination

- List endpoints accept `cursor` (opaque) and `limit` (1-100, default 20).
- Responses wrap items in `data` plus `pagination: { nextCursor, hasMore }`.
- Cursors are opaque to clients and encode the sort column and offset/page boundary.
- Default ordering is descending by creation time (newest first).
- An absent or null `nextCursor` means there are no more pages.

## Privacy notes

- Email addresses are stored encrypted; only masked forms are returned to the client.
- Sessions are bearer tokens; refresh tokens are stored as hashes.
- Reports are linked to the reporter's real profile internally but never exposed.
- Blocks are directional and private; no notification is sent to the blocked party.
- Account deletion enters a 30-day grace period; personas are hidden during grace.
- Data exports include the requester's own data only and exclude other users' content.
- IP/email authentication logs are retained for 90 days then purged.

## Migration range mapping

| Domain area | Migration range |
|-------------|-----------------|
| Auth (sessions, email codes) | `001-009` |
| Identity (real profiles, personas, exports) | `010-019` |
| Content (topics, posts, comments, reactions, media) | `020-029` |
| Moderation (reports, blocks, moderation cases) | `030-039` |
