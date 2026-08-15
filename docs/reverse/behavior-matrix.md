# Behavior Matrix — 一罐 (Yi Guan) APK

## Method
This matrix is built from **static analysis** of `/home/jichi/yiguan/一罐.apk`. Evidence sources are:
- `AndroidManifest.xml` for activities, services, receivers, providers, and permissions.
- `res/navigation/nav_index.xml` for fragment routes and arguments.
- `res/layout/` and `res/values/strings.xml` for visible UI and actions.
- Decompiled Kotlin/Java sources (`jadx`) for navigation logic, API calls, and SDK initialisation.
- Runtime observation is noted only for the privacy-policy prompt; all deeper routes were blocked by an ARM-native-library crash on the x86_64 emulator.

## Matrix

| Route | Entry condition | Visible data | Allowed actions | Back target | Network dependency | Privacy-sensitive behavior | Evidence |
|-------|-----------------|--------------|-----------------|-------------|--------------------|---------------------------|----------|
| **Startup / Splash** | App launch | Splash image, bottom illustration, optional ad container | Wait; ad dismiss if shown | Launcher | Yes (boot config + splash ad) | Reads agreement flag; initializes push; creates notification channels; ad tracking | `SplashActivity.java`, `activity_splash.xml` |
| **Privacy-policy prompt** | `key_login_user_agreement` false | Modal card with agreement summary and two buttons | Agree / Disagree (or open Settings) | Launcher (blocked by modal) | No | Explicit consent to privacy policy, ToS, children’s notice | Runtime screenshot; `LoginActivity.java` `LP()` |
| **Onboarding** | New user after splash | Onboarding slides | Swipe / complete | Splash | Likely | Analytics | `SplashActivity.java` `Mo()` |
| **Login phone** | Not logged in | Phone field, agreement checkbox, WeChat/QQ/password icons | Enter phone, toggle agreement, submit, switch login method | Splash | Yes (send SMS) | Phone number collected; agreement persisted | `fragment_login_phone.xml`, `data/net/a.java` |
| **Login code** | After phone submit | Masked phone, 4-digit code input, resend countdown | Enter code, resend | Login phone | Yes (login/register) | SMS verification; auto-registration | `fragment_login_code.xml`, `nav_index.xml` |
| **Login password** | Password icon tapped | Phone + password fields | Login | Login phone | Yes (`/user/login`) | Password credentials | `data/net/a.java` |
| **Profile setup** | First login missing gender/birthday | Gender and birthday selectors, skip/complete | Select gender/birthday, skip, complete | Blocked | Yes (save profile) | Gender/birthday for matching | `fragment_init_info.xml`, strings |
| **Main feed (一罐)** | Logged in | Collapsible header, mood tabs, diary list | Scroll, switch mood, refresh, tap diary, create diary | Home | Yes (`feed/listRecommend`) | Mood/tag preferences | `fragment_index_diary.xml`, `IndexFragment.java` |
| **Drift bottle (漂流瓶)** | Tab 1 | Web-based surface | Browse, interact | Tab 0 | Yes (H5) | WebView third-party content | `IndexFragment.java` adapter |
| **Messages (消息)** | Tab 2 | Session list, unread badges | Open chat, refresh | Tab 1 | Yes | Message metadata; push | `fragment_index_session.xml`, `nav_index.xml` |
| **Profile (我)** | Tab 3 | Profile top/page, liked diaries, settings entry | Scroll, edit, view likes, settings | Tab 2 | Yes | Avatar, real-name data | `fragment_profile.xml` |
| **Diary detail** | Tap diary | Post, author, comments, actions | Like, comment, share, report, PM | Feed/list | Yes (`comment/list`, reports) | UGC; report reasons | `fragment_diary_detail.xml`, strings |
| **Add/edit diary** | Center + / own diary | Editor, mood/tag/vote/album options | Compose, publish, save draft | Feed/detail | Yes | UGC, images, possible location | `nav_index.xml`, strings |
| **Mood list** | From feed | Moods | Select mood | Feed | Yes | Mood preferences | `nav_index.xml` |
| **Tag/topic list** | From feed/mood | Topic tags | Select tag | Previous | Yes | Topic interests | `nav_index.xml`, strings |
| **Flash chat** | Drift-bottle tab | Cards, filters (gender, intent, same-city) | Create, filter, chat, delete | Drift bottle | Yes | Gender/sexual orientation/city filters; location prompt | Strings `fc_*`, `nav_index.xml` |
| **Flash chat random** | Random match action | Matching surface | Match / cancel | Flash chat | Yes | Matching data | `nav_index.xml` |
| **Session list** | Messages tab | Conversations | Open, delete, mark important | Messages tab | Yes | Metadata | `nav_index.xml` |
| **Direct message** | Open session | Messages, input bar | Text/image/voice, gift clover, report/close/delete | Session list | Yes (`chat/list`) | Message content; voice uses `RECORD_AUDIO` | Strings `chat_*`, `IMActivity` |
| **Chat room** | Messages/feed | Categories, rooms | Join, create, kick, report | Previous | Yes | Group content | Strings `chatroom_*`, `nav_index.xml` |
| **Audio live** | Feed/profile | Room list, host start, ranking | Host/join, apply connect, gifts, BGM | Previous | Yes (live APIs, Agora) | Microphone; real-name to connect; shell charges | `activity_audio_live_*.xml`, strings |
| **Wallet** | Settings/host flow | Balance, recharge, records, withdrawal | Recharge, view records, withdraw | Settings | Yes (payment products, withdrawal) | Payment/Alipay identity; contract for >400 RMB | `fragment_my_wallet.xml`, strings |
| **Settings** | Profile tab | Long settings list | Toggle modes, navigate, logout, clear cache | Profile | Yes (sub-screens) | Privacy controls, data-collection disclosures | `fragment_settings.xml` |
| **Account & security** | Settings | Phone/password management | Bind phone, change password | Settings | Yes | Account credentials | `nav_index.xml` |
| **Push settings** | Settings | Push toggles | Toggle channels | Settings | Yes | Push preferences | `nav_index.xml` |
| **Black screen** | Banned account | Block message | Dismiss | Launcher | No | Account status | `SplashActivity.java` `Mo()` |
| **Teenager mode** | Enabled / splash gate | PIN/time-limit screen | Disable with PIN | Launcher | No | Age-gating | `activity_teenager_mode.xml` |
| **Feedback** | Settings | Feedback input | Submit | Settings | Yes | User feedback | `activity_feedback.xml` |
| **Web view** | Various links | In-app browser | Browse, back | Caller | Yes | External URLs | `activity_webview.xml` |

## Cross-screen behaviors

| Behavior | Routes involved | Evidence |
|----------|-----------------|----------|
| **Pull-to-refresh** | Feed, messages, profile | `SwipeRefreshLayout` in layouts |
| **Report content** | Diary detail, comments, chat, chat room | Strings `rd_report_*`, `chat_report_*`, endpoints `/diary/report`, `/comment/report`, `/session/report` |
| **Gift virtual currency** | Chat (clover), audio live (gifts) | Strings `chat_send_gift`, `chatroom_colver`, `set_gift_clover` |
| **Location permission prompt** | Flash-chat same-city filter | Strings `fc_open_loc_question`, `share_request_loc` |
| **Storage permission prompt** | Image sharing, cache, downloads | Manifest `WRITE_EXTERNAL_STORAGE`, `READ_EXTERNAL_STORAGE` |
| **Microphone permission prompt** | Voice messages, audio live | Manifest `RECORD_AUDIO`, `MODIFY_AUDIO_SETTINGS` |
| **Camera permission prompt** | Image capture | Manifest `CAMERA` |
| **Phone-state read** | Login/device identification | Manifest `READ_PHONE_STATE` |
| **Push registration** | Splash, login | Manifest push permissions; `SplashActivity.java` `Mp()`; JPush/MiPush/Huawei/Meizu/Oppo receivers |
| **Ad loading** | Splash | `SplashActivity.java` `Ml()`; AdBright/GDT/Pangle |
| **AB test / boot config** | Splash, login | `BootModel` in `SplashActivity.java` / `LoginActivity.java` |
| **Real-name / real profile** | Profile, audio live connect | Strings `live_connect_audience_real_tip_*`, `realFragment`, `editRealFragment` |
| **Withdrawal identity verification** | Wallet | Strings `withdrawal_tip_content`; `EditWithdrawalInfoFragment` |

## Network and security observations

| Observation | Evidence |
|-------------|----------|
| Production API base URL | `https://api.jijigugu.club/` in `data/net/h.java` |
| Update check endpoint | `https://api.midway.run/1.0/settings/checkUpdate` |
| Payment products endpoint | `https://api.widgetbox.top/v1/payment/products/list` |
| Cleartext traffic permitted globally | `network_security_config.xml`, `network_config.xml` |
| User-installed certs trusted (debug config) | `network_config.xml` |
| Third-party tracking/ad domains | `snssdk.com`, `bdurl.net`, `fengkongcloud.com`, push vendor domains |

## Privacy-sensitive SDKs and permissions

| SDK / Permission | Purpose | Evidence |
|------------------|---------|----------|
| JPush | Push notifications | Manifest permissions/receivers; `push_jiguang_process_name` |
| MiPush / Huawei / Meizu / Oppo | OEM push channels | Manifest permissions/receivers |
| AdBright / GDT / Pangle | Splash and in-feed ads | `SplashActivity.java`; ad layouts/strings |
| QQ / WeChat SDK | Social login and share | Manifest queries; strings `login_qq`, `login_wechat` |
| Alipay SDK | Payments | Manifest queries; `AliPayResult` model |
| SensorsData | Analytics | `SensorsDataAutoTrackHelper` in click listeners |
| Agora | Audio live rooms | `agora_app_id`; `AudioLive*` fragments/activities |
| Fengkong / anti-fraud | Risk control | `fengkongcloud.com` in network config |
| `READ_PHONE_STATE` | Device identification | Manifest |
| `QUERY_ALL_PACKAGES` | Package visibility | Manifest |
| `GET_TASKS` / `REORDER_TASKS` | Task management | Manifest |

## Key runtime events (from earlier session)
1. **Cold launch:** `SplashActivity` launched.
2. **Privacy prompt:** Modal dialog appeared with `tvTitle`, `tvContent`, `tvQuite`, `tvAgree`.
3. **Consent action:** Tapping **Agree** triggered native library initialisation (`MMKV`), which crashed on x86 due to ARM-only `libmmkv.so`.
4. **Result:** No account, feed, chat, wallet, or settings routes could be reached at runtime.

## Unknown / runtime-unverified
- Exact animation/interaction timing on deeper screens.
- Whether all declared permissions are actually requested at runtime.
- Server response shapes and error states.
- In-room audio-live UX details.
- Complete moderation/filter flows.

## Notes
- Documents contain only original behavior descriptions; no extracted source code, proprietary graphics, or copied long-form text.
