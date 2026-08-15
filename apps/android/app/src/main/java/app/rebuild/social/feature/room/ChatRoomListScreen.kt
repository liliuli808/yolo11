package app.rebuild.social.feature.room

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.EmptyState
import app.rebuild.social.core.design.components.LanternTopBar

data class ChatRoom(
    val id: String,
    val name: String,
    val description: String,
    val members: String,
    val emoji: String
)

// Clean-room placeholder rooms / templates.
private val Rooms = listOf(
    ChatRoom("r1", "今晚一起失眠", "睡不着就来聊聊，树洞永远营业", "1.2k 人", "🌙"),
    ChatRoom("r2", "夜跑打卡群", "每天一条夜跑记录，互相监督", "86 人", "🏃"),
    ChatRoom("r3", "写作练习室", "每天写 500 字，有人一起才不孤单", "233 人", "✍️"),
    ChatRoom("r4", "粤语歌同好会", "分享你最近循环的那首歌", "419 人", "🎵")
)

private val RoomTemplates = listOf(
    "深夜电台" to "🎙", "读书会" to "📚", "树洞小站" to "🌳",
    "音乐点歌" to "🎵", "游戏搭子" to "🎮", "职场吐槽" to "💼"
)

/**
 * Chat-room list, 树洞 tab (evidence: ui-structures §20).
 * Room card list (9dp padding) + bottom drawer: 1dp top underline, "开启新房间"
 * 40dp bold row with chevron, expanding 160dp template card row.
 */
@Composable
fun ChatRoomListScreen(
    onBack: () -> Unit,
    onOpenFlash: () -> Unit,
    onOpenChat: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    var drawerExpanded by remember { mutableStateOf(false) }
    val chevronRotation by animateFloatAsState(
        targetValue = if (drawerExpanded) 180f else 0f,
        label = "drawerChevron"
    )

    Scaffold(
        modifier = modifier.testTag("chat-room-list-screen"),
        topBar = {
            LanternTopBar(
                title = "树洞",
                onBack = onBack,
                rightText = "闪聊",
                onRight = onOpenFlash
            )
        },
        containerColor = MaterialTheme.colorScheme.background,
        bottomBar = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(MaterialTheme.colorScheme.surface)
            ) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(1.dp)
                        .background(MaterialTheme.colorScheme.outline)
                )
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(40.dp)
                        .clickable { drawerExpanded = !drawerExpanded }
                        .padding(horizontal = 16.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "开启新房间",
                        style = LanternType.bodyLarge.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(modifier = Modifier.weight(1f))
                    Icon(
                        imageVector = Icons.Default.ExpandMore,
                        contentDescription = if (drawerExpanded) "收起" else "展开",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.rotate(chevronRotation)
                    )
                }
                AnimatedVisibility(
                    visible = drawerExpanded,
                    enter = expandVertically(),
                    exit = shrinkVertically()
                ) {
                    LazyRow(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(160.dp)
                            .padding(12.dp),
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        items(RoomTemplates) { (name, emoji) ->
                            RoomTemplateCard(name = name, emoji = emoji, onClick = {
                                onOpenChat(name)
                            })
                        }
                    }
                }
            }
        }
    ) { padding ->
        if (Rooms.isEmpty()) {
            EmptyState(
                title = "还没有房间",
                message = "点下方「开启新房间」创建一个吧。",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
            )
        } else {
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .padding(9.dp),
                verticalArrangement = Arrangement.spacedBy(9.dp)
            ) {
                items(Rooms, key = { it.id }) { room ->
                    RoomCard(room = room, onClick = { onOpenChat(room.id) })
                }
            }
        }
    }
}

@Composable
private fun RoomCard(room: ChatRoom, onClick: () -> Unit) {
    Card(
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .background(LanternColors.primaryContainer, RoundedCornerShape(12.dp)),
                contentAlignment = Alignment.Center
            ) {
                Text(text = room.emoji, fontSize = 22.sp)
            }
            Spacer(modifier = Modifier.width(14.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = room.name,
                    style = LanternType.bodyLarge.copy(fontWeight = FontWeight.Bold),
                    color = MaterialTheme.colorScheme.onSurface
                )
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    text = room.description,
                    style = LanternType.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1
                )
            }
            Spacer(modifier = Modifier.width(10.dp))
            Text(
                text = room.members,
                style = LanternType.bodyTiny,
                color = LanternColors.cardTimeText
            )
        }
    }
}

@Composable
private fun RoomTemplateCard(name: String, emoji: String, onClick: () -> Unit) {
    Card(
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
        modifier = Modifier
            .width(104.dp)
            .height(136.dp)
            .clickable(onClick = onClick)
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(12.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            Text(text = emoji, fontSize = 30.sp)
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = name,
                style = LanternType.bodySmall.copy(fontWeight = FontWeight.Medium),
                color = MaterialTheme.colorScheme.onSurface
            )
        }
    }
}