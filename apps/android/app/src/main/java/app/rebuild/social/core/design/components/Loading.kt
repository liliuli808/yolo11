package app.rebuild.social.core.design.components

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import app.rebuild.social.core.design.LanternIcon

@Composable
fun LoadingIndicator(
    modifier: Modifier = Modifier,
    testTag: String = "loading-indicator"
) {
    Box(
        modifier = modifier.testTag(testTag),
        contentAlignment = Alignment.Center
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(LanternIcon.large),
            color = MaterialTheme.colorScheme.primary
        )
    }
}
