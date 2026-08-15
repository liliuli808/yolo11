# Content Policy — Lantern Anonymous Social Client

## Scope

This document defines the content rules for the first slice of the Lantern anonymous
social client. It governs user-generated notes and replies, reports, blocks,
moderation outcomes, account deletion, and data export. All language is original and
does not reproduce the source application's terms or policies.

## Core principle

Lantern is a persona-based sharing space. Every public action is tied to an anonymous
persona, not to a real profile. The policy exists to keep the space safe, respectful,
and legally compliant while preserving anonymity.

## Prohibited content categories

The following categories of content are not allowed in notes, replies, persona aliases,
bios, channel names, or any other user-submitted text.

| Category | Definition | Examples |
|----------|------------|----------|
| **Illegal activity** | Content that violates applicable law or promotes illegal acts. | Drug trafficking, fraud, hacking services, evasion of sanctions |
| **Violence and physical harm** | Threats, incitement, or graphic depictions of violence. | Death threats, assault instructions, gore, torture |
| **Sexually explicit material** | Pornography or sexual services involving adults; any sexual content involving minors is absolutely prohibited. | Nude imagery, sexual solicitations, links to adult services |
| **Child sexual abuse material (CSAM)** | Any visual, textual, or linked material sexualizing minors. | Sexualized images of children, grooming text, links to CSAM |
| **Non-consensual intimate imagery** | Intimate images or recordings shared without consent. | Revenge porn, hidden-camera imagery |
| **Harassment and bullying** | Targeted abuse, intimidation, or repeated unwanted contact. | Coordinated harassment, doxxing-adjacent targeting, stalking |
| **Hate speech** | Content that attacks people based on protected characteristics. | Racist, sexist, homophobic, transphobic, ableist slurs or incitement |
| **Self-harm and suicide** | Content that encourages, glorifies, or provides instructions for self-harm or suicide. | Suicide methods, pro-ana content, self-injury imagery |
| **Dangerous misinformation** | demonstrably false content that causes real-world harm. | Dangerous health hoaxes, election fraud incitement, false emergency claims |
| **Impersonation** | Pretending to be another person, brand, or official entity. | Fake celebrity accounts, fraudulent customer support personas |
| **Doxxing and privacy violations** | Sharing private identifying information without consent. | Home addresses, phone numbers, government IDs, private messages |
| **Spam and scams** | Repeated unsolicited content, phishing, financial fraud, or manipulation. | Pyramid schemes, phishing links, bot-generated replies, repetitive ads |
| **Intellectual-property infringement** | Sharing content that violates another party's copyright or trademark. | Pirated media, unauthorized brand logos |
| **Evading moderation** | Attempts to circumvent enforcement or automated filters. | Masked profanity, encoded CSAM hashes, ban evasion via new personas |

## Persona identity rules

- A persona alias must not impersonate a real individual, public figure, or official
  Lantern account.
- A persona alias must not contain slurs, sexual content, contact information, or
  promotional URLs.
- A persona bio must not link to external payment, chat, or adult services.
- One real profile may create up to a configurable maximum number of personas
  (default: 5). This limit reduces ban evasion.

## Channel rules

- Channels are curated by the platform in the first slice; users may follow them but
  may not create them.
- Channel names and descriptions must not contain prohibited content categories.
- A channel moderator (platform staff) may hide or restrict a channel.

## Report categories and required evidence

Users may report a note, a reply, or a persona. The report flow requires a category
and allows optional details.

| Category | Required evidence | Optional evidence |
|----------|-------------------|-------------------|
| **Harassment or bullying** | None required initially | Screenshots, note/reply IDs, context |
| **Hate speech** | None required initially | Targeted group, context |
| **Harmful or dangerous content** | None required initially | Description of harm |
| **Spam or scam** | None required initially | External links, patterns |
| **Sexual content** | None required initially | Description of explicit material |
| **Doxxing or privacy violation** | None required initially | What information was shared |
| **Impersonation** | Who is being impersonated | Supporting links |
| **Illegal content** | None required initially | Jurisdiction, law violated |
| **Other** | Description required | Any relevant context |

### Report handling

- Reports are stored with the reporter's real profile ID (not persona) to prevent
  abuse of the reporting system.
- The reported content and reporter identity are visible to moderators.
- A single user may submit one open report per target; duplicate reports are grouped.
- False or abusive reporting may result in a warning or suspension.

## Block behavior

Blocking in Lantern is **directional** and **private**.

| Aspect | Behavior |
|--------|----------|
| **Direction** | If Persona A blocks Persona B, A will not see B's content and B cannot interact with A's content. B is not notified that they were blocked. |
| **Visibility** | A's notes and replies are hidden from B. B's notes and replies are hidden from A. |
| **Interaction** | B cannot reply to A's notes, like A's notes, or view A's persona profile. A cannot do the same to B. |
| **Persistence** | Blocks are stored at the persona level, not the real-profile level. If A archives that persona, the block remains until explicitly removed. |
| **Reversal** | Either persona may unblock the other from the Block list in Privacy settings. |
| **Reporting overlap** | Blocking does not prevent either party from reporting content that was visible before the block. |

## Moderation outcomes

Moderators may apply the following outcomes to content or accounts.

| Outcome | Applies to | Effect | Notification |
|---------|------------|--------|--------------|
| **No action** | Note, reply, persona | None | None |
| **Warn** | Real profile | Account-level warning banner; no content removed | Email to real profile |
| **Hide note/reply** | Note or reply | Content hidden from public view; remains visible to author | In-app notice to author persona |
| **Remove note/reply** | Note or reply | Content removed from public view and author view; retained in moderation logs | In-app notice to author persona |
| **Restrict persona** | Persona | Persona cannot publish new notes or replies for a period | In-app notice |
| **Suspend account** | Real profile | All personas disabled; read-only access to own content | Email to real profile |
| **Ban account** | Real profile | Account permanently disabled; content retained per retention rules | Email to real profile |

### Appeals

- In the first slice, warnings and suspensions may be appealed once within 30 days.
- The appeal is submitted from the Privacy settings / Account status screen.
- Appeals are reviewed by a human moderator; the decision is final.
- Bans are not appealable in the first slice.

### Moderator transparency

- Authors receive an in-app notice when their content is hidden or removed, including
  the policy category violated.
- Reporters receive an in-app notice when a report is resolved (without revealing
  moderator actions beyond "reviewed").

## Account deletion

A user may delete their real profile from the Privacy settings screen.

| Action | Behavior |
|--------|----------|
| **Initiation** | User requests deletion; email verification code required to confirm. |
| **Grace period** | 30-day grace period during which the account is deactivated but not yet purged. The user may cancel deletion by logging in. |
| **Public effect during grace** | Personas are hidden; notes and replies become unattributed and marked "former member". |
| **Final deletion** | After the grace period, the real profile, email, and direct identifiers are purged. |
| **Content retention** | Anonymized notes and replies may be retained for legal, safety, and research purposes unless prohibited by law. |
| **Re-registration** | The same email may not be re-registered for 90 days after final deletion. |

## Data export

Users may request a copy of their data from Privacy settings.

| Included | Excluded |
|----------|----------|
| Real profile email (in cleartext) | Other users' content not authored by the requester |
| Persona aliases, bios, and avatar metadata | Other users' real profile data |
| Notes and replies authored by the requester's personas | Moderator-only notes and internal logs |
| Channel follow list | Report details about other users |
| Block list (persona IDs only) | Recommendation algorithms or derived scores |

| Property | Value |
|----------|-------|
| **Format** | Machine-readable JSON archive, optionally compressed as ZIP. |
| **Delivery** | Download link sent to verified email; link expires after 7 days. |
| **Frequency** | One export per 30 days per real profile. |
| **Preparation time** | Up to 48 hours; in-app notification when ready. |

## Content retention and anonymization

| Data type | Retention rule |
|-----------|----------------|
| Notes and replies | Retained indefinitely in anonymized form after account deletion unless the user requested deletion of all content and law permits removal. |
| Report records | Retained for 2 years after resolution for safety and legal compliance. |
| Moderation logs | Retained for 2 years. |
| Block records | Deleted when both personas are deleted or when the block is explicitly removed. |
| IP/email authentication logs | Retained for 90 days, then purged unless required by law. |
| Export archives | Deleted 7 days after generation or upon download, whichever is earlier. |

## Automated safety layers

- Text input is filtered for known prohibited patterns at compose time; matches block
  publish and show a generic safety message.
- Uploaded media (future slice) will be hashed and checked against known CSAM databases
  before storage.
- Repeated posting in a short window triggers rate limiting.
- New personas are rate-limited per real profile.

## Enforcement principles

- **Proportionality**: Minor first-time violations receive a warning; repeated or severe
  violations receive stronger action.
- **Consistency**: The same policy category is enforced the same way regardless of the
  persona's popularity or age.
- **Privacy**: Moderators see persona aliases, not real names or emails, unless a legal
  request or safety investigation requires it.
- **Appeal**: Users have a clear path to contest enforcement actions.

## First-slice implementation checklist

- [ ] Compose-time text filter for prohibited patterns.
- [ ] Report flow with category selection and optional details.
- [ ] Block/unblock flow with directional hide behavior.
- [ ] Moderation state machine for content: published, pendingReview, rejected,
      hidden, deleted. (`warned` and `suspended` are account/persona-level enforcement
      states, not content states.)
- [ ] Account deletion flow with email verification and 30-day grace period.
- [ ] Data export request flow with 30-day cooldown.
- [ ] In-app moderation notices for authors.
- [ ] Report resolution notices for reporters.

## Notes

- This policy is a first-slice baseline. Legal review is required before public launch.
- The policy intentionally omits chat, audio rooms, payments, and virtual currency;
  those surfaces are out of scope for the first slice.
- All terms are original and do not copy the source application's policy text.
