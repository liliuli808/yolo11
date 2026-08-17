package app.rebuild.social.feature.auth

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.hilt.navigation.compose.hiltViewModel
import app.rebuild.social.BuildConfig
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.ButtonVariant
import app.rebuild.social.core.design.components.LanternButton
import app.rebuild.social.core.design.components.LanternInput
import app.rebuild.social.core.design.components.TurnstileWebView

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RegisterScreen(
    onRegistered: () -> Unit,
    onGoToLogin: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    viewModel: AuthViewModel = hiltViewModel()
) {
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    var agreed by remember { mutableStateOf(false) }
    var turnstileToken by remember { mutableStateOf<String?>(null) }
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        modifier = modifier.testTag("register-screen"),
        topBar = {
            TopAppBar(
                title = { Text("注册", style = LanternType.headingLarge) },
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
                .padding(horizontal = LanternSpacing.screenHorizontal)
        ) {
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            LanternInput(
                value = username,
                onValueChange = { username = it },
                label = "用户名",
                placeholder = "3-20位字母数字下划线"
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            LanternInput(
                value = password,
                onValueChange = { password = it },
                label = "密码",
                placeholder = "至少8位",
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password)
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            LanternInput(
                value = confirm,
                onValueChange = { confirm = it },
                label = "确认密码",
                placeholder = "再次输入密码",
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password)
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(
                    checked = agreed,
                    onCheckedChange = { agreed = it }
                )
                Text(
                    text = "我已阅读并同意服务协议与隐私政策",
                    style = LanternType.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            TurnstileWebView(
                siteKey = BuildConfig.TURNSTILE_SITE_KEY,
                onSuccess = { turnstileToken = it },
                onError = { turnstileToken = null },
                modifier = Modifier.fillMaxWidth()
            )
            if (uiState is AuthUiState.Error) {
                Spacer(modifier = Modifier.height(LanternSpacing.space3))
                Text(
                    text = (uiState as AuthUiState.Error).message,
                    style = LanternType.bodyMedium,
                    color = MaterialTheme.colorScheme.error
                )
            }
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            LanternButton(
                label = "注册",
                onClick = {
                    val token = turnstileToken
                    if (token != null && password == confirm) {
                        viewModel.register(username, password, token, onRegistered)
                    }
                },
                variant = ButtonVariant.FilledPrimary,
                modifier = Modifier.fillMaxWidth(),
                enabled = username.isNotBlank() && password.length >= 8 && password == confirm &&
                    agreed && turnstileToken != null && uiState != AuthUiState.Loading
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Text(
                text = "已有账号？去登录",
                style = LanternType.bodyMedium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier
                    .align(Alignment.CenterHorizontally)
                    .clickable { onGoToLogin() }
            )
        }
    }
}
