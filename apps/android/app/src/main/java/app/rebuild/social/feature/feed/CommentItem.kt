package app.rebuild.social.feature.feed

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import app.rebuild.social.core.design.LanternIcon
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.LanternAvatar
import app.rebuild.social.core.network.Comment

@Composable
fun CommentItem(
    comment: Comment,
    onReport: (commentId: String) -> Unit = {},
    onBlock: (personaId: String) -> Unit = {},
    modifier: Modifier = Modifier
) {
    var menuExpanded by remember { mutableStateOf(false) }
    Row(
        modifier = modifier
            .testTag("comment-item-${comment.id}")
            .fillMaxWidth()
            .padding(vertical = LanternSpacing.space2),
        verticalAlignment = Alignment.Top
    ) {
        LanternAvatar(
            imageUrl = null,
            alias = comment.persona.alias,
            size = LanternIcon.avatarSmall
        )
        Spacer(modifier = Modifier.width(LanternSpacing.space2))
        Column(modifier = Modifier.fillMaxWidth()) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(LanternSpacing.space1)
            ) {
                Text(
                    text = comment.persona.alias,
                    style = LanternType.labelLarge,
                    color = MaterialTheme.colorScheme.onSurface
                )
                Spacer(modifier = Modifier.weight(1f))
                Text(
                    text = formatRelativeTime(comment.createdAt),
                    style = LanternType.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                IconButton(onClick = { menuExpanded = true }) {
                    Icon(imageVector = Icons.Default.MoreVert, contentDescription = "更多", modifier = Modifier.size(LanternIcon.small))
                }
                DropdownMenu(
                    expanded = menuExpanded,
                    onDismissRequest = { menuExpanded = false }
                ) {
                    DropdownMenuItem(
                        text = { Text("举报") },
                        onClick = {
                            menuExpanded = false
                            onReport(comment.id)
                        }
                    )
                    DropdownMenuItem(
                        text = { Text("拉黑该用户") },
                        onClick = {
                            menuExpanded = false
                            onBlock(comment.persona.id)
                        }
                    )
                }
            }
            Spacer(modifier = Modifier.height(LanternSpacing.space1))
            Text(
                text = comment.content,
                style = LanternType.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface
            )
            val likes = comment.reactionCounts["like"] ?: 0
            if (likes > 0) {
                Spacer(modifier = Modifier.height(LanternSpacing.space1))
                Text(
                    text = "♡ $likes",
                    style = LanternType.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}
