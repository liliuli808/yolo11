package app.rebuild.social.feature.persona

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.LanternAvatar

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PersonaScreen(
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    onOpenFlash: () -> Unit = {}
) {
    Scaffold(
        modifier = modifier.testTag("persona-screen"),
        topBar = {
            TopAppBar(
                title = { Text("Persona", style = LanternType.headingLarge) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "Back"
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
                .padding(horizontal = LanternSpacing.screenHorizontal),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            LanternAvatar(imageUrl = null, alias = "You")
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Text(
                text = "Persona profile",
                style = LanternType.displaySmall,
                color = MaterialTheme.colorScheme.onBackground
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space3))
            Text(
                text = "Create and manage your anonymous personas here.",
                style = LanternType.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(onClick = onOpenFlash)
                    .padding(vertical = LanternSpacing.space2),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "闪聊",
                    style = LanternType.bodyLarge.copy(color = LanternColors.primary),
                    modifier = Modifier.weight(1f)
                )
                Icon(
                    imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                    contentDescription = null,
                    tint = LanternColors.primary
                )
            }
        }
    }
}
