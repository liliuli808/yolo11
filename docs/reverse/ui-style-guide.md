# 一罐 UI Style Guide — Design Tokens (Reverse Engineered)

> **Clean-room note:** This document records *visual/UX evidence only* for replicating the
> frontend look-and-feel. Original identifiers, endpoints, assets, branding, and code must
> NOT be copied into `app.rebuild.social`. Colors/sizes below are empirical reference data.

Source: jadx decode of `一罐.apk` v3.16.10 → `decompiled/jadx-out/resources/res/values/`
(`styles.xml` AppTheme / AppTheme.Dark, `colors.xml`, `dimens.xml`, `attrs.xml`).

## 1. Brand accent (primary blue)

The app's accent color is **`#56a5ff`** (light blue). Related variants:

| Token | Light | Dark |
|-------|-------|------|
| `blue` (accent) | `#56a5ff` | `#56a5ff` |
| `blue_active` / `globalLight65A5FF` | `#65a5ff` | selected-tab tint |
| `light_blue` (tab default) | `#cbe1ff` | — |
| `color_70cea7` (green accent) | `#70cea7` | `#e570cea7` |

## 2. Global theme (surfaces & text)

AppTheme (light) and AppTheme.Dark (night) define the shared attribute surface:

| Attribute | Light | Dark | Usage |
|-----------|-------|------|-------|
| `windowBackground` | `#ffffff` | `#1a1a1a` | Root |
| `globalBackgroundColor` | `#ffffff` | `#1a1a1a` | Page bg |
| `globalGrayBackgroundColor` | `#f7f7f7` | `#1a1a1a` | Gray sections |
| `globalCardTintColor` | `@null` (none) | `#262626` | Card bg |
| `globalTitleTextColor` | `#333333` | `#d3d3d3` | Page titles |
| `globalContentTextColor` | `#333333` | `#d3d3d3` | Primary text |
| `globalNewContentTextColor` | `#262626` | `#d3d3d3` | Body text alt |
| `globalSubTextColor` | `#666666` | `#808080` | Secondary text |
| `globalDescTextColor` | `#999999` | `#808080` | Tertiary/desc |
| `globalCardDividerLineColor` | `#f2f2f2` | `#19ffffff` | Card dividers |
| `globalUnderlineColor` | `#ececec` | `#333333` | Hairlines/underlines |
| `globalTipLinkColor` | `#56a5ff` | `#808080` | Links / tips |
| `globalTabDefaultTextColor` | `#333333` | `#808080` | Unselected tab |
| `globalTagTextColor` | `#333333` | `#808080` | Tag chips |
| `navigationDefaultColor` | `#333333` | `#808080` | Bottom nav idle |
| `navigationBackgroundColor` | transparent | `#262626` | Bottom nav bg |
| `toolBarTitle` | `#333333` | `#d3d3d3` | Top bar title |

Additional fixed grays: `c808080=#808080`, `c262626=#262626`, `cd3d3d3=#d3d3d3`,
`cececec=#ececec`, `white_70=#b3ffffff`, `black_70=#b3000000`.

## 3. Index / main tab top bar (sticky header)

The 一罐 tab top header is a **blue (#56a5ff) expanded area** that collapses to white:

| Attribute | Light | Dark |
|-----------|-------|------|
| `indexTopExpandedColor` | `#56a5ff` (blue) | `#262626` |
| `indexTopExpandedTitleColor` | `#ffffff` | `#d3d3d3` |
| `indexTopCollapsingColor` | `#ffffff` | `#1a1a1a` |
| `indexTitleColor` | `#333333` | `#d3d3d3` |
| `indexDiaryTitleColor` | `#ffffff` | `#d3d3d3` |
| `toolBarBackground` | `#ffffff` | transparent |

So the diary header behaves like a Material CollapsingToolbar: blue while expanded
(white titles), white after collapse. Mood/`全部` chip on the expanded header uses
`indexDiaryMoodTabNormalColor`/`SelectedColor` = `#ffffff` (light), `#808080`/`#56a5ff` (dark).

## 4. Chat / IM surfaces

| Token | Light | Dark |
|-------|-------|------|
| Message bubble (self) | white text `#ffffff` on `shape_bg_chat_text_self` | `#e6e6e6` on `..._dark` |
| Message bubble (other) | `#666666` text on `shape_bg_chat_text_customer` | `#d3d3d3` on `..._dark` |
| System notices | `#666666` w/ `shape_chat_notice` border | `#808080` w/ `..._dark` |
| `chatRoomTitleColor` | `#56a5ff` | `#d3d3d3` |
| `inputPanelHintColor` | `#33000000` | `#80808080` |
| `inputPanelTextColor` | `#333333` | `#d3d3d3` |
| Session preview (`indexSessionContentTextColor`) | `#666666` | `#808080` |
| Chat input field (`shape_chat_input`) | rounded field | dark variant |
| Link color inside bubbles | `#56a5ff` | — |

## 5. Navigation & toolbar chroma

| Attribute | Light | Dark |
|-----------|-------|------|
| `navigationUnreadBorderColor` | `#ffffff` | `#262626` |
| `globalSwitchBackgroundColor` | `#ffffff` | `#1a1a1a` |
| `android:navigationBarColor` | `#cccccc` | `#262626` |
| `loginTitleBackgroundColor` | `#56a5ff` | `#262626` |
| `loginSubmitButtonDisable` | `shape_blue_button_disable` | `..._dark` |

## 6. Layout cadence (from dimens.xml)

- Tab strip indicator: selected indicator height `0dp` — **no underline on the top TabLayout**,
  tab strip uses the `SlidingTabLayout` (TabLayout) w/ custom colors.
- Cards use radius helpers `shape_rounded_12_*` (`12dp` corners) and `shape_rounded_4_*` (`4dp`).
- Avatar views at 32dp / rounded square chips for mood.

### Text size scale (`dimens.xml`)

The app-wide type scale (`text_*` + literal sp values observed in layouts):

| Token / value | Usage |
|---------------|-------|
| `text_title` 18sp | `TitleView` centered app-bar title (bold) |
| `text_common` 15sp | body text, list rows, buttons, inputs |
| `text_small` 14sp | secondary rows, action labels, comment counts |
| `text_tiny` 12sp | tertiary/labels, timestamps alt, counters |
| `text_mini` 11sp | HOT/RULES badges, small chips |
| 20sp bold | login page titles, profile `realName` |
| 18sp | login phone/password inputs |
| 16sp bold | comment section header, mood field labels |
| 24sp bold | 一罐 collapsed title strip |
| TabTextSize16 | mood tab strip text (16dp via Material style) |

Spacing scale (`size_*`): 1dp hairline, 4/5/6/8/12/15/16/20/28/32/48/64dp — with 16dp as
the standard content horizontal padding and 20dp for editor/row inner padding.

## 7. Key notes for the rebuild

1. Entire theme is driven by a **custom attribute set** (`global*` re-mappable per
   day/night) — mirror this in Compose with a small design-token layer instead of hard-coding.
2. Day/night switching is supported via `AppTheme` / `AppTheme.Dark` (light `#fff` surfaces,
   dark `#1a1a1a` surfaces with `#d3d3d3` text, muted `#808080` secondary).
3. The famous "全部/心情 filter + collapsing blue header" on the diary feed is the signature
   interaction: blue (`#56a5ff`) expanded → white collapsed, content `#333333`→`#d3d3d3`;
   mood tab strip sits *inside* the header on `?indexTopExpandedColor` (blue when expanded,
   darkened `#262626` surfaces in night), unselected text `#cbe1ff` light / `#808080` dark.
4. Buttons/badges: blue capsule `#56a5ff`, green accent `#70cea7` for "green group" diary albums.
5. Status/gold accents: pink error `#ff8a8f`, red `#ff6565` ("仅自己可见"), gold `#ffb854`
   (真身/real-identity badge).
6. Day/night mapping table for the extended surfaces:

| Attribute | Light | Dark |
|-----------|-------|------|
| `indexDiaryTitleColor` | `#ffffff` | `#d3d3d3` |
| `indexDiaryMoodTabSelectedColor` | `#ffffff` | `#56a5ff` |
| `indexDiaryMoodTabNormalColor` | `#ffffff` | `#808080` |
| `globalBlueTabDefaultTextColor` | `#cbe1ff` | `#808080` |
| `globalLightC262626` (body neutral) | `#262626` | `#d3d3d3` |
| `loginIconTintColor` | `#333333` | `#56a5ff` |
| `globalButtonBlueBackground` | `blue_button` | `shape_blue_button_dark` |
| `blueGradientButton` | blue gradient png | dark variant |
| `chatRoomTitleColor` | `#56a5ff` | `#d3d3d3` |
| `sessionUnreadBackground` | `shape_session_unread` | `shape_session_unread_dark` |
| `chatTipBorder` | `shape_rounded_4_eaeaea` | `shape_rounded_4_0cffffff` |
| `indexFlashChatFilterBorder` | `shape_flash_card_menu` | `shape_flash_card_menu_dark` |
| `globalDialogBorder` | `shape_diary_filter_dialog` | `shape_diary_filter_dialog_dark` |
| `chatItemCustomerTextColor` | `#666666` | `#d3d3d3` |
| `chatItemSelfTextColor` | `#ffffff` | `#e6e6e6` |
| `globalSwitchBackgroundColor` | `#ffffff` | `#1a1a1a` |
| `SwitchView` off/on | off `#ececec`, on `#56a5ff` | same |

## 8. Compose alignment record (2026-08-13)

The existing `apps/android` Material3 token layer was aligned to this guide. Changes
landed in `core/design/LanternColors.kt`, `LanternType.kt`, `LanternTheme.kt`,
`LanternSpacing.kt`; every screen consumes the scheme so the change is global.

Direct evidence applied:

- Accent: `primary` → `#56a5ff` (was a green `#4F8A6E`); links `textLink` → `#56a5ff`.
- Light surfaces: `background`/`surface` → `#ffffff`, `surfaceVariant` → `#f7f7f7`.
- Light text: `onSurface`/`textPrimary` → `#333333`, `onSurfaceVariant`/`textSecondary` → `#666666`,
  `textTertiary` → `#999999`.
- Dividers: `outline` → `#ececec` (hairline/underline), `outlineVariant` → `#f2f2f2` (card divider).
- Dark surfaces: `background`/`surfaceVariant` → `#1a1a1a`, `surface` (card) → `#262626`.
- Dark text: `onSurface` → `#d3d3d3`, `onSurfaceVariant`/`textSecondary`/`textTertiary` → `#808080`,
  `textLink` → `#808080` (tip link dark).
- Status: `error` → `#ff8a8f` (pink), `warning`/`gold` → `#ffb854`, new `danger` → `#ff6565`.
- Green accent `#70cea7` → `secondary`/`success`.
- Navigation bar: `#cccccc` (light) / `#262626` (dark).
- Type: `text_common` 15sp → `bodyLarge`/`labelLarge` (was 16/14sp); `text_title` 18sp bold → `headingMedium`;
  20sp bold → `headingLarge`; 16sp bold → `headingSmall`; 24sp bold → `displaySmall` (collapsed strip).
- Spacing: added 5/15/28dp to the scale.

Derived (no direct evidence — tints of the evidence palette):

- `primaryContainer` `#dbe9ff`/`#2a4a6e`, `onPrimaryContainer` `#123a63`/`#d6e6ff`.
- `secondaryContainer` `#e2f5ec`/`#2a4a3c`, `onSecondaryContainer` `#11402c`/`#d7efe4`.
- `errorContainer` `#ffdfe0`/`#5a2525`, `onErrorContainer` `#571d1f`.
- `disabled` `#bdbdbd`, `disabledContainer` `#ececec` (dark `#5e5e5e`/`#353535`).
- Sticky-header tokens reserved for the feed collapsing header (WP10): `indexTopExpanded`,
  `indexTopExpandedTitle`, `indexTopCollapsing`, `indexDiaryTitle`, `indexDiaryMoodTabSelected`,
  `indexDiaryMoodTabNormal`, `lightBlueTab` — values per §3/§7 tables.