package app.rebuild.social.feature.feed

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.LinearOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.core.view.WindowCompat
import kotlinx.coroutines.flow.distinctUntilChanged
import android.app.Activity
import app.rebuild.social.core.design.LanternColors
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.EmptyState
import app.rebuild.social.core.design.components.ErrorState
import app.rebuild.social.core.design.components.LoadingIndicator
import app.rebuild.social.core.network.Post

@Composable
fun FeedScreen(
    onNavigateToChatRooms: () -> Unit,
    onNavigateToInbox: () -> Unit,
    onNavigateToProfile: () -> Unit,
    onNavigateToPost: (String) -> Unit,
    modifier: Modifier = Modifier,
    viewModel: FeedViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val listState = rememberLazyListState()
    var selectedMood by remember { mutableStateOf<String?>(null) }
    val layoutDirection = LocalLayoutDirection.current

    val density = LocalDensity.current
    val collapseRangePx = with(density) { FeedHeaderCollapseRange.toPx() }
    val statusBarInset = WindowInsets.statusBars.asPaddingValues().calculateTopPadding()

    val headerProgress by remember {
        derivedStateOf {
            val offset = listState.firstVisibleItemScrollOffset.toFloat()
            (offset / collapseRangePx).coerceIn(0f, 1f)
        }
    }

    val darkTheme = isSystemInDarkTheme()
    val view = LocalView.current
    val themeBackground = MaterialTheme.colorScheme.background
    val headerBg = lerp(
        if (darkTheme) LanternColors.darkIndexTopExpanded else LanternColors.indexTopExpanded,
        if (darkTheme) LanternColors.darkIndexTopCollapsing else LanternColors.indexTopCollapsing,
        headerProgress
    )
    SideEffect {
        val window = (view.context as Activity).window
        window.statusBarColor = headerBg.toArgb()
        WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars =
            !darkTheme && headerProgress > 0.5f
    }
    DisposableEffect(Unit) {
        onDispose {
            val window = (view.context as Activity).window
            window.statusBarColor = themeBackground.toArgb()
            WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars = !darkTheme
        }
    }

    LaunchedEffect(listState.firstVisibleItemIndex) {
        val posts = state.posts
        if (state.hasMore && !state.isLoading && !state.isRefreshing &&
            posts.isNotEmpty() && listState.firstVisibleItemIndex >= posts.size - 3
        ) {
            viewModel.loadMore()
        }
    }

    LaunchedEffect(listState) {
        snapshotFlow { listState.isScrollInProgress }
            .distinctUntilChanged()
            .collect { inProgress ->
                if (!inProgress) {
                    val p = headerProgress
                    if (p > 0.02f && p < 0.98f) {
                        val animator = Animatable(p)
                        val target = if (p > 0.5f) 1f else 0f
                        listState.scroll {
                            var last = animator.value
                            animator.animateTo(
                                targetValue = target,
                                animationSpec = tween(durationMillis = 300, easing = LinearOutSlowInEasing)
                            ) {
                                val delta = value - last
                                last = value
                                scrollBy(delta * collapseRangePx)
                            }
                        }
                    }
                }
            }
    }

    if (state.isComposerOpen) {
        ComposerSheet(
            topics = state.topics,
            isPosting = state.isPosting,
            onDismiss = viewModel::closeComposer,
            onPublish = { content, topicId -> viewModel.createPost(content, topicId) }
        )
    }

    Scaffold(
        modifier = modifier.testTag("feed-screen"),
        bottomBar = {
            LanternBottomBar(
                onOpenComposer = viewModel::openComposer,
                onNavigateToChatRooms = onNavigateToChatRooms,
                onNavigateToInbox = onNavigateToInbox,
                onNavigateToProfile = onNavigateToProfile
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(
                    start = padding.calculateLeftPadding(layoutDirection),
                    end = padding.calculateRightPadding(layoutDirection),
                    bottom = padding.calculateBottomPadding()
                )
        ) {
            when {
                state.isLoading && state.posts.isEmpty() -> {
                    LoadingIndicator(modifier = Modifier.fillMaxSize())
                }
                state.error != null && state.posts.isEmpty() -> {
                    ErrorState(
                        title = "加载失败",
                        message = state.error ?: "请稍后重试。",
                        modifier = Modifier.fillMaxSize(),
                        action = {
                            TextButton(onClick = viewModel::refresh) {
                                Text("重试")
                            }
                        }
                    )
                }
                state.posts.isEmpty() -> {
                    EmptyState(
                        title = "还没有内容",
                        message = "成为第一个在这里分享的人。",
                        modifier = Modifier.fillMaxSize(),
                        action = {
                            TextButton(onClick = viewModel::openComposer) {
                                Text("发布第一条")
                            }
                        }
                    )
                }
                else -> {
                    LazyColumn(
                        state = listState,
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(top = statusBarInset + FeedHeaderExpandedHeight),
                        verticalArrangement = Arrangement.spacedBy(LanternSpacing.cardGap)
                    ) {
                        items(state.posts, key = Post::id) { post ->
                            PostCard(
                                post = post,
                                onClick = { onNavigateToPost(post.id) },
                                modifier = Modifier.padding(horizontal = LanternSpacing.screenHorizontal)
                            )
                        }
                        if (state.hasMore) {
                            item(key = "feed-loading-more") {
                                Box(
                                    modifier = Modifier.fillMaxWidth().padding(LanternSpacing.space3),
                                    contentAlignment = Alignment.Center
                                ) {
                                    LoadingIndicator()
                                }
                            }
                        }
                    }
                }
            }

            FeedCollapsingHeader(
                progress = headerProgress,
                selectedMood = selectedMood,
                onMoodSelected = { selectedMood = it },
                modifier = Modifier.align(Alignment.TopStart)
            )
        }
    }
}

/**
 * Custom bottom nav (evidence: ui-structures §8.3): 4 tabs + center "+" create action,
 * the FAB sitting over the tab bar.
 */
@Composable
private fun LanternBottomBar(
    onOpenComposer: () -> Unit,
    onNavigateToChatRooms: () -> Unit,
    onNavigateToInbox: () -> Unit,
    onNavigateToProfile: () -> Unit
) {
    Box(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp)
                .navigationBarsPadding()
                .background(MaterialTheme.colorScheme.surface),
            verticalAlignment = Alignment.CenterVertically
        ) {
            BottomBarItem(icon = Icons.Default.Home, label = "广场", selected = true, onClick = {})
            BottomBarItem(
                icon = Icons.Outlined.ChatBubbleOutline,
                label = "树洞",
                selected = false,
                onClick = onNavigateToChatRooms
            )
            Spacer(modifier = Modifier.width(56.dp))
            BottomBarItem(
                icon = Icons.Default.Email,
                label = "消息",
                selected = false,
                onClick = onNavigateToInbox
            )
            BottomBarItem(
                icon = Icons.Default.Person,
                label = "我的",
                selected = false,
                onClick = onNavigateToProfile
            )
        }
        FloatingActionButton(
            onClick = onOpenComposer,
            modifier = Modifier
                .align(Alignment.Center)
                .offset(y = (-40).dp),
            containerColor = LanternColors.primary,
            contentColor = LanternColors.onPrimary
        ) {
            Icon(Icons.Default.Add, contentDescription = "发布")
        }
    }
}

@Composable
private fun RowScope.BottomBarItem(
    icon: ImageVector,
    label: String,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val tint = if (selected) LanternColors.primary else MaterialTheme.colorScheme.onSurfaceVariant
    Column(
        modifier = modifier
            .weight(1f)
            .clickable(onClick = onClick)
            .padding(vertical = 4.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Icon(
            imageVector = icon,
            contentDescription = label,
            tint = tint,
            modifier = Modifier.size(24.dp)
        )
        Text(
            text = label,
            style = LanternType.bodySmall,
            color = tint
        )
    }
}
