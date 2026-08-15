package app.rebuild.social.feature.auth

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AlternateEmail
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.QrCode2
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.withStyle
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternIcon
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.ButtonVariant
import app.rebuild.social.core.design.components.LanternButton
import app.rebuild.social.core.design.components.LanternInput

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EmailSignInScreen(
    onSignInSubmitted: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    var phone by remember { mutableStateOf("") }
    var agreed by remember { mutableStateOf(false) }

    Scaffold(
        modifier = modifier.testTag("email-signin-screen"),
        topBar = {
            TopAppBar(
                title = { Text("登录", style = LanternType.headingLarge) }
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = LanternSpacing.screenHorizontal)
        ) {
            Spacer(modifier = Modifier.height(LanternSpacing.space8))
            Text(
                text = "登录即表示你已阅读并同意服务协议与隐私政策",
                style = LanternType.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "+86",
                    style = LanternType.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurface
                )
                Spacer(modifier = Modifier.size(LanternSpacing.space3))
                LanternInput(
                    value = phone,
                    onValueChange = { phone = it },
                    label = "",
                    placeholder = "请输入手机号"
                )
            }
            Spacer(modifier = Modifier.height(LanternSpacing.space2))
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Checkbox(
                    checked = agreed,
                    onCheckedChange = { agreed = it },
                    colors = CheckboxDefaults.colors(
                        checkedColor = LanternColors.primary,
                        uncheckedColor = LanternColors.outline
                    )
                )
                Text(
                    text = buildAnnotatedString {
                        append("我已阅读并同意")
                        withStyle(
                            style = SpanStyle(
                                color = LanternColors.primary,
                                textDecoration = TextDecoration.Underline
                            )
                        ) { append("《服务协议》") }
                        append("与")
                        withStyle(
                            style = SpanStyle(
                                color = LanternColors.primary,
                                textDecoration = TextDecoration.Underline
                            )
                        ) { append("《隐私政策》") }
                    },
                    style = LanternType.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            LanternButton(
                label = "一键登录",
                onClick = onSignInSubmitted,
                variant = ButtonVariant.FilledPrimary,
                modifier = Modifier.fillMaxWidth(),
                enabled = agreed
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space8))
            Text(
                text = "其他登录方式",
                style = LanternType.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.align(Alignment.CenterHorizontally)
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Outlined.ChatBubbleOutline,
                    contentDescription = "验证码登录",
                    modifier = Modifier
                        .size(LanternIcon.medium)
                        .clickable { onSignInSubmitted() },
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Spacer(modifier = Modifier.size(LanternSpacing.space6))
                Icon(
                    imageVector = Icons.Outlined.AlternateEmail,
                    contentDescription = "邮箱登录",
                    modifier = Modifier
                        .size(LanternIcon.medium)
                        .clickable { onSignInSubmitted() },
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Spacer(modifier = Modifier.size(LanternSpacing.space6))
                Icon(
                    imageVector = Icons.Outlined.QrCode2,
                    contentDescription = "扫码登录",
                    modifier = Modifier
                        .size(LanternIcon.medium)
                        .clickable { onSignInSubmitted() },
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}
