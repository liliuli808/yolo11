package app.rebuild.social.core.network

import retrofit2.Response
import java.util.UUID
import javax.inject.Inject

interface ContentApiClient {
    suspend fun listPosts(
        topicId: String? = null,
        cursor: String? = null,
        limit: Int? = null
    ): Result<CursorPagePost>

    suspend fun createPost(request: PostCreateRequest): Result<Post>

    suspend fun listTopics(
        query: String? = null,
        limit: Int? = null
    ): Result<CursorPageTopic>

    suspend fun getPost(postId: String): Result<Post>

    suspend fun listComments(
        postId: String,
        cursor: String? = null,
        limit: Int? = null
    ): Result<CursorPageComment>

    suspend fun createComment(postId: String, request: CommentCreateRequest): Result<Comment>

    suspend fun createReaction(postId: String, request: ReactionCreateRequest): Result<ReactionSummary>

    suspend fun deleteReaction(postId: String, reactionType: String): Result<Unit>

    suspend fun createReport(request: ReportCreateRequest): Result<Report>

    suspend fun createBlock(request: BlockCreateRequest): Result<Block>
}

class LanternContentApiClient @Inject constructor(
    private val service: ContentApiService
) : ContentApiClient {

    override suspend fun listPosts(
        topicId: String?,
        cursor: String?,
        limit: Int?
    ): Result<CursorPagePost> = runApiCall { service.listPosts(topicId, cursor, limit) }

    override suspend fun createPost(request: PostCreateRequest): Result<Post> =
        runApiCall { service.createPost(UUID.randomUUID().toString(), request) }

    override suspend fun listTopics(query: String?, limit: Int?): Result<CursorPageTopic> =
        runApiCall { service.listTopics(query, limit) }

    override suspend fun getPost(postId: String): Result<Post> =
        runApiCall { service.getPost(postId) }

    override suspend fun listComments(
        postId: String,
        cursor: String?,
        limit: Int?
    ): Result<CursorPageComment> = runApiCall { service.listComments(postId, cursor, limit) }

    override suspend fun createComment(postId: String, request: CommentCreateRequest): Result<Comment> =
        runApiCall { service.createComment(postId, UUID.randomUUID().toString(), request) }

    override suspend fun createReaction(postId: String, request: ReactionCreateRequest): Result<ReactionSummary> =
        runApiCall { service.createReaction(postId, UUID.randomUUID().toString(), request) }

    override suspend fun deleteReaction(postId: String, reactionType: String): Result<Unit> =
        runApiCall { service.deleteReaction(postId, reactionType) }

    override suspend fun createReport(request: ReportCreateRequest): Result<Report> =
        runApiCall { service.createReport(UUID.randomUUID().toString(), request) }

    override suspend fun createBlock(request: BlockCreateRequest): Result<Block> =
        runApiCall { service.createBlock(UUID.randomUUID().toString(), request) }

    private suspend fun <T : Any> runApiCall(call: suspend () -> Response<T>): Result<T> {
        return try {
            val response = call()
            if (response.isSuccessful) {
                val body = response.body()
                if (body != null) {
                    Result.success(body)
                } else {
                    Result.failure(
                        ApiException(
                            ApiError.Malformed(
                                message = "The server returned an empty response.",
                                requestId = response.headers()["X-Request-Id"]
                            )
                        )
                    )
                }
            } else {
                Result.failure(ApiException(response.toApiError()))
            }
        } catch (e: Exception) {
            Result.failure(ApiException(e.toApiError()))
        }
    }
}
