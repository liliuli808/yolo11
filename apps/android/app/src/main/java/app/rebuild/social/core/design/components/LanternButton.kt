package app.rebuild.social.core.design.components

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternShapes
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType

sealed class ButtonVariant {
    data object FilledPrimary : ButtonVariant()
    data object FilledSecondary : ButtonVariant()
    data object TonalPrimary : ButtonVariant()
    data object Outlined : ButtonVariant()
    data object Text : ButtonVariant()
    data object Destructive : ButtonVariant()
    data object DestructiveText : ButtonVariant()
}

@Composable
fun LanternButton(
    label: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    variant: ButtonVariant = ButtonVariant.FilledPrimary,
    enabled: Boolean = true
) {
    val shape = when (variant) {
        ButtonVariant.Text, ButtonVariant.DestructiveText -> LanternShapes.none
        else -> LanternShapes.medium
    }

    val contentPadding = PaddingValues(
        horizontal = LanternSpacing.space3,
        vertical = LanternSpacing.space2
    )

    val labelContent = @Composable {
        Text(text = label, style = LanternType.labelLarge)
    }

    when (variant) {
        ButtonVariant.FilledPrimary -> Button(
            onClick = onClick,
            modifier = modifier.heightIn(min = LanternSpacing.buttonMinHeight),
            enabled = enabled,
            shape = shape,
            colors = ButtonDefaults.buttonColors(
                containerColor = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary,
                disabledContainerColor = LanternColors.disabledContainer,
                disabledContentColor = LanternColors.disabled
            ),
            contentPadding = contentPadding,
            content = { labelContent() }
        )

        ButtonVariant.FilledSecondary -> Button(
            onClick = onClick,
            modifier = modifier.heightIn(min = LanternSpacing.buttonMinHeight),
            enabled = enabled,
            shape = shape,
            colors = ButtonDefaults.buttonColors(
                containerColor = LanternColors.secondary,
                contentColor = LanternColors.onSecondary,
                disabledContainerColor = LanternColors.disabledContainer,
                disabledContentColor = LanternColors.disabled
            ),
            contentPadding = contentPadding,
            content = { labelContent() }
        )

        ButtonVariant.TonalPrimary -> Button(
            onClick = onClick,
            modifier = modifier.heightIn(min = LanternSpacing.buttonMinHeight),
            enabled = enabled,
            shape = shape,
            colors = ButtonDefaults.buttonColors(
                containerColor = MaterialTheme.colorScheme.primaryContainer,
                contentColor = MaterialTheme.colorScheme.onPrimaryContainer,
                disabledContainerColor = LanternColors.disabledContainer,
                disabledContentColor = LanternColors.disabled
            ),
            contentPadding = contentPadding,
            content = { labelContent() }
        )

        ButtonVariant.Outlined -> OutlinedButton(
            onClick = onClick,
            modifier = modifier.heightIn(min = LanternSpacing.buttonMinHeight),
            enabled = enabled,
            shape = shape,
            colors = ButtonDefaults.outlinedButtonColors(
                contentColor = MaterialTheme.colorScheme.primary,
                disabledContentColor = LanternColors.disabled
            ),
            contentPadding = contentPadding,
            content = { labelContent() }
        )

        ButtonVariant.Text -> TextButton(
            onClick = onClick,
            modifier = modifier.heightIn(min = LanternSpacing.buttonMinHeight),
            enabled = enabled,
            contentPadding = contentPadding,
            content = { labelContent() }
        )

        ButtonVariant.Destructive -> Button(
            onClick = onClick,
            modifier = modifier.heightIn(min = LanternSpacing.buttonMinHeight),
            enabled = enabled,
            shape = shape,
            colors = ButtonDefaults.buttonColors(
                containerColor = MaterialTheme.colorScheme.error,
                contentColor = MaterialTheme.colorScheme.onError,
                disabledContainerColor = LanternColors.disabledContainer,
                disabledContentColor = LanternColors.disabled
            ),
            contentPadding = contentPadding,
            content = { labelContent() }
        )

        ButtonVariant.DestructiveText -> TextButton(
            onClick = onClick,
            modifier = modifier.heightIn(min = LanternSpacing.buttonMinHeight),
            enabled = enabled,
            colors = ButtonDefaults.textButtonColors(
                contentColor = MaterialTheme.colorScheme.error,
                disabledContentColor = LanternColors.disabled
            ),
            contentPadding = contentPadding,
            content = { labelContent() }
        )
    }
}
