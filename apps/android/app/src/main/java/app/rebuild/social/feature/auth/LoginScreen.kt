package app.rebuild.social.feature.auth

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
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
fun LoginScreen(
    onSignInSubmitted: () -> Unit,
    onGoToRegister: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    viewModel: AuthViewModel = hiltViewModel()
) {
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var turnstileEpoch by remember { mutableStateOf(0) }
    var turnstileToken by remember { mutableStateOf<String?>(null) }
    val uiState by viewModel.uiState.collectAsState()

    LaunchedEffect(uiState) {
        if (uiState is AuthUiState.Error) {
            turnstileToken = null
            turnstileEpoch++
        }
    }

    Scaffold(
        modifier = modifier.testTag("login-screen"),
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
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            LanternInput(
                value = username,
                onValueChange = {
                    username = it
                    if (uiState is AuthUiState.Error) viewModel.clearError()
                },
                label = "用户名",
                placeholder = "请输入用户名"
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            LanternInput(
                value = password,
                onValueChange = {
                    password = it
                    if (uiState is AuthUiState.Error) viewModel.clearError()
                },
                label = "密码",
                placeholder = "请输入密码",
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password)
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            key(turnstileEpoch) {
                TurnstileWebView(
                    siteKey = BuildConfig.TURNSTILE_SITE_KEY,
                    onSuccess = { turnstileToken = it },
                    onError = { turnstileToken = null },
                    modifier = Modifier.fillMaxWidth()
                )
            }
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
                label = "登录",
                onClick = {
                    val token = turnstileToken
                    if (token != null) {
                        viewModel.login(username, password, token, onSignInSubmitted)
                    }
                },
                variant = ButtonVariant.FilledPrimary,
                modifier = Modifier.fillMaxWidth(),
                enabled = username.isNotBlank() && password.length >= 8 && turnstileToken != null &&
                    uiState != AuthUiState.Loading
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Text(
                text = "没有账号？去注册",
                style = LanternType.bodyMedium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier
                    .align(Alignment.CenterHorizontally)
                    .clickable { onGoToRegister() }
            )
        }
    }
}
