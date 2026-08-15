package app.rebuild.social.feature.inbox

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.LanternAvatar
import app.rebuild.social.core.design.components.LanternTopBar

data class SessionItem(
    val id: String,
    val alias: String,
    val preview: String,
    val time: String,
    val unread: Int,
    val muted: Boolean = false
)

private val InboxTabs = listOf("树洞", "私信")

// Clean-room placeholder data; messaging backend is out of scope for this UI pass.
private val Sessions = listOf(
    SessionItem("1", "小岛", "今晚的月亮很圆，忽然就想找人聊聊", "21:04", 2),
    SessionItem("2", "薄荷", "你上次说的那个方法也不对呀", "20:51", 1),
    SessionItem("3", "长岛冰茶", "语音消息 · 3″", "20:27", 0, muted = true),
    SessionItem("4", "玻璃珠", "[图片]", "18:02", 5),
    SessionItem("5", "酥饼", "晚安啦，明天再聊", "昨天", 0),
    SessionItem("6", "原野", "在吗？今天过得怎么样", "周二", 0),
    SessionItem("7", "椿", "我想听听你的看法", "周一", 0)
)

/**
 * Inbox / 消息 tab (evidence: ui-structures §16 + §17 header).
 * Title bar + 2-tab strip (树洞/私信) + session rows: 47dp avatar, bold single-line
 * name + right timestamp, 1-line preview, unread badge, inset divider (80/20dp).
 */
@Composable
fun InboxScreen(
    onBack: () -> Unit,
    onOpenChat: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    var tab by remember { mutableIntStateOf(0) }

    Scaffold(
        modifier = modifier.testTag("inbox-screen"),
        topBar = { LanternTopBar(title = "消息", onBack = onBack) },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Column(modifier = Modifier.padding(padding)) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(MaterialTheme.colorScheme.surface)
                    .padding(horizontal = 16.dp)
            ) {
                InboxTabs.forEachIndexed { index, label ->
                    val selected = tab == index
                    Column(
                        modifier = Modifier
                            .weight(1f)
                            .clickable { tab = index }
                            .padding(vertical = 10.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = label,
                            style = LanternType.bodyLarge,
                            fontWeight = if (selected) FontWeight.Medium else FontWeight.Normal,
                            color = if (selected) LanternColors.primary else MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Box(
                            modifier = Modifier
                                .padding(top = 6.dp)
                                .size(width = 24.dp, height = 2.dp)
                                .clip(RoundedCornerShape(50))
                                .background(if (selected) LanternColors.primary else MaterialTheme.colorScheme.background)
                        )
                    }
                }
            }
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(1.dp)
                    .background(MaterialTheme.colorScheme.outline)
            )
            LazyColumn(modifier = Modifier.fillMaxWidth()) {
                items(Sessions, key = { it.id }) { session ->
                    SessionRow(
                        session = session,
                        onClick = { onOpenChat(session.id) }
                    )
                }
            }
        }
    }
}

@Composable
private fun SessionRow(session: SessionItem, onClick: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = 20.dp, top = 10.dp, end = 20.dp, bottom = 10.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(contentAlignment = Alignment.TopEnd) {
                LanternAvatar(imageUrl = null, alias = session.alias, size = 47.dp)
                if (session.muted) {
                    Box(
                        modifier = Modifier
                            .size(12.dp)
                            .clip(CircleShape)
                            .background(LanternColors.unreadBadge)
                    )
                } else if (session.unread > 0) {
                    Box(
                        modifier = Modifier
                            .size(18.dp)
                            .clip(CircleShape)
                            .background(LanternColors.unreadBadge),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(
                            text = "${session.unread}",
                            style = LanternType.bodyTiny.copy(fontSize = 10.sp),
                            color = LanternColors.onPrimary,
                            fontWeight = FontWeight.Medium
                        )
                    }
                }
            }
            Spacer(modifier = Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = session.alias,
                        style = LanternType.bodyLarge.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.onSurface,
                        maxLines = 1,
                        modifier = Modifier.weight(1f)
                    )
                    Text(
                        text = session.time,
                        style = LanternType.bodyTiny,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                Text(
                    text = session.preview,
                    style = LanternType.bodyLarge,
                    color = LanternColors.cardTimeText,
                    maxLines = 1
                )
            }
        }
        Box(
            modifier = Modifier
                .padding(start = 80.dp, end = 20.dp)
                .fillMaxWidth()
                .height(1.dp)
                .background(MaterialTheme.colorScheme.outline)
        )
    }
}