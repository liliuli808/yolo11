package app.rebuild.social.feature.feed

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import app.rebuild.social.core.design.LanternIcon
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.EmptyState
import app.rebuild.social.core.design.components.ErrorState
import app.rebuild.social.core.design.components.LanternAvatar
import app.rebuild.social.core.design.components.LanternCard
import app.rebuild.social.core.design.components.LanternInput
import app.rebuild.social.core.design.components.LoadingIndicator
import app.rebuild.social.core.network.Comment
import app.rebuild.social.core.network.Post

private val REPORT_CATEGORIES = listOf(
    "harassment" to "骚扰",
    "hateSpeech" to "仇恨言论",
    "harmfulContent" to "有害内容",
    "spam" to "垃圾广告",
    "sexualContent" to "色情内容",
    "doxxing" to "人肉搜索",
    "impersonation" to "冒充他人",
    "illegalContent" to "违法内容",
    "other" to "其他"
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PostDetailScreen(
    postId: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    viewModel: PostDetailViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val listState = rememberLazyListState()
    val snackbarHostState = remember { SnackbarHostState() }
    var commentText by remember { mutableStateOf("") }
    var menuExpanded by remember { mutableStateOf(false) }
    var reportTarget by remember { mutableStateOf<Pair<String, String>?>(null) }
    var blockPersonaId by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(postId) { viewModel.load(postId) }
    LaunchedEffect(listState.firstVisibleItemIndex) {
        val comments = state.comments
        if (state.hasMore && !state.isLoading &&
            comments.isNotEmpty() && listState.firstVisibleItemIndex >= comments.size - 4
        ) {
            viewModel.loadMore(postId)
        }
    }
    LaunchedEffect(state.notice) {
        state.notice?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearNotice()
        }
    }
    LaunchedEffect(state.moderationError) {
        state.moderationError?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearModerationError()
        }
    }

    val authorPersonaId = state.post?.persona?.id

    Scaffold(
        modifier = modifier.testTag("post-detail-screen"),
        topBar = {
            TopAppBar(
                title = { Text("帖子", style = LanternType.headingLarge) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "返回"
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { menuExpanded = true }) {
                        Icon(imageVector = Icons.Default.MoreVert, contentDescription = "更多")
                    }
                    DropdownMenu(
                        expanded = menuExpanded,
                        onDismissRequest = { menuExpanded = false }
                    ) {
                        DropdownMenuItem(
                            text = { Text("举报") },
                            onClick = {
                                menuExpanded = false
                                reportTarget = "post" to postId
                            }
                        )
                        DropdownMenuItem(
                            text = { Text("拉黑作者") },
                            enabled = authorPersonaId != null,
                            onClick = {
                                menuExpanded = false
                                blockPersonaId = authorPersonaId
                            }
                        )
                    }
                }
            )
        },
        bottomBar = {
            CommentComposer(
                value = commentText,
                onValueChange = { commentText = it },
                isSending = state.isCommenting,
                onSend = {
                    viewModel.createComment(postId, commentText)
                    commentText = ""
                }
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
        containerColor = MaterialTheme.colorScheme.surfaceVariant
    ) { padding ->
        when {
            state.isLoading && state.post == null -> {
                LoadingIndicator(modifier = Modifier.fillMaxSize().padding(padding))
            }
            state.error != null && state.post == null -> {
                ErrorState(
                    title = "加载失败",
                    message = state.error ?: "请稍后重试。",
                    modifier = Modifier.fillMaxSize().padding(padding),
                    action = {
                        TextButton(onClick = { viewModel.refresh(postId) }) { Text("重试") }
                    }
                )
            }
            state.post != null -> {
                LazyColumn(
                    state = listState,
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(
                        vertical = LanternSpacing.space3
                    ),
                    verticalArrangement = Arrangement.spacedBy(LanternSpacing.space2)
                ) {
                    item(key = "post-header") {
                        state.post?.let { post ->
                            LanternCard(
                                modifier = Modifier
                                    .padding(top = LanternSpacing.space2)
                                    .padding(horizontal = LanternSpacing.space2)
                            ) {
                                PostDetailHeader(post = post, onLike = { viewModel.like(postId) })
                            }
                        }
                    }
                    if (state.comments.isEmpty()) {
                        item(key = "comments-empty") {
                            LanternCard(modifier = Modifier.padding(horizontal = LanternSpacing.space2)) {
                                EmptyState(
                                    title = "还没有评论",
                                    message = "来抢沙发吧。",
                                    modifier = Modifier.fillMaxWidth()
                                )
                            }
                        }
                    }
                    items(state.comments, key = Comment::id) { comment ->
                        LanternCard(modifier = Modifier.padding(horizontal = LanternSpacing.space2)) {
                            CommentItem(
                                comment = comment,
                                onReport = { reportTarget = "comment" to it },
                                onBlock = { blockPersonaId = it }
                            )
                        }
                    }
                    if (state.hasMore) {
                        item(key = "comments-loading-more") {
                            Box(
                                modifier = Modifier.fillMaxWidth().padding(LanternSpacing.space3),
                                contentAlignment = Alignment.Center
                            ) { LoadingIndicator() }
                        }
                    }
                }
            }
        }
    }

    if (reportTarget != null) {
        ReportDialog(
            isSubmitting = state.isReporting,
            onDismiss = { reportTarget = null },
            onSubmit = { category, details ->
                reportTarget?.let { (type, id) -> viewModel.report(type, id, category, details) }
                reportTarget = null
            }
        )
    }

    if (blockPersonaId != null) {
        AlertDialog(
            onDismissRequest = { blockPersonaId = null },
            title = { Text("拉黑该用户", style = LanternType.headingMedium) },
            text = { Text("拉黑后你将不再看到对方的内容，对方也无法与你互动。", style = LanternType.bodyMedium) },
            confirmButton = {
                TextButton(
                    onClick = {
                        blockPersonaId?.let { viewModel.blockPersona(it) }
                        blockPersonaId = null
                    }
                ) { Text("拉黑") }
            },
            dismissButton = {
                TextButton(onClick = { blockPersonaId = null }) { Text("取消") }
            }
        )
    }
}

@Composable
private fun ReportDialog(
    isSubmitting: Boolean,
    onDismiss: () -> Unit,
    onSubmit: (category: String, details: String?) -> Unit,
    modifier: Modifier = Modifier
) {
    var selectedCategory by remember { mutableStateOf<String?>(null) }
    var details by remember { mutableStateOf("") }

    AlertDialog(
        modifier = modifier,
        onDismissRequest = onDismiss,
        title = { Text("举报内容", style = LanternType.headingMedium) },
        text = {
            Column(modifier = Modifier.verticalScroll(rememberScrollState())) {
                Text("选择举报理由", style = LanternType.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Spacer(modifier = Modifier.height(LanternSpacing.space2))
                Column(verticalArrangement = Arrangement.spacedBy(LanternSpacing.space2)) {
                    REPORT_CATEGORIES.forEach { (value, label) ->
                        FilterChip(
                            selected = selectedCategory == value,
                            onClick = { selectedCategory = value },
                            label = { Text(label) }
                        )
                    }
                }
                Spacer(modifier = Modifier.height(LanternSpacing.space3))
                OutlinedTextField(
                    value = details,
                    onValueChange = { details = it.take(2000) },
                    modifier = Modifier.fillMaxWidth().heightIn(min = 80.dp),
                    placeholder = { Text("补充说明（选填）", style = LanternType.bodyMedium) },
                    textStyle = LanternType.bodyMedium,
                    maxLines = 4
                )
            }
        },
        confirmButton = {
            TextButton(
                enabled = !isSubmitting && selectedCategory != null,
                onClick = { selectedCategory?.let { onSubmit(it, details) } }
            ) { Text(if (isSubmitting) "提交中…" else "提交") }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("取消") }
        }
    )
}

@Composable
private fun PostDetailHeader(post: Post, onLike: () -> Unit, modifier: Modifier = Modifier) {
    LanternCard(modifier = modifier.testTag("post-detail-header")) {
        Row(verticalAlignment = Alignment.Top) {
            LanternAvatar(imageUrl = null, alias = post.persona.alias, size = LanternIcon.avatarDetail)
            Spacer(modifier = Modifier.width(LanternSpacing.space2))
            Column(modifier = Modifier.fillMaxWidth()) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(LanternSpacing.space1)
                ) {
                    Text(
                        text = post.persona.alias,
                        style = LanternType.labelLarge,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(modifier = Modifier.weight(1f))
                    Text(
                        text = "· ${post.topic.name}",
                        style = LanternType.labelSmall,
                        color = MaterialTheme.colorScheme.primary
                    )
                }
                Spacer(modifier = Modifier.height(LanternSpacing.space2))
                Text(
                    text = post.content,
                    style = LanternType.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurface
                )
                Spacer(modifier = Modifier.height(LanternSpacing.space2))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(LanternSpacing.space3),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    val liked = post.userReaction == "like"
                    val likeCount = post.reactionCounts["like"] ?: 0
                    TextButton(onClick = onLike) {
                        Text(
                            text = if (liked) "♥ 已赞 $likeCount" else "♡ 赞 $likeCount",
                            style = LanternType.labelMedium,
                            color = if (liked) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    Text(
                        text = "💬 ${post.replyCount}",
                        style = LanternType.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    if (post.isSaved) {
                        Text(
                            text = "已收藏",
                            style = LanternType.labelMedium,
                            color = MaterialTheme.colorScheme.primary
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun CommentComposer(
    value: String,
    onValueChange: (String) -> Unit,
    isSending: Boolean,
    onSend: () -> Unit,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier
            .fillMaxWidth()
            .imePadding(),
        color = MaterialTheme.colorScheme.surfaceVariant,
        shadowElevation = LanternSpacing.cardGap
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(
                    horizontal = LanternSpacing.screenHorizontal,
                    vertical = LanternSpacing.space1
                ),
            verticalAlignment = Alignment.CenterVertically
        ) {
            LanternInput(
                value = value,
                onValueChange = { onValueChange(it.take(2000)) },
                modifier = Modifier.weight(1f),
                placeholder = "说点什么…",
                singleLine = false,
                maxLines = 4,
                minHeight = 44.dp,
                trailingIcon = {
                    IconButton(
                        onClick = onSend,
                        enabled = !isSending && value.trim().isNotEmpty()
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.Send,
                            contentDescription = "发送",
                            modifier = Modifier.size(LanternIcon.default)
                        )
                    }
                }
            )
        }
    }
}
