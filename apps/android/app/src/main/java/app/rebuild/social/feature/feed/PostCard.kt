package app.rebuild.social.feature.feed

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternIcon
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.LanternAvatar
import app.rebuild.social.core.network.MediaAsset
import app.rebuild.social.core.network.Post
import coil.compose.AsyncImage

/**
 * Flat diary-style feed row (evidence: ui-structures §4).
 * 16dp horizontal padding (applied by the list), 32dp avatar w/ 15dp top margin,
 * 14sp bold name / 10sp time, blue topic tag, 15sp body (line spacing 3dp) clamped
 * to 6 lines with 展开/收起, 1dp bottom divider.
 */
@Composable
fun PostCard(
    post: Post,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    var expanded by remember { mutableStateOf(false) }
    var overflow by remember { mutableStateOf(false) }
    val darkTheme = isSystemInDarkTheme()
    val nameColor = if (darkTheme) LanternColors.darkCardText else LanternColors.cardText

    Column(
        modifier = modifier
            .testTag("post-card-${post.id}")
            .clickable(onClick = onClick)
    ) {
        Row(
            verticalAlignment = Alignment.Top,
            modifier = Modifier.padding(top = 15.dp)
        ) {
            LanternAvatar(
                imageUrl = null,
                alias = post.persona.alias,
                size = LanternIcon.avatarSmall
            )
            Spacer(modifier = Modifier.width(LanternSpacing.space2))
            Column(modifier = Modifier.fillMaxWidth()) {
                Text(
                    text = post.persona.alias,
                    style = LanternType.bodyMedium.copy(fontWeight = FontWeight.Bold),
                    color = nameColor,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    text = formatRelativeTime(post.createdAt),
                    style = LanternType.bodyTiny,
                    color = LanternColors.cardTimeText
                )
            }
        }
        Text(
            text = "· ${post.topic.name}",
            style = LanternType.bodyMedium,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.padding(top = 4.dp)
        )
        Text(
            text = post.content,
            style = LanternType.bodyLarge.copy(lineHeight = 25.sp),
            color = nameColor,
            maxLines = if (expanded) Int.MAX_VALUE else 6,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(top = 6.dp),
            onTextLayout = { result ->
                if (!expanded) {
                    overflow = result.hasVisualOverflow
                }
            }
        )
        if (post.media.isNotEmpty()) {
            Spacer(modifier = Modifier.height(LanternSpacing.space2))
            PostMediaRow(media = post.media)
        }
        if (overflow) {
            Text(
                text = if (expanded) "收起" else "展开",
                style = LanternType.bodyMedium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier
                    .padding(top = 2.dp)
                    .clickable { expanded = !expanded }
            )
        }
        Spacer(modifier = Modifier.height(LanternSpacing.space3))
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(MaterialTheme.colorScheme.outline)
        )
    }
}

@Composable
private fun PostMediaRow(media: List<MediaAsset>) {
    val first = media.first()
    AsyncImage(
        model = first.url,
        contentDescription = null,
        modifier = Modifier
            .fillMaxWidth()
            .height(200.dp),
        contentScale = ContentScale.Crop
    )
}
