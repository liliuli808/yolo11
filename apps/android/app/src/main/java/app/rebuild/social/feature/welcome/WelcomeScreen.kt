package app.rebuild.social.feature.welcome

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Lightbulb
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.style.TextAlign
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternIcon
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.ButtonVariant
import app.rebuild.social.core.design.components.LanternButton

@Composable
fun WelcomeScreen(
    onGetStarted: () -> Unit,
    modifier: Modifier = Modifier
) {
    Scaffold(
        modifier = modifier.testTag("welcome-screen"),
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = LanternSpacing.screenHorizontal),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            Icon(
                imageVector = Icons.Outlined.Lightbulb,
                contentDescription = null,
                modifier = Modifier.size(LanternIcon.xLarge),
                tint = LanternColors.primary
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space10))
            Text(
                text = "灯笼",
                style = LanternType.displayLarge,
                color = MaterialTheme.colorScheme.onBackground,
                textAlign = TextAlign.Center
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space3))
            Text(
                text = "一个安静的地方，分享不必被聚光灯照亮。",
                style = LanternType.bodyLarge,
                color = LanternColors.textSecondary,
                textAlign = TextAlign.Center
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space8))
            LanternButton(
                label = "开始",
                onClick = onGetStarted,
                variant = ButtonVariant.FilledPrimary
            )
        }
    }
}
