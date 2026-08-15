package app.rebuild.social.core.network

import kotlinx.serialization.Serializable

@Serializable
data class Avatar(
    val seed: String = "",
    val color: String = "#000000"
)

@Serializable
data class Persona(
    val id: String,
    val alias: String,
    val bio: String? = null,
    val avatar: Avatar = Avatar(),
    val createdAt: String = "",
    val noteCount: Int = 0,
    val isBlocked: Boolean = false
)

@Serializable
data class Topic(
    val id: String,
    val name: String,
    val description: String = "",
    val category: String = "",
    val noteCount: Int = 0,
    val followerCount: Int = 0,
    val isFollowed: Boolean = false,
    val status: String = "active",
    val createdAt: String = ""
)

@Serializable
data class MediaAsset(
    val id: String = "",
    val url: String,
    val mimeType: String = "",
    val width: Int = 0,
    val height: Int = 0,
    val thumbnailUrl: String? = null
)

@Serializable
data class Post(
    val id: String,
    val persona: Persona,
    val topic: Topic,
    val content: String,
    val media: List<MediaAsset> = emptyList(),
    val reactionCounts: Map<String, Int> = emptyMap(),
    val userReaction: String? = null,
    val isSaved: Boolean = false,
    val replyCount: Int = 0,
    val moderationState: String = "published",
    val createdAt: String = "",
    val updatedAt: String = ""
)

@Serializable
data class PostCreateRequest(
    val content: String,
    val topicId: String,
    val personaId: String? = null,
    val mediaIds: List<String> = emptyList()
)

@Serializable
data class Pagination(
    val nextCursor: String? = null,
    val hasMore: Boolean = false
)

@Serializable
data class CursorPagePost(
    val data: List<Post> = emptyList(),
    val pagination: Pagination = Pagination()
)

@Serializable
data class CursorPageTopic(
    val data: List<Topic> = emptyList(),
    val pagination: Pagination = Pagination()
)

@Serializable
data class Comment(
    val id: String,
    val persona: Persona,
    val postId: String,
    val content: String,
    val reactionCounts: Map<String, Int> = emptyMap(),
    val userReaction: String? = null,
    val moderationState: String = "published",
    val createdAt: String = "",
    val updatedAt: String = ""
)

@Serializable
data class CommentCreateRequest(
    val content: String,
    val personaId: String? = null
)

@Serializable
data class ReactionCreateRequest(
    val type: String
)

@Serializable
data class ReactionSummary(
    val reactionCounts: Map<String, Int> = emptyMap(),
    val userReaction: String? = null
)

@Serializable
data class CursorPageComment(
    val data: List<Comment> = emptyList(),
    val pagination: Pagination = Pagination()
)

@Serializable
data class Report(
    val id: String = "",
    val targetType: String,
    val targetId: String,
    val category: String,
    val status: String = "open",
    val createdAt: String = ""
)

@Serializable
data class ReportCreateRequest(
    val targetType: String,
    val targetId: String,
    val category: String,
    val details: String? = null
)

@Serializable
data class Block(
    val id: String = "",
    val persona: Persona,
    val createdAt: String = ""
)

@Serializable
data class BlockCreateRequest(
    val personaId: String
)
