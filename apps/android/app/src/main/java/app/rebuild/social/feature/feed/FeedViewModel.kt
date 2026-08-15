package app.rebuild.social.feature.feed

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.rebuild.social.core.network.ContentApiClient
import app.rebuild.social.core.network.Post
import app.rebuild.social.core.network.PostCreateRequest
import app.rebuild.social.core.network.Topic
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class FeedUiState(
    val isLoading: Boolean = false,
    val isRefreshing: Boolean = false,
    val isPosting: Boolean = false,
    val posts: List<Post> = emptyList(),
    val nextCursor: String? = null,
    val hasMore: Boolean = false,
    val error: String? = null,
    val topics: List<Topic> = emptyList(),
    val selectedTopicId: String? = null,
    val isComposerOpen: Boolean = false
)

@HiltViewModel
class FeedViewModel @Inject constructor(
    private val contentClient: ContentApiClient
) : ViewModel() {

    private val _uiState = MutableStateFlow(FeedUiState())
    val uiState: StateFlow<FeedUiState> = _uiState.asStateFlow()

    init {
        loadTopics()
        loadFeed()
    }

    fun loadFeed() {
        if (_uiState.value.isLoading) return
        _uiState.update { it.copy(isLoading = true, error = null) }
        loadPage(cursor = null, append = false)
    }

    fun refresh() {
        if (_uiState.value.isRefreshing) return
        _uiState.update { it.copy(isRefreshing = true, error = null) }
        loadPage(cursor = null, append = false)
    }

    fun loadMore() {
        val s = _uiState.value
        if (s.isLoading || s.isRefreshing || !s.hasMore || s.nextCursor == null) return
        loadPage(cursor = s.nextCursor, append = true)
    }

    fun selectTopic(topicId: String?) {
        if (_uiState.value.selectedTopicId == topicId) return
        _uiState.update { it.copy(selectedTopicId = topicId, posts = emptyList(), nextCursor = null, hasMore = false) }
        loadFeed()
    }

    fun openComposer() = _uiState.update { it.copy(isComposerOpen = true) }
    fun closeComposer() = _uiState.update { it.copy(isComposerOpen = false) }

    fun createPost(content: String, topicId: String) {
        val trimmed = content.trim()
        if (trimmed.isBlank() || topicId.isBlank()) return
        _uiState.update { it.copy(isPosting = true, error = null) }
        viewModelScope.launch {
            val result = contentClient.createPost(PostCreateRequest(content = trimmed, topicId = topicId))
            result
                .onSuccess {
                    _uiState.update { it.copy(isPosting = false, isComposerOpen = false) }
                    refresh()
                }
                .onFailure { e ->
                    _uiState.update { it.copy(isPosting = false, error = e.message ?: "Failed to publish.") }
                }
        }
    }

    private fun loadPage(cursor: String?, append: Boolean) {
        val topicId = _uiState.value.selectedTopicId
        viewModelScope.launch {
            val result = contentClient.listPosts(topicId = topicId, cursor = cursor, limit = 20)
            result
                .onSuccess { page ->
                    _uiState.update { state ->
                        val posts = if (append) state.posts + page.data else page.data
                        state.copy(
                            isLoading = false,
                            isRefreshing = false,
                            posts = posts,
                            nextCursor = page.pagination.nextCursor,
                            hasMore = page.pagination.hasMore,
                            error = if (posts.isEmpty() && !append) null else state.error
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            isRefreshing = false,
                            error = e.message ?: "Failed to load the feed."
                        )
                    }
                }
        }
    }

    private fun loadTopics() {
        viewModelScope.launch {
            contentClient.listTopics(limit = 50)
                .onSuccess { page ->
                    _uiState.update { it.copy(topics = page.data) }
                }
        }
    }
}
