package app.rebuild.social.core.design.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.painter.Painter
import androidx.compose.ui.platform.testTag
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternIcon
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType

@Composable
fun EmptyState(
    title: String,
    message: String,
    modifier: Modifier = Modifier,
    icon: Painter? = null,
    action: @Composable (() -> Unit)? = null,
    testTag: String = "empty-state"
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .testTag(testTag),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        if (icon != null) {
            Icon(
                painter = icon,
                contentDescription = null,
                modifier = Modifier.size(LanternIcon.xLarge),
                tint = LanternColors.secondary
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space8))
        }
        Text(
            text = title,
            style = LanternType.displayMedium,
            color = LanternColors.textPrimary
        )
        Spacer(modifier = Modifier.height(LanternSpacing.space3))
        Text(
            text = message,
            style = LanternType.bodyMedium,
            color = LanternColors.textSecondary
        )
        if (action != null) {
            Spacer(modifier = Modifier.height(LanternSpacing.space3))
            action()
        }
    }
}
