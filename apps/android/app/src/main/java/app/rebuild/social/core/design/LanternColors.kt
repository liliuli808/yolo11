package app.rebuild.social.core.design

import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.ui.graphics.Color

/**
 * Design tokens aligned to `docs/reverse/ui-style-guide.md`.
 *
 * Colors with direct evidence are marked "evidence"; Material3 semantic roles
 * without a direct source are derived as tints/shades of the evidence palette.
 */
object LanternColors {
    // Core brand — evidence: accent blue #56a5ff
    val primary = Color(0xFF56A5FF)
    val onPrimary = Color(0xFFFFFFFF)
    val primaryContainer = Color(0xFFDBE9FF) // derived light blue tint
    val onPrimaryContainer = Color(0xFF123A63) // derived dark blue

    // Secondary — evidence: green accent #70cea7
    val secondary = Color(0xFF70CEA7)
    val onSecondary = Color(0xFFFFFFFF)
    val secondaryContainer = Color(0xFFE2F5EC) // derived green tint
    val onSecondaryContainer = Color(0xFF11402C) // derived

    // Neutral surfaces — evidence: ui-style-guide §2
    val background = Color(0xFFFFFFFF) // windowBackground / globalBackgroundColor
    val surface = Color(0xFFFFFFFF) // card tint (@null on light = white)
    val surfaceVariant = Color(0xFFF7F7F7) // globalGrayBackgroundColor
    val onSurface = Color(0xFF333333) // globalTitleTextColor / globalContentTextColor
    val onSurfaceVariant = Color(0xFF666666) // globalSubTextColor
    val outline = Color(0xFFECECEC) // globalUnderlineColor
    val outlineVariant = Color(0xFFF2F2F2) // globalCardDividerLineColor

    // Semantic state
    val error = Color(0xFFFF8A8F) // evidence: pink error
    val onError = Color(0xFFFFFFFF)
    val errorContainer = Color(0xFFFFDFE0) // derived pink tint
    val onErrorContainer = Color(0xFF571D1F) // derived
    val success = Color(0xFF70CEA7) // evidence: green accent
    val successContainer = Color(0xFFE2F5EC) // derived
    val warning = Color(0xFFFFB854) // evidence: gold
    val warningContainer = Color(0xFFF8EDD3) // derived gold tint
    val info = Color(0xFF56A5FF)
    val infoContainer = Color(0xFFE0ECF8)
    val disabled = Color(0xFFBDBDBD)
    val disabledContainer = Color(0xFFECECEC)

    // Text levels — evidence: ui-style-guide §2
    val textPrimary = Color(0xFF333333)
    val textSecondary = Color(0xFF666666)
    val textTertiary = Color(0xFF999999)
    val textOnPrimary = Color(0xFFFFFFFF)
    val textOnError = Color(0xFFFFFFFF)
    val textLink = Color(0xFF56A5FF) // evidence: globalTipLinkColor

    // Diary card text — evidence: ui-structures §4 (globalLightC262626 / C808080)
    val cardText = Color(0xFF262626)
    val cardTimeText = Color(0xFF808080)
    val darkCardText = Color(0xFFD3D3D3)
    val darkCardTimeText = Color(0xFF808080)
    val textLinkVisited = Color(0xFF3A7FC4) // derived darker blue

    // Form fields — evidence: ui-style-guide §4/§12
    val fieldLabel = Color(0xFF808080) // c808080, 16sp bold labels
    val inputHint = Color(0x33000000) // inputPanelHintColor light

    // Status accents — evidence: ui-style-guide §7
    val danger = Color(0xFFFF6565) // red (仅自己可见)
    val gold = Color(0xFFFFB854) // badge / 真身

    // Index / diary sticky header — evidence: ui-style-guide §3
    val indexTopExpanded = Color(0xFF56A5FF) // indexTopExpandedColor
    val indexTopExpandedTitle = Color(0xFFFFFFFF) // indexTopExpandedTitleColor
    val indexTopCollapsing = Color(0xFFFFFFFF) // indexTopCollapsingColor
    val indexDiaryTitle = Color(0xFFFFFFFF) // indexDiaryTitleColor
    val indexDiaryMoodTabSelected = Color(0xFFFFFFFF) // indexDiaryMoodTabSelectedColor
    val indexDiaryMoodTabNormal = Color(0xFFFFFFFF) // indexDiaryMoodTabNormalColor
    val lightBlueTab = Color(0xFFCBE1FF) // globalBlueTabDefaultTextColor

    val darkIndexTopExpanded = Color(0xFF262626) // evidence §3 dark
    val darkIndexTopCollapsing = Color(0xFF1A1A1A) // evidence §3 dark
    val darkIndexDiaryTitle = Color(0xFFD3D3D3) // evidence §3 dark
    val darkIndexDiaryMoodTabSelected = Color(0xFF56A5FF) // evidence §7
    val darkIndexDiaryMoodTabNormal = Color(0xFF808080) // evidence §7

    // Moderation
    val moderationPendingReview = Color(0xFFF8EDD3)
    val moderationRejected = Color(0xFFFFDFE0)
    val moderationHidden = Color(0xFFF7F7F7)
    val moderationWarned = Color(0xFFF8EDD3)
    val moderationSuspended = Color(0xFFFFDFE0)

    // System chrome — evidence: android:navigationBarColor
    val systemNavigationBar = Color(0xFFCCCCCC)

    // Chat bubbles — evidence: ui-style-guide §bubble (self = white text on blue,
    // other = #666 text on gray); bubble shapes in ui-structures §17
    val chatBubbleSelf = Color(0xFF56A5FF)
    val chatBubbleSelfText = Color(0xFFFFFFFF)
    val chatBubbleOther = Color(0xFFF2F2F2)
    val chatBubbleOtherText = Color(0xFF666666)
    val darkChatBubbleSelf = Color(0xFF3B7FD4) // derived darker blue
    val darkChatBubbleSelfText = Color(0xFFE6E6E6)
    val darkChatBubbleOther = Color(0xFF262626)
    val darkChatBubbleOtherText = Color(0xFFD3D3D3)

    // Unread bubble — evidence: ui-structures §16 (session unread badge, red)
    val unreadBadge = Color(0xFFFF6565)

    // ---- Dark mode ----
    val darkPrimary = Color(0xFF56A5FF) // evidence: accent stays blue in dark
    val darkOnPrimary = Color(0xFFFFFFFF)
    val darkPrimaryContainer = Color(0xFF2A4A6E) // derived
    val darkOnPrimaryContainer = Color(0xFFD6E6FF) // derived

    val darkSecondary = Color(0xFF70CEA7)
    val darkOnSecondary = Color(0xFFFFFFFF)
    val darkSecondaryContainer = Color(0xFF2A4A3C) // derived
    val darkOnSecondaryContainer = Color(0xFFD7EFE4) // derived

    val darkBackground = Color(0xFF1A1A1A) // evidence: windowBackground dark
    val darkSurface = Color(0xFF262626) // evidence: globalCardTintColor dark
    val darkSurfaceVariant = Color(0xFF1A1A1A) // evidence: gray bg dark
    val darkOnSurface = Color(0xFFD3D3D3) // evidence: title/content text dark
    val darkOnSurfaceVariant = Color(0xFF808080) // evidence: sub text dark
    val darkOutline = Color(0xFF333333) // evidence: underline dark
    val darkOutlineVariant = Color(0x19FFFFFF) // evidence: divider dark

    val darkError = Color(0xFFFF8A8F)
    val darkOnError = Color(0xFF1A1A1A)
    val darkErrorContainer = Color(0xFF5A2525)
    val darkSuccess = Color(0xFF70CEA7)
    val darkSuccessContainer = Color(0xFF24533B)
    val darkWarning = Color(0xFFFFB854)
    val darkWarningContainer = Color(0xFF5A451E)
    val darkInfo = Color(0xFF8BB8E8)
    val darkInfoContainer = Color(0xFF25405E)
    val darkDisabled = Color(0xFF5E5E5E)
    val darkDisabledContainer = Color(0xFF353535)

    val darkTextPrimary = Color(0xFFD3D3D3)
    val darkTextSecondary = Color(0xFF808080)
    val darkTextTertiary = Color(0xFF808080) // evidence: desc text dark == sub
    val darkTextOnPrimary = Color(0xFFFFFFFF)
    val darkTextOnError = Color(0xFF1A1A1A)
    val darkTextLink = Color(0xFF808080) // evidence: tip link dark
    val darkTextLinkVisited = Color(0xFF86C2AC) // derived

    val darkFieldLabel = Color(0xFF808080)
    val darkInputHint = Color(0x80808080) // inputPanelHintColor dark

    val darkModerationPendingReview = Color(0xFF5A451E)
    val darkModerationRejected = Color(0xFF5A2525)
    val darkModerationHidden = Color(0xFF262626)
    val darkModerationWarned = Color(0xFF5A451E)
    val darkModerationSuspended = Color(0xFF5A2525)

    val darkSystemNavigationBar = Color(0xFF262626) // evidence: nav bar dark
}

internal val LightColorScheme = lightColorScheme(
    primary = LanternColors.primary,
    onPrimary = LanternColors.onPrimary,
    primaryContainer = LanternColors.primaryContainer,
    onPrimaryContainer = LanternColors.onPrimaryContainer,
    secondary = LanternColors.secondary,
    onSecondary = LanternColors.onSecondary,
    secondaryContainer = LanternColors.secondaryContainer,
    onSecondaryContainer = LanternColors.onSecondaryContainer,
    background = LanternColors.background,
    surface = LanternColors.surface,
    surfaceVariant = LanternColors.surfaceVariant,
    onSurface = LanternColors.onSurface,
    onSurfaceVariant = LanternColors.onSurfaceVariant,
    outline = LanternColors.outline,
    outlineVariant = LanternColors.outlineVariant,
    error = LanternColors.error,
    onError = LanternColors.onError,
    errorContainer = LanternColors.errorContainer,
)

internal val DarkColorScheme = darkColorScheme(
    primary = LanternColors.darkPrimary,
    onPrimary = LanternColors.darkOnPrimary,
    primaryContainer = LanternColors.darkPrimaryContainer,
    onPrimaryContainer = LanternColors.darkOnPrimaryContainer,
    secondary = LanternColors.darkSecondary,
    onSecondary = LanternColors.darkOnSecondary,
    secondaryContainer = LanternColors.darkSecondaryContainer,
    onSecondaryContainer = LanternColors.darkOnSecondaryContainer,
    background = LanternColors.darkBackground,
    surface = LanternColors.darkSurface,
    surfaceVariant = LanternColors.darkSurfaceVariant,
    onSurface = LanternColors.darkOnSurface,
    onSurfaceVariant = LanternColors.darkOnSurfaceVariant,
    outline = LanternColors.darkOutline,
    outlineVariant = LanternColors.darkOutlineVariant,
    error = LanternColors.darkError,
    onError = LanternColors.darkOnError,
    errorContainer = LanternColors.darkErrorContainer,
)
