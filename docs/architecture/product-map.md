# Product Map — Lantern Anonymous Social Client

## Project identifier

- Android application ID: `app.rebuild.social`
- Internal product code name: **Lantern**
- This document describes the clean-room first-slice information architecture.

## Purpose

This product map converts the reverse-engineered behavior evidence into an original,
buildable page architecture. It covers the first releasable vertical slice only:

- Email authentication and verification
- Real profile and anonymous persona management
- Feed, topic discovery, composer, post detail
- Profile, privacy settings, report flow, and block flow

Later slices (chat, audio rooms, payments, onboarding slides, ads, teen mode) are
explicitly out of scope for this document and must not be designed into the first
slice.

## Core concepts

| Original concept | Lantern term | Meaning |
|------------------|--------------|---------|
| Real account | **Real profile** | The email-backed account. It owns personas but never appears in public content. |
| Anonymous identity | **Persona** | The public mask used to author posts and replies. A real profile may switch among personas. |
| Feed item | **Note** | A short public post authored by a persona. |
| Topic / mood | **Channel** | A thematic bucket that notes can be tagged with. Channels are curated by the platform. |
| Comment | **Reply** | A response to a note, authored by a persona. |

## Navigation model

The first slice uses a single-Activity architecture with a Navigation Component graph
for all authenticated destinations. The entry gate is a lightweight splash/check screen
that routes to the welcome flow when no session exists, or to the main tabs when a
session exists.

```
Cold start
  │
  ▼
Splash / gate
  │
  ├── No accepted privacy notice ──► Privacy notice
  │                                  Agree  ──► continue
  │                                  Decline ──► exit app
  │
  ├── No session ──► Welcome
  │                    │
  │                    ▼
  │                  Email login
  │                    │
  │                    ▼
  │                  Verification
  │                    │
  │                    ▼
  │                  Persona setup / switcher
  │                    │
  │                    ▼
  │                  Main tabs
  │
  └── Session exists ──► Persona switcher (if no default) or Main tabs

Main tabs (bottom navigation)
  ├── Stream (feed)
  ├── Discover (channels)
  └── Me (profile / settings)

Floating action (center) ──► Composer
```

## Page definitions

### 1. Splash / gate

| Field | Value |
|-------|-------|
| **Route id** | `splash-gate` |
| **Purpose** | Show a minimal branded placeholder while checking local session state and the accepted-privacy flag. No network boot config or ads in the first slice. |
| **Entry points** | Launcher icon, cold start, process restart |
| **Visible data** | Product logo/wordmark placeholder, version/build label (debug only) |
| **User actions** | None — automatic routing after local checks complete |
| **Empty/error states** | None; if session check fails, treat as logged out |
| **Back behavior** | Back exits the app |
| **Screenshot acceptance** | Logo centered; no original brand imagery, colors, or mascots; routes within 500 ms to welcome, privacy notice, or main tabs. |

### 2. Privacy notice

| Field | Value |
|-------|-------|
| **Route id** | `privacy-notice` |
| **Purpose** | Obtain explicit consent before any account data is collected or network calls are made. |
| **Entry points** | First launch, any launch where privacy flag is false, login entry gate |
| **Visible data** | Summary card: "Before you begin"; bullet points covering Terms of Service, Privacy Policy, and data use; two bottom actions |
| **User actions** | Agree and continue, Decline and exit, open linked policy documents in an in-app web view |
| **Empty/error states** | None |
| **Back behavior** | Back is blocked until the user taps Agree or Decline; Decline exits the app |
| **Screenshot acceptance** | Modal or full-screen card; agree button uses `colorPrimary`; decline button uses `textSecondary`; links are underlined; no pre-ticked checkbox. |

### 3. Welcome

| Field | Value |
|-------|-------|
| **Route id** | `welcome` |
| **Purpose** | Introduce the product value proposition and route to email login. |
| **Entry points** | Splash gate when no session exists |
| **Visible data** | Illustration placeholder, headline, one-line value statement, primary "Continue with email" button, secondary link to privacy notice |
| **User actions** | Continue to email login, open privacy notice |
| **Empty/error states** | None |
| **Back behavior** | Back exits the app |
| **Screenshot acceptance** | Illustration is a generic geometric placeholder (no original art); headline is original copy; single primary CTA; no social-login icons in first slice. |

### 4. Email login

| Field | Value |
|-------|-------|
| **Route id** | `email-login` |
| **Purpose** | Collect the user's email address to start authentication. |
| **Entry points** | Welcome, deep link to login, back from verification |
| **Visible data** | Email input field, keyboard type `email`, inline validation hint, "Send code" primary action, link to privacy notice |
| **User actions** | Type email, submit to request verification code, open privacy notice |
| **Empty/error states** | Empty field disables submit; invalid email shows inline error; network error shows a snackbar with retry |
| **Back behavior** | Back returns to Welcome |
| **Screenshot acceptance** | Email field uses `textField` style; submit disabled until valid email; error color is `colorError`; no phone number field, no third-party login icons. |

### 5. Verification

| Field | Value |
|-------|-------|
| **Route id** | `email-verification` |
| **Purpose** | Verify email ownership via a one-time code. |
| **Entry points** | Email login after submit |
| **Visible data** | Masked email, 6-digit code input, countdown timer, resend action (disabled until timer expires) |
| **User actions** | Enter code, resend code, change email address |
| **Empty/error states** | Wrong/expired code shows inline error; too many attempts routes back to email login with explanation; network error shows snackbar with retry |
| **Back behavior** | Back returns to Email login |
| **Screenshot acceptance** | Code field is 6 boxes or single field with monospace digits; masked email shows first 2 and last 2 characters (e.g., `ab•••••@example.com`); resend disabled state uses `textDisabled`; success auto-advances. |

### 6. Persona switcher / creation

| Field | Value |
|-------|-------|
| **Route id** | `persona-switcher` |
| **Purpose** | Let the user pick an existing anonymous persona or create a new one before entering public areas. |
| **Entry points** | After first verification, after launching with session but no default persona, from Profile |
| **Visible data** | List of existing personas (avatar placeholder + alias + creation date), "New persona" card, maximum-persona hint |
| **User actions** | Select persona to enter main tabs, create new persona, delete/archive persona (with confirmation), set default persona |
| **Empty/error states** | Empty state: "You don't have a persona yet" with "Create one" action; max personas reached disables creation with explanation |
| **Back behavior** | Back is blocked if the user has no persona; otherwise back exits to launcher (logout required to change account) |
| **Screenshot acceptance** | Each persona card shows a generated color avatar and original alias; selected persona has a check indicator; create card uses `colorSecondary` outline; no real-name info on this screen. |

### 7. Persona creation

| Field | Value |
|-------|-------|
| **Route id** | `persona-create` |
| **Purpose** | Create a new anonymous public identity. |
| **Entry points** | Persona switcher |
| **Visible data** | Alias input, auto-generated color/avatar preview, optional short bio, guidelines link |
| **User actions** | Edit alias, regenerate avatar, enter bio, save, cancel |
| **Empty/error states** | Alias required; profanity filter blocks disallowed aliases with inline error; duplicate alias handled server-side |
| **Back behavior** | Back discards and returns to Persona switcher |
| **Screenshot acceptance** | Avatar is a deterministic color + initial shape (not a photo picker); alias max length visible; save disabled until valid; no connection to real profile fields. |

### 8. Main tabs

| Field | Value |
|-------|-------|
| **Route id** | `main-tabs` |
| **Purpose** | Host the three first-slice bottom-navigation destinations and the composer entry point. |
| **Entry points** | Persona switcher after selection, cold start with session and default persona |
| **Visible data** | Bottom bar with Stream, Discover, Me; center elevated "Compose" button |
| **User actions** | Tap tab to switch, tap compose to create a note |
| **Empty/error states** | None |
| **Back behavior** | Back from any tab exits the app |
| **Screenshot acceptance** | Bottom bar uses `colorSurface` and `elevationSmall`; active tab uses `colorPrimary`; inactive uses `textSecondary`; compose button is elevated and uses `colorPrimary`. |

### 9. Stream (feed)

| Field | Value |
|-------|-------|
| **Route id** | `stream` |
| **Purpose** | Browse notes from the selected channel or all channels. |
| **Entry points** | Main tabs default, deep link to a channel |
| **Visible data** | Top channel filter chips, pull-to-refresh, infinite-scroll note list, note cards (alias, channel tag, text, reply count, timestamp) |
| **User actions** | Scroll, pull to refresh, tap channel chip to filter, tap note to open detail, long-press own note to edit/delete, tap compose |
| **Empty/error states** | Empty: "No notes here yet. Be the first to share." with compose action; error: snackbar with retry; loading: skeleton list |
| **Back behavior** | Back exits app from main tabs |
| **Screenshot acceptance** | Channel chips scroll horizontally; selected chip uses `colorPrimaryContainer`; note cards use `radiusMedium` and `elevationNone`; avatar placeholders are color-generated; timestamps use `textTertiary`. |

### 10. Discover (channels)

| Field | Value |
|-------|-------|
| **Route id** | `discover` |
| **Purpose** | Find and follow channels. |
| **Entry points** | Main tabs second tab |
| **Visible data** | Search field, category sections (e.g., Everyday, Reflection, Creative), channel tiles with name, description, note count, follow state |
| **User actions** | Search channels, tap channel to open channel feed, follow/unfollow channel, browse categories |
| **Empty/error states** | Search empty: "No channels match your search"; general empty: "More channels coming soon" |
| **Back behavior** | Back exits app from main tabs |
| **Screenshot acceptance** | Search field uses `searchField` style; channel tiles use `radiusLarge`; followed state shows filled icon + `colorPrimary`; categories use `textHeading` section headers. |

### 11. Channel feed

| Field | Value |
|-------|-------|
| **Route id** | `channel-feed` |
| **Purpose** | Show notes scoped to a single channel. |
| **Entry points** | Tap channel from Stream filter chips, Discover, or deep link |
| **Visible data** | Channel header (name, description, follower count), note list, follow/unfollow action |
| **User actions** | Follow/unfollow, scroll notes, tap note, compose into this channel |
| **Empty/error states** | Empty: "This channel is quiet. Add a note."; error: snackbar with retry |
| **Back behavior** | Back returns to Stream or Discover depending on entry point |
| **Screenshot acceptance** | Header uses `colorSurface` with `elevationSmall`; follow button uses `colorPrimary` when not following, `colorSurfaceVariant` when following; note list matches Stream. |

### 12. Composer

| Field | Value |
|-------|-------|
| **Route id** | `composer` |
| **Purpose** | Create a new note as the active persona. |
| **Entry points** | Center compose button, "Be the first" empty state, "Add a note" from channel feed |
| **Visible data** | Multi-line text input, character counter, channel selector, active persona indicator, publish action |
| **User actions** | Type note, select channel, switch persona (pre-filled from main tabs), publish, save draft, discard with confirmation |
| **Empty/error states** | Empty note disables publish; over limit shows `colorError` counter; channel required if none pre-selected; network error shows retry snackbar |
| **Back behavior** | Back with unsaved changes shows discard confirmation; back after publish returns to Stream |
| **Screenshot acceptance** | Input area uses `colorSurface` with `radiusMedium`; character counter uses `textTertiary`; channel selector shows current channel name with dropdown chevron; persona chip shows alias + avatar. |

### 13. Post detail

| Field | Value |
|-------|-------|
| **Route id** | `note-detail` |
| **Purpose** | Read a note and its replies, and take moderation actions. |
| **Entry points** | Tap note in Stream/Channel feed/Profile, deep link |
| **Visible data** | Note author alias, channel tag, note text, timestamp, reply count, reply list (alias, text, timestamp), own-content edit/delete actions |
| **User actions** | Reply to note, like note, report note or reply, block author, copy text, share link, edit/delete own note or reply |
| **Empty/error states** | No replies: "No replies yet. Start the conversation."; deleted note: "This note is no longer available."; blocked author: note hidden with "Unblock to see" action |
| **Back behavior** | Back returns to the calling list |
| **Screenshot acceptance** | Author alias is tappable and opens persona profile; report/block available via overflow menu; reply input pinned to bottom with send button; timestamps align to `textTertiary`; moderation states use `moderation-*` tokens. |

### 14. Persona profile

| Field | Value |
|-------|-------|
| **Route id** | `persona-profile` |
| **Purpose** | Show public information about a persona. |
| **Entry points** | Tap alias in Stream or note detail |
| **Visible data** | Persona avatar, alias, bio, joined date, note count, recent notes |
| **User actions** | Follow persona (future slice), view notes, report persona, block persona |
| **Empty/error states** | No public notes: "This persona hasn't shared anything yet."; blocked persona: profile masked with unblock action |
| **Back behavior** | Back returns to caller |
| **Screenshot acceptance** | Avatar is large generated shape; alias uses `textDisplaySmall`; bio uses `textBody`; note list matches Stream; no real-name, email, or phone fields shown. |

### 15. Real profile

| Field | Value |
|-------|-------|
| **Route id** | `real-profile` |
| **Purpose** | Manage the email-backed account and its personas. |
| **Entry points** | Me tab "Profile" row, settings |
| **Visible data** | Email (masked), persona list, created-at date |
| **User actions** | Switch default persona, add/edit/archive persona, change email, request data export, delete account |
| **Empty/error states** | None |
| **Back behavior** | Back returns to Me tab |
| **Screenshot acceptance** | Email masked like `a•••@example.com`; persona list uses switcher cards; destructive actions grouped at bottom with `colorError`; no public content appears here. |

### 16. Privacy settings

| Field | Value |
|-------|-------|
| **Route id** | `privacy-settings` |
| **Purpose** | Control visibility and consent settings. |
| **Entry points** | Me tab, settings list |
| **Visible data** | Block list entry, data export entry, account deletion entry, privacy policy link, analytics consent toggle (if applicable) |
| **User actions** | Open block list, request export, delete account, review privacy policy, toggle optional consent |
| **Empty/error states** | None |
| **Back behavior** | Back returns to Me tab or settings |
| **Screenshot acceptance** | Grouped list with `textHeading` section titles; destructive rows use `colorError` text; toggle uses `colorPrimary` when on. |

### 17. Block list

| Field | Value |
|-------|-------|
| **Route id** | `block-list` |
| **Purpose** | Review and manage blocked personas. |
| **Entry points** | Privacy settings |
| **Visible data** | List of blocked personas with alias, avatar, block date, unblock action |
| **User actions** | Unblock persona (with confirmation) |
| **Empty/error states** | Empty: "You haven't blocked anyone." |
| **Back behavior** | Back returns to Privacy settings |
| **Screenshot acceptance** | Each row shows avatar + alias + `textTertiary` block date; unblock button is text-button style in `colorError`. |

### 18. Report flow

| Field | Value |
|-------|-------|
| **Route id** | `report` |
| **Purpose** | Submit a report against a note, reply, or persona. |
| **Entry points** | Overflow menu in note detail, persona profile, reply row |
| **Visible data** | Reported target summary, category list, optional details field, submit action |
| **User actions** | Select category, enter details, submit report, cancel |
| **Empty/error states** | Category required; network error shows retry; success shows confirmation and routes back |
| **Back behavior** | Back returns to caller; success confirmation auto-dismisses |
| **Screenshot acceptance** | Category list uses radio buttons or selectable rows; details field uses `textField` multiline; submit disabled until category selected; confirmation uses `colorSuccess` icon. |

## First-slice screenshot acceptance criteria summary

| Page | Screenshot must show |
|------|---------------------|
| Splash / gate | Original placeholder logo; no original branding; routes within 500 ms. |
| Privacy notice | Two clear consent actions; no pre-ticked boxes; linked policies. |
| Welcome | Original headline/copy; email-login CTA only; no third-party login icons. |
| Email login | Email input only; inline validation; no phone/password fields. |
| Verification | 6-digit code input; masked email; disabled resend with countdown. |
| Persona switcher | Anonymous persona cards; create action; no real-name fields. |
| Persona creation | Alias + generated avatar/bio; no photo picker tied to real profile. |
| Main tabs | Three tabs + elevated compose; no chat/message/payment tabs. |
| Stream | Channel chips + note cards; no chat, audio, or payment surfaces. |
| Discover | Search + channel categories; no H5/web surfaces. |
| Channel feed | Channel header + scoped note list. |
| Composer | Text + channel + persona selector; character limit visible. |
| Note detail | Note + replies + reply input + report/block overflow. |
| Persona profile | Avatar + alias + bio + persona notes; no real identity. |
| Real profile | Masked email + persona management; no public content. |
| Privacy settings | Block list, export, deletion entries; policy link. |
| Block list | Blocked personas with unblock action. |
| Report | Category list + details + confirmation. |

## Out-of-scope (later slices)

- Real-time chat and direct messages
- Audio rooms / live streaming
- Virtual currency or payment flows
- Social login providers
- Splash ads and boot-config fetching
- Push notification settings beyond a placeholder
- Teen mode / age gating
- Onboarding carousel slides
- Location-based matching
- Group chat rooms

## Notes

- All copy is original and does not reproduce strings from the source APK.
- No original graphics, icons, color values, or API endpoints are referenced.
- Every public-facing action is attributed to a persona, never to the real profile.
