# API Error Catalog — Lantern Anonymous Social Client

## Scope

This document lists the stable error codes returned by the Lantern API. Error codes
are used in the `code` field of the standard error envelope:

```json
{
  "code": "persona.not_found",
  "message": "Persona not found.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

All error responses are produced by the handler in
`services/api/internal/platform/httpx/response.go`. The HTTP status code and error
code are independent: the status indicates the response class, while the code gives
the client enough information to show a localized message or take corrective action.

## General errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `unauthorized` | 401 | No access token, malformed Authorization header, or expired/invalid token. | "Please sign in again." |
| `forbidden` | 403 | Authenticated principal lacks permission for the resource or action. | "You don't have permission to do that." |
| `not_found` | 404 | Resource or path does not exist. | "That doesn't exist." |
| `method_not_allowed` | 405 | HTTP method not allowed on the path. | "That action isn't supported here." |
| `conflict` | 409 | Generic state conflict not covered by a domain-specific code. | "That action conflicts with the current state." |
| `rate_limited` | 429 | Client exceeded the rate limit for the endpoint. | "Too many attempts. Please try again later." |
| `internal_error` | 500 | Unexpected server failure. | "Something went wrong. Please try again." |
| `not_implemented` | 501 | Endpoint is reserved for a future slice. | "That feature isn't available yet." |
| `validation.failed` | 400 | Request body failed schema validation (missing field, wrong type, etc.). | "Please check your input and try again." |

## Idempotency errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `idempotency.conflict` | 409 | Same `Idempotency-Key` reused with a different request body or principal. | "This request was already submitted with different data." |
| `idempotency.missing_key` | 400 | Mutating endpoint called without a required `Idempotency-Key` header. | "Please provide an idempotency key." |

## Auth errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `auth.invalid_email` | 400 | Email address is malformed or exceeds length limits. | "Please enter a valid email address." |
| `auth.invalid_code` | 401 | Verification code is wrong, expired, or exhausted allowed attempts. | "The code is incorrect or expired." |
| `auth.rate_limited` | 429 | Too many code requests or verification attempts for the email/IP. | "Too many attempts. Please try again later." |
| `auth.session_expired` | 401 | Refresh token expired or revoked. | "Your session expired. Please sign in again." |
| `auth.session_revoked` | 401 | Session was explicitly revoked. | "This session was signed out." |
| `auth.invalid_token` | 401 | Access token is malformed or signature invalid. | "Please sign in again." |

## Identity / real-profile errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `me.not_found` | 404 | Authenticated real profile no longer exists. | "Account not found." |
| `me.deletion_invalid_code` | 403 | Account deletion confirmation code is wrong or expired. | "The confirmation code is incorrect or expired." |
| `me.deletion_already_pending` | 409 | Deletion is already in progress. | "Account deletion is already in progress." |
| `me.email_change_invalid_code` | 403 | Email change confirmation code is wrong or expired. | "The confirmation code is incorrect or expired." |
| `me.email_already_used` | 409 | New email is already associated with another real profile. | "That email is already in use." |
| `me.export_rate_limited` | 429 | Data export requested within the 30-day cooldown. | "You can request another export in {days} days." |
| `me.export_not_ready` | 400 | Export download requested before status is `ready`. | "Your export isn't ready yet." |

## Persona errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `persona.not_found` | 404 | Persona ID does not exist or is not visible to the caller. | "That persona doesn't exist." |
| `persona.not_owned` | 403 | Caller does not own the persona. | "You don't manage this persona." |
| `persona.max_reached` | 403 | Real profile already has the maximum number of active personas. | "You've reached the persona limit." |
| `persona.alias_taken` | 409 | Alias is already in use by an active persona. | "That alias is already taken." |
| `persona.alias_disallowed` | 422 | Alias violates the content policy or profanity filter. | "That alias isn't allowed." |
| `persona.archived` | 400 | Action requires an active persona but the target is archived. | "This persona has been archived." |
| `persona.restricted` | 403 | Persona is temporarily restricted from publishing. | "This persona can't post right now." |
| `persona.default_required` | 400 | Action requires a default persona but none is set. | "Please select a default persona first." |

## Topic errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `topic.not_found` | 404 | Topic ID does not exist or is hidden. | "That channel doesn't exist." |
| `topic.already_followed` | 409 | Persona already follows the topic. | "You're already following this channel." |
| `topic.not_followed` | 409 | Persona does not follow the topic. | "You aren't following this channel." |
| `topic.hidden` | 403 | Topic is hidden by platform curators. | "This channel isn't available." |

## Post errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `post.not_found` | 404 | Post ID does not exist, is deleted, or is blocked. | "That note isn't available." |
| `post.not_author` | 403 | Caller is not the post's author. | "You can only edit your own notes." |
| `post.invalid_state` | 409 | Edit/delete attempted on a post in an incompatible state (e.g., rejected). | "This note can't be edited right now." |
| `post.content_disallowed` | 422 | Post content violates the content policy. | "This note couldn't be published due to safety guidelines." |
| `post.topic_required` | 422 | Post create/update is missing a topic. | "Please choose a channel." |
| `post.rate_limited` | 429 | Too many posts from this persona in a short window. | "You're posting too quickly. Please slow down." |

## Comment errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `comment.not_found` | 404 | Comment ID does not exist, is deleted, or is blocked. | "That reply isn't available." |
| `comment.not_author` | 403 | Caller is not the comment's author. | "You can only edit your own replies." |
| `comment.invalid_state` | 409 | Edit/delete attempted on a comment in an incompatible state. | "This reply can't be edited right now." |
| `comment.content_disallowed` | 422 | Comment content violates the content policy. | "This reply couldn't be posted due to safety guidelines." |
| `comment.parent_not_found` | 404 | Parent post does not exist or is deleted. | "The note you're replying to isn't available." |
| `comment.rate_limited` | 429 | Too many replies from this persona in a short window. | "You're replying too quickly. Please slow down." |

## Reaction errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `reaction.invalid_type` | 400 | Reaction type is not supported in this slice. | "That reaction isn't supported." |
| `reaction.already_exists` | 409 | Persona already reacted with the same type. | "You've already reacted to this." |
| `reaction.not_found` | 404 | Reaction to remove does not exist. | "Reaction not found." |
| `reaction.target_not_found` | 404 | Target post or comment does not exist. | "That note or reply isn't available." |

## Block errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `block.not_found` | 404 | Block record does not exist. | "Block not found." |
| `block.already_exists` | 409 | Caller already blocked the target persona. | "You've already blocked this persona." |
| `block.self` | 422 | Caller attempted to block their own persona. | "You can't block yourself." |

## Report errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `report.invalid_target` | 422 | Report target type/id combination is invalid. | "That can't be reported." |
| `report.target_not_found` | 404 | Reported post, comment, or persona does not exist. | "The reported content isn't available." |
| `report.duplicate` | 409 | Caller already has an open report for the same target. | "You've already reported this." |
| `report.details_required` | 422 | Category `other` selected without details. | "Please provide more details." |
| `report.self` | 422 | Caller attempted to report their own content. | "You can't report your own content." |

## Moderation errors

| Code | HTTP | When returned | Client-facing message guideline |
|------|------|---------------|---------------------------------|
| `moderation.case_not_found` | 404 | Moderation case ID does not exist. | "Case not found." |
| `moderation.invalid_outcome` | 422 | Outcome is not valid for the target type or state. | "That outcome isn't valid for this case." |
| `moderation.report_not_found` | 404 | Report referenced in a case does not exist. | "Report not found." |
| `moderation.not_moderator` | 403 | Caller lacks moderator role. | "You don't have moderator access." |

## HTTP status mapping

| Status | Meaning | Typical codes |
|--------|---------|---------------|
| 200 OK | Successful read or update. | (no error) |
| 201 Created | Resource created successfully. | (no error) |
| 202 Accepted | Async request accepted. | (no error) |
| 204 No Content | Success with empty body. | (no error) |
| 400 Bad Request | Malformed request or validation failure. | `validation.failed`, `idempotency.missing_key`, domain-specific validation codes. |
| 401 Unauthorized | Authentication missing or invalid. | `unauthorized`, `auth.*` |
| 403 Forbidden | Authenticated but not permitted. | `forbidden`, `*.not_owned`, `*.restricted`, `moderation.not_moderator` |
| 404 Not Found | Resource or endpoint not found. | `not_found`, `*.not_found` |
| 405 Method Not Allowed | HTTP method not allowed. | `method_not_allowed` |
| 409 Conflict | State conflict or duplicate. | `conflict`, `idempotency.conflict`, `*.already_*` |
| 422 Unprocessable Entity | Syntactically valid but semantically invalid. | `*.disallowed`, `*.required`, `*.invalid_*` |
| 429 Too Many Requests | Rate limit hit. | `rate_limited`, `auth.rate_limited`, `*.rate_limited` |
| 500 Internal Server Error | Unexpected server error. | `internal_error` |
| 501 Not Implemented | Reserved for future slices. | `not_implemented` |

## Client-facing message guidelines

1. **Do not expose internal details.** Messages should describe the user-facing
   condition, not database IDs, stack traces, or internal state.
2. **Use stable `code` values for localization.** Clients should map `code` to
   localized strings rather than parsing `message`.
3. **Keep messages actionable.** When possible, tell the user what to do next
   ("Please sign in again", "Please try again later").
4. **Respect the `Retry-After` header.** On 429 responses, clients should wait at
   least the specified number of seconds before retrying.
5. **Include `requestId` in support flows.** When showing a generic error message,
   offer the `requestId` for support or debugging.
6. **Avoid blame.** Use neutral language for policy blocks ("This note couldn't be
   published due to safety guidelines") rather than accusatory phrasing.

## Consistency with OpenAPI

The error codes in this document match the `code` values produced by the error
responses declared in `contracts/openapi/openapi.yaml`. Each endpoint references a
reusable response component (e.g., `#/components/responses/BadRequest`) that returns
the `Error` schema. Implementers must ensure that the `code` returned in production
appears in this catalog so clients can rely on it.
