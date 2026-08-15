package app.rebuild.social.feature.feed

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.rebuild.social.core.network.BlockCreateRequest
import app.rebuild.social.core.network.Comment
import app.rebuild.social.core.network.CommentCreateRequest
import app.rebuild.social.core.network.ContentApiClient
import app.rebuild.social.core.network.Post
import app.rebuild.social.core.network.ReactionCreateRequest
import app.rebuild.social.core.network.ReportCreateRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class PostDetailUiState(
    val isLoading: Boolean = false,
    val isCommenting: Boolean = false,
    val isReporting: Boolean = false,
    val isBlocking: Boolean = false,
    val post: Post? = null,
    val comments: List<Comment> = emptyList(),
    val nextCursor: String? = null,
    val hasMore: Boolean = false,
    val error: String? = null,
    val moderationError: String? = null,
    val notice: String? = null
)

@HiltViewModel
class PostDetailViewModel @Inject constructor(
    private val contentClient: ContentApiClient
) : ViewModel() {

    private val _uiState = MutableStateFlow(PostDetailUiState())
    val uiState: StateFlow<PostDetailUiState> = _uiState.asStateFlow()

    fun load(postId: String) {
        if (_uiState.value.isLoading) return
        _uiState.update { it.copy(isLoading = true, error = null) }
        loadPost(postId)
        loadComments(postId, cursor = null, append = false)
    }

    fun refresh(postId: String) {
        if (_uiState.value.isLoading) return
        _uiState.update { it.copy(isLoading = true, error = null) }
        loadPost(postId)
        loadComments(postId, cursor = null, append = false)
    }

    fun loadMore(postId: String) {
        val s = _uiState.value
        if (s.isLoading || !s.hasMore || s.nextCursor == null) return
        loadComments(postId, cursor = s.nextCursor, append = true)
    }

    fun like(postId: String) {
        val post = _uiState.value.post ?: return
        val isLiked = post.userReaction == "like"

        val optimistic = post.copy(
            userReaction = if (isLiked) null else "like",
            reactionCounts = post.reactionCounts.toMutableMap().apply {
                val current = this["like"] ?: 0
                this["like"] = if (isLiked) maxOf(0, current - 1) else current + 1
            }
        )
        _uiState.update { it.copy(post = optimistic) }

        viewModelScope.launch {
            if (isLiked) {
                contentClient.deleteReaction(postId, "like")
                    .onFailure { e ->
                        _uiState.update { it.copy(post = post, error = e.message ?: "操作失败，请重试。") }
                    }
            } else {
                contentClient.createReaction(postId, ReactionCreateRequest(type = "like"))
                    .onSuccess { summary ->
                        _uiState.update {
                            it.copy(post = it.post?.copy(userReaction = summary.userReaction, reactionCounts = summary.reactionCounts))
                        }
                    }
                    .onFailure { e ->
                        _uiState.update { it.copy(post = post, error = e.message ?: "操作失败，请重试。") }
                    }
            }
        }
    }

    fun createComment(postId: String, content: String) {
        val trimmed = content.trim()
        if (trimmed.isBlank()) return
        _uiState.update { it.copy(isCommenting = true, error = null) }
        viewModelScope.launch {
            val result = contentClient.createComment(postId, CommentCreateRequest(content = trimmed))
            result
                .onSuccess { comment ->
                    _uiState.update { state ->
                        val post = state.post?.copy(replyCount = state.post.replyCount + 1)
                        state.copy(
                            isCommenting = false,
                            post = post,
                            comments = listOf(comment) + state.comments
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(isCommenting = false, error = e.message ?: "评论发布失败。") }
                }
        }
    }

    fun report(targetType: String, targetId: String, category: String, details: String? = null) {
        if (_uiState.value.isReporting) return
        if (category.isBlank() || targetId.isBlank()) return
        _uiState.update { it.copy(isReporting = true, error = null, notice = null) }
        viewModelScope.launch {
            val result = contentClient.createReport(
                ReportCreateRequest(
                    targetType = targetType,
                    targetId = targetId,
                    category = category,
                    details = details?.takeIf { it.isNotBlank() }
                )
            )
            result
                .onSuccess {
                    _uiState.update { it.copy(isReporting = false, notice = "已提交举报，我们会尽快处理。") }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(isReporting = false, moderationError = e.message ?: "举报提交失败。") }
                }
        }
    }

    fun blockPersona(personaId: String) {
        if (_uiState.value.isBlocking) return
        if (personaId.isBlank()) return
        _uiState.update { it.copy(isBlocking = true, error = null, notice = null) }
        viewModelScope.launch {
            val result = contentClient.createBlock(BlockCreateRequest(personaId = personaId))
            result
                .onSuccess {
                    _uiState.update { it.copy(isBlocking = false, notice = "已拉黑该用户。") }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(isBlocking = false, moderationError = e.message ?: "拉黑失败。") }
                }
        }
    }

    fun clearNotice() = _uiState.update { it.copy(notice = null) }
    fun clearModerationError() = _uiState.update { it.copy(moderationError = null) }

    private fun loadPost(postId: String) {
        viewModelScope.launch {
            contentClient.getPost(postId)
                .onSuccess { post -> _uiState.update { it.copy(post = post) } }
                .onFailure { e ->
                    if (_uiState.value.post == null) {
                        _uiState.update { it.copy(error = e.message ?: "加载帖子失败。") }
                    }
                }
        }
    }

    private fun loadComments(postId: String, cursor: String?, append: Boolean) {
        viewModelScope.launch {
            val result = contentClient.listComments(postId, cursor = cursor, limit = 30)
            result
                .onSuccess { page ->
                    _uiState.update { state ->
                        val comments = if (append) state.comments + page.data else page.data
                        state.copy(
                            isLoading = false,
                            comments = comments,
                            nextCursor = page.pagination.nextCursor,
                            hasMore = page.pagination.hasMore,
                            error = if (comments.isEmpty() && !append) null else state.error
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = e.message ?: "加载评论失败。"
                        )
                    }
                }
        }
    }
}
