package app.rebuild.social.feature.feed

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternRadius
import app.rebuild.social.core.design.LanternType

private val MoodTabs = listOf(
    "此刻", "沙雕", "求撩", "自拍", "求助", "游戏", "壁纸", "追剧", "安利"
)

val FeedHeaderExpandedHeight = 128.dp
val FeedHeaderCollapsedHeight = 56.dp
val FeedHeaderCollapseRange = FeedHeaderExpandedHeight - FeedHeaderCollapsedHeight

/**
 * Sticky collapsing header (evidence: ui-structures §3/§5/§9).
 *
 * The whole header box rises with scroll (`translationY = -progress * range`) while the
 * title bar counter-translates (`+progress * range`), pinning it at the top — matching the
 * original `CollapsingTitleView` (fitsSystemWindows, stays visible as the mood-tab area
 * scrolls away). Progress is 0 (expanded, blue) → 1 (collapsed, white title bar).
 *
 * Mood area mirrors the original `TabLayout`: 16sp labels, 46dp strip, 3dp pill indicator,
 * selected white (on blue) → primary (on white), unselected #cbe1ff → #808080 (§9).
 */
@Composable
fun FeedCollapsingHeader(
    progress: Float,
    selectedMood: String?,
    onMoodSelected: (String?) -> Unit,
    modifier: Modifier = Modifier
) {
    val darkTheme = isSystemInDarkTheme()
    val expandedBg = if (darkTheme) LanternColors.darkIndexTopExpanded else LanternColors.indexTopExpanded
    val collapsedBg = if (darkTheme) LanternColors.darkIndexTopCollapsing else LanternColors.indexTopCollapsing
    val expandedTitle = LanternColors.indexDiaryTitle
    val collapsedTitle = if (darkTheme) LanternColors.darkIndexDiaryTitle else MaterialTheme.colorScheme.onSurface

    val bg = lerp(expandedBg, collapsedBg, progress)
    val titleColor = lerp(expandedTitle, collapsedTitle, progress)
    val tabSelected = lerp(Color.White, LanternColors.darkIndexDiaryMoodTabSelected, progress)
    val tabNormal = lerp(LanternColors.lightBlueTab, LanternColors.darkIndexDiaryMoodTabNormal, progress)
    val tabIndicator = lerp(Color.White, LanternColors.primary, progress)
    val moodAlpha = 1f - progress
    val selectedIndex = MoodTabs.indexOf(selectedMood).takeIf { it >= 0 } ?: 0

    Box(
        modifier = modifier
            .fillMaxWidth()
            .height(FeedHeaderExpandedHeight)
            .graphicsLayer { translationY = -progress * FeedHeaderCollapseRange.toPx() }
            .background(bg)
    ) {
        Column(modifier = Modifier.fillMaxWidth().statusBarsPadding()) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .graphicsLayer { translationY = progress * FeedHeaderCollapseRange.toPx() }
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(FeedHeaderCollapsedHeight)
                        .padding(horizontal = 16.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Lantern",
                        style = MaterialTheme.typography.displaySmall.copy(fontWeight = FontWeight.Bold),
                        color = titleColor,
                        maxLines = 1
                    )
                }
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(1.dp)
                        .alpha(progress)
                        .background(if (darkTheme) LanternColors.darkOutlineVariant else LanternColors.outline)
                )
            }
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(start = 6.dp, end = 6.dp, top = 10.dp)
                    .alpha(moodAlpha)
                    .horizontalScroll(rememberScrollState()),
                verticalAlignment = Alignment.Bottom
            ) {
                MoodTabs.forEachIndexed { index, mood ->
                    val selected = selectedIndex == index
                    Column(
                        modifier = Modifier
                            .clip(RoundedCornerShape(LanternRadius.full))
                            .clickable { onMoodSelected(mood) }
                            .padding(horizontal = 18.dp)
                            .height(46.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = mood,
                            style = LanternType.bodyMedium.copy(fontSize = 16.sp),
                            color = if (selected) tabSelected else tabNormal,
                            modifier = Modifier.weight(1f)
                        )
                        Box(
                            modifier = Modifier
                                .width(24.dp)
                                .height(3.dp)
                                .clip(RoundedCornerShape(50))
                                .background(if (selected) tabIndicator else Color.Transparent)
                        )
                    }
                }
            }
        }
    }
}
