package app.rebuild.social.feature.chat

import androidx.compose.foundation.background
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.Photo
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.LanternAvatar
import app.rebuild.social.core.design.components.LanternTopBar

data class ChatMessage(
    val id: String,
    val fromSelf: Boolean,
    val text: String,
    val time: String? = null
)

// Clean-room placeholder conversation.
private val Messages = listOf(
    ChatMessage("t1", fromSelf = false, text = "", time = "今天 20:31"),
    ChatMessage("1", fromSelf = false, text = "今天工作还顺利吗"),
    ChatMessage("2", fromSelf = true, text = "还好，就是有点累"),
    ChatMessage("3", fromSelf = false, text = "那早点休息呀"),
    ChatMessage("4", fromSelf = true, text = "嗯嗯，你呢"),
    ChatMessage("5", fromSelf = false, text = "我这边也挺忙的，刚下班"),
    ChatMessage("6", fromSelf = true, text = "辛苦了"),
    ChatMessage("t2", fromSelf = false, text = "", time = "今天 21:02"),
    ChatMessage("7", fromSelf = true, text = "对了，你上次推荐的歌我听了，很好听"),
    ChatMessage("8", fromSelf = false, text = "哈哈，就知道你会喜欢"),
    ChatMessage("9", fromSelf = true, text = "下次再推荐几首给我")
)

private val MockPeerNames = mapOf(
    "1" to "小岛", "2" to "薄荷", "3" to "长岛冰茶", "4" to "玻璃珠",
    "r1" to "今晚一起失眠", "r2" to "夜跑打卡群", "r3" to "写作练习室", "r4" to "粤语歌同好会"
)

/**
 * Chat session (evidence: ui-structures §17).
 * Centered time separators, left bubbles (avatar + nickname, gray bg, #666 text),
 * right bubbles (blue bg, white text, 62dp max-width margin), bottom 48dp input bar.
 */
@Composable
fun ChatScreen(
    peerId: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    val peerName = remember(peerId) {
        MockPeerNames[peerId] ?: peerId.takeIf { name -> name.any { it.code > 0x7F } } ?: "匿名"
    }
    val listState = rememberLazyListState()
    var input by remember { mutableStateOf("") }

    Scaffold(
        modifier = modifier.testTag("chat-screen"),
        topBar = { LanternTopBar(title = peerName, onBack = onBack) },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            LazyColumn(
                state = listState,
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .padding(top = 6.dp, bottom = 16.dp)
            ) {
                items(Messages, key = { it.id }) { message ->
                    if (message.time != null) {
                        TimeSeparator(text = message.time)
                    }
                    MessageBubble(message = message)
                }
            }
            ChatInputBar(
                value = input,
                onValueChange = { input = it },
                onSend = { input = "" },
                modifier = Modifier.fillMaxWidth()
            )
        }
    }
}

@Composable
private fun TimeSeparator(text: String) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 12.dp),
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = text,
            style = LanternType.bodyTiny.copy(fontSize = 11.sp),
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}

@Composable
private fun MessageBubble(message: ChatMessage) {
    val darkTheme = isSystemInDarkTheme()
    val selfColor = if (darkTheme) LanternColors.darkChatBubbleSelf else LanternColors.chatBubbleSelf
    val selfTextColor =
        if (darkTheme) LanternColors.darkChatBubbleSelfText else LanternColors.chatBubbleSelfText
    val otherColor = if (darkTheme) LanternColors.darkChatBubbleOther else LanternColors.chatBubbleOther
    val otherTextColor =
        if (darkTheme) LanternColors.darkChatBubbleOtherText else LanternColors.chatBubbleOtherText

    if (message.fromSelf) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 4.dp)
        ) {
            Spacer(modifier = Modifier.weight(1f))
            Box(
                modifier = Modifier
                    .padding(start = 48.dp)
                    .clip(BubbleShapeSelf)
                    .background(selfColor)
                    .padding(horizontal = 15.dp, vertical = 10.dp)
            ) {
                Text(
                    text = message.text,
                    style = LanternType.bodyLarge,
                    color = selfTextColor
                )
            }
        }
    } else {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 4.dp)
        ) {
            LanternAvatar(
                imageUrl = null,
                alias = "聊",
                size = 40.dp,
                modifier = Modifier.align(Alignment.Top)
            )
            Spacer(modifier = Modifier.width(10.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "对方昵称",
                    style = LanternType.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Spacer(modifier = Modifier.size(2.dp))
                Box(
                    modifier = Modifier
                        .padding(end = 62.dp)
                        .clip(BubbleShapeOther)
                        .background(otherColor)
                        .padding(horizontal = 15.dp, vertical = 10.dp)
                ) {
                    Text(
                        text = message.text,
                        style = LanternType.bodyLarge,
                        color = otherTextColor
                    )
                }
            }
        }
    }
}

private val BubbleShapeSelf = RoundedCornerShape(
    topStart = 12.dp, topEnd = 12.dp, bottomStart = 12.dp, bottomEnd = 2.dp
)
private val BubbleShapeOther = RoundedCornerShape(
    topStart = 12.dp, topEnd = 12.dp, bottomStart = 2.dp, bottomEnd = 12.dp
)

@Composable
private fun ChatInputBar(
    value: String,
    onValueChange: (String) -> Unit,
    onSend: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .background(MaterialTheme.colorScheme.surfaceVariant)
            .navigationBarsPadding()
            .imePadding()
            .padding(horizontal = 8.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        IconButton(onClick = {}) {
            Icon(
                imageVector = Icons.Default.Photo,
                contentDescription = "图片",
                tint = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        TextField(
            value = value,
            onValueChange = onValueChange,
            placeholder = {
                Text(
                    text = "想和他说点什么…",
                    style = LanternType.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            },
            textStyle = LanternType.bodyLarge.copy(color = MaterialTheme.colorScheme.onSurface),
            shape = RoundedCornerShape(8.dp),
            colors = TextFieldDefaults.colors(
                focusedIndicatorColor = Color.Transparent,
                unfocusedIndicatorColor = Color.Transparent,
                disabledIndicatorColor = Color.Transparent,
                focusedContainerColor = MaterialTheme.colorScheme.surface,
                unfocusedContainerColor = MaterialTheme.colorScheme.surface
            ),
            modifier = Modifier
                .weight(1f)
                .heightIn(min = 36.dp, max = 88.dp)
        )
        IconButton(onClick = onSend) {
            Icon(
                imageVector = Icons.AutoMirrored.Filled.Send,
                contentDescription = "发送",
                tint = if (value.isBlank()) MaterialTheme.colorScheme.onSurfaceVariant else LanternColors.primary
            )
        }
    }
}