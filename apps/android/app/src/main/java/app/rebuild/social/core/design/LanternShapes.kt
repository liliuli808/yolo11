package app.rebuild.social.core.design

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Shapes
import androidx.compose.ui.unit.dp

object LanternShapes {
    val none = RoundedCornerShape(0.dp)
    val small = RoundedCornerShape(LanternRadius.small)
    val medium = RoundedCornerShape(LanternRadius.medium)
    val large = RoundedCornerShape(LanternRadius.large)
    val xLarge = RoundedCornerShape(LanternRadius.xLarge)
    val full = RoundedCornerShape(LanternRadius.full)
    val bottomSheet = RoundedCornerShape(topStart = LanternRadius.large, topEnd = LanternRadius.large)
}

internal fun lanternShapes(): Shapes = Shapes(
    small = LanternShapes.small,
    medium = LanternShapes.medium,
    large = LanternShapes.large,
    extraLarge = LanternShapes.xLarge
)
