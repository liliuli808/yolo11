package app.rebuild.social.core.design

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Inbox
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import app.rebuild.social.core.design.components.ButtonVariant
import app.rebuild.social.core.design.components.EmptyState
import app.rebuild.social.core.design.components.ErrorState
import app.rebuild.social.core.design.components.LanternButton
import app.rebuild.social.core.design.components.LoadingIndicator
import org.junit.Rule
import org.junit.Test

class DesignComponentsTest {

    @get:Rule
    val composeTestRule = createAndroidComposeRule<androidx.activity.ComponentActivity>()

    @Test
    fun loadingIndicator_isDisplayed() {
        composeTestRule.setContent {
            LanternTheme {
                LoadingIndicator()
            }
        }
        composeTestRule.onNodeWithTag("loading-indicator").assertIsDisplayed()
    }

    @Test
    fun emptyState_displaysTitleAndMessage() {
        composeTestRule.setContent {
            LanternTheme {
                EmptyState(
                    title = "Nothing here",
                    message = "When notes arrive, they will appear here.",
                    icon = Icons.Outlined.Inbox
                )
            }
        }
        composeTestRule.onNodeWithTag("empty-state").assertIsDisplayed()
        composeTestRule.onNodeWithText("Nothing here").assertIsDisplayed()
        composeTestRule.onNodeWithText("When notes arrive, they will appear here.").assertIsDisplayed()
    }

    @Test
    fun errorState_displaysTitleAndMessage() {
        composeTestRule.setContent {
            LanternTheme {
                ErrorState(
                    title = "Something went wrong",
                    message = "Please check your connection and try again."
                )
            }
        }
        composeTestRule.onNodeWithTag("error-state").assertIsDisplayed()
        composeTestRule.onNodeWithText("Something went wrong").assertIsDisplayed()
        composeTestRule.onNodeWithText("Please check your connection and try again.").assertIsDisplayed()
    }

    @Test
    fun lanternButton_displaysLabel() {
        composeTestRule.setContent {
            LanternTheme {
                LanternButton(
                    label = "Continue",
                    onClick = {},
                    variant = ButtonVariant.FilledPrimary
                )
            }
        }
        composeTestRule.onNodeWithText("Continue").assertIsDisplayed()
    }
}
