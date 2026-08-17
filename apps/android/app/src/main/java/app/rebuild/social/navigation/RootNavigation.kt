package app.rebuild.social.navigation

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.key
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import app.rebuild.social.core.design.components.LoadingIndicator
import app.rebuild.social.feature.auth.LoginScreen
import app.rebuild.social.feature.auth.RegisterScreen
import app.rebuild.social.feature.chat.ChatScreen
import app.rebuild.social.feature.feed.FeedScreen
import app.rebuild.social.feature.feed.PostDetailScreen
import app.rebuild.social.feature.flash.FlashCardScreen
import app.rebuild.social.feature.inbox.InboxScreen
import app.rebuild.social.feature.persona.PersonaScreen
import app.rebuild.social.feature.profile.ProfileScreen
import app.rebuild.social.feature.room.ChatRoomListScreen
import app.rebuild.social.feature.settings.SettingsScreen
import app.rebuild.social.feature.welcome.WelcomeScreen

@Composable
fun RootNavigation(
    isAuthenticated: Boolean,
    modifier: Modifier = Modifier,
    navController: NavHostController = rememberNavController()
) {
    key(isAuthenticated) {
        val startDestination = if (isAuthenticated) Routes.Feed.route else Routes.Welcome.route

        NavHost(
            navController = navController,
            startDestination = startDestination,
            modifier = modifier
        ) {
            composable(Routes.Welcome.route) {
                WelcomeScreen(
                    onGetStarted = { navController.navigate(Routes.Login.route) }
                )
            }
            composable(Routes.Login.route) {
                LoginScreen(
                    onSignInSubmitted = {
                        navController.navigate(Routes.Feed.route) {
                            popUpTo(Routes.Welcome.route) { inclusive = true }
                        }
                    },
                    onGoToRegister = { navController.navigate(Routes.Register.route) },
                    onBack = { navController.popBackStack() }
                )
            }
            composable(Routes.Register.route) {
                RegisterScreen(
                    onRegistered = {
                        navController.navigate(Routes.Feed.route) {
                            popUpTo(Routes.Welcome.route) { inclusive = true }
                        }
                    },
                    onGoToLogin = { navController.popBackStack() },
                    onBack = { navController.popBackStack() }
                )
            }
            composable(Routes.Feed.route) {
                FeedScreen(
                    onNavigateToChatRooms = { navController.navigate(Routes.ChatRoomList.route) },
                    onNavigateToInbox = { navController.navigate(Routes.Inbox.route) },
                    onNavigateToProfile = { navController.navigate(Routes.Profile.route) },
                    onNavigateToPost = { navController.navigate(Routes.PostDetail.build(it)) }
                )
            }
            composable(Routes.PostDetail.route) { backStackEntry ->
                val postId = backStackEntry.arguments?.getString(Routes.PostDetail.ARG) ?: ""
                PostDetailScreen(
                    postId = postId,
                    onBack = { navController.popBackStack() }
                )
            }
            composable(Routes.Persona.route) {
                PersonaScreen(
                    onBack = { navController.popBackStack() },
                    onOpenFlash = { navController.navigate(Routes.Flash.route) }
                )
            }
            composable(Routes.Settings.route) {
                SettingsScreen(
                    onBack = { navController.popBackStack() }
                )
            }
            composable(Routes.Inbox.route) {
                InboxScreen(
                    onBack = { navController.popBackStack() },
                    onOpenChat = { navController.navigate(Routes.Chat.build(it)) }
                )
            }
            composable(Routes.Profile.route) {
                ProfileScreen(
                    onOpenSettings = { navController.navigate(Routes.Settings.route) },
                    onBack = { navController.popBackStack() }
                )
            }
            composable(Routes.Flash.route) {
                FlashCardScreen(
                    onBack = { navController.popBackStack() },
                    onOpenChat = { navController.navigate(Routes.Chat.build(it)) }
                )
            }
            composable(Routes.ChatRoomList.route) {
                ChatRoomListScreen(
                    onBack = { navController.popBackStack() },
                    onOpenFlash = { navController.navigate(Routes.Flash.route) },
                    onOpenChat = { navController.navigate(Routes.Chat.build(it)) }
                )
            }
            composable(Routes.Chat.route) { backStackEntry ->
                val peerId = backStackEntry.arguments?.getString(Routes.Chat.ARG) ?: ""
                ChatScreen(
                    peerId = peerId,
                    onBack = { navController.popBackStack() }
                )
            }
        }
    }
}

@Composable
fun RootNavigationWithSession(
    viewModel: SessionViewModel = hiltViewModel(),
    modifier: Modifier = Modifier
) {
    val session by viewModel.session.collectAsStateWithLifecycle()
    val isLoading by viewModel.isLoading.collectAsStateWithLifecycle()

    if (isLoading) {
        LoadingIndicator(modifier = modifier.fillMaxSize())
    } else {
        RootNavigation(
            isAuthenticated = session != null,
            modifier = modifier
        )
    }
}
