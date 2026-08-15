package app.rebuild.social.feature.settings

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternIcon
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType

/**
 * Settings list aligned to ui-structures §22: 40dp rows with 20dp start padding,
 * trailing value/switch/chip + chevron, 1dp section dividers.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    var teenMode by remember { mutableStateOf(false) }
    var nightMode by remember { mutableStateOf(true) }

    Scaffold(
        modifier = modifier.testTag("settings-screen"),
        topBar = {
            TopAppBar(
                title = { Text("设置", style = LanternType.headingLarge) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "返回"
                        )
                    }
                }
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
        ) {
            SettingsRow("个人资料", onClick = {})
            SettingsRow("评论昵称", value = "漂流木", onClick = {})
            SettingsRow("我喜欢的帖子", onClick = {})
            SettingsRow("账户与安全", onClick = {})
            SettingsRow("青少年模式", trailing = { LanternSwitch(checked = teenMode, onCheckedChange = { teenMode = it }) })
            SectionDivider()
            SettingsRow("声音提示", onClick = {})
            SettingsRow("夜间模式", trailing = { LanternSwitch(checked = nightMode, onCheckedChange = { nightMode = it }) })
            SectionDivider()
            SettingsRow("隐私设置", onClick = {})
            SettingsRow("用户协议", onClick = {})
            SettingsRow("隐私政策", onClick = {})
            SettingsRow("个人信息收集清单", onClick = {})
            SettingsRow("第三方信息共享清单", onClick = {})
            SettingsRow("儿童个人信息保护规则及监护人须知", onClick = {})
            SectionDivider()
            SettingsRow("联系人工客服", onClick = {})
            SettingsRow("给 Lantern 提意见", onClick = {})
            SettingsRow("去应用商店评价", onClick = {})
            SettingsRow("清除缓存", value = "12.3 MB", onClick = {})
            SettingsRow("检查更新", value = "0.1.0", onClick = {})
            SectionDivider()
            SettingsRow("退出登录", onClick = {})
            Spacer(modifier = Modifier.height(LanternSpacing.space6))
        }
    }
}

@Composable
private fun SettingsRow(
    label: String,
    modifier: Modifier = Modifier,
    value: String? = null,
    onClick: (() -> Unit)? = null,
    trailing: @Composable (() -> Unit)? = null
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .heightIn(min = LanternSpacing.space8)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(start = LanternSpacing.space5, end = LanternSpacing.space3),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = label,
            style = LanternType.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface,
            modifier = Modifier.weight(1f)
        )
        if (value != null) {
            Text(
                text = value,
                style = LanternType.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(end = LanternSpacing.space1)
            )
        }
        if (trailing != null) {
            trailing()
        } else if (onClick != null) {
            Icon(
                imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(LanternIcon.medium)
            )
        }
    }
}

@Composable
private fun SectionDivider() {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = LanternSpacing.space3)
            .height(1.dp)
            .background(MaterialTheme.colorScheme.outline)
    )
}

@Composable
private fun LanternSwitch(
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit
) {
    Switch(
        checked = checked,
        onCheckedChange = onCheckedChange,
        colors = SwitchDefaults.colors(
            checkedTrackColor = LanternColors.primary,
            checkedThumbColor = LanternColors.onPrimary,
            uncheckedTrackColor = LanternColors.outline,
            uncheckedThumbColor = LanternColors.surface
        )
    )
}
