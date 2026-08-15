package app.rebuild.social.core.design

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

object LanternType {
    val displayLarge = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 32.sp,
        lineHeight = 40.sp,
        fontWeight = FontWeight.Bold,
        letterSpacing = (-0.5).sp
    )

    val displayMedium = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 28.sp,
        lineHeight = 36.sp,
        fontWeight = FontWeight.Bold,
        letterSpacing = (-0.25).sp
    )

    // text_title 24sp bold — 一罐 collapsed title strip (evidence: 24sp bold)
    val displaySmall = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 24.sp,
        lineHeight = 32.sp,
        fontWeight = FontWeight.Bold,
        letterSpacing = 0.sp
    )

    // 20sp bold — login page titles, profile realName (evidence)
    val headingLarge = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 20.sp,
        lineHeight = 28.sp,
        fontWeight = FontWeight.Bold,
        letterSpacing = 0.sp
    )

    // text_title 18sp bold — centered app-bar title (evidence)
    val headingMedium = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 18.sp,
        lineHeight = 26.sp,
        fontWeight = FontWeight.Bold,
        letterSpacing = 0.sp
    )

    // 16sp bold — comment section header, mood field labels (evidence)
    val headingSmall = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 16.sp,
        lineHeight = 24.sp,
        fontWeight = FontWeight.Bold,
        letterSpacing = 0.1.sp
    )

    // text_common 15sp — body text, list rows, buttons, inputs (evidence)
    val bodyLarge = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 15.sp,
        lineHeight = 22.sp,
        fontWeight = FontWeight.Normal,
        letterSpacing = 0.25.sp
    )

    // text_small 14sp — secondary rows, action labels (evidence)
    val bodyMedium = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 14.sp,
        lineHeight = 20.sp,
        fontWeight = FontWeight.Normal,
        letterSpacing = 0.25.sp
    )

    // text_tiny 12sp — tertiary/labels, timestamps (evidence)
    val bodySmall = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 12.sp,
        lineHeight = 16.sp,
        fontWeight = FontWeight.Normal,
        letterSpacing = 0.4.sp
    )

    // 10sp — diary card timestamp (evidence: ui-structures §4, time 10sp)
    val bodyTiny = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 10.sp,
        lineHeight = 14.sp,
        fontWeight = FontWeight.Normal,
        letterSpacing = 0.4.sp
    )

    // text_common 15sp for buttons (evidence)
    val labelLarge = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 15.sp,
        lineHeight = 22.sp,
        fontWeight = FontWeight.Medium,
        letterSpacing = 0.5.sp
    )

    val labelMedium = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 12.sp,
        lineHeight = 16.sp,
        fontWeight = FontWeight.Medium,
        letterSpacing = 0.5.sp
    )

    val labelSmall = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 11.sp,
        lineHeight = 14.sp,
        fontWeight = FontWeight.Medium,
        letterSpacing = 0.6.sp
    )

    val caption = TextStyle(
        fontFamily = FontFamily.Default,
        fontSize = 12.sp,
        lineHeight = 16.sp,
        fontWeight = FontWeight.Normal,
        letterSpacing = 0.4.sp
    )
}

internal fun lanternTypography(): Typography = Typography(
    displayLarge = LanternType.displayLarge,
    displayMedium = LanternType.displayMedium,
    displaySmall = LanternType.displaySmall,
    headlineLarge = LanternType.headingLarge,
    headlineMedium = LanternType.headingMedium,
    headlineSmall = LanternType.headingSmall,
    titleLarge = LanternType.headingLarge,
    titleMedium = LanternType.headingMedium,
    titleSmall = LanternType.headingSmall,
    bodyLarge = LanternType.bodyLarge,
    bodyMedium = LanternType.bodyMedium,
    bodySmall = LanternType.bodySmall,
    labelLarge = LanternType.labelLarge,
    labelMedium = LanternType.labelMedium,
    labelSmall = LanternType.labelSmall,
)
