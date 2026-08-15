# UI Structure Trees — 一罐 APK (v3.16.10)

> **Clean-room note:** Structural/UX evidence for frontend replication only. No original
> identifiers, assets, branding, or code may be copied to `app.rebuild.social`. View ids
> below are descriptive pointers, not names to reuse.

Source: jadx decode → `decompiled/jadx-out/resources/res/layout/*.xml` (546 layouts).

---

## 1. IndexActivity (main shell)

```
ConstraintLayout (activity_index)
├── IndexFragment (fills screen)          → hosts the 4-tab ViewPager
├── GNavHostFragment (fills screen,       → the single nav graph (slide transitions)
│    defaultNavHost, navGraph=nav_index)
├── HoverMessengerView (@messenger)       → floating chat bubbles / session messenger
└── HoverAudioLiveView (@audioLiveHover)  → floating live-audio room pill
     marginBottom 140dp, marginEnd 15dp, gone by default
```

Notable: the tab + nav-host are **both** full-screen ConstraintLayout children; floating
messenger/live pills sit on top with bottom padding above the nav bar.

## 2. IndexFragment — four-tab main screen (`fragment_index.xml`)

```
ConstraintLayout
├── ScrollableViewPager (@viewPager)      → 4 tabs, pinned to top of nav bar
└── FrameLayout (@layNavigation)          → bottom tab bar
│   └── SlidingTabLayout (@navigation)    height 59dp, bg ?navigationBackgroundColor
└── View (1dp)                            hairline above the nav bar (?globalUnderlineColor)
```

Tab bar: `SlidingTabLayout`, height **59dp**, `?navigationBackgroundColor` bg
(transparent light / `#262626` dark), with a 1dp hairline (`globalUnderlineColor`) sitting
on top of the bar. Tabs:
- Tab 0  `IndexDiaryFragment`   一罐（diary feed）
- Tab 1  `IndexWebFragment`     漂流瓶（drift bottle / web）
- Tab 2  `IndexSessionFragment` 消息（sessions; EmptyFragment if logged out）
- Tab 3  `ProfileFragment`      我（profile）

## 3. IndexDiaryFragment — 一罐 feed (signature collapsing header)

```
ConstraintLayout
└── SwipeRefreshLayout (@refreshLayout)
    └── CoordinatorLayout
        ├── IndexDiaryTopLayout (@top)    height: 1800dp (fixed, tall canvas)
        │     └── mood chip row, expanding blue header (expanded #56a5ff / collapsed white)
        └── IndexDiaryPageLayout (@page)  initially invisible, match_parent
             app:layout_behavior = StickyPageBehavior
```

Interaction model (verified in `StickyPageBehavior.java`, 13.4 KB):
- `StickyTopLayout` (the tall 1800dp canvas) and `StickyPageLayout` (the content panel) are
  the two sticks; the behavior translates **both** views in sync on scroll.
- `StickyPageBehavior.onLayoutChild`: content panel height = `parent.height - topLayout.Hk()`
  (i.e., the sticky content starts exactly below the collapsed top). It is laid out over the
  top canvas.
- On scroll: content + top translate together by the consumed delta. The content panel is
  clamped to `[0 .. bUb]` and the top canvas to `[-Hk() .. 0]`.
- On stop: a **settle animation (300 ms, DecelerateInterpolator)** snaps to either
  `translationY == 0` (state tag `"expanded"` → blue header fully shown) or
  `-Hm()`/`-Hk()` (state tag `"collapsing"` → header collapsed to title bar). Which one is
  decided by the current velocity + threshold (`touch slop`), and `onNestedPreFling` feeds
  fling velocity into the same snap decision.
- So this is effectively a hand-rolled CollapsingToolbarLayout: expanded = blue header with
  white title + mood chips; collapsed = white title bar. Same pattern as Profile tab
  (ProfileTopLayout 1800dp + ProfilePageLayout).

## 4. Item: diary card (`item_diary`) — ConstraintLayout, wrap_content

```
ConstraintLayout
├─ debugPosition (gone) text red
├─ AvatarView         32×32dp, marginStart 16dp, marginTop 15dp
├─ name               TextView 14sp bold, color ?globalLightC262626
│                      singleLine maxLength 20, minHeight 18dp, constrained to avatar
├─ time               TextView 10sp, color ?globalLightC808080, under name
├─ hot / rules        TextView 10sp white on ic_tag_icon_hot bg, gone, min 44×18dp (HOT/RULES badge)
├─ locked             ImageView 18×18dp ic_tag_icon_locked, gone (private/secret diary)
├─ belong             LinearLayout (horizontal) — moodText + tag chips, blue_active
├─ content            TextView text_common, color ?globalLightC262626, maxLines 6,
│                      lineSpacingExtra 3dp, margins 16dp
├─ expand             TextView "展开" blue_active, gone (long-post expander)
├─ audioView          AudioView (voice post), gone, margin 16dp
├─ vote               VoteView (poll card), gone, margin 16dp
├─ layPic             PicturesLayout (grid images), 12dp top, margin 16dp
├─ layGroup           ConstraintLayout bg ?diaryGroupBackground (rounded 12dp green)
│                      ├ groupLogo ImageView (grouptalk icon, tinted)
│                      ├ groupText bold, ?diaryGroupTextColor, singleLine
│                      ├ groupContent TextFlipper (rotating group reply preview)
│                      └ groupArrow chevron (ic_setting_go_right_white, tinted)
├─ album              TextView 14sp bold blue "album", bg ?diaryAlbumBackground,
│                      12dp radius, drawableStart ic_home_icon_tag_album_blue, gone
├─ groupTip           TextView text_small ?globalDescTextColor, bg diaryAlbumBackground, gone
├─ chat               ImageView 18dp icon_20px_feed_chat, gone (chat entry)
├─ commentIcon/comment 18dp icon_20px_feed_comment + 12sp count, gone
├─ likeIcon/like      18dp icon_20px_feed_like + 12sp count (?globalSubTextColor), gone
├─ underline          View 1dp, bg ?globalUnderlineColor, margins 16dp
└─ likePop            ImageView 40×40dp ic_like_big_size, gone (like burst animation)
```

**Card anatomy (top→bottom):** avatar+name+time / moods+tags / body text (max 6 lines,
"展开" expander) / optional voice / optional poll / optional image grid / optional
group-talk card / optional album chip / optional chat-comment-like row / 1dp divider.
All rows separated by 4–12dp; standard horizontal padding 16dp.

## 5. ProfileFragment — 我 (profile)

```
SwipeRefreshLayout
└── CoordinatorLayout
    ├── ProfileTopLayout (1800dp)            → profile header canvas (avatar/bio/verifications)
    └── ProfilePageLayout (StickyPageBehavior) → scrollable content (albums, likes, settings)
```

Same sticky-header pattern as the feed tab.

## 5a. Flash-chat tab (电台 / 闪聊 — `fragment_index_flash_chat`)

```
ConstraintLayout (bg ?globalBackgroundColor, fitsSystemWindows)
├── TitleView (@topTitle)            title="电台", right action "闪聊"
├── LongitudinalSwipeRefreshLayout
│   └── IndexFlashChatPageLayout (@page, StickyPageBehavior)
└── startLive   TextView white text "我要开播", bg shape_blue_circle,
                64×64dp, floating bottom-end margin 15dp
```

Same sticky-page pattern here too, plus a floating circular "我要开播" button
(`shape_blue_circle` = blue accent circle).

## 6. Activity: splash (`activity_splash`)

```
ConstraintLayout
├── ivTitle (ic_splash_title)      top title artwork, marginTop 200dp
├── ivBottom                       bottom illustration (ratio ~360:194)
└── adContainer (weight ~8/9)      full-screen ad region under title, ad close button overlay
```

(Verified from earlier reading of `activity_splash.xml`.)

## 7. Common widget inventory (used across screens)

| Widget | Typical size | Use |
|--------|-------------|-----|
| `AvatarView` | 32dp | post/chat avatars (rounded square) |
| `PicturesLayout` | wrap | diary photo grid |
| `AudioView` | wrap | voice posts / voice messages |
| `VoteView` | 0dp/match | poll options |
| `TextFlipper` | 0dp | rotating group reply text |
| `SlidingTabLayout` | 59dp h | top tab strip |
| `HoverMessengerView` / `HoverAudioLiveView` | wrap | floating bubbles above nav |

## 8. Takeaways for the Compose rebuild

1. Feed/profile tabs share one architecture: **very tall header canvas + sticky content
   panel**, hand-rolled over `CoordinatorLayout`:
   - Content panel height = viewport − collapsed header height.
   - Both layers translate together on scroll; content clamps to `[0..headerOffset]`,
     top canvas to `[-headerHeight..0]`.
   - 300 ms DecelerateInterpolator settle to "expanded" (`y=0`) or "collapsing" (`y=−H`)
     chosen by velocity at scroll stop (fling-aware).
   - In Compose approximate with `Modifier.nestedScroll` on a `LazyColumn` + two translated
     layers, or an equivalent of `HeaderBehavior`/`AppBarLayout`; replicate the
     expand/collapse settle animation and tag-driven state (`expanded`/`collapsing`).
2. The diary card is a fixed-row layout: cost to replicate is low but fiducial — keep
   spacing 16dp, dividers 1dp `#ececec`(light)/`#333333`(dark), body 6-line clamp +
   "展开".
3. Bottom nav is a custom bar with 4 tabs + center "+" create action (mentioned by
   page-map evidence) — the center FAB sits over the tab bar.
4. Floating messenger/live pills anchored above the bottom nav (marginBottom 140dp)
   appear on the whole app shell — a reusable overlay layer.

## 9. Diary feed header internals (`layout_index_diary_top.xml` + CollapsingTitleView)

The blue collapsing header is a `ConstraintLayout` (58 lines) inside `IndexDiaryTopLayout`:

```
ConstraintLayout (@content)
├── ScrollableViewPager (@moodsPage)          mood sub-tabs M-page (fills under moodsTab)
├── View (@moodsTabBackground)                bg ?indexTopExpandedColor (blue!!) behind mood tabs
├── TabLayout (@moodsTab, MaterialComponents) height 46dp, margins 6dp
│     tabMode=scrollable, tabMinWidth=20dp, tabMaxWidth=200dp
│     tabIndicator=shape_tab_layout_indicator_18, indicatorHeight=3dp, indicatorFullWidth=false
│     tabSelectedTextColor ?indexDiaryMoodTabSelectedColor
│     tabTextAppearance=TabTextSize16 (16sp)
├── View (@underline)                         1dp ?globalUnderlineLightColor, alpha=0 (fades in)
└── CollapsingTitleView (@collapsingTopTitle) the app-name bar, fitsSystemWindows
```

**Mood tab strip colors (light):** selected `#56a5ff`-tinted text w/ 3dp indicator,
unselected `?globalBlueTabDefaultTextColor` (`#cbe1ff`). Both sit on the **blue expanded
header** (`?indexTopExpandedColor` = `#56a5ff`), so tabs read light-on-blue while expanded
and dark-on-white after collapse.

The standalone title strip (`view_index_diary_top_title.xml`, used in `CollapsingTitleView`)
is: `一罐` **24sp bold** `?indexDiaryTitleColor`, 52dp tall, marginTop 19dp/start 16dp,
includeFontPadding=false.

## 10. Diary detail page (`fragment_diary_detail.xml` + `item_diary_detail.xml`)

```
ConstraintLayout ?globalBackgroundColor, fitsSystemWindows
├── TitleView (@topTitle)                       standard nav bar
├── ConstraintLayout (@layContent)  bg ?globalGrayBackgroundColor
│   ├── View (edit_card drawable)               the white card, 8dp side margin, 9dp top
│   ├── SwipeRefreshLayout (@refreshLayout)     RecyclerView (@recyclerView) post+comments
│   ├── View (1dp, ?globalUnderlineLightColor)  divider (margins 12dp)
│   └── ConstraintLayout (@layMenu, 54dp)       bottom action bar, spread chain:
│       ├── 喜欢 (likeIcon 24dp icon_24px_feed_detail_like + like text_small)
│       ├── 评论 (commentIcon 24dp + commentText)
│       └── 私聊 (24dp icon_24px_feed_detail_chat + label)
├── likePop  ImageView icon_like_big_size (centered burst)  32dp
└── MaterialProgressView (@loading) 56dp   full-page loading
```

`item_diary_detail.xml` (the post body row inside the RecyclerView):
- Top row: `AvatarView 40dp` (gone if real-匿名), `nickname` bold text_common
  `?globalContentTextColor`, `time 13sp ?globalSubTextColor`, `tvIp 12sp`, `locked`
  (red "仅自己可见").
- `tag` bold `#56a5ff` (mood/tag), `content` text_common `?globalContentTextColor`
  with `autoLink=web` + `textIsSelectable`, then optional `audioView`, `voteView`,
  `layPic` (DetailPicturesLayout), `layGroup` (group card), `album` chip (15ems
  max, 33dp min height, blue, drawableStart ic_home_icon_tag_album_blue).
- `scheme` button: `?globalButtonBlueBackground` (blue pill), min 136×40dp — CTA
  (e.g. "参与方案") when diary has a vote.
- Comment header block: `commentCount` (16sp bold), `likeCount` (right-aligned),
  `commentLine` divider, `tip` (yellow-info "长按评论可举报不友善内容", drawableStart
  ic_pop_left_yellow), `tempComment` row (21dp avatar + "发布第一条评论…" input), `bottom` spacer.
- **Card style** = light-gray page bg (`?globalGrayBackgroundColor`) + white
  `edit_card` rounded card (9dp top, 8dp side).

## 11. Write/edit diary (`fragment_edit_diary.xml`, 481 lines)

```
ConstraintLayout ?globalBackgroundColor, fitsSystemWindows
├── TitleView (@topTitle)
├── ConstraintLayout (bg ?globalGrayBackgroundColor)
│   ├── View (edit_card white card, 9dp top / 8dp side)
│   ├── NestedScrollView (@scrollView, padding 10dp top)
│   │   └── ConstraintLayout (padding 20dp top, 12dp sides)
│   │       ├── EditText (@topTag)   text_common bold #65a5ff, no bg, "置顶话题" (gone)
│   │       ├── EditText (@input)    text_common ?globalContentTextColor,
│   │       │                          hint #80999999, no bg, 40dp bottom, style defaultTextSpace
│   │       ├── AudioView (@audioView, gone)
│   │       ├── EditPhotosLayout (@layPic, gone)   photo grid
│   │       ├── VoteView (@voteView)               optional poll
│   │       ├── ConstraintLayout (@layReal)        发布身份 row (selectable identity)
│   │       │     realIcon ic_edit_id + realText "发布身份" + 26dp avatar + real
│   │       │     ("分身匿名发布" desc) + realArrow chevron
│   │       ├── ConstraintLayout (@layMood)        频道与话题
│   │       │     moodIcon ic_edit_mood + moodText "频道与话题" + mood value + chevron
│   │       ├── ConstraintLayout (@layAlbum)       专辑
│   │       │     albumIcon ic_edit_album + albumText "专辑" + albumValue + chevron
│   │       └── ConstraintLayout (@layInteractive) 互动权限
│   │             interactiveIcon ic_edit_lock + interactiveText "互动权限"
│   │             + interactive "允许互动" + chevron
│   └── ConstraintLayout (@layMenu, 54dp)  bottom toolbar, spread chain:
│       ├── 图片 (ic_input_box_pic_fault + label)
│       ├── 语音 (ic_input_box_voice_fault_3 + label)
│       └── 投票 (ic_input_box_vote_fault + label)
├── HorizontalProgressView (@loadingProgress) 3dp, gone — publish progress
```

Publisher bottom bar is a ***3-item toolbar*** (图片/语音/投票) with icon+text, spread
evenly; each row `text_small` `?globalSubTextColor` with icon tinted
`?globalSubTextColor`. Meta rows (`layReal/layMood/layAlbum/layInteractive`) all share the
same anatomy: icon (24px, `?globalTagTextColor`) + bold-ish label + value + chevron
(`ic_setting_go_right` tinted `?globalArrowTintColor`), padded 20dp, stacked with no gaps.

## 12. Login flow (account password page `fragment_login.xml` + SMS)

`fragment_login.xml` (账号密码登录) — `fitsSystemWindows`, `?globalBackgroundColor`:
- `navbarBack` 20dp back arrow (12dp margins).
- `title` "账号密码登录" 20sp bold `?globalLightC262626`, centered.
- Field rows (phone / password) identical anatomy: label ("手机"/"密码", 16sp bold c808080)
  + `EditText` 18sp minHeight 53dp hint `?inputPanelHintColor` + 1dp underline
  (`?globalUnderlineLightColor`, horizontal margins 32dp, 8dp below field).
- `phoneToast` error (pink "格式错误", right-aligned); `pwdSecret` eye toggle
  (`icon_hide_password`, tinted `?loginIconTintColor`).
- `submit` ="登录" blue gradient pill `blueGradientButton`, 0dp×46dp, margins 16dp,
  15sp bold white.
- `pwdReset` "忘记密码？" 12sp `#56a5ff`, centered below.
- Loading overlay: full-screen `layLoading` + `MaterialProgressView 36dp`.

`fragment_login_phone.xml` (手机号登录): logo `illustrations_login_logo_with_name` 83×121dp,
40dp top; phone field + 1dp underline; agreement `CheckBox` (16dp margins, 1.25 spacing,
"我已阅读并同意《用户协议》…"); submit "下一步" blue pill margins 32dp; "其他登录方式"
12sp (65dp below submit) then 28dp icon row: WeChat(`icon_28px_login_wechat`, dark bg),
QQ (`icon_28px_login_qq`), 密码 (`icon_28px_login_password`), 40dp apart.

`fragment_login_code.xml` (短信验证码): navbarBack + title "输入短信验证码" 20sp bold +
subtitle "我们已发送验证码到你的手机" 16sp c808080 + phone "137****2890" +
`VerificationCodeInputView` (4 boxes, 48×58dp each, 15dp gap, focus/normal bg) +
tip "60S后重新发送" 14sp.

**Login form token summary:** blue gradient submit pill (`blueGradientButton`, 46dp h),
1dp field underlines, `?inputPanelHintColor` hints (`#33000000` light / `#80808080` dark),
error text pink, links `#56a5ff`, back arrow 20dp, title 20sp bold.

## 13. Profile top header (`view_index_profile_top_title.xml`, 163 lines = header content)

Clarifies what the ProfileTopLayout paints inside its 242dp-tall title block:
```
ConstraintLayout (@topRootContainer, height 242dp)
├── ImageView (@cover)            cover art illustrations_me_default_bg, centerCrop
├── View (@layMask)               shape_real_user_bg_mask (gradient darkening)
├── menuIcon                      settings gear icon_20px_nav_setting 40dp (25dp top)
├── menuTip                       "NEW" badge on white rounded 12 (gone)
├── switchIcon                    icon_20px_nav_switch 40dp (账号分身切换)
├── AvatarView (@avatar)          64dp at left
├── tvUserFlag                    "真身" badge, color_ffb854 gold, shape_real_user_flag_bg
├── realName                      20sp bold ?globalLightContentTextColor (white, 29dp min h)
├── tvUid                         12sp white
├── desc (TagFlowLayout)          tag chips row (gender/age/location/city tags)
├── tvIp                          12sp white (IP location)
├── tvEditProfile                 "编辑资料" pill shape_real_user_edit_bg, icon_16px_me_edit
└── underline                     1dp, alpha 0 (fades with collapse)
```
So the profile header is a **cover illustration + dark gradient mask + white text + tag
chips + edit-profile pill**; the "真身"(real identity)/switch affordances sit top-right.
`fragment_profile` uses the same StickyPageBehavior collapse to shrink this into a title bar.

## 14. TitleView widget (`view_title.xml` — the shared app bar)

Used by virtually every pushed screen (`topTitle`):
```
ConstraintLayout (@layTitle, height 43dp)
├── title       "全局标题" text_title sp bold ?globalTitleTextColor, centered, minHeight 24dp
├── underline   1dp ?globalUnderlineColor at bottom
├── navbarBack  back arrow, 50dp×match w, padding 12/18, start-aligned
├── menus       LinearLayout (right-side icon menus, marginEnd 12dp)
└── tvRight     right text action 14sp, end-aligned, marginEnd 12dp
```
**43dp app bar** with centered bold title + optional left back / right text or icon menus,
plus 1dp bottom hairline — the standard pushed-screen chrome.

## 15. Mood diary page (`fragment_mood_diary.xml`)

```
ConstraintLayout ?globalBackgroundColor, fitsSystemWindows
├── TitleView (@topTitle)
├── FrameLayout (@adContainer)        16dp margins (middle ad)
├── LongitudinalSwipeRefreshLayout (@refreshLayout)
│   └── MoodDiaryPageLayout (@page, StickyPageBehavior, bg ?globalBackgroundColor)
└── ImageView (@write)  blue_button_circle 48dp, ic_button_add,
    margin 25dp, bottom-end FAB → opens AddDiary
```

Even the mood filter page reuses the sticky pattern with a floating circular `+` FAB.

## 16. Messages tab (消息, `fragment_index_session.xml` + `item_session.xml`)

```
SwipeRefreshLayout
└── CoordinatorLayout
    ├── SessionListTopLayout (@top)            wrap_content header
    └── SessionListPageLayout (@page, StickyPageBehavior)
```

Yet another sticky-page pattern. **Session list item** (`item_session.xml`, ConstraintLayout):

```
ConstraintLayout (bg ?globalItemBackgroundFakeBlue — tappable ripple)
├── layIcon (47×47dp, marginStart 20dp)
│   ├── icon (47×47dp)
│   ├── View (?globalOpacityTintCircle10)       circular press tint
│   └── AvatarView (@avatar 47dp)
├── unread          text_mini white, ?sessionUnreadBackground badge, gone (count bubble)
├── unreadNoDisturbing 12dp red dot (muted dot), top-end of layIcon
├── title            text_common bold ?globalContentTextColor, singleLine, 20dp end margin
├── time             text_mini ?globalDescTextColor, top-right
├── content          text_common ?indexSessionContentTextColor, singleLine, under title
└── underline        1dp ?globalUnderlineColor, inset start 80dp / end 20dp
```

Row: 47dp avatar, bold single-line name + right timestamp, 1-line preview
(`#666` light / `#808080` dark), unread badge on avatar top-end, inset divider 80dp.

## 17. Chat session (`fragment_session_msg.xml` + `fragment_chat_input.xml`)

`fragment_session.xml` (inbox, 2 tabs) and `fragment_session_msg.xml` (single chat):

**Inbox header (`fragment_session.xml`):**
```
ConstraintLayout (@topTitle, 56dp)
├── TabLayout (@tab)  TabTextSize15, indicator 2dp ?shape_tab_layout_indicator_34,
│                      selected #56a5ff, unselected ?globalTabDefaultTextColor (#333/light)
├── underline 1dp ?globalUnderlineLightColor
├── navbarBack 50dp back arrow
└── menus  right icon row
└── ViewPager (@pager)
```

**Chat message list (`fragment_session_msg.xml`):** FrameLayout with RecyclerView
(`paddingTop 6dp`, bottom 16dp, `clipToPadding=false`) + a floating `switchRealLayout`
chip centered-bottom ("开始对话前，你可以 **切换为分身**" — blue link text, 12sp)
inside `?attr/chatTipBorder`.

**Text bubble (`item_msg_text_left/right.xml`):**
- `tv_timestamp` centered top (chat_text_date_style) — time separator.
- `avatar` 40dp (gone when sender hidden), marginStart 10dp.
- `tv_room_msg_nickname` (chat_sender style) above the bubble when avatar shown.
- Bubble: `tv_room_content` minHeight 40dp, padding 15/10/15/10, `autoLink=web`,
  textColorLink `#56a5ff`; **left** bubble bg `?attr/chatItemCustomerBackground`
  (other → `#666` text light), **right** `?attr/chatItemSelfBackground` (self → white
  text), marginEnd 62dp for max bubble width.

**Chat input bar (`fragment_chat_input.xml`, LinearLayout minHeight 48dp, bg gray):**
```
LinearLayout (center_vertical, bg ?globalGrayBackgroundColor, minHeight 48dp)
├── actionLayout     channel(off,gone)/image(ic_input_pic)/voice(ic_input_voice,gone) icons
├── actionEllipsis   ic_input_ellipsis, gone
├── FrameLayout(weight=1)  EditText (@chatInput): text_common, hint chat_input_tips,
│                          bg shape_chat_input (rounded field), min 36dp max 88dp, 3 lines,
│                          padding 16/32, style defaultTextSpace
│                          + actionEmoji (ic_input_emoji, overlaid end, gone)
├── actionBottle      ic_input_box_bottle_fault_copy_10 (漂流瓶 send toggle)
└── actionSend        ic_input_send (send, gone until text)
```
Left: image/voice/emoji tools; center: rounded input field; right: bottle→send toggle.

## 18. Flash-chat feed (`fragment_flash_card.xml` + `item_flash_card.xml`)

```
LinearLayout (vertical, ?globalBackgroundColor, fitsSystemWindows)
├── TitleView (@topTitle, tv_right="我的卡片")
├── SwipeRefreshLayout (@refreshLayout)
│   └── ConstraintLayout (bg ?globalGrayBackgroundColor)
│       ├── shimmerLayout (loading shimmer include)
│       ├── empty  "此刻寂静无人" 32sp ?globalEmptyTextColor, gone
│       ├── GRecyclerView (@recyclerView, paddingBottom 80dp, clipToPadding=false)
│       ├── MaterialProgressView (@cityLoading) 36dp
│       └── submit  "随机匹配" ?globalButtonBlueBackground blue pill, min 96×40dp, bottom
```

**Flash card item (`item_flash_card.xml`, ConstraintLayout bg `@drawable/card` 11dp margins):**
```
ConstraintLayout (card bg, 11dp side margins, bgTint ?globalCardTintColor)
├── laySelf (chain packed)
│   ├── avatar 40dp (20dp top/12dp bottom, start 15dp)
│   ├── nickname  text_common bold ?globalContentTextColor, singleLine
│   ├── tvIp     12sp ?globalSubTextColor (next to nickname)
│   └── desc     13sp ?globalSubTextColor, singleLine
├── status      text_small #33000000, top-right, gone
├── content     text_common ?globalContentTextColor, maxLines 2, autoLink=web, 15dp pads
├── audioView   AudioView, gone
├── more        → layPic PicturesLayout (images), gone
├── divider     1dp
└── menu (43dp, gone)  dash divider (globalDashLine 2dp) + menuText
     "私聊" bold #56a5ff, drawableStart ic_pop_left_blue (chat CTA)
```
Card: avatar+name+desc row, 2-line content, optional audio/images, dashed divider,
single "私聊" blue CTA. Empty state text is the distinctive **"此刻寂静无人"**.

## 19. Flash-card filter dialog (`dialog_flash_card_filter.xml`)

Bottom-sheet style dialog (`?attr/globalDialogBorder`, margins 15dp, center_horizontal):
- Section 性别: **男生 / 女生** rows (text_common, 32dp min height, margins 25dp,
  `ic_edit_icon_unselected` trailing radio).
- 1dp divider (`?globalUnderlineLightColor`, 10dp margins).
- Section 年龄段: 05后 / 00后 / 95后 / 90后 / 85后 / 80后 / 80前 (7 rows, same anatomy).
- Section 同城 (conditional, location-enabled): **同城** row.
- `submit` "确定" blue pill (`?globalButtonBlueBackground`, min 102×40dp, 11dp top margin).

Selection rows = full-width list lines with trailing radio dot; blue "确定" pill bottom.

## 20. Chat-room list (`fragment_chat_room_list.xml`)

```
ConstraintLayout ?globalBackgroundColor, fitsSystemWindows
├── empty  ic_can (罐 illustration) centered, ?globalEmptyIconColor tint
├── SwipeRefreshLayout (@refreshLayout, marginBottom 40dp)
│   └── RecyclerView (@recyclerView, padding 9dp)
├── drawer  (bottom sheet, translationY -40dp, bg ?globalBackgroundColor)
│   ├── underline 1dp top
│   ├── menu   "开启新房间" text_common bold ?chatRoomTitleColor, 40dp h, padding 16dp
│   ├── arrow  ic_room_icon_down rotated 180 (chevron-up), ?chatRoomTitleColor
│   └── RecyclerView (@template, height 160dp, padding 12dp)  room templates
└── TitleView (@topTitle)
```
Bottom drawer with "开启新房间" + template cards row (160dp) expanding up.

## 21. Wallet / money (`fragment_my_wallet.xml` + settings)

`fragment_my_wallet.xml`:
```
ConstraintLayout ?globalBackgroundColor, fitsSystemWindows
├── title  "我的钱包" 16sp bold ?globalTitleTextColor, centered, 12dp top
├── navbarBack 50×32dp back
├── TabLayout (@tab, fixed, 40dp, margins 16dp)
│     indicator 2dp ?globalLight65A5FF (blue), tabText TabTextSize15sp
│     selected #65a5ff / unselected ?globalLightC808080
├── underline 1dp ?globalUnderlineLightColor
└── ViewPager (@pager)  (Balance / Recharge / Bills tabs)
```

## 22. Settings (`fragment_settings.xml`, LinearLayout list — the canonical settings screen)

```
LinearLayout vertical ?globalBackgroundColor
├── TitleView "设置"
└── NestedScrollView
    └── LinearLayout (vertical, padding 10dp)
        ├── 个人资料 (no bg, value via @info)
        ├── 评论昵称 (value + chevron)                     [?globalItemBackground rows]
        ├── 我喜欢的罐头                       (chevron)
        ├── 账户与安全                         (chevron)
        ├── 青少年模式   SwitchView (51dp, primary blue #56a5ff, off gray #ececec)
        ├── divider (1dp ?globalUnderlineColor, 13/14dp)
        ├── 我的钱包                           (chevron)
        ├── 申请成为主播                       (chevron, gone)
        ├── divider
        ├── 声音提示   chip selector (layPushSound, ?indexFlashChatFilterBorder,
        │              "总是开启" + chevron-down, min 96dp)
        ├── 夜间模式   SwitchView (default on)
        ├── divider
        ├── 隐私设置 / 用户协议 / 隐私政策 / 个人信息收集清单 /
        │   第三方信息共享清单 / 儿童个人信息保护规则及监护人须知   (chevron rows)
        ├── divider
        ├── 联系人工客服 / 给一罐提意见 (meg icon) / 去应用商店评价一罐 /
        │   清除缓存 (value) / 检查更新 (version value)          (chevron rows)
        └── 退出登录 (chevron row)
```
**Settings row anatomy:** label text_common `?globalContentTextColor` (minHeight 40dp,
padding 20dp) + optional value/`SwitchView`/select-chip + `ic_setting_go_right` chevron
tinted `?globalArrowTintColor` (margin 7dp). Rows share `?globalItemBackground`
(selector white/blue). Section dividers 1dp with 13/14dp margins.

## 23. Album / profile sub-pages & misc

**Album (`fragment_album.xml`):** plain RecyclerView + floating `followText`
"关注" pill (`?globalButtonBlueBackground`, min 137×40dp, bottom) + `FadingTitleView`.

**Audio-live top inside flash-chat tab (`layout_index_flash_chat_top.xml`):**
- `audioLayout` section: "语音电台" 16sp + "排行榜" chip (12sp, bg
  `shape_live_ranking_entrance_bg`, icon_12px_audio_live_rank) + "开播"/"更多" links +
  `DZStickyNavLayouts` room category strip + `ivWave` wave illustration (32dp, fitXY).

**Random match (`fragment_flash_card_random.xml`):** 43dp black title bar + centered
`ic_flash_card_random` art (73dp top) + full-screen `splash` ratio 750:600.

**My diaries (`fragment_self_diary_list.xml`):** plain vertical RecyclerView + refresh.

## 24. Compose alignment record (2026-08-13)

Layout behavior aligned into `apps/android` (alongside the token alignment recorded in
`ui-style-guide.md` §8). Changes landed in `core/design/components` and `feature/feed`.

- `LanternInput` (core/design/components/LanternInput.kt) rebuilt as the **underline-style
  field** from §12: label 16sp bold `#808080`, input 18sp, 1dp bottom underline
  (primary `#56a5ff` focused / `#ececec`–`#333333` idle / pink error), hint
  `#33000000` light / `#80808080` dark. Tokens `fieldLabel`, `inputHint` + dark variants
  added to `LanternColors`; `fieldMinHeight` (53dp) added to `LanternSpacing`.
- `PostCard` (feature/feed/PostCard.kt) rebuilt as a **flat feed row** per §4 (no card
  container): 32dp avatar (`avatarSmall`), 14sp bold name, body `text_common` clamped to
  **6 lines with 展开/收起** (via `onTextLayout.hasVisualOverflow`), 1dp bottom divider
  (`?globalUnderlineColor`). Horizontal 16dp padding is owned by the feed list.
- `PostDetailScreen` aligned to §10: content area on `?globalGrayBackgroundColor`
  (`surfaceVariant`), post/comments on a white card with **9dp top / 8dp side** margins,
  post avatar **40dp** (`avatarDetail`), comment composer on gray with underline `LanternInput`.
- `ComposerSheet` input swapped to underline-style field (body `text_common` 15sp).
- `LanternIcon.avatarDetail` (40dp) added for the diary detail avatar.

Second pass (2026-08-13):

- `SettingsScreen` (feature/settings) rebuilt per §22: 40dp rows with 20dp start padding +
  trailing value/chevron, blue `#56a5ff` SwitchView (off `#ececec`), 1dp section dividers.
  Rows use clean-room copy (e.g. 我喜欢的帖子 / 给 Lantern 提意见).
- `VerificationCodeInput` (feature/auth/VerificationCodeInput.kt) added per §12:
  4 boxes of 48x58dp with 15dp gaps, primary border when focused/filled, invisible
  BasicTextField overlay; `VerificationScreen` now uses it with aligned copy.

Still pending (future WP work): sticky header behavior (§3/§5), and the 声音提示 chip
selector detail (§22).