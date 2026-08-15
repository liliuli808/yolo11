# Design Tokens — Lantern Anonymous Social Client

## Scope

This document defines the visual contract consumed by Android implementers and the
admin console. All values are original and avoid reproducing the source APK's colors,
graphics, or naming conventions.

Tokens are grouped by category. Each token includes a semantic name, a reference
purpose, and a default light-mode value. Dark-mode equivalents are listed where they
differ; a separate dark-mode pass is expected after the first slice is stable.

## Application identifier

- Android application ID: `app.rebuild.social`
- Product code name: **Lantern**

## Color palette

### Core brand colors

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `colorPrimary` | `#4F8A6E` | `#6FA68E` | Primary buttons, active tab, selected chips, links |
| `colorOnPrimary` | `#FFFFFF` | `#FFFFFF` | Text/icons on primary surfaces |
| `colorPrimaryContainer` | `#D7ECE3` | `#365C4C` | Selected chip background, faint emphasis |
| `colorOnPrimaryContainer` | `#0D3B2A` | `#D7ECE3` | Text on primary containers |
| `colorSecondary` | `#7B8FA6` | `#9DAFC2` | Secondary buttons, toggles, compose FAB ring |
| `colorOnSecondary` | `#FFFFFF` | `#FFFFFF` | Text/icons on secondary surfaces |
| `colorSecondaryContainer` | `#E3EAF1` | `#4A5A6A` | Secondary chip backgrounds |
| `colorOnSecondaryContainer` | `#1D2A36` | `#E3EAF1` | Text on secondary containers |

### Neutral surfaces

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `colorBackground` | `#FAFAF8` | `#121212` | App background behind scrollable content |
| `colorSurface` | `#FFFFFF` | `#1E1E1E` | Cards, sheets, dialogs, input backgrounds |
| `colorSurfaceVariant` | `#F1F1EE` | `#2A2A2A` | Elevated surfaces, disabled buttons, alternate rows |
| `colorOnSurface` | `#1C1C1A` | `#E8E8E6` | Primary text on surfaces |
| `colorOnSurfaceVariant` | `#5A5A56` | `#A8A8A4` | Secondary text on surfaces |
| `colorOutline` | `#D6D6D2` | `#474743` | Borders, dividers, disabled strokes |
| `colorOutlineVariant` | `#EBEBE7` | `#30302C` | Subtle separators |

### Semantic state colors

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `colorError` | `#B84444` | `#EF8A8A` | Errors, destructive actions, block/unblock |
| `colorOnError` | `#FFFFFF` | `#1C1C1A` | Text/icons on error surfaces |
| `colorErrorContainer` | `#F9DEDE` | `#5A2525` | Error banner background |
| `colorSuccess` | `#3A8A5F` | `#6BC295` | Success confirmations, report submitted |
| `colorSuccessContainer` | `#D7F0E3` | `#24533B` | Success banner background |
| `colorWarning` | `#C28A2A` | `#E8C06E` | Warnings, pending-review highlights |
| `colorWarningContainer` | `#F8EDD3` | `#5A451E` | Warning banner background |
| `colorInfo` | `#4A7FB8` | `#8BB8E8` | Informational hints, tips |
| `colorInfoContainer` | `#E0ECF8` | `#25405E` | Info banner background |
| `colorDisabled` | `#BDBDB8` | `#5E5E5A` | Disabled text/icon |
| `colorDisabledContainer` | `#E8E8E4` | `#353531` | Disabled background |

### Text colors

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `textPrimary` | `#1C1C1A` | `#E8E8E6` | Headlines, body, primary content |
| `textSecondary` | `#5A5A56` | `#A8A8A4` | Captions, metadata, placeholders |
| `textTertiary` | `#8A8A86` | `#6E6E6A` | Timestamps, disabled hints |
| `textOnPrimary` | `#FFFFFF` | `#FFFFFF` | Buttons, primary chip text |
| `textOnError` | `#FFFFFF` | `#1C1C1A` | Error button text |
| `textLink` | `#4F8A6E` | `#6FA68E` | Inline links |
| `textLinkVisited` | `#3A6A55` | `#86C2AC` | Visited inline links |

### Moderation state colors

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `moderationPublished` | transparent | transparent | Default published state (no extra tint) |
| `moderationPendingReview` | `#F8EDD3` | `#5A451E` | Note under review, reduced visibility |
| `moderationRejected` | `#F9DEDE` | `#5A2525` | Note rejected from feed, visible only to author |
| `moderationHidden` | `#F1F1EE` | `#2A2A2A` | Note hidden by author or policy, with overlay |
| `moderationWarned` | `#F8EDD3` | `#5A451E` | Account warning banner |
| `moderationSuspended` | `#F9DEDE` | `#5A2525` | Account suspended banner |

## Typography scale

Font family: use the platform default (`Roboto` on Android, system serif/sans in admin
console). Weight tokens map to Android `Typeface` / Compose `FontWeight` values.

| Token | Size | Line height | Weight | Letter spacing | Usage |
|-------|------|-------------|--------|----------------|-------|
| `textDisplayLarge` | 32 sp | 40 sp | 700 | -0.5 sp | Splash headline |
| `textDisplayMedium` | 28 sp | 36 sp | 700 | -0.25 sp | Empty-state headline |
| `textDisplaySmall` | 24 sp | 32 sp | 600 | 0 sp | Persona alias, large |
| `textHeadingLarge` | 20 sp | 28 sp | 600 | 0 sp | Screen title |
| `textHeadingMedium` | 18 sp | 26 sp | 600 | 0 sp | Section header |
| `textHeadingSmall` | 16 sp | 24 sp | 600 | 0.1 sp | Card title |
| `textBodyLarge` | 16 sp | 24 sp | 400 | 0.25 sp | Long-form body, composer input |
| `textBodyMedium` | 14 sp | 20 sp | 400 | 0.25 sp | Default body, note text |
| `textBodySmall` | 12 sp | 16 sp | 400 | 0.4 sp | Metadata, timestamps |
| `textLabelLarge` | 14 sp | 20 sp | 500 | 0.5 sp | Button label, chip text |
| `textLabelMedium` | 12 sp | 16 sp | 500 | 0.5 sp | Form labels, badge text |
| `textLabelSmall` | 11 sp | 14 sp | 500 | 0.6 sp | Overline, helper text |
| `textCaption` | 12 sp | 16 sp | 400 | 0.4 sp | Footnotes, character counter |

### Typography rules

- Minimum body text size: `textBodyMedium` (14 sp).
- `textBodyLarge` is the default for editable multi-line fields.
- Labels in buttons and chips must use `textLabelLarge`.
- Section headers use `textHeadingMedium` with `textPrimary`.
- Screen titles use `textHeadingLarge` with `textPrimary`.

## Spacing scale

All spacing values are in `dp` (Android) or `px` equivalent in admin console.

| Token | Value | Usage |
|-------|-------|-------|
| `space0` | 0 dp | No spacing |
| `space1` | 4 dp | Tight icon padding, hairline gaps |
| `space2` | 8 dp | Inline element gaps, small button padding |
| `space3` | 12 dp | Card internal padding, text-to-icon distance |
| `space4` | 16 dp | Default screen padding, card margin |
| `space5` | 20 dp | Relaxed list item padding |
| `space6` | 24 dp | Section breaks, dialog padding |
| `space7` | 32 dp | Large section separation |
| `space8` | 40 dp | Hero spacing, empty-state vertical gaps |
| `space9` | 48 dp | Major vertical rhythm |
| `space10` | 64 dp | Splash centering gaps |

### Spacing rules

- Horizontal screen margins default to `space4` (16 dp).
- Vertical gaps between cards default to `space3` (12 dp).
- Card internal padding defaults to `space3` (12 dp) horizontal and `space4` (16 dp) vertical.
- Bottom sheet internal padding defaults to `space6` (24 dp).
- Touch targets are at least 48 dp × 48 dp; use `space6` (24 dp) as minimum visual
  button height for non-filled targets.

## Elevation and shadow tokens

Use `elevationNone` for flat lists, `elevationSmall` for cards and bottom bars,
`elevationMedium` for bottom sheets and dialogs, `elevationLarge` for modals and FABs.

| Token | Shadow / elevation | Usage |
|-------|--------------------|-------|
| `elevationNone` | 0 dp | List rows, plain surfaces |
| `elevationSmall` | 2 dp | Cards, bottom app bar, channel tiles |
| `elevationMedium` | 4 dp | Bottom sheets, menus, snackbars |
| `elevationLarge` | 8 dp | Dialogs, modal cards, FAB |
| `elevationXLarge` | 16 dp | Full-screen modal scrim overlay |

### Shadow values (Compose / XML reference)

```xml
<!-- elevationSmall -->
<shape>
    <solid android:color="@color/colorSurface"/>
    <corners android:radius="@dimen/radiusMedium"/>
</shape>
```

For Compose, use `MaterialTheme.shapes` with the radius tokens below and apply
`shadowElevation` values of `2.dp`, `4.dp`, `8.dp`, or `16.dp`.

## Radius tokens

| Token | Value | Usage |
|-------|-------|-------|
| `radiusNone` | 0 dp | Full-bleed surfaces, dividers |
| `radiusSmall` | 4 dp | Small chips, badges |
| `radiusMedium` | 8 dp | Cards, input fields, buttons |
| `radiusLarge` | 12 dp | Bottom sheets, dialogs, channel tiles |
| `radiusXLarge` | 16 dp | Modals, large cards |
| `radiusFull` | 999 dp | Pills, FAB, avatars, full-round buttons |

### Radius rules

- Buttons: `radiusMedium` (8 dp) default, `radiusFull` for pill-shaped actions.
- Input fields: `radiusMedium`.
- Cards: `radiusMedium`.
- Bottom sheets and dialogs: `radiusLarge` top corners only.
- Persona avatars: `radiusFull` (circle).
- Small filter chips: `radiusFull`.

## Iconography conventions

### Style

- Default icon style: **outline**.
- Active/selected state: **filled** for navigation and toggle icons.
- Avoid mixing filled and outline in the same toolbar.

### Sizes

| Token | Size | Usage |
|-------|------|-------|
| `iconSmall` | 16 dp | Inline status icons, inline actions |
| `iconMedium` | 20 dp | Dense toolbars, chips |
| `iconDefault` | 24 dp | Standard toolbar, list icons |
| `iconLarge` | 32 dp | Empty-state icons, moderate emphasis |
| `iconXLarge` | 40 dp | Feature icons, empty states |
| `iconAvatarSmall` | 32 dp | Persona avatar in note rows |
| `iconAvatarMedium` | 48 dp | Persona avatar in profile header |
| `iconAvatarLarge` | 80 dp | Persona avatar in full profile |

### Color usage for icons

- Default icon color: `textSecondary`.
- Active icon color: `colorPrimary`.
- Destructive icon color: `colorError`.
- Disabled icon color: `colorDisabled`.
- Icon on primary surface: `colorOnPrimary`.

## Empty-state and loading-state treatments

### Empty state

| Element | Token |
|---------|-------|
| Container | `colorBackground` |
| Icon | `iconXLarge` with `colorSecondary` |
| Headline | `textDisplayMedium` with `textPrimary` |
| Body | `textBodyMedium` with `textSecondary` |
| Action button | `colorPrimary` with `textOnPrimary`, `textLabelLarge` |
| Vertical spacing | `space8` between icon and headline; `space3` between headline, body, and action |

### Loading state

- **Skeleton**: rounded rectangles using `colorSurface` with a `colorSurfaceVariant`
  shimmer overlay. Use `radiusMedium` for cards and `radiusFull` for avatars.
- **Spinner**: circular indeterminate progress with `colorPrimary` and size `iconLarge`.
- **Pull-to-refresh**: `SwipeRefreshLayout` indicator color set to `colorPrimary`.
- **Lazy list placeholders**: 3 skeleton note cards before first page loads.

### Error state

- Inline errors: `colorError` text below input using `textCaption`.
- Snackbar errors: `colorSurface` background, `textPrimary` text, `colorError` action
  text, `elevationMedium`.
- Full-screen error: empty-state layout with `colorError` `iconXLarge` and retry action.

## Moderation-state treatments

These treatments apply to notes, replies, and account banners.

| State | Background | Border/icon | Copy style | Visible to |
|-------|------------|-------------|------------|------------|
| `published` | Default | None | Normal | Everyone |
| `pendingReview` | `moderationPendingReview` | `colorWarning` 1 dp left border | `textBodyMedium` in `textSecondary` | Author + moderators; reduced distribution to others |
| `rejected` | `moderationRejected` | `colorError` 1 dp left border | `textBodyMedium` in `textSecondary` with "This note was removed." | Author only |
| `hidden` | `moderationHidden` | Dashed `colorOutline` border | `textSecondary` with "Hidden" label | Author + moderators |
| `warned` | `moderationWarned` | `colorWarning` icon | Banner `textBodyMedium` | Account owner |
| `suspended` | `moderationSuspended` | `colorError` icon | Banner `textBodyMedium` | Account owner |

### Banner component

- Position: anchored below top app bar or at top of profile.
- Background: `moderationWarned` or `moderationSuspended`.
- Icon: `iconDefault` `colorWarning` or `colorError`.
- Text: `textBodyMedium` `textPrimary`.
- Action (if appeal supported): text button in `colorPrimary`.

### Reduced-distribution card

- A note under review appears in the author's own stream with the pending-review
  treatment; to others it is withheld entirely until a moderator resolves it.
- A rejected note remains visible only to the author with the rejected treatment
  and an option to delete it.

## Component-specific token mappings

### Bottom navigation

| Element | Token |
|---------|-------|
| Background | `colorSurface` |
| Elevation | `elevationSmall` |
| Active icon | `colorPrimary` filled |
| Inactive icon | `textSecondary` outline |
| Active label | `textLabelSmall` `colorPrimary` |
| Inactive label | `textLabelSmall` `textSecondary` |
| Compose FAB | `colorPrimary` background, `colorOnPrimary` icon, `elevationLarge`, `radiusFull` |

### Note card

| Element | Token |
|---------|-------|
| Background | `colorSurface` |
| Elevation | `elevationNone` (flat list) or `elevationSmall` (separated grid) |
| Corner radius | `radiusMedium` |
| Internal padding | `space3` horizontal, `space4` vertical |
| Avatar | `iconAvatarSmall`, `radiusFull` |
| Alias | `textBodyMedium` `textPrimary` weight 500 |
| Channel tag | `textLabelSmall` `colorPrimary` in `colorPrimaryContainer` pill |
| Note text | `textBodyMedium` `textPrimary` |
| Metadata | `textCaption` `textTertiary` |
| Reply icon | `iconSmall` `textSecondary` |

### Input field

| Element | Token |
|---------|-------|
| Background | `colorSurface` |
| Border | `colorOutline`, 1 dp; focused `colorPrimary` |
| Corner radius | `radiusMedium` |
| Padding | `space3` horizontal, `space3` vertical |
| Text | `textBodyLarge` `textPrimary` |
| Hint | `textBodyLarge` `textTertiary` |
| Error text | `textCaption` `colorError` |

### Button

| Variant | Background | Text | Radius | Padding |
|---------|------------|------|--------|---------|
| Filled primary | `colorPrimary` | `textOnPrimary` | `radiusMedium` | `space3` horizontal, `space2` vertical |
| Filled secondary | `colorSecondary` | `textOnSecondary` | `radiusMedium` | `space3` horizontal, `space2` vertical |
| Tonal primary | `colorPrimaryContainer` | `colorOnPrimaryContainer` | `radiusMedium` | `space3` horizontal, `space2` vertical |
| Outlined | transparent, 1 dp `colorOutline` | `colorPrimary` | `radiusMedium` | `space3` horizontal, `space2` vertical |
| Text | transparent | `colorPrimary` | `radiusNone` | `space2` horizontal, `space1` vertical |
| Destructive | `colorError` | `textOnError` | `radiusMedium` | `space3` horizontal, `space2` vertical |
| Destructive text | transparent | `colorError` | `radiusNone` | `space2` horizontal, `space1` vertical |

### Dialog / bottom sheet

| Element | Token |
|---------|-------|
| Background | `colorSurface` |
| Scrim | 32 % black (`#00000052`) |
| Corner radius | `radiusLarge` (top only for bottom sheet) |
| Padding | `space6` |
| Title | `textHeadingMedium` `textPrimary` |
| Body | `textBodyMedium` `textSecondary` |
| Actions | `textLabelLarge`; primary action `colorPrimary`, destructive `colorError` |

## Dark mode considerations

The tokens above include dark-mode values. First-slice Android implementation should
prepare the token table so dark mode can be enabled by providing the dark column to
`MaterialTheme.colorScheme`. Admin console should consume the same JSON token source.
Recommended dark-mode extras:

- Status bar: use system default dark icons on `colorBackground`.
- Splash background: same as `colorBackground` dark.
- Shadows: reduce shadow opacity by ~30 % in dark mode; rely more on surface color
  shifts than shadows.

## Token delivery format

Android implementers should expose tokens as:

- `LanternColors.kt` / `colors.xml` for colors.
- `LanternType.kt` / `type.xml` for typography.
- `LanternSpacing.kt` / `dimens.xml` for spacing, elevation, radius, icon sizes.
- `LanternShapes.kt` / `shapes.xml` for shapes.

The admin console should read the same semantic names from a shared JSON file
(`docs/architecture/design-tokens.json`) if one is added later; until then, keep the
names in this document synchronized with any web implementation.

## Notes

- All color values are original hex selections, not extracted from the source APK.
- All names are generic/semantic to avoid brand-specific language.
- Moderation tokens are designed to be color-blind safe when paired with text and
  icon labels; do not rely on color alone to communicate moderation status.
