package app.rebuild.social.feature.feed

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.ButtonVariant
import app.rebuild.social.core.design.components.LanternButton
import app.rebuild.social.core.design.components.LanternInput
import app.rebuild.social.core.network.Topic

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ComposerSheet(
    topics: List<Topic>,
    isPosting: Boolean,
    onDismiss: () -> Unit,
    onPublish: (content: String, topicId: String) -> Unit
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var content by remember { mutableStateOf("") }
    var selectedTopicId by remember { mutableStateOf<String?>(topics.firstOrNull()?.id) }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(
                    horizontal = LanternSpacing.bottomSheetPadding,
                    vertical = LanternSpacing.space3
                )
        ) {
            Text(
                text = "写点什么",
                style = LanternType.headingMedium,
                color = MaterialTheme.colorScheme.onSurface
            )

            if (topics.isNotEmpty()) {
                Text(
                    text = "选择话题",
                    style = LanternType.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = LanternSpacing.space3, bottom = LanternSpacing.space1)
                )
                LazyRow(
                    horizontalArrangement = Arrangement.spacedBy(LanternSpacing.space2),
                    contentPadding = PaddingValues(vertical = LanternSpacing.space1)
                ) {
                    items(topics, key = { it.id }) { topic ->
                        FilterChip(
                            selected = selectedTopicId == topic.id,
                            onClick = { selectedTopicId = topic.id },
                            label = { Text(topic.name) }
                        )
                    }
                }
            }

            LanternInput(
                value = content,
                onValueChange = { content = it.take(2000) },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = LanternSpacing.space3),
                placeholder = "分享你的想法…",
                singleLine = false,
                maxLines = 8,
                minHeight = 120.dp,
                textStyle = LanternType.bodyLarge
            )

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = LanternSpacing.space3, bottom = LanternSpacing.space4),
                horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Box(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "${content.length}/2000",
                        style = LanternType.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                LanternButton(
                    label = if (isPosting) "发布中…" else "发布",
                    onClick = {
                        selectedTopicId?.let { onPublish(content, it) }
                    },
                    enabled = !isPosting && selectedTopicId != null && content.trim().isNotEmpty(),
                    variant = ButtonVariant.FilledPrimary
                )
            }
        }
    }
}
