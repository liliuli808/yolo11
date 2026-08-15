# Screen Inventory — 一罐 (Yi Guan) APK

## Legend
- **Route id:** stable identifier used in the page map and behavior matrix.
- **Screenshot:** file in `docs/reverse/screens/` if captured.
- **Evidence:** `manifest` = `AndroidManifest.xml`; `layout` = `res/layout/` XML; `nav` = `res/navigation/nav_index.xml`; `code` = decompiled Kotlin/Java.
- **Network dependency:** whether the screen appears to require connectivity.
- **Privacy-sensitive behavior:** data collection or permission prompts inferred from the APK.

## 1. Startup / Splash

| Field | Value |
|-------|-------|
| **Route id** | `startup-splash` |
| **Activity** | `club.jijigugu.yiguan.ui.activities.SplashActivity` (launcher) |
| **Entry condition** | Cold launch from launcher / `am start` |
| **Visible data** | Branded splash image (`ivTitle`), bottom illustration (`ivBottom`), optional full-screen ad container (`adContainer`) |
| **Allowed actions** | Wait for automatic transition; ad dismiss if shown |
| **Back target** | Launcher |
| **Network dependency** | Yes — fetches boot config with 3s timeout; loads splash ad when conditions met |
| **Privacy-sensitive behavior** | Reads agreement flag; initializes push SDK; creates notification channels; ad load/click/show tracked |
| **Evidence** | `AndroidManifest.xml` MAIN/LAUNCHER filter; `activity_splash.xml`; `SplashActivity.java` |
| **Screenshot** | *(captured as part of transition; rendered surface was blank in headless capture)* |

## 2. Privacy-Policy Prompt

| Field | Value |
|-------|-------|
| **Route id** | `privacy-prompt` |
| **Activity / Dialog** | `UserAgreementDialog` shown from `SplashActivity` and `LoginActivity` |
| **Entry condition** | `key_login_user_agreement` is false |
| **Visible data** | Modal card titled “用户协议与隐私政策提示” with scrollable explanatory text and two bottom actions |
| **Allowed actions** | **不同意** (Disagree) — dismisses/closes the app; **同意** (Agree) — continues, saves flag, fetches boot config |
| **Back target** | Launcher (dialog blocks back navigation) |
| **Network dependency** | No |
| **Privacy-sensitive behavior** | Presents privacy policy, user agreement, and children’s data notice; requires explicit consent before proceeding |
| **Evidence** | Runtime screenshot; `SplashActivity.java` `onCreate` / `LP()`; `LoginActivity.java` `LP()` |
| **Screenshot** | `screens/001_privacy_prompt.png` |

## 3. Onboarding

| Field | Value |
|-------|-------|
| **Route id** | `onboarding` |
| **Fragment** | `StartFragment` |
| **Entry condition** | New user passing splash gate (`key_new_user` true + backend condition) |
| **Visible data** | Onboarding slides (exact content not fully decoded) |
| **Allowed actions** | Swipe through / complete onboarding |
| **Back target** | Splash / launcher |
| **Network dependency** | Likely |
| **Privacy-sensitive behavior** | Introductory analytics possible |
| **Evidence** | `SplashActivity.java` `Mo()` |

## 4. Login / Account

### 4a. Phone Login

| Field | Value |
|-------|-------|
| **Route id** | `login-phone` |
| **Activity / Fragment** | `LoginActivity` hosting `LoginInputPhoneFragment` |
| **Entry condition** | Not logged in after splash |
| **Visible data** | App logo, phone number field, agreement checkbox, WeChat/QQ/password icons |
| **Allowed actions** | Enter phone, toggle agreement, submit, switch to QQ/WeChat/password login |
| **Back target** | Splash / launcher |
| **Network dependency** | Yes — `/user/sendVCode/V2` |
| **Privacy-sensitive behavior** | Phone number collected; agreement flag persisted; agreement dialog reshown if needed |
| **Evidence** | `activity_login.xml`; `fragment_login_phone.xml`; `LoginActivity.java`; `data/net/a.java` |

### 4b. SMS Code

| Field | Value |
|-------|-------|
| **Route id** | `login-code` |
| **Fragment** | `LoginInputCodeFragment` |
| **Entry condition** | After phone submit |
| **Visible data** | Masked phone, 4-digit code input, resend countdown |
| **Allowed actions** | Enter code, resend SMS |
| **Back target** | Phone login |
| **Network dependency** | Yes — `/user/phoneLoginOrRegister` |
| **Privacy-sensitive behavior** | SMS verification; auto-login/registration |
| **Evidence** | `fragment_login_code.xml`; `nav_index.xml`; `data/net/a.java` |

### 4c. Password Login

| Field | Value |
|-------|-------|
| **Route id** | `login-password` |
| **Fragment** | `LoginFragment` |
| **Entry condition** | Password icon tapped |
| **Visible data** | Phone + password fields |
| **Allowed actions** | Login with phone and password |
| **Back target** | Phone login |
| **Network dependency** | Yes — `/user/login` |
| **Privacy-sensitive behavior** | Password credentials sent |
| **Evidence** | `data/net/a.java`; `LoginActivity.java` |

## 5. Profile Setup (Persona)

| Field | Value |
|-------|-------|
| **Route id** | `profile-setup` |
| **Fragment** | `InitInfoFragment` |
| **Entry condition** | After first login if gender/birthday missing |
| **Visible data** | “资料填写” title, gender selector (男生/女生), birthday selector, skip/complete buttons |
| **Allowed actions** | Select gender, select birthday, skip, complete |
| **Back target** | Blocked during setup (`LoginActivity` intercepts back) |
| **Network dependency** | Yes (save profile) |
| **Privacy-sensitive behavior** | Gender and birthday collected for “intelligent matching” |
| **Evidence** | `fragment_init_info.xml`; strings `profile_gender_qa`, `profile_age_qa`, `profile_tips` |

## 6. Main Tabs (IndexActivity)

### 6a. Feed / 一罐

| Field | Value |
|-------|-------|
| **Route id** | `main-feed` |
| **Activity / Fragment** | `IndexActivity` / `IndexFragment` tab 0 / `IndexDiaryFragment` |
| **Entry condition** | Logged in, not blacklisted/teen-mode |
| **Visible data** | Collapsible title header, mood tabs, diary/feed list, HOT tags, online count |
| **Allowed actions** | Scroll, switch mood, pull-to-refresh, tap diary, tap center + to create |
| **Back target** | Home / launcher |
| **Network dependency** | Yes — `feed/listRecommend`, `feed/listRecommendByTag`, `diary/listSelfByMood` |
| **Privacy-sensitive behavior** | Mood/tag preferences; location prompt elsewhere for same-city features |
| **Evidence** | `fragment_index.xml`; `fragment_index_diary.xml`; `layout_index_diary_top.xml`; `IndexFragment.java`; `nav_index.xml` |

### 6b. Drift Bottle / 漂流瓶

| Field | Value |
|-------|-------|
| **Route id** | `main-drift-bottle` |
| **Fragment** | `IndexWebFragment` |
| **Entry condition** | Tab 1 selection |
| **Visible data** | Web-based surface (flash-chat / drift-bottle content) |
| **Allowed actions** | Browse, interact with web content |
| **Back target** | Tab 0 |
| **Network dependency** | Yes (H5/web) |
| **Privacy-sensitive behavior** | WebView may load third-party content |
| **Evidence** | `IndexFragment.java` adapter `getItem(1)`; tab title "漂流瓶" |

### 6c. Messages / 消息

| Field | Value |
|-------|-------|
| **Route id** | `main-messages` |
| **Fragment** | `IndexSessionFragment` (or `EmptyFragment` if not logged in) |
| **Entry condition** | Tab 2 selection |
| **Visible data** | Session list, system messages, interactive messages, unread badges |
| **Allowed actions** | Open chat, refresh, view system/interactive lists |
| **Back target** | Tab 1 |
| **Network dependency** | Yes |
| **Privacy-sensitive behavior** | Message content; push notification handling |
| **Evidence** | `fragment_index_session.xml`; `nav_index.xml` destinations `sessionFragment`, `systemMsgListFragment`, `interactiveMsgListFragment` |

### 6d. Profile / 我

| Field | Value |
|-------|-------|
| **Route id** | `main-profile` |
| **Fragment** | `ProfileFragment` |
| **Entry condition** | Tab 3 selection |
| **Visible data** | Profile top layout, page content, liked diaries, settings entry |
| **Allowed actions** | Scroll, edit real profile, view liked diaries, open settings |
| **Back target** | Tab 2 |
| **Network dependency** | Yes |
| **Privacy-sensitive behavior** | Avatar, real-name profile, liked content |
| **Evidence** | `fragment_profile.xml`; `layout_profile_top.xml`; `layout_profile_page.xml` |

## 7. Feed / Topic / Post Detail

### 7a. Mood List

| Field | Value |
|-------|-------|
| **Route id** | `mood-list` |
| **Fragment** | `IndexMoodListFragment` |
| **Entry condition** | Navigate from feed |
| **Visible data** | List of moods |
| **Allowed actions** | Select a mood |
| **Back target** | Feed |
| **Network dependency** | Yes |
| **Evidence** | `nav_index.xml` `moodListFragment` |

### 7b. Tag / Topic List

| Field | Value |
|-------|-------|
| **Route id** | `tag-list` |
| **Fragment** | `IndexTagListFragment`, `MoodTagListFragment` |
| **Entry condition** | Navigate from feed/mood |
| **Visible data** | Topic tags |
| **Allowed actions** | Select a tag/topic |
| **Back target** | Previous |
| **Network dependency** | Yes |
| **Evidence** | `nav_index.xml` `indexTagListFragment`, `moodTagListFragment`; strings `st_title`, `st_hint` |

### 7c. Post Detail

| Field | Value |
|-------|-------|
| **Route id** | `diary-detail` |
| **Fragment** | `DiaryDetailFragment` |
| **Entry condition** | Tap a diary/can |
| **Visible data** | Post content, author info, comments, like/share/report actions |
| **Allowed actions** | Like, comment, share, report, private message author, copy/delete own comments |
| **Back target** | Feed / previous list |
| **Network dependency** | Yes — `comment/list`, report endpoints |
| **Privacy-sensitive behavior** | Report reasons sent to server; UGC content |
| **Evidence** | `fragment_diary_detail.xml`; `nav_index.xml`; strings `rd_report_diary_*`, `rd_like`, `rd_comment` |

### 7d. Add / Edit Diary

| Field | Value |
|-------|-------|
| **Route id** | `add-diary` / `edit-diary` |
| **Fragment** | `AddDiaryFragment`, `EditDiaryFragment` |
| **Entry condition** | Center + button or own diary |
| **Visible data** | Text editor, mood/tag/vote/album/interactive-state options |
| **Allowed actions** | Compose, add vote, select mood/tag, choose visibility, publish, save draft |
| **Back target** | Main feed / diary detail |
| **Network dependency** | Yes |
| **Privacy-sensitive behavior** | UGC, images, possible location; draft stored locally |
| **Evidence** | `nav_index.xml`; strings `send_diary`, `wd_max_length_alert`, `share_save_as_private` |

## 8. Flash Chat

| Field | Value |
|-------|-------|
| **Route id** | `flash-chat` |
| **Fragments** | `FlashCardFragment`, `SelfFlashCardFragment`, `AddFlashCardFragment`, `EditFlashCardFragment`, `FlashCardDetailFragment` |
| **Entry condition** | Drift-bottle tab / messages |
| **Visible data** | Flash-chat cards, filters (gender, intent, same-city), my cards |
| **Allowed actions** | Create card, filter, start 1v1 chat, delete card |
| **Back target** | Drift-bottle tab |
| **Network dependency** | Yes |
| **Privacy-sensitive behavior** | Gender, sexual orientation, city/location filters; location permission prompt for same-city (`fc_open_loc_question`) |
| **Evidence** | Strings `fc_*`; `nav_index.xml`; `FlashCardRandomActivity` |

## 9. Chat

### 9a. Session List

| Field | Value |
|-------|-------|
| **Route id** | `main-messages` (also `session`) |
| **Fragment** | `SessionFragment` |
| **Entry condition** | Messages tab or deep link |
| **Visible data** | Conversation list with unread counts |
| **Allowed actions** | Open chat, mark important, delete |
| **Back target** | Main messages |
| **Network dependency** | Yes |
| **Privacy-sensitive behavior** | Message metadata |
| **Evidence** | `nav_index.xml` `sessionFragment` |

### 9b. Direct Message

| Field | Value |
|-------|-------|
| **Route id** | `chat` |
| **Activity / Fragment** | `IMActivity`; chat message screens |
| **Entry condition** | Open a session |
| **Visible data** | Message list (text, image, voice, album, diary, emoticon), input bar |
| **Allowed actions** | Send text/image/voice, gift clover, report/close/delete session |
| **Back target** | Session list |
| **Network dependency** | Yes — `chat/list`, send message APIs |
| **Privacy-sensitive behavior** | Message content; `RECORD_AUDIO` for voice messages; report to `/session/report` |
| **Evidence** | Strings `chat_*`; `IMActivity` in manifest; `data/net/a.java` |

### 9c. Chat Room

| Field | Value |
|-------|-------|
| **Route id** | `chat-room` |
| **Fragments** | `ChatRoomListFragment`, `ChatRoomTipFragment` |
| **Entry condition** | From messages/feed |
| **Visible data** | Chat room categories, room list, create-room tip |
| **Allowed actions** | Join room, create room, kick/report |
| **Back target** | Previous |
| **Network dependency** | Yes |
| **Privacy-sensitive behavior** | Group chat content; moderation actions |
| **Evidence** | `nav_index.xml`; strings `chatroom_*` |

## 10. Audio Room

| Field | Value |
|-------|-------|
| **Route id** | `audio-live` |
| **Fragments / Activities** | `StartAudioLiveFragment`, `MoreAudioLiveListFragment`, `LiveRankingFragment`, `AudioLiveHostEndActivity`, `AudioLiveAudienceEndActivity` |
| **Entry condition** | From feed/profile |
| **Visible data** | Audio live room list, host start, ranking, end screens |
| **Allowed actions** | Start/host room, join as audience, apply to connect, send gifts, BGM |
| **Back target** | Previous |
| **Network dependency** | Yes (live room APIs, Agora audio SDK) |
| **Privacy-sensitive behavior** | `RECORD_AUDIO`, microphone; real-name required to connect; virtual currency (shell) charged for connect; commission split on withdrawals |
| **Evidence** | `activity_audio_live_end.xml`, `activity_audio_live_audience_end.xml`; `nav_index.xml`; strings `live_*`, `agora_app_id`; `data/net/a.java` |

## 11. Wallet

| Field | Value |
|-------|-------|
| **Route id** | `wallet` |
| **Fragments** | `MyWalletFragment`, `TradingRecordFragment`, `WithdrawalFragment`, `EditWithdrawalInfoFragment`, `WithdrawalSuccessFragment`, `IncomeRecordFragment` |
| **Entry condition** | Settings / live host flow |
| **Visible data** | Balance (shells/clover), recharge products (100–50000 shells), trading records, withdrawal info |
| **Allowed actions** | Recharge via Alipay/WeChat, view records, withdraw to Alipay, edit withdrawal identity |
| **Back target** | Settings / profile |
| **Network dependency** | Yes — `api.widgetbox.top` products, recharge, withdrawal endpoints |
| **Privacy-sensitive behavior** | Payment info, real-name/Alipay identity, commission split; withdrawal requires electronic contract for amounts > 400 RMB |
| **Evidence** | `fragment_my_wallet.xml`; strings `recharge_*`, `withdrawal_tip_content`; `data/net/a.java` |

## 12. Settings

| Field | Value |
|-------|-------|
| **Route id** | `settings` |
| **Fragment / Activity** | `SettingsFragment` (also `SettingsActivity` for agreement links) |
| **Entry condition** | Profile tab |
| **Visible data** | Personal info, comment nickname, liked diaries, account/security, teen mode, wallet, sound, night mode, privacy, agreements, feedback, cache, logout |
| **Allowed actions** | Toggle modes, navigate to sub-screens, logout, clear cache, check update |
| **Back target** | Main profile |
| **Network dependency** | Yes for sub-screens |
| **Privacy-sensitive behavior** | Aggregated privacy controls and data-collection disclosures |
| **Evidence** | `fragment_settings.xml`; strings `setting_*`; `nav_index.xml` |

## 13. Other notable screens

| Route id | Activity / Fragment | Purpose | Evidence |
|----------|---------------------|---------|----------|
| `black-screen` | `BlackActivity` | Account banned/blacklisted | Manifest; `SplashActivity.java` `Mo()` |
| `teenager-mode` | `TeenagerModeActivity` | Age-gate / restricted mode | Manifest; `activity_teenager_mode.xml` |
| `feedback` | `FeedBackActivity` | User feedback | Manifest; `activity_feedback.xml` |
| `web-view` | `WebViewActivity`, `AbWebViewActivity` | In-app browser | Manifest; `activity_webview.xml` |
| `night-mode` | `NightActivity` | Night-mode transition surface | Manifest |
| `im-helper` | `IMHelperActivity` | IM helper / assistant | Manifest; `activity_imhelper.xml` |

## Reachability status

| Route | Static evidence | Runtime reached |
|-------|-----------------|-----------------|
| Startup / Splash | Yes | Yes |
| Privacy-policy prompt | Yes | Yes (`screens/001_privacy_prompt.png`) |
| Login / onboarding / profile setup | Yes (layouts + code) | No (ARM-native crash) |
| Main tabs (feed, drift bottle, messages, profile) | Yes (layouts + nav graph + code) | No |
| Feed / topic / post detail | Yes (layouts + nav graph + API endpoints) | No |
| Flash chat | Yes (layouts + strings + nav graph) | No |
| Chat / chat room | Yes (layouts + strings + API) | No |
| Audio room | Yes (layouts + strings + Agora SDK) | No |
| Wallet / settings | Yes (layouts + strings + API) | No |

## Notes
- Screens are documented from static evidence only, except for the privacy-policy prompt which was reached at runtime.
- No proprietary graphics, branding, or source code is reproduced in this inventory.
