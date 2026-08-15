# UI Widget Internals — 一罐 APK (v3.16.10)

> **Clean-room note:** Implementation *behavior evidence* for replicating the frontend.
> No original identifiers, assets, branding, or code may be copied to `app.rebuild.social`.
> Class names/attrs below are descriptive pointers only.

Source: jadx decompiled sources under `decompiled/jadx-out/sources/club/jijigugu/yiguan/`.
This document covers the custom widgets' *behavior*, complementing `ui-structures.md`.

## 1. TitleView (shared app bar)

Base: `ConstraintLayout` inflating `view_title.xml` (43dp). Reads `R.styleable.TitleView`
attrs: `title`, `menus` (pipe-joined menu labels), `tv_right`, background color
(`?globalTitleBackgroundColor`).

Behavior:
- `navbarBack` click → `NavController.popBackStack()` (close current screen).
- `menus` attr (pipe `|`-separated) → for each label, `appendMenu()` builds a **TextView**
  at 17sp, gravity center, minHeight 24dp, **color `@color/blue` (#56a5ff)**, padding
  start 12dp; tap via `setMenuListener`.
- `tvRight` = right text action (e.g. "我的卡片"), 14sp, end-aligned.
- `Ab()` clears all listeners (used when back should be disabled); `setBackClickListener`,
  `setTvRightClick` assign callbacks; `setMenus` rebuilds the menu row.
- Taps are auto-tracked (SensorsData `trackViewOnClick`).

Rebuild note: one shared TopBar composable with left-back (pop), centered bold title,
right text/icon menu; all push screens reuse it. Menu item color is **always `#56a5ff`**.

## 2. AvatarView (multi-mode avatar)

Base: `ConstraintLayout` inflating `item_msg_avatar`. `bind(avatarUrl, gender, name, radius)`
resolves to one of four visual modes:

| Priority | Trigger | Render |
|----------|---------|--------|
| 1 | non-empty avatar URL | Glide round image (`GlideUrl` w/ header `X-Can-Image-Fallback`); fallback = dynamic avatar placeholder; error placeholder `ic_canman_is_watching_you_avatar` |
| 2 | non-empty name | colored oval with **first character uppercase** (`shape_oval_avatar_male/female_bg`); font size auto = `height / (2.4 × density)` |
| 3 | `gender==1` | `shape_oval_avatar_male_bg` + `icon_16px_me_male` |
| 4 | `gender==2` | `shape_oval_avatar_female_bg` + `icon_16px_me_female` |
| else | — | green circle `?globalGreenCircleBackground` |

So a contact without photo renders as a colored circle with the first letter; gender-only
renders a male/female icon chip. Rebuild: an Avatar(loading→image→letter/icon→green)
state machine.

## 3. CollapsingTitleView (feed sticky title — the blue→white header)

Subclass of `ConstraintLayout` inflating `view_index_diary_top_title.xml` (the "一罐"
24sp bold strip, 52dp). Driven by scroll offset `N(translationY)`:

- `getEffectRange()` = `52dp` (titleMax 52dp − titleMin 0).
- Progress `t = clamp(-transY / 52dp, 0..1)`.
- **Background:** `ArgbEvaluator.evaluate(t, expandedColor, collapsingColor)` where
  expanded = `?indexTopExpandedColor` (**#56a5ff**), collapsing = `?indexTopCollapsingColor`
  (**#ffffff**). The whole header bar color-blerps blue→white as you scroll.
- **Title:** alpha `1−t`; translationY `t × 10dp` (title slides down slightly while
  fading). At full collapse title hidden; the view itself translates up by `-transY−52`.

This is exactly the visual contract of a Material collapsing header, hand-rolled.
Rebuild: header background = lerp(blue, white, t); title = fade + slide 10dp over a 52dp
range.

## 4. FadingTitleView (album/profile overlay title)

Extends `TitleView`. Attaches a `RecyclerView.OnScrollListener` computing scroll ratio:
`t = clamp(scrollOffset / range, 0..1)` then `Q(t)`:
- **Background** `ArgbEvaluator(t, startColor, endColor)` — darkens/whitens with scroll.
- Icon/drawables tinted via `DrawableCompat`; text color crossfades (default → target).
Used on image-list screens (album) where the title starts transparent over art and fades
into a solid bar.

## 5. ScrollableViewPager (main-tab ViewPager)

Extends `ViewPager`. Wraps `onInterceptTouchEvent`/`onTouchEvent` in try/catch and gates
both on `setScrollable(boolean)` flag (scroll disabled until tabs loaded / during
transitions). Swallow exceptions to avoid crashes on rapid swipes.
Rebuild: a ViewPager/`PagerState` with an enabled flag and crash-safe gesture handling.

## 6. StickyPageBehavior (already covered in ui-structures §3)

Behavior details confirmed from source: `onLayoutChild` fixes page height to
`parent.height − top.Hk()`; `layoutDependsOn`/`onDependentViewChanged` always true
(full coupling); `onNestedPreFling` feeds fling velocity into the settle decision;
settle anim 300ms `DecelerateInterpolator`, target `0` (expanded) or `−Hm()` (collapsing),
state tag `tag_view_state` ∈ {`"expanded"`, `"collapsing"`}; `bUe = touchSlop` used to
decide drag vs tap.

## 7. VerificationCodeInputView (SMS code boxes)

Custom `ViewGroup`; creates `box`(default 4) `EditText`s, each:
- size from `child_width`/`child_height` (e.g. 48×58dp), gravity center, `ems=1`,
  `InputFilter.LengthFilter(1)`, inputType per `inputType` attr (number/password/text/phone).
- Background swaps normal↔focus drawables (`box_bg_normal`/`box_bg_focus`).
- First box auto-focuses after 1s.
- TextWatcher: after typing, focus jumps to next empty box (backward-fill from first empty);
  `onKey(DEL)` moves focus back.
- When all boxes filled → concat code, invoke listener, `setEnabled(false)`.
Rebuild: `n` text fields, each single-char, auto-advance + backspace rewind, submit when
complete.

## 8. MaterialProgressView (loading spinner)

Custom `View` (not the platform ProgressBar). Uses an `AnimationDrawable` whose frames are
drawn by a Material-style **ring** painter (`Paint.Style.STROKE`, square cap, width ~6dp):
- stroke color from theme (`?globalButtonTextColor`-like), ring rotates with
  `LinearInterpolator`; start/end sweeps expand/contract with
  `AccelerateDecelerateInterpolator` (i.e. the Material indeterminate circular pattern).
Sizes seen in layouts: 36dp (inline, login/city loading), 56dp (full-page detail).
Rebuild: use a Material `CircularProgressIndicator`; keep sizes 36/56.

## 9. AudioView (voice post / voice message)

Extends `ConstraintLayout` inflating `layout_diary_edit_audio`. Model = `Audio{duration,
displayUrl}`:
- `bind(audio, uniqueId)` sets duration text (`"${duration}s"`) and **left padding =
  37dp + 132dp × (duration/60)** — the voice waveform bar grows with duration.
- Playback: icon cycles 3 white mic/wave drawables (`ic_button_voice_left_{1,2,3}_white`)
  every 300ms via RxJava interval while playing (`setKeepScreenOn(true)`).
- `Eh()` disables (gray `#808080` icon/text on `?audioDisableBackground`).
- A singleton audio player service coordinates one-playing-at-a-time (attach/detach
  listeners; tapping another AudioView stops the current).
Rebuild: a VoiceRow = icon + duration + proportional-width bar + play/pause state,
with a shared audio player singleton.

## 10. VoteView (poll card)

Pure-code `ConstraintLayout` (no layout XML): a `title` TextView (15sp, minHeight 20dp,
gravity center_vertical) + a vertical `LinearLayout` of `VoteOptionView`s. Options render
per `Option` model with selected/unselected + percent states (option bars with results
after voting).
Rebuild: PollCard(title, options[]) with selectable options and post-vote percentage bars.

## 11. TextFlipper (rotating group reply text)

Extends `TextSwitcher`. Creates 12sp single-line TextView (color `?diaryGroupTextColor`,
ellipsize end). Rotation: `interval = 1400ms`, in/out anims `push_up_in/out`; only runs
when visible and `items.size > 1`; cycles `index = (index+1) % size`.
Rebuild: a `LaunchedEffect` cycling group-reply previews every 1.4s with a vertical
slide-fade.

## 12. TagFlowLayout (tag chips flow)

Extends a custom `FlowLayout`. Adds child views wrapped in `TagView` (checkable chip
state); optional single/multi-select via `maxSelect` (1 = radio-like, swaps selection;
else allow toggle). Selection set tracked & reported via listener; defaults:
`maxSelect=-1` (multi), margins 0/2/16/2 dp, `dip2px` helper. Used for mood/tag/desc chips.
Rebuild: FlowRow with selectable chip composables (single/multi toggle semantics).

## 13. GRecyclerView / EditPhotosLayout / PicturesLayout (misc lists)

- `GRecyclerView`: app's RecyclerView wrapper (used for feed/flash lists, `clipToPadding`
  false + bottom padding for floating buttons).
- `PicturesLayout`: horizontal image strip used in diary card `layPic`.
- `EditPhotosLayout`: horizontal scroller of photo thumbs with add/delete (editor).

## 14. HoverMessengerView / HoverAudioLiveView (floating pills)

Docked over the shell (activity_index): `HoverMessengerView` = floating chat-bubble
messenger; `HoverAudioLiveView` = floating live-audio pill (`gone` until a live is
active), bottom margin 140dp above nav. Rebuild as shell-level overlays (like a Compose
`Box` layer above the tab content) driven by messenger/live state.

## 15. Key implementation patterns for the rebuild

1. **Theme resolution everywhere** via `?attr/global*` — the Compose equivalent is a
   `CompositionLocal` token map, resolved per light/dark.
2. **Shared audio player singleton** coordinates all `AudioView`s (one plays at a time).
3. **Collapsing headers are hand-rolled** (lerp + translate + fade) — match ranges
   exactly (52dp effect range, 10dp title slide).
4. **Avatars are a 4-mode state machine** (image/letter/gender-icon/green-default).
5. **Animations:** 300ms decelerate settle (sticky), 300ms audio icon ticker,
   1.4s text-flipper cycle, `push_up_in/out` flips.
6. All clickable widgets fire SensorsData auto-track — mirror with an analytics hook if
   desired, but this is optional for the clean-room rebuild.