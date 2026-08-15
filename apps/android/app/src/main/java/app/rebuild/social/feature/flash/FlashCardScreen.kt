package app.rebuild.social.feature.flash

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FilterList
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.platform.testTag
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.LanternAvatar
import app.rebuild.social.core.design.components.LanternTopBar

enum class GenderFilter(val label: String) { ALL("不限"), MALE("男生"), FEMALE("女生") }

enum class AgeFilter(val label: String) {
    ALL("不限"), G1824("18-24"), G2530("25-30"), G3140("31-40")
}

data class FlashCardItem(
    val id: String,
    val title: String,
    val subtitle: String,
    val alias: String,
    val meta: String,
    val gender: GenderFilter,
    val age: AgeFilter
)

private val Cards = listOf(
    FlashCardItem("f1", "喜欢夜跑的人", "九点后的跑道最安静。想找一个能一起慢慢跑的搭子，不用说话。", "鹿鲸", "29 岁 · 2.3km", GenderFilter.MALE, AgeFilter.G2530),
    FlashCardItem("f2", "周五的蓝色时刻", "图书馆天台傍晚的晚霞最好看，等一个同样爱拍照的人。", "南栀", "26 岁 · 1.1km", GenderFilter.FEMALE, AgeFilter.G2530),
    FlashCardItem("f3", "把遗憾写进歌里", "会弹一点吉他。想把你的故事收集起来，写成一首旋律。", "木吉", "31 岁 · 5km", GenderFilter.MALE, AgeFilter.G3140),
    FlashCardItem("f4", "凌晨四点的便利店", "加班到很晚，困到不行却舍不得睡，来聊聊吗？", "匿名", "0.8km", GenderFilter.ALL, AgeFilter.G1824)
)

/**
 * Flash card flow (evidence: ui-structures §18/§19).
 * 11dp rounded cards with dashed divider, persona row and 私聊 CTA; right-top filter
 * opens a bottom sheet with gender / age range single-select chips.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FlashCardScreen(
    onBack: () -> Unit,
    onOpenChat: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    var showFilter by remember { mutableStateOf(false) }
    var gender by remember { mutableStateOf(GenderFilter.ALL) }
    var age by remember { mutableStateOf(AgeFilter.ALL) }

    val filtered = Cards.filter {
        (gender == GenderFilter.ALL || it.gender == GenderFilter.ALL || it.gender == gender) &&
            (age == AgeFilter.ALL || it.age == age)
    }

    Scaffold(
        modifier = modifier.testTag("flash-screen"),
        topBar = {
            LanternTopBar(
                title = "闪聊",
                onBack = onBack,
                rightText = "筛选",
                onRight = { showFilter = true }
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
            verticalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(14.dp)
        ) {
            items(filtered, key = { it.id }) { card ->
                FlashCard(card = card, onOpenChat = { onOpenChat(card.id) })
            }
        }
    }

    if (showFilter) {
        FlashFilterSheet(
            initialGender = gender,
            initialAge = age,
            onGender = { gender = it },
            onAge = { age = it },
            onDismiss = { showFilter = false }
        )
    }
}

@Composable
private fun FlashCard(card: FlashCardItem, onOpenChat: () -> Unit) {
    Card(
        shape = RoundedCornerShape(11.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(horizontal = 20.dp, vertical = 18.dp)) {
            Text(
                text = card.title,
                style = LanternType.headingSmall.copy(fontSize = 18.sp),
                color = MaterialTheme.colorScheme.onSurface
            )
            Spacer(modifier = Modifier.height(6.dp))
            Text(
                text = card.subtitle,
                style = LanternType.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            val dividerColor = MaterialTheme.colorScheme.outlineVariant
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(18.dp),
                contentAlignment = Alignment.Center
            ) {
                Canvas(modifier = Modifier.fillMaxWidth()) {
                    val dash = 6.dp.toPx()
                    val gap = 4.dp.toPx()
                    val y = size.height / 2
                    var x = 0f
                    while (x < size.width) {
                        drawLine(
                            color = dividerColor,
                            start = androidx.compose.ui.geometry.Offset(x, y),
                            end = androidx.compose.ui.geometry.Offset(x + dash, y),
                            strokeWidth = 1.dp.toPx()
                        )
                        x += dash + gap
                    }
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                LanternAvatar(imageUrl = null, alias = card.alias, size = 36.dp)
                Spacer(modifier = Modifier.width(10.dp))
                Column {
                    Text(
                        text = card.alias,
                        style = LanternType.bodyMedium.copy(fontSize = 14.sp),
                        fontWeight = androidx.compose.ui.text.font.FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Text(
                        text = card.meta,
                        style = LanternType.bodyTiny,
                        color = LanternColors.cardTimeText
                    )
                }
                Spacer(modifier = Modifier.weight(1f))
                Text(
                    text = "私聊 TA",
                    style = LanternType.bodyMedium,
                    color = LanternColors.primary,
                    modifier = Modifier
                        .clip(RoundedCornerShape(50))
                        .clickable(onClick = onOpenChat)
                        .padding(horizontal = 14.dp, vertical = 8.dp)
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun FlashFilterSheet(
    initialGender: GenderFilter,
    initialAge: AgeFilter,
    onGender: (GenderFilter) -> Unit,
    onAge: (AgeFilter) -> Unit,
    onDismiss: () -> Unit
) {
    var gender by remember { mutableStateOf(initialGender) }
    var age by remember { mutableStateOf(initialAge) }

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(modifier = Modifier.padding(horizontal = 20.dp, vertical = 16.dp)) {
            Text(
                text = "筛选",
                style = LanternType.headingMedium,
                color = MaterialTheme.colorScheme.onSurface
            )
            Text(
                text = "性别",
                style = LanternType.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 16.dp, bottom = 8.dp)
            )
            Row {
                GenderFilter.entries.forEach {
                    FilterChip(
                        selected = gender == it,
                        onClick = { gender = it },
                        label = { Text(it.label, style = LanternType.bodyMedium) },
                        colors = FilterChipDefaults.filterChipColors(
                            selectedContainerColor = LanternColors.primary,
                            selectedLabelColor = LanternColors.onPrimary,
                            containerColor = MaterialTheme.colorScheme.surfaceVariant,
                            labelColor = MaterialTheme.colorScheme.onSurfaceVariant
                        ),
                        modifier = Modifier.padding(end = 8.dp)
                    )
                }
            }
            Text(
                text = "年龄段",
                style = LanternType.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 16.dp, bottom = 8.dp)
            )
            Row {
                AgeFilter.entries.forEach {
                    FilterChip(
                        selected = age == it,
                        onClick = { age = it },
                        label = { Text(it.label, style = LanternType.bodyMedium) },
                        colors = FilterChipDefaults.filterChipColors(
                            selectedContainerColor = LanternColors.primary,
                            selectedLabelColor = LanternColors.onPrimary,
                            containerColor = MaterialTheme.colorScheme.surfaceVariant,
                            labelColor = MaterialTheme.colorScheme.onSurfaceVariant
                        ),
                        modifier = Modifier.padding(end = 8.dp)
                    )
                }
            }
            Button(
                onClick = {
                    onGender(gender)
                    onAge(age)
                    onDismiss()
                },
                shape = RoundedCornerShape(50),
                colors = ButtonDefaults.buttonColors(
                    containerColor = LanternColors.primary,
                    contentColor = LanternColors.onPrimary
                ),
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 24.dp, bottom = 24.dp)
            ) {
                Text(text = "确定", style = LanternType.bodyMedium)
            }
        }
    }
}