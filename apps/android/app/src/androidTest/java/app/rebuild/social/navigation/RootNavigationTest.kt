package app.rebuild.social.navigation

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.navigation.compose.rememberNavController
import org.junit.Rule
import org.junit.Test

class RootNavigationTest {

    @get:Rule
    val composeTestRule = createAndroidComposeRule<androidx.activity.ComponentActivity>()

    @Test
    fun unauthenticatedFlow_startsAtWelcomeAndNavigatesToLogin() {
        composeTestRule.setContent {
            RootNavigation(isAuthenticated = false)
        }

        composeTestRule.onNodeWithTag("welcome-screen").assertIsDisplayed()
        composeTestRule.onNodeWithText("开始").performClick()
        composeTestRule.onNodeWithTag("login-screen").assertIsDisplayed()
    }

    @Test
    fun authenticatedFlow_startsAtFeed() {
        composeTestRule.setContent {
            RootNavigation(isAuthenticated = true)
        }

        composeTestRule.onNodeWithTag("feed-screen").assertIsDisplayed()
    }

    @Test
    fun feedNavigatesToPersonaAndBack() {
        composeTestRule.setContent {
            RootNavigation(
                isAuthenticated = true,
                navController = rememberNavController()
            )
        }

        composeTestRule.onNodeWithTag("feed-screen").assertIsDisplayed()
        composeTestRule.onNodeWithContentDescription("Persona").performClick()
        composeTestRule.onNodeWithTag("persona-screen").assertIsDisplayed()
    }
}
