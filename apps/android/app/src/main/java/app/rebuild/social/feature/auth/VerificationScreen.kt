package app.rebuild.social.feature.auth

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.ButtonVariant
import app.rebuild.social.core.design.components.LanternButton

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VerificationScreen(
    onVerified: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    var code by remember { mutableStateOf("") }

    Scaffold(
        modifier = modifier.testTag("verification-screen"),
        topBar = {
            TopAppBar(
                title = { Text("输入验证码", style = LanternType.headingLarge) },
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
            Spacer(modifier = Modifier.height(LanternSpacing.space6))
            Text(
                text = "我们已发送验证码到你的邮箱",
                style = LanternType.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            VerificationCodeInput(
                value = code,
                onValueChange = { code = it },
                modifier = Modifier.fillMaxWidth()
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            Text(
                text = "60 秒后重新发送",
                style = LanternType.bodyMedium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.align(Alignment.CenterHorizontally)
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            LanternButton(
                label = "验证",
                onClick = onVerified,
                variant = ButtonVariant.FilledPrimary,
                modifier = Modifier.fillMaxWidth(),
                enabled = code.length >= 4
            )
        }
    }
}
