package app.rebuild.social.core.network

import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

/**
 * Content subsystem (posts, topics, feed). Mirrors the OpenAPI contract under `/v1`.
 * See `contracts/openapi/openapi.yaml`.
 */
interface ContentApiService {
    @GET("v1/posts")
    suspend fun listPosts(
        @Query("topicId") topicId: String? = null,
        @Query("cursor") cursor: String? = null,
        @Query("limit") limit: Int? = null
    ): Response<CursorPagePost>

    @POST("v1/posts")
    suspend fun createPost(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body request: PostCreateRequest
    ): Response<Post>

    @GET("v1/topics")
    suspend fun listTopics(
        @Query("q") query: String? = null,
        @Query("limit") limit: Int? = null
    ): Response<CursorPageTopic>

    @GET("v1/posts/{postId}")
    suspend fun getPost(
        @Path("postId") postId: String
    ): Response<Post>

    @GET("v1/posts/{postId}/comments")
    suspend fun listComments(
        @Path("postId") postId: String,
        @Query("cursor") cursor: String? = null,
        @Query("limit") limit: Int? = null
    ): Response<CursorPageComment>

    @POST("v1/posts/{postId}/comments")
    suspend fun createComment(
        @Path("postId") postId: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body request: CommentCreateRequest
    ): Response<Comment>

    @POST("v1/posts/{postId}/reactions")
    suspend fun createReaction(
        @Path("postId") postId: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body request: ReactionCreateRequest
    ): Response<ReactionSummary>

    @DELETE("v1/posts/{postId}/reactions/{reactionType}")
    suspend fun deleteReaction(
        @Path("postId") postId: String,
        @Path("reactionType") reactionType: String
    ): Response<Unit>

    @POST("v1/reports")
    suspend fun createReport(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body request: ReportCreateRequest
    ): Response<Report>

    @POST("v1/me/blocks")
    suspend fun createBlock(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body request: BlockCreateRequest
    ): Response<Block>
}
