package app.rebuild.social.feature.profile

import androidx.compose.foundation.background
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.LanternAvatar

data class ProfileInfo(
    val realName: String = "一只夜航船",
    val uid: String = "UID 10902217",
    val meta: String = "来自南方的小城",
    val tags: List<String> = listOf("树洞", "音乐", "失眠"),
    val verified: Boolean = true
)

/**
 * Profile page (evidence: ui-structures §5/§13, §14 TitleView icons).
 * 242dp cover block with dark gradient bottom mask, white content: 40dp top-row
 * actions (back / settings), 64dp avatar, gold 真身 pill, real name 20sp bold,
 * 12sp uid/meta, tag row, real-name pill button below.
 */
@Composable
fun ProfileScreen(
    profile: ProfileInfo = ProfileInfo(),
    onBack: () -> Unit,
    onOpenSettings: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .testTag("profile-screen")
            .background(MaterialTheme.colorScheme.background)
    ) {
        ProfileHeader(profile = profile, onBack = onBack, onOpenSettings = onOpenSettings)
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceAround
            ) {
                StatItem("树洞", "23")
                StatItem("关注", "128")
                StatItem("粉丝", "84")
            }
            Button(
                onClick = {},
                shape = RoundedCornerShape(50),
                colors = ButtonDefaults.buttonColors(
                    containerColor = LanternColors.primary,
                    contentColor = LanternColors.onPrimary
                ),
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 6.dp)
                    .height(40.dp)
            ) {
                Text(text = "编辑资料", style = LanternType.bodyMedium)
            }
            Text(
                text = "他补充了自我介绍：",
                style = LanternType.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 24.dp, bottom = 8.dp)
            )
            Text(
                text = "「想在这里找一个能说话的人，也听听别人的心事。」",
                style = LanternType.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface
            )
        }
    }
}

@Composable
private fun ProfileHeader(
    profile: ProfileInfo,
    onBack: () -> Unit,
    onOpenSettings: () -> Unit
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(242.dp)
            .background(LanternColors.primary)
    ) {
        Box(
            modifier = Modifier
                .matchParentSize()
                .background(
                    Brush.verticalGradient(
                        colors = listOf(Color.Transparent, Color(0xB3000000))
                    )
                )
        )
        IconButton(
            onClick = onBack,
            modifier = Modifier
                .align(Alignment.TopStart)
                .padding(start = 10.dp, top = 10.dp)
        ) {
            Icon(
                imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                contentDescription = "返回",
                tint = Color.White
            )
        }
        IconButton(
            onClick = onOpenSettings,
            modifier = Modifier
                .align(Alignment.TopEnd)
                .padding(end = 10.dp, top = 10.dp)
        ) {
            Icon(
                imageVector = Icons.Default.Settings,
                contentDescription = "设置",
                tint = Color.White
            )
        }
        Column(
            modifier = Modifier
                .align(Alignment.BottomStart)
                .padding(start = 20.dp, end = 20.dp, bottom = 18.dp)
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                LanternAvatar(
                    imageUrl = null,
                    alias = profile.realName,
                    size = 64.dp
                )
                Spacer(modifier = Modifier.width(14.dp))
                Column {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            text = profile.realName,
                            fontSize = 20.sp,
                            fontWeight = FontWeight.Bold,
                            color = Color.White
                        )
                        if (profile.verified) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                modifier = Modifier
                                    .padding(start = 8.dp)
                                    .clip(RoundedCornerShape(4.dp))
                                    .background(LanternColors.gold)
                                    .padding(horizontal = 6.dp, vertical = 2.dp)
                            ) {
                                Icon(
                                    imageVector = Icons.Default.AutoAwesome,
                                    contentDescription = null,
                                    tint = Color.White,
                                    modifier = Modifier.size(12.dp)
                                )
                                Text(
                                    text = "真身",
                                    fontSize = 11.sp,
                                    fontWeight = FontWeight.Medium,
                                    color = Color.White,
                                    modifier = Modifier.padding(start = 2.dp)
                                )
                            }
                        }
                    }
                    Text(
                        text = "${profile.uid} · ${profile.meta}",
                        fontSize = 12.sp,
                        color = Color(0xE6FFFFFF),
                        modifier = Modifier.padding(top = 3.dp)
                    )
                }
            }
            Row(modifier = Modifier.padding(top = 12.dp)) {
                profile.tags.forEach { tag ->
                    Text(
                        text = tag,
                        fontSize = 12.sp,
                        color = Color(0xE6FFFFFF),
                        modifier = Modifier
                            .clip(RoundedCornerShape(50))
                            .background(Color(0x33FFFFFF))
                            .padding(horizontal = 10.dp, vertical = 4.dp)
                    )
                    Spacer(modifier = Modifier.width(6.dp))
                }
            }
        }
    }
}

@Composable
private fun StatItem(label: String, value: String) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = Modifier.padding(vertical = 18.dp)
    ) {
        Text(
            text = value,
            style = LanternType.headingMedium.copy(fontWeight = FontWeight.Bold),
            color = MaterialTheme.colorScheme.onSurface
        )
        Text(
            text = label,
            style = LanternType.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}