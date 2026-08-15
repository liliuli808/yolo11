# Page Map — 一罐 (Yi Guan) APK

## Scope
This document maps the screens and routes discovered through **static analysis** of `/home/jichi/yiguan/一罐.apk` (`club.jijigugu.yiguan`, v3.16.10, code 316020). Where runtime observations from the earlier emulator session are available, they are noted explicitly. Everything else is inferred from the decompiled APK resources, `AndroidManifest.xml`, navigation graphs, layout XML, and decompiled Kotlin/Java.

## Environment
- **APK:** `club.jijigugu.yiguan`, version `3.16.10` (code `316020`)
- **minSdk:** 21, **targetSdk:** 30, **compileSdk:** 30
- **Activities:** 84 (17 app-owned, remainder from third-party SDKs)
- **Services:** 34, **Receivers:** 21, **Providers:** 22
- **Permissions:** 38 declared
- **Main navigation graph:** `res/navigation/nav_index.xml` (single Navigation Component graph)
- **API base URL:** `https://api.jijigugu.club/` (production; test variants exist)

## Evidence summary
- `SplashActivity` is the launcher. It checks a privacy-policy flag, fetches a boot config, optionally shows a splash ad, then routes to onboarding, login, black-screen, teen-mode, or the main `IndexActivity`.
- `IndexActivity` hosts a four-tab main screen via `IndexFragment` + `ViewPager`:
  1. 一罐 (feed / diary list by mood)
  2. 漂流瓶 (drift-bottle / web tab)
  3. 消息 (messages / sessions)
  4. 我 (profile)
- A center floating action in the tab bar opens the **create-diary** flow.
- All deeper screens are destinations inside `nav_index.xml` and are pushed with slide animations.
- Login is phone/SMS-based with QQ and WeChat as alternatives, followed by a profile-setup (gender/birthday) fragment.
- The app declares live-audio, wallet/recharge/withdrawal, flash-chat, chat-room, and album flows.

## Entry flow

```
Launcher icon / cold start
    │
    ▼
SplashActivity
    │
    ├── Privacy-policy not accepted ──► Show UserAgreementDialog
    │                                   Agree  ──► save flag + fetch boot config
    │                                   Disagree ──► close app
    │
    ├── Fetch boot config (timeout 3s) ──► set token, ad vendor, AB test, live-host flag
    │
    ├── Load splash ad (conditional: new user / first install / daily cap / debug skip)
    │
    ▼
Routing gate (Mo)
    │
    ├── New user + condition ──► StartFragment (onboarding)
    ├── Blacklisted account ──► BlackActivity
    ├── Not logged in ──► LoginActivity
    ├── Teenager mode active ──► TeenagerModeActivity
    └── Else ──► IndexActivity (main tabs)
```

## Page-by-page route map

| Route id | Screen / Fragment | Entry condition | Visible data | Allowed actions | Back target | Network dependency | Privacy-sensitive behavior |
|----------|-------------------|-----------------|--------------|-----------------|-------------|--------------------|----------------------------|
| `startup-splash` | `SplashActivity` | Cold launch from launcher | Branded splash image, bottom illustration, optional full-screen ad container | Wait for ad/config gates | Launcher / home | Yes (boot config + splash ad) | Reads `key_login_user_agreement`; may load personalized ad; sets notification channels on API 26+ |
| `privacy-prompt` | `UserAgreementDialog` (shown in `SplashActivity` and `LoginActivity`) | First launch or before login if agreement flag is false | Modal card summarising privacy policy, user agreement, children’s data notice | Agree / Disagree (or open Settings from dialog) | Launcher (blocks back) | No | Explicit consent gate; links to privacy policy, ToS, children’s notice |
| `onboarding` | `StartFragment` | New user passing splash gate | Onboarding content (exact pages not fully decoded) | Swipe / complete onboarding | Splash / launcher | Likely | May collect intro analytics |
| `login-phone` | `LoginInputPhoneFragment` in `LoginActivity` | Not logged in | Phone number field, agreement checkbox, WeChat/QQ/password icons | Enter phone, toggle agreement, choose WeChat/QQ/password, submit | Splash (back closes activity) | Yes (send SMS code) | Phone number collected; agreement flag persisted |
| `login-code` | `LoginInputCodeFragment` | After phone submit | Masked phone number, 4-digit code input, resend countdown | Enter SMS code, resend | Login phone screen | Yes (login/register) | SMS verification; phone login/register endpoint `/user/phoneLoginOrRegister` |
| `login-password` | `LoginFragment` | Password icon tapped | Password input | Login with phone+password | Login phone screen | Yes | Credentials sent to `/user/login` |
| `profile-setup` | `InitInfoFragment` | After first login if gender/birthday missing | Gender selector, birthday selector, skip/complete buttons | Select gender/birthday, skip | Blocked during setup | Yes (save profile) | Gender, birthday collected for matching |
| `main-feed` | `IndexActivity` + `IndexFragment` tab 0 (`IndexDiaryFragment`) | Logged in, not blacklisted/teen-mode | Collapsible header, mood tabs, diary/feed list | Scroll, switch mood, tap diary, pull-to-refresh, tap center + to create | Home / launcher | Yes (feed/listRecommend, listByTag, etc.) | Location permission requested for same-city matching elsewhere; feed list sends mood/tag IDs |
| `main-drift-bottle` | `IndexFragment` tab 1 (`IndexWebFragment`) | Logged in | Web-based drift-bottle / flash-chat surface | Browse, interact with web content | Tab 0 | Yes (H5/web) | WebView may load third-party content |
| `main-messages` | `IndexFragment` tab 2 (`IndexSessionFragment` or `EmptyFragment` if not logged in) | Logged in | Session list, system/interactive message entries | Open chat, refresh | Tab 1 | Yes (session list, chat list) | Push notification handling; message content |
| `main-profile` | `IndexFragment` tab 3 (`ProfileFragment`) | Logged in | Profile top, page content, liked diaries, settings entry | Scroll, open settings, edit real profile, view likes | Tab 2 | Yes (profile data) | Avatar, real-name profile data |
| `diary-detail` | `DiaryDetailFragment` | Tap a diary/can from feed/profile | Post content, author info, comments, like/share/report actions | Like, comment, share, report, private message author | Main feed / previous list | Yes (comment/list, diary detail) | Post and comment content; report reasons sent to `/diary/report`, `/comment/report` |
| `add-diary` | `AddDiaryFragment` | Center + button or mood/tag entry | Text editor, mood/tag selectors, vote/album/interactive-state options | Compose, add vote, select mood/tag, choose visibility, publish | Main feed | Yes (publish diary) | User-generated content, images, location if allowed |
| `edit-diary` | `EditDiaryFragment` | Own diary | Edit existing diary | Edit, save | Diary detail | Yes | Content update |
| `mood-list` | `IndexMoodListFragment` | Navigate from feed | List of moods | Select mood | Feed | Yes | Mood preferences |
| `tag-list` | `IndexTagListFragment` / `MoodTagListFragment` | Navigate from feed/mood | Topic tags | Select tag | Previous | Yes | Topic interests |
| `flash-chat` | `FlashCardFragment`, `SelfFlashCardFragment`, `AddFlashCardFragment`, `FlashCardDetailFragment` | Drift-bottle tab / messages | Flash-chat cards, filters (gender, intent, same-city), my cards | Create card, filter, start 1v1 chat, delete card | Main drift-bottle | Yes (flash-chat list, create, match) | Gender, sexual orientation, city/location filters; location permission prompt for same-city |
| `flash-chat-random` | `FlashCardRandomFragment` | Random match action | Random matching surface | Match / cancel | Flash chat | Yes | Matching data |
| `chat` | `SessionFragment` / direct-message screens | Open a session | Message list (text, image, voice, album, diary, emoticon) | Send text/image/voice, gift clover, report/close/delete session | Messages list | Yes (chat/list, send message) | Message content; voice recording uses `RECORD_AUDIO`; report to `/session/report` |
| `chat-room` | `ChatRoomListFragment`, `ChatRoomTipFragment` | From messages/feed | Chat room categories, room list, create room | Join room, create room, kick/report | Previous | Yes | Group chat content |
| `audio-live` | `StartAudioLiveFragment`, `MoreAudioLiveListFragment`, `LiveRankingFragment` | From feed/profile | Audio live room list, host start, ranking | Start/host room, join as audience, apply to connect, send gifts | Previous | Yes (live room APIs, Agora audio SDK) | `RECORD_AUDIO`, microphone; real-name required to connect; virtual currency (shell) charged |
| `wallet` | `MyWalletFragment`, `TradingRecordFragment`, `WithdrawalFragment`, `IncomeRecordFragment` | Settings / live host | Balance, recharge products, trading records, withdrawal flow | Recharge (Alipay/WeChat), view records, withdraw | Settings / profile | Yes (payment products, recharge, withdrawal) | Payment info, real-name/Alipay identity, commission split |
| `settings` | `SettingsFragment` | Profile tab | Personal info, nickname, liked diaries, account/security, teen mode, wallet, sound/night, privacy, agreements, feedback, cache, logout | Toggle modes, navigate to sub-screens, logout | Main profile | Yes (some sub-screens) | Extensive privacy controls and data-collection disclosures |
| `account-security` | `AccountAndSecurityFragment` | Settings | Phone bind, password, etc. | Bind phone, change password | Settings | Yes | Phone/account credentials |
| `push-settings` | `PushSettingFragment` | Settings / privacy settings | Push notification toggles | Toggle push channels | Settings | Yes | Push preferences |
| `teenager-mode` | `TeenagerModeActivity` | Enabled in settings or splash gate | PIN/time-limit screen | Disable (with PIN) / use teen mode | Launcher | No/minimal | Age-gating |
| `black-screen` | `BlackActivity` | Account banned/blacklisted | Block message | Dismiss / exit | Launcher | No | Account status |
| `feedback` | `FeedBackActivity` | Settings | Feedback input | Submit feedback | Settings | Yes | User feedback |
| `web-view` | `WebViewActivity` / `AbWebViewActivity` | Various links | In-app browser | Browse, back | Caller screen | Yes | External URLs |

## Navigation back behavior
- The single `nav_index.xml` graph handles almost all fragment transitions with standard `popBackStack()` (slide-out-right animation).
- `LoginActivity` blocks back when `InitInfoFragment` is on screen.
- The privacy dialog blocks back until a choice is made.
- Main-tab switching is via `ViewPager.setCurrentItem()`; the system back button exits the app from the main tabs.

## Network dependency
- Almost every screen after login requires `https://api.jijigugu.club/`.
- Boot config is fetched with a 3–4 second timeout in `SplashActivity` / `LoginActivity`.
- The app also contacts `api.midway.run` (update check), `api.widgetbox.top` (payment products), ByteDance/GDT ad domains, and push vendors (JPush, MiPush, Huawei, Meizu, Oppo).
- `network_security_config.xml` and `network_config.xml` permit cleartext traffic globally and trust user-installed certificates in debug configuration.

## Privacy-sensitive behaviors observed statically
- **Consent gate:** `UserAgreementDialog` requires explicit agreement before login and is also checked at splash.
- **Phone number:** collected at login and bound to account (`/user/sendVCode/V2`, `/user/phoneLoginOrRegister`, `/user/bindPhoneNoPass`).
- **Gender & birthday:** collected during onboarding (`InitInfoFragment`) for matching.
- **Location:** requested for same-city flash-chat filtering (`share_request_loc`, `fc_open_loc_question`).
- **Storage:** `WRITE_EXTERNAL_STORAGE`, `READ_EXTERNAL_STORAGE` declared; used for images/audio/cache.
- **Camera & microphone:** `CAMERA`, `RECORD_AUDIO`, `MODIFY_AUDIO_SETTINGS` for chat voice messages and live audio rooms.
- **Phone state:** `READ_PHONE_STATE` declared.
- **Contacts / package queries:** `QUERY_ALL_PACKAGES`, `GET_TASKS`, `REORDER_TASKS`; queries for QQ, WeChat, Alipay packages.
- **Third-party SDKs:** advertising (AdBright, GDT, Pangle/TikTok), push (JPush, MiPush, Huawei, Meizu, Oppo), sharing (QQ/WeChat), payment (Alipay), analytics (SensorsData), anti-fraud (Fengkong).
- **Tracking identifiers:** multiple MSA/OAID related permissions and ad SDK identifiers.

## Unknown / not fully decoded
- Exact onboarding slides in `StartFragment`.
- Complete in-room UI for audio live (host vs. audience).
- Full set of moderation/filter rules.
- Whether `QUERY_ALL_PACKAGES` is exercised at runtime.

## Notes
- No source code, proprietary graphics, branding, or long-form copy was reproduced in this document.
- Runtime screenshots beyond the privacy prompt could not be captured because the APK ships ARM-only native libraries and cannot execute on the available x86_64 emulator.
