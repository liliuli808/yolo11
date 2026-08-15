package app.rebuild.social.core.network

sealed class ApiError(
    open val code: String,
    open val message: String,
    open val requestId: String?
) {
    data class Http(
        override val code: String,
        override val message: String,
        val statusCode: Int,
        override val requestId: String?
    ) : ApiError(code, message, requestId)

    data class Network(
        override val message: String,
        override val requestId: String?
    ) : ApiError("network_error", message, requestId)

    data class Malformed(
        override val message: String,
        override val requestId: String?
    ) : ApiError("malformed_response", message, requestId)

    data class SessionExpired(
        override val requestId: String?
    ) : ApiError("session_expired", "Your session has expired. Please sign in again.", requestId)

    data class Unknown(
        override val message: String,
        override val requestId: String?
    ) : ApiError("unknown_error", message, requestId)
}

class ApiException(val apiError: ApiError) : Exception(apiError.message)

fun ApiError.userSafeMessage(): String = message

@kotlinx.serialization.Serializable
internal data class ErrorResponse(
    val code: String,
    val message: String,
    val requestId: String? = null
)

internal const val ERROR_CODE_SESSION_EXPIRED = "session_expired"
internal const val ERROR_CODE_UNAUTHORIZED = "unauthorized"
internal const val HTTP_UNAUTHORIZED = 401
internal const val HTTP_FORBIDDEN = 403
