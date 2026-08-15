# Navigation Graph — nav_index.xml (一罐 APK, v3.16.10)

> **Clean-room note:** Records *route/structure evidence only* for frontend replication.
> No original identifiers, endpoints, branding, or code may be copied to `app.rebuild.social`.

Source: `decompiled/jadx-out/resources/res/navigation/nav_index.xml`
(single Navigation Component graph; **61 fragments**, **59 actions**).

## 1. Transition system (uniform)

Every action uses the same four slide animations (right-push):

| Slot | Animation |
|------|-----------|
| `enterAnim` | `@anim/slide_in_right` |
| `exitAnim` | `@anim/slide_out_left` |
| `popEnterAnim` | `@anim/slide_in_left` |
| `popExitAnim` | `@anim/slide_out_right` |

Two destinations (`to_flash_card_random` only) additionally use `app:anim/keep_origin`
for a tab-overlay effect.

`app:launchSingleTop="true"` is set on 47 of 59 actions (all leaf/editor screens).

## 2. Fragment inventory (61)

### Index / main tabs (hosted by `IndexFragment`, 4 tabs via ViewPager)
| Destination | Class | Notes |
|-------------|-------|-------|
| `indexFragment` | `ui.index.IndexFragment` | main 4-tab host |
| `indexPlaceHolderFragment` | `ui.fragments.IndexPlaceHolderFragment` | blank placeholder tab |
| `indexTagListFragment` | `plugin.index.ui.IndexTagListFragment` | 话题 tags |
| `moodListFragment` | `plugin.index.ui.IndexMoodListFragment` | 全部/心情 filter list |
| `moodDiaryFragment` | `plugin.mood.ui.MoodDiaryFragment` | args: `mid` (mood id), `tid` (tag id) |
| `tagDiaryFragment` | `plugin.tag.ui.TagDiaryFragment` | args: `mid`, `tid`, `refer` |
| `indexFlashChatFragment` | `ui.index.IndexFlashChatFragment` | flash-chat tab surface |

### Diary
| Destination | Class | Args |
|-------------|-------|------|
| `diaryDetailFragment` | `plugin.diarydetail.ui.DiaryDetailFragment` | `did`, `refer`, `scrollToComment`, `readOnly` |
| `addDiaryFragment` | `plugin.edit.ui.AddDiaryFragment` | `mid`, `tid` |
| `editDiaryFragment` | `plugin.edit.ui.EditDiaryFragment` | `did` |
| `moodTagListFragment` | `plugin.mood.ui.MoodTagListFragment` | `mid` |
| `editDiaryVoteFragment` | `plugin.edit.ui.EditDiaryVoteFragment` | — |
| `editDiaryTagFragment` | `plugin.edit.ui.EditDiaryTagFragment` | `mid` |
| `editDiaryMoodTagFragment` | `plugin.edit.ui.EditDiaryMoodTagFragment` | `lastMid` |
| `editDiaryInteractiveStateFragment` | `plugin.edit.ui.EditDiaryInteractiveStateFragment` | `select`, `editMode`, `isPublished` |
| `likedDiaryListFragment` | `plugin.self.ui.like.LikedDiaryListFragment` | — |

### Mood / tag
| Destination | Args |
|-------------|------|
| `moodDiaryFragment` (above) | `mid`, `tid` |
| `tagDiaryFragment` (above) | `mid`, `tid`, `refer` |

### Flash-chat (漂流瓶 / flash card)
| Destination | Class | Args |
|-------------|-------|------|
| `flashCardFragment` | `ui.flashchat.FlashCardFragment` | — |
| `selfFlashCardFragment` | `ui.flashchat.SelfFlashCardFragment` | — |
| `addFlashCardFragment` | `ui.flashchat.AddFlashCardFragment` | — |
| `editFlashCardFragment` | `ui.flashchat.EditFlashCardFragment` | `isRealTop` |
| `flashCardDetailFragment` | `ui.flashchat.FlashCardDetailFragment` | `fid`, `readOnly` |
| `flashCardRandomFragment` | `ui.flashchat.FlashCardRandomFragment` | — (tab overlay) |

### Chat / chat rooms
| Destination | Class | Args |
|-------------|-------|------|
| `sessionFragment` | `plugin.session.ui.SessionFragment` | session 1v1 chat |
| `systemMsgListFragment` | `plugin.session.ui.system.SystemMsgListFragment` | — |
| `interactiveMsgListFragment` | `plugin.session.ui.interactive.InteractiveMsgListFragment` | — |
| `interactiveUserListFragment` | same pkg `.interactive.InteractiveUserListFragment` | `messageId`, `type` |
| `chatRoomListFragment` | `ui.chatroom.ChatRoomListFragment` | `categoryId`, `categoryName` |
| `chatRoomTipFragment` | `ui.chatroom.ChatRoomTipFragment` | `categoryId`, `roomTemplate` |
| `groupBgPreviewFragment` | `plugin.group.ui.GroupBgPreviewFragment` | `session`, `groupBgPreview` |
| `editEmoticonFragment` | `im.ui.fragments.EditEmoticonFragment` | — |
| `emoticonDetailFragment` | `im.ui.fragments.EmoticonDetailFragment` | `emoticon` |

### Audio live (audio live rooms)
| Destination | Class | Args |
|-------------|-------|------|
| `startAudioLive` (fragment) | `im.ui.fragments.live.StartAudioLiveFragment` | — |
| `toAudioLive` (fragment) | `ui.index.IndexFlashChatFragment` sibling | — |
| `moreAudioLive` | `ui.audiolive.MoreAudioLiveListFragment` | — |
| `liveRankingFragment` | `ui.audiolive.LiveRankingFragment` | `source` |
| `addBGMFragment` | `im.ui.fragments.live.AudioLiveAddBGMFragment` | — |

### Albums
| Destination | Class | Args |
|-------------|-------|------|
| `albumFragment` | `plugin.album.ui.AlbumFragment` | `albumId` |
| `albumDiaryListFragment` | `plugin.album.ui.AlbumDiaryListFragment` | `id`, `type`, `title` |
| `editDiaryAlbumFragment` | `plugin.edit.ui.EditDiaryAlbumFragment` | `diary`, `readOnly` |
| `editAlbumFragment` | `plugin.album.ui.EditAlbumFragment` | `albumId` |
| `addAlbumFragment` | `plugin.album.ui.AddAlbumFragment` | — |
| `editAlbumTagListFragment` | `plugin.album.ui.EditAlbumTagListFragment` | — |
| `selfAlbumListFragment` | `plugin.self.ui.album.SelfAlbumListFragment` | — |

### Real profile / follow
| Destination | Class | Args |
|-------------|-------|------|
| `realFragment` | `plugin.real.ui.RealFragment` | `realId` |
| `editRealFragment` | `plugin.real.ui.EditRealFragment` | — |
| `editRealEmotionFragment` | `plugin.real.ui.EditRealEmotionFragment` | — |
| `followFragment` | `plugin.self.ui.FollowFragment` | — |

### Login / on-ramp
| Destination | Class | Args |
|-------------|-------|------|
| `bindInputPhoneFragment` | `ui.account.BindInputPhoneFragment` | — |
| `initInfoFragment` | `ui.account.InitInfoFragment` | profile setup after first login |
| `personalInfoCollectFragment` | `ui.profile.PersonalInfoCollectSettingFragment` | — |
| `loginInputCodeFragment` | `ui.account.LoginInputCodeFragment` | `extra_phone`, `extra_type` |

### Wallet / money
| Destination | Class | Args |
|-------------|-------|------|
| `myWalletFragment` | `ui.profile.MyWalletFragment` | — |
| `tradingRecordFragment` | `ui.profile.TradingRecordFragment` | `type` |
| `editWithdrawalInfo` (fragment) | `ui.profile.EditWithdrawalInfoFragment` | — |
| `withdrawal` (fragment) | `ui.profile.WithdrawalFragment` | — |
| `withdrawalSuccess` (fragment) | `ui.profile.WithdrawalSuccessFragment` | `status` |
| `incomeRecordFragment` | `ui.profile.IncomeRecordFragment` | — |

### Settings / profile
| Destination | Class | Args |
|-------------|-------|------|
| `settingsFragment` | `ui.profile.SettingsFragment` | — |
| `debugFragment` | `ui.profile.DebugFragment` | — |
| `accountAndSecurityFragment` | `ui.profile.AccountAndSecurityFragment` | — |
| `pushSettingFragment` | `ui.profile.PushSettingFragment` | — |
| `applyHostFragment` | `ui.profile.ApplyHostFragment` | — |

## 3. Routing map (actions with launch flags)

Editor/leaf routes use `app:launchSingleTop="true"`:
`to_add_diary`, `to_edit_diary`, `to_add_flash_card`, `to_edit_flash_card`,
`to_edit_diary_vote`, `to_edit_diary_tag`, `to_edit_diary_mood_tag`,
`to_edit_diary_interactive_state`, `to_edit_diary_album`, `to_edit_album`,
`to_add_album`, `to_edit_album_tag_list`, `to_edit_real`, `to_edit_real_emotion`,
`to_edit_emoticon`, `to_add_bgm`, `to_edit_withdrawal_info`,
`to_bind_phone`, `to_account_security`, `to_push_setting`, `to_settings`,
`to_personal_info_collect`, `to_init_info`, `to_input_phone_code`,
`to_my_wallet`, `to_income_record`, `to_trading_record`, `to_withdrawal`.

Implication for the rebuild: every screen push = right slide-in; the tab bar is a
ViewPager (not Navigation). Editors are always singleTop to prevent duplicate stacks.

## 4. Tab structure (from `IndexFragment` / `activity_index`)

```
IndexFragment (ScrollableViewPager)
├── Tab 0: IndexDiaryFragment     (一罐 — diary feed, collapsing blue header)
├── Tab 1: IndexWebFragment       (漂流瓶 — drift bottle / web)
├── Tab 2: IndexSessionFragment   (消息 — sessions / chat ; EmptyFragment when logged out)
└── Tab 3: ProfileFragment        (我 — profile)
```
Tab strip: SlidingTabLayout height 59dp, 1dp underline (color from `globalUnderlineColor`).